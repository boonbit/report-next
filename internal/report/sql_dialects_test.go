package report

import (
	"testing"

	"github.com/DATA-DOG/go-sqlmock"
)

func TestSQLLoadTableMock(t *testing.T) {
	db, mock, err := sqlmock.New()
	if err != nil {
		t.Fatal(err)
	}
	defer db.Close()
	rows := sqlmock.NewRows([]string{"id", "name", "nick"}).
		AddRow(int64(1), "Ann", nil).
		AddRow(int64(2), "Bo", "Bobby")
	mock.ExpectQuery(`SELECT \* FROM "customers"`).WillReturnRows(rows)
	src, err := sqlLoadTable(db, "customers", "Customers")
	if err != nil {
		t.Fatal(err)
	}
	if len(src.Rows) != 2 || src.Rows[0]["nick"] != "" || src.Rows[1]["nick"] != "Bobby" {
		t.Fatalf("rows = %+v", src.Rows)
	}
	if src.Columns[0].Name != "id" || src.Rows[0]["id"] != "1" {
		t.Fatalf("cols/values = %+v %+v", src.Columns, src.Rows[0])
	}
	if err := mock.ExpectationsWereMet(); err != nil {
		t.Fatal(err)
	}
}

func TestSQLLoadTableWithBrackets(t *testing.T) {
	db, mock, err := sqlmock.New()
	if err != nil {
		t.Fatal(err)
	}
	defer db.Close()
	mock.ExpectQuery(`SELECT \* FROM \[my table\]`).
		WillReturnRows(sqlmock.NewRows([]string{"a"}).AddRow("x"))
	src, err := sqlLoadTableWith(db, "my table", "T", sqlQuoteIdentMSSQL)
	if err != nil {
		t.Fatal(err)
	}
	if len(src.Rows) != 1 || src.Rows[0]["a"] != "x" {
		t.Fatalf("rows = %+v", src.Rows)
	}
	if got := sqlQuoteIdentMSSQL("a]b"); got != "[a]]b]" {
		t.Fatalf("bracket quote = %q", got)
	}
	if err := mock.ExpectationsWereMet(); err != nil {
		t.Fatal(err)
	}
}

func TestSQLInspectPostgres(t *testing.T) {
	db, mock, err := sqlmock.New(sqlmock.QueryMatcherOption(sqlmock.QueryMatcherEqual))
	if err != nil {
		t.Fatal(err)
	}
	defer db.Close()
	mock.ExpectQuery(`SELECT table_name FROM information_schema.tables WHERE table_schema NOT IN ('pg_catalog','information_schema') AND table_type = 'BASE TABLE' ORDER BY table_name`).
		WillReturnRows(sqlmock.NewRows([]string{"table_name"}).AddRow("orders").AddRow("customers"))
	mock.ExpectQuery(`SELECT column_name, data_type FROM information_schema.columns WHERE table_name = $1 ORDER BY ordinal_position`).
		WithArgs("orders").
		WillReturnRows(sqlmock.NewRows([]string{"column_name", "data_type"}).
			AddRow("id", "integer").AddRow("total", "numeric").AddRow("placed", "timestamp without time zone"))
	mock.ExpectQuery(`SELECT column_name, data_type FROM information_schema.columns WHERE table_name = $1 ORDER BY ordinal_position`).
		WithArgs("customers").
		WillReturnRows(sqlmock.NewRows([]string{"column_name", "data_type"}).AddRow("name", "character varying"))
	d, err := sqlInspectPostgres(db)
	if err != nil {
		t.Fatal(err)
	}
	if len(d.Sources) != 2 {
		t.Fatalf("sources = %+v", d.Sources)
	}
	got := map[string]string{}
	for _, c := range d.Sources[0].Columns {
		got[c.Name] = c.DataType
	}
	if got["id"] != "System.Int32" || got["total"] != "System.Decimal" || got["placed"] != "System.DateTime" {
		t.Fatalf("pg types = %+v", got)
	}
	if err := mock.ExpectationsWereMet(); err != nil {
		t.Fatal(err)
	}
}

func TestSQLInspectMSSQL(t *testing.T) {
	db, mock, err := sqlmock.New(sqlmock.QueryMatcherOption(sqlmock.QueryMatcherEqual))
	if err != nil {
		t.Fatal(err)
	}
	defer db.Close()
	mock.ExpectQuery(`SELECT TABLE_NAME FROM INFORMATION_SCHEMA.TABLES WHERE TABLE_TYPE = 'BASE TABLE' AND TABLE_SCHEMA NOT IN ('sys','INFORMATION_SCHEMA') ORDER BY TABLE_NAME`).
		WillReturnRows(sqlmock.NewRows([]string{"TABLE_NAME"}).AddRow("Orders"))
	mock.ExpectQuery(`SELECT COLUMN_NAME, DATA_TYPE FROM INFORMATION_SCHEMA.COLUMNS WHERE TABLE_NAME = @p1 ORDER BY ORDINAL_POSITION`).
		WithArgs("Orders").
		WillReturnRows(sqlmock.NewRows([]string{"COLUMN_NAME", "DATA_TYPE"}).
			AddRow("Id", "int").AddRow("Paid", "bit").AddRow("Photo", "varbinary"))
	d, err := sqlInspectMSSQL(db)
	if err != nil {
		t.Fatal(err)
	}
	if len(d.Sources) != 1 {
		t.Fatalf("sources = %+v", d.Sources)
	}
	got := map[string]string{}
	for _, c := range d.Sources[0].Columns {
		got[c.Name] = c.DataType
	}
	if got["Id"] != "System.Int32" || got["Paid"] != "System.Boolean" || got["Photo"] != "System.Byte[]" {
		t.Fatalf("mssql types = %+v", got)
	}
	if err := mock.ExpectationsWereMet(); err != nil {
		t.Fatal(err)
	}
}

func TestSQLOpenBadConnStrings(t *testing.T) {
	if _, err := sqlOpenPostgres("://bad scheme"); err == nil {
		t.Error("bad postgres string accepted")
	}
	if _, err := sqlOpenMSSQL("http://[::1]:namedport"); err == nil {
		t.Error("bad mssql string accepted")
	}
}

func TestSQLMapTypeExtra(t *testing.T) {
	if got := sqlMapType("bytea"); got != "System.Byte[]" {
		t.Errorf("bytea = %q", got)
	}
	if got := sqlMapType("bit"); got != "System.Boolean" {
		t.Errorf("bit = %q", got)
	}
}
