package report

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"path/filepath"
	"strconv"
	"strings"
	"testing"
)

// schemaPost posts one JSON body to /api/schema on the test mux.
func schemaPost(t *testing.T, body string) *httptest.ResponseRecorder {
	t.Helper()
	res := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodPost, "/api/schema", strings.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	srvTestMux(t).ServeHTTP(res, req)
	return res
}

func schemaDecode(t *testing.T, res *httptest.ResponseRecorder) schemaResponseJSON {
	t.Helper()
	var out schemaResponseJSON
	if err := json.Unmarshal(res.Body.Bytes(), &out); err != nil {
		t.Fatal(err)
	}
	return out
}

func TestSchemaCSVColumns(t *testing.T) {
	res := schemaPost(t, `{"kind":"csv","content":"name,age\nAlice,30\nBob,3\n"}`)
	if res.Code != http.StatusOK {
		t.Fatalf("status = %d, body = %q", res.Code, res.Body.String())
	}
	out := schemaDecode(t, res)
	if len(out.Tables) != 1 {
		t.Fatalf("tables = %d, want 1", len(out.Tables))
	}
	var names []string
	for _, c := range out.Tables[0].Columns {
		names = append(names, c.Name)
	}
	if strings.Join(names, ",") != "name,age" {
		t.Fatalf("columns = %v, want [name age]", names)
	}
}

func TestSchemaBadKind(t *testing.T) {
	res := schemaPost(t, `{"kind":"bogus","dsn":"","content":""}`)
	if res.Code != http.StatusBadRequest {
		t.Fatalf("status = %d, want 400", res.Code)
	}
}

func TestSchemaSQLiteMemory(t *testing.T) {
	dict, err := schemaInspect("sqlite", ":memory:", "")
	if err != nil {
		t.Fatal(err)
	}
	if len(dict.Sources) != 0 {
		t.Fatalf("sources = %d, want 0 for empty memory db", len(dict.Sources))
	}
}

func TestSchemaSQLiteEmptyDSN(t *testing.T) {
	res := schemaPost(t, `{"kind":"sqlite","dsn":"","content":""}`)
	if res.Code != http.StatusBadRequest {
		t.Fatalf("status = %d, want 400", res.Code)
	}
}

func TestSchemaSQLiteFile(t *testing.T) {
	path := filepath.Join(t.TempDir(), "s.db")
	db, err := sqlOpenDB(path)
	if err != nil {
		t.Fatal(err)
	}
	if _, err = db.Exec(`CREATE TABLE customers (id INTEGER PRIMARY KEY, name TEXT)`); err != nil {
		t.Fatal(err)
	}
	db.Close()
	res := schemaPost(t, `{"kind":"sqlite","dsn":`+strconv.Quote(path)+`,"content":""}`)
	if res.Code != http.StatusOK {
		t.Fatalf("status = %d, body = %q", res.Code, res.Body.String())
	}
	out := schemaDecode(t, res)
	if len(out.Tables) != 1 || out.Tables[0].Name != "customers" {
		t.Fatalf("tables = %+v, want [customers]", out.Tables)
	}
}
