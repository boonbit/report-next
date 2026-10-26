package report

import (
	"database/sql"
	"path/filepath"
	"testing"
)

// sqlTestDB creates a throwaway SQLite file under t.TempDir() (never the
// repo testdata dir) with a customers table holding two rows, one with a
// NULL note.
func sqlTestDB(t *testing.T) *sql.DB {
	t.Helper()
	db, err := sqlOpenDB(filepath.Join(t.TempDir(), "t.db"))
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { db.Close() })
	_, err = db.Exec(`CREATE TABLE customers (id INTEGER PRIMARY KEY, name TEXT, price REAL, note TEXT)`)
	if err != nil {
		t.Fatal(err)
	}
	if _, err = db.Exec(`INSERT INTO customers (id, name, price, note) VALUES (1, 'Alice', 19.99, NULL), (2, 'Bob', 3.5, 'hi')`); err != nil {
		t.Fatal(err)
	}
	return db
}

func sqlColumnTypeOf(s *DataSource, col string) string {
	for _, c := range s.Columns {
		if c.Name == col {
			return c.DataType
		}
	}
	return ""
}

func TestSQLLoadTableValues(t *testing.T) {
	db := sqlTestDB(t)
	src, err := sqlLoadTable(db, "customers", "Customers")
	if err != nil {
		t.Fatal(err)
	}
	if src.Name != "Customers" {
		t.Errorf("name = %q", src.Name)
	}
	if len(src.Rows) != 2 {
		t.Fatalf("rows = %d, want 2", len(src.Rows))
	}
	if got := src.Rows[0]["id"]; got != "1" {
		t.Errorf("id = %q, want 1", got)
	}
	if got := src.Rows[0]["name"]; got != "Alice" {
		t.Errorf("name = %q, want Alice", got)
	}
	if got := src.Rows[0]["price"]; got != "19.99" {
		t.Errorf("price = %q, want 19.99", got)
	}
	if got := src.Rows[0]["note"]; got != "" {
		t.Errorf("NULL note = %q, want empty", got)
	}
	if got := src.Rows[1]["note"]; got != "hi" {
		t.Errorf("note = %q, want hi", got)
	}
	if got := sqlColumnTypeOf(src, "id"); got != "System.Int32" {
		t.Errorf("id type = %q", got)
	}
	if got := sqlColumnTypeOf(src, "name"); got != "System.String" {
		t.Errorf("name type = %q", got)
	}
	if got := sqlColumnTypeOf(src, "price"); got != "System.Double" {
		t.Errorf("price type = %q", got)
	}
}

func TestSQLLoadTableQuery(t *testing.T) {
	db := sqlTestDB(t)
	src, err := sqlLoadTable(db, "SELECT name FROM customers ORDER BY id", "Names")
	if err != nil {
		t.Fatal(err)
	}
	if len(src.Columns) != 1 || src.Columns[0].Name != "name" {
		t.Fatalf("columns = %+v", src.Columns)
	}
	if len(src.Rows) != 2 || src.Rows[0]["name"] != "Alice" || src.Rows[1]["name"] != "Bob" {
		t.Fatalf("rows = %+v", src.Rows)
	}
}

func TestSQLLoadTableBadQuery(t *testing.T) {
	db := sqlTestDB(t)
	for _, bad := range []string{"SELECT * FROM missing_table", "missing_table", ""} {
		if _, err := sqlLoadTable(db, bad, "X"); err == nil {
			t.Errorf("query %q: want error, got nil", bad)
		}
	}
	if _, err := sqlLoadTable(nil, "customers", "X"); err == nil {
		t.Error("nil db: want error, got nil")
	}
}

func TestSQLInspectSchema(t *testing.T) {
	db := sqlTestDB(t)
	if _, err := db.Exec(`CREATE TABLE orders (oid INTEGER, cid INTEGER, placed TEXT)`); err != nil {
		t.Fatal(err)
	}
	d, err := sqlInspectSchema(db)
	if err != nil {
		t.Fatal(err)
	}
	if len(d.Sources) != 2 {
		t.Fatalf("sources = %d, want 2", len(d.Sources))
	}
	if d.Sources[0].Name != "customers" || d.Sources[1].Name != "orders" {
		t.Fatalf("tables = %q, %q", d.Sources[0].Name, d.Sources[1].Name)
	}
	var names []string
	for _, c := range d.Sources[0].Columns {
		names = append(names, c.Name+":"+c.DataType)
	}
	want := []string{"id:System.Int32", "name:System.String", "price:System.Double", "note:System.String"}
	if len(names) != len(want) {
		t.Fatalf("columns = %v", names)
	}
	for i := range want {
		if names[i] != want[i] {
			t.Errorf("column %d = %q, want %q", i, names[i], want[i])
		}
	}
	for _, s := range d.Sources {
		if len(s.Rows) != 0 {
			t.Errorf("table %q skeleton has %d rows", s.Name, len(s.Rows))
		}
	}
	if _, err := sqlInspectSchema(nil); err == nil {
		t.Error("nil db: want error, got nil")
	}
}

func TestSQLOpenDBBadPath(t *testing.T) {
	if _, err := sqlOpenDB(filepath.Join(t.TempDir(), "no-such-dir", "t.db")); err == nil {
		t.Error("bad path: want error, got nil")
	}
}
