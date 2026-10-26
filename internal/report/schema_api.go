package report

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"
)

// Schema inspection endpoint (track U3): POST /api/schema with JSON
// {kind, dsn, content} returns {tables:[{name, columns:[{name, type}]}]}.
// Database kinds open a live DB and read its schema; file kinds parse the
// posted content with the existing DataLoadCSV/JSON/XML loaders.

type schemaColumnJSON struct {
	Name string `json:"name"`
	Type string `json:"type"`
}

type schemaTableJSON struct {
	Name    string             `json:"name"`
	Columns []schemaColumnJSON `json:"columns"`
}

type schemaRequestJSON struct {
	Kind    string `json:"kind"`
	DSN     string `json:"dsn"`
	Content string `json:"content"`
}

type schemaResponseJSON struct {
	Tables []schemaTableJSON `json:"tables"`
}

// schemaInspect returns a dictionary skeleton for one data source without
// touching any report. kind is one of sqlite|postgres|mssql|csv|json|xml;
// dsn carries the DB path or connection string, content the file bytes.
func schemaInspect(kind, dsn, content string) (*DataDictionary, error) {
	switch strings.ToLower(strings.TrimSpace(kind)) {
	case "sqlite":
		if strings.TrimSpace(dsn) == "" {
			return nil, errSchema("sqlite: missing dsn")
		}
		db, err := sqlOpenDB(dsn)
		if err != nil {
			return nil, err
		}
		defer db.Close()
		return sqlInspectSchema(db)
	case "postgres":
		if strings.TrimSpace(dsn) == "" {
			return nil, errSchema("postgres: missing dsn")
		}
		db, err := sqlOpenPostgres(dsn)
		if err != nil {
			return nil, err
		}
		defer db.Close()
		return sqlInspectPostgres(db)
	case "mssql":
		if strings.TrimSpace(dsn) == "" {
			return nil, errSchema("mssql: missing dsn")
		}
		db, err := sqlOpenMSSQL(dsn)
		if err != nil {
			return nil, err
		}
		defer db.Close()
		return sqlInspectMSSQL(db)
	case "csv":
		src, err := DataLoadCSV(strings.NewReader(content), "data")
		if err != nil {
			return nil, err
		}
		return &DataDictionary{Sources: []DataSource{*src}}, nil
	case "json":
		src, err := DataLoadJSON(strings.NewReader(content), "data")
		if err != nil {
			return nil, err
		}
		return &DataDictionary{Sources: []DataSource{*src}}, nil
	case "xml":
		src, err := DataLoadXML(strings.NewReader(content), "data")
		if err != nil {
			return nil, err
		}
		return &DataDictionary{Sources: []DataSource{*src}}, nil
	default:
		return nil, errSchema("unknown kind %q: want sqlite|postgres|mssql|csv|json|xml", kind)
	}
}

// schemaError is a plain inspection failure (reported as HTTP 400).
type schemaError string

func (e schemaError) Error() string { return "report: schema: " + string(e) }

func errSchema(format string, args ...any) error {
	if len(args) == 0 {
		return schemaError(format)
	}
	return schemaError(fmt.Sprintf(format, args...))
}

// srvHandleSchema serves POST /api/schema. Errors are plain-text HTTP 400.
func srvHandleSchema(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}
	raw, err := io.ReadAll(io.LimitReader(r.Body, 32<<20))
	if err != nil {
		http.Error(w, "bad body", http.StatusBadRequest)
		return
	}
	var req schemaRequestJSON
	if err := json.Unmarshal(raw, &req); err != nil {
		http.Error(w, "parse: "+err.Error(), http.StatusBadRequest)
		return
	}
	dict, err := schemaInspect(req.Kind, req.DSN, req.Content)
	if err != nil {
		http.Error(w, "schema: "+err.Error(), http.StatusBadRequest)
		return
	}
	res := schemaResponseJSON{Tables: []schemaTableJSON{}}
	for _, s := range dict.Sources {
		t := schemaTableJSON{Name: s.Name, Columns: []schemaColumnJSON{}}
		for _, c := range s.Columns {
			t.Columns = append(t.Columns, schemaColumnJSON{Name: c.Name, Type: c.DataType})
		}
		res.Tables = append(res.Tables, t)
	}
	out, err := json.Marshal(res)
	if err != nil {
		http.Error(w, "encode: "+err.Error(), http.StatusInternalServerError)
		return
	}
	w.Header().Set("Content-Type", "application/json")
	_, _ = w.Write(out)
}
