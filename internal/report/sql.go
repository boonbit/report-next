// SQLite live source (track "sql"): database/sql loaders that turn live
// SQLite tables or SELECT queries into DataSource rows and DataDictionary
// skeletons.
//
// Column DataType names mirror real .frx dictionaries, e.g.
// FastReport/Demos/Reports/Sales Statistics.frx TableDataSource Column
// values: System.String, System.Int32, System.Int16, System.Decimal,
// System.Double, System.Single, System.Boolean, System.DateTime,
// System.Byte[]. Element and attribute names mirror that file too:
// TableDataSource attrs Name/ReferenceName/DataType/Enabled with Column
// children (Name, DataType, optional Calculated, Expression).
package report

import (
	"database/sql"
	"fmt"
	"strconv"
	"strings"
	"time"

	_ "modernc.org/sqlite"
)

// sqlOpenDB opens a SQLite database file through database/sql using the
// modernc.org/sqlite driver and verifies it with Ping.
func sqlOpenDB(path string) (*sql.DB, error) {
	db, err := sql.Open("sqlite", path)
	if err != nil {
		return nil, fmt.Errorf("report: sql open %q: %w", path, err)
	}
	if err := db.Ping(); err != nil {
		db.Close()
		return nil, fmt.Errorf("report: sql ping %q: %w", path, err)
	}
	return db, nil
}

// sqlLoadTable loads tableOrQuery into a DataSource named name.
// tableOrQuery is either a plain table name (loaded as SELECT * FROM it)
// or a SELECT/WITH query used verbatim. Every cell is stored as text;
// NULL becomes "".
func sqlLoadTable(db *sql.DB, tableOrQuery string, name string) (*DataSource, error) {
	return sqlLoadTableWith(db, tableOrQuery, name, sqlQuoteIdent)
}

// sqlRunQuery runs a verbatim SELECT query into a DataSource.
func sqlRunQuery(db *sql.DB, q string, name string) (*DataSource, error) {
	if db == nil {
		return nil, fmt.Errorf("report: sql: nil db")
	}
	rows, err := db.Query(q)
	if err != nil {
		return nil, fmt.Errorf("report: sql query: %w", err)
	}
	defer rows.Close()
	cols, err := rows.Columns()
	if err != nil {
		return nil, fmt.Errorf("report: sql columns: %w", err)
	}
	types, err := rows.ColumnTypes()
	if err != nil {
		return nil, fmt.Errorf("report: sql column types: %w", err)
	}
	src := &DataSource{Name: name}
	for i, c := range cols {
		decl := ""
		if i < len(types) {
			decl = types[i].DatabaseTypeName()
		}
		src.Columns = append(src.Columns, DataColumn{Name: c, DataType: sqlMapType(decl)})
	}
	for rows.Next() {
		vals := make([]any, len(cols))
		ptrs := make([]any, len(cols))
		for i := range vals {
			ptrs[i] = &vals[i]
		}
		if err := rows.Scan(ptrs...); err != nil {
			return nil, fmt.Errorf("report: sql scan: %w", err)
		}
		row := DataRow{}
		for i, c := range cols {
			row[c] = sqlCellString(vals[i])
		}
		src.Rows = append(src.Rows, row)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("report: sql rows: %w", err)
	}
	return src, nil
}

// sqlInspectSchema builds a DataDictionary skeleton (sources with columns,
// no rows) from sqlite_master plus PRAGMA table_info per table.
func sqlInspectSchema(db *sql.DB) (*DataDictionary, error) {
	if db == nil {
		return nil, fmt.Errorf("report: sql: nil db")
	}
	rows, err := db.Query(`SELECT name FROM sqlite_master WHERE type = 'table' AND name NOT LIKE 'sqlite_%' ORDER BY name`)
	if err != nil {
		return nil, fmt.Errorf("report: sql tables: %w", err)
	}
	var tables []string
	for rows.Next() {
		var name string
		if err := rows.Scan(&name); err != nil {
			rows.Close()
			return nil, fmt.Errorf("report: sql table scan: %w", err)
		}
		tables = append(tables, name)
	}
	if err := rows.Err(); err != nil {
		rows.Close()
		return nil, fmt.Errorf("report: sql tables: %w", err)
	}
	rows.Close()
	d := &DataDictionary{}
	for _, tbl := range tables {
		cols, err := sqlTableColumns(db, tbl)
		if err != nil {
			return nil, err
		}
		d.Sources = append(d.Sources, DataSource{Name: tbl, Columns: cols})
	}
	return d, nil
}

