package report

import (
	"fmt"
	"os"
	"strconv"
	"strings"
	"testing"
	"time"
)

// Live-source verification for PostgreSQL and SQL Server. These tests run
// only when a DSN is provided:
//
//	REPORT_PG_DSN    e.g. postgres://user:pass@localhost:5432/reportnext?sslmode=disable
//	REPORT_MSSQL_DSN e.g. sqlserver://sa:Pass@localhost:1433?database=master
//
// Each test creates a throwaway table, round-trips rows (including NULL)
// through sqlLoadTableWith, checks the dictionary skeleton from
// sqlInspect{Postgres,MSSQL}, then drops the table. Without the env vars
// they skip, keeping the default suite green and offline.
func TestSQLLivePostgres(t *testing.T) {
	dsn := os.Getenv("REPORT_PG_DSN")
	if dsn == "" {
		t.Skip("REPORT_PG_DSN not set: live PostgreSQL verification skipped")
	}
	db, err := sqlOpenPostgres(dsn)
	if err != nil {
		t.Fatal(err)
	}
	defer db.Close()
	tbl := fmt.Sprintf("rn_live_pg_%d", time.Now().UnixNano()%1000000)
	t.Cleanup(func() { db.Exec("DROP TABLE IF EXISTS " + sqlQuoteIdent(tbl)) })
	if _, err := db.Exec(`CREATE TABLE ` + sqlQuoteIdent(tbl) + ` (
		id integer PRIMARY KEY, name varchar(50),
		price numeric(10,2), paid boolean, note text)`); err != nil {
		t.Fatal(err)
	}
	if _, err := db.Exec(`INSERT INTO ` + sqlQuoteIdent(tbl) + ` (id, name, price, paid, note) VALUES
		(1, 'Alice', 19.99, true, NULL), (2, 'Bob', 3.5, false, 'hi')`); err != nil {
		t.Fatal(err)
	}
	src, err := sqlLoadTableWith(db, tbl, "Live", sqlQuoteIdent)
	if err != nil {
		t.Fatal(err)
	}
	if len(src.Rows) != 2 || src.Rows[0]["name"] != "Alice" || src.Rows[0]["note"] != "" {
		t.Fatalf("rows = %+v", src.Rows)
	}
	if src.Rows[0]["paid"] != "true" || src.Rows[1]["paid"] != "false" {
		t.Fatalf("paid = %q, %q", src.Rows[0]["paid"], src.Rows[1]["paid"])
	}
	if f, err := strconv.ParseFloat(src.Rows[0]["price"], 64); err != nil || f != 19.99 {
		t.Fatalf("price = %q", src.Rows[0]["price"])
	}
	got := map[string]string{}
	for _, c := range src.Columns {
		got[c.Name] = c.DataType
	}
	for col, want := range map[string]string{
		"id": "System.Int32", "name": "System.String",
		"price": "System.Decimal", "paid": "System.Boolean", "note": "System.String",
	} {
		if got[col] != want {
			t.Errorf("column %s type = %q, want %q", col, got[col], want)
		}
	}
	d, err := sqlInspectPostgres(db)
	if err != nil {
		t.Fatal(err)
	}
	found := false
	for _, s := range d.Sources {
		if s.Name == tbl && len(s.Columns) == 5 {
			found = true
		}
	}
	if !found {
		t.Fatalf("inspect missed %q: %+v", tbl, d.Sources)
	}
}

func TestSQLLiveMSSQL(t *testing.T) {
	dsn := os.Getenv("REPORT_MSSQL_DSN")
	if dsn == "" {
		t.Skip("REPORT_MSSQL_DSN not set: live SQL Server verification skipped")
	}
	db, err := sqlOpenMSSQL(dsn)
	if err != nil {
		t.Fatal(err)
	}
	defer db.Close()
	tbl := fmt.Sprintf("rn_live_ms_%d", time.Now().UnixNano()%1000000)
	t.Cleanup(func() {
		db.Exec("DROP TABLE IF EXISTS " + sqlQuoteIdentMSSQL(tbl))
	})
	if _, err := db.Exec(`CREATE TABLE ` + sqlQuoteIdentMSSQL(tbl) + ` (
		id int PRIMARY KEY, name nvarchar(50),
		price decimal(10,2), paid bit, placed datetime2, note nvarchar(200) NULL)`); err != nil {
		t.Fatal(err)
	}
	if _, err := db.Exec(`INSERT INTO ` + sqlQuoteIdentMSSQL(tbl) + ` (id, name, price, paid, placed, note) VALUES
		(1, 'Alice', 19.99, 1, '2024-01-02T03:04:05', NULL),
		(2, 'Bob', 3.5, 0, '2024-06-07T08:09:10', 'hi')`); err != nil {
		t.Fatal(err)
	}
	src, err := sqlLoadTableWith(db, tbl, "Live", sqlQuoteIdentMSSQL)
	if err != nil {
		t.Fatal(err)
	}
	if len(src.Rows) != 2 || src.Rows[0]["name"] != "Alice" || src.Rows[0]["note"] != "" {
		t.Fatalf("rows = %+v", src.Rows)
	}
	if src.Rows[0]["paid"] != "true" || src.Rows[1]["paid"] != "false" {
		t.Fatalf("paid = %q, %q", src.Rows[0]["paid"], src.Rows[1]["paid"])
	}
	if f, err := strconv.ParseFloat(src.Rows[0]["price"], 64); err != nil || f != 19.99 {
		t.Fatalf("price = %q", src.Rows[0]["price"])
	}
	if !strings.HasPrefix(src.Rows[0]["placed"], "2024-01-02T03:04:05") {
		t.Fatalf("placed = %q", src.Rows[0]["placed"])
	}
	got := map[string]string{}
	for _, c := range src.Columns {
		got[c.Name] = c.DataType
	}
	for col, want := range map[string]string{
		"id": "System.Int32", "name": "System.String",
		"price": "System.Decimal", "paid": "System.Boolean",
		"placed": "System.DateTime", "note": "System.String",
	} {
		if got[col] != want {
			t.Errorf("column %s type = %q, want %q", col, got[col], want)
		}
	}
	d, err := sqlInspectMSSQL(db)
	if err != nil {
		t.Fatal(err)
	}
	found := false
	for _, s := range d.Sources {
		if strings.EqualFold(s.Name, tbl) && len(s.Columns) == 6 {
			found = true
		}
	}
	if !found {
		t.Fatalf("inspect missed %q", tbl)
	}
}