// sqlTableColumns lists one table's columns via PRAGMA table_info.
func sqlTableColumns(db *sql.DB, table string) ([]DataColumn, error) {
	rows, err := db.Query("PRAGMA table_info(" + sqlQuoteIdent(table) + ")")
	if err != nil {
		return nil, fmt.Errorf("report: sql pragma %q: %w", table, err)
	}
	defer rows.Close()
	var cols []DataColumn
	for rows.Next() {
		var cid, notnull, pk int
		var name, decl string
		var dflt any
		if err := rows.Scan(&cid, &name, &decl, &notnull, &dflt, &pk); err != nil {
			return nil, fmt.Errorf("report: sql pragma scan %q: %w", table, err)
		}
		cols = append(cols, DataColumn{Name: name, DataType: sqlMapType(decl)})
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("report: sql pragma %q: %w", table, err)
	}
	return cols, nil
}

// sqlIsSelect reports whether q is a SELECT/WITH query to run verbatim
// rather than a plain table name.
func sqlIsSelect(q string) bool {
	up := strings.ToUpper(q)
	for _, kw := range []string{"SELECT", "WITH"} {
		if strings.HasPrefix(up, kw) {
			rest := up[len(kw):]
			if rest == "" || strings.ContainsRune(" \t\n\r(", rune(rest[0])) {
				return true
			}
		}
	}
	return false
}

// sqlQuoteIdent double-quotes an SQLite identifier.
func sqlQuoteIdent(s string) string {
	return `"` + strings.ReplaceAll(s, `"`, `""`) + `"`
}

// sqlMapType maps an SQLite declared type to an .frx-style DataType name.
func sqlMapType(decl string) string {
	up := strings.ToUpper(strings.TrimSpace(decl))
	switch {
	case strings.Contains(up, "BOOL"):
		return "System.Boolean"
	case strings.Contains(up, "CHAR") || strings.Contains(up, "CLOB") || strings.Contains(up, "TEXT"):
		return "System.String"
	case strings.Contains(up, "BLOB") || strings.Contains(up, "BYTEA") || strings.Contains(up, "BINARY"):
		return "System.Byte[]"
	case strings.Contains(up, "BIT"):
		return "System.Boolean"
	case strings.Contains(up, "REAL") || strings.Contains(up, "FLOA") || strings.Contains(up, "DOUB"):
		return "System.Double"
	case strings.Contains(up, "DATE") || strings.Contains(up, "TIME"):
		return "System.DateTime"
	case strings.Contains(up, "DECIMAL") || strings.Contains(up, "NUMERIC"):
		return "System.Decimal"
	case strings.Contains(up, "INT"):
		return "System.Int32"
	default:
		return "System.String"
	}
}

// sqlCellString renders one database/sql cell value as text.
func sqlCellString(v any) string {
	switch t := v.(type) {
	case nil:
		return ""
	case string:
		return t
	case []byte:
		return string(t)
	case bool:
		return strconv.FormatBool(t)
	case int:
		return strconv.FormatInt(int64(t), 10)
	case int32:
		return strconv.FormatInt(int64(t), 10)
	case int64:
		return strconv.FormatInt(t, 10)
	case float32:
		return strconv.FormatFloat(float64(t), 'g', -1, 32)
	case float64:
		return strconv.FormatFloat(t, 'g', -1, 64)
	case time.Time:
		return t.Format(time.RFC3339Nano)
	default:
		return fmt.Sprintf("%v", t)
	}
}
