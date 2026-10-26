package report

import (
	"database/sql"
	"fmt"
	"strings"

	_ "github.com/jackc/pgx/v5/stdlib"
	_ "github.com/microsoft/go-mssqldb"
)

// PostgreSQL and SQL Server live sources. Both drivers speak database/sql,
// so loading reuses sqlLoadTable; only connecting and schema inspection are
// dialect-specific (information_schema instead of sqlite_master/PRAGMA).
// sqlMapType already covers the shared type names; BYTEA/BIT are added here
// via sqlMapTypeAny.

// sqlOpenPostgres opens a PostgreSQL database (pgx stdlib, "pgx" driver).
// connStr is a postgres:// URL or key=value string.
func sqlOpenPostgres(connStr string) (*sql.DB, error) {
	db, err := sql.Open("pgx", connStr)
	if err != nil {
		return nil, fmt.Errorf("report: sql open postgres: %w", err)
	}
	if err := db.Ping(); err != nil {
		db.Close()
		return nil, fmt.Errorf("report: sql ping postgres: %w", err)
	}
	return db, nil
}

// sqlOpenMSSQL opens a SQL Server database ("sqlserver" driver).
// connStr is a sqlserver:// URL or ADO-style string.
func sqlOpenMSSQL(connStr string) (*sql.DB, error) {
	db, err := sql.Open("sqlserver", connStr)
	if err != nil {
		return nil, fmt.Errorf("report: sql open mssql: %w", err)
	}
	if err := db.Ping(); err != nil {
		db.Close()
		return nil, fmt.Errorf("report: sql ping mssql: %w", err)
	}
	return db, nil
}

// sqlLoadTableWith is sqlLoadTable with a caller-chosen identifier quoter
// (double quotes for SQLite/PostgreSQL, brackets for SQL Server).
func sqlLoadTableWith(db *sql.DB, tableOrQuery, name string, quote func(string) string) (*DataSource, error) {
	if db == nil {
		return nil, fmt.Errorf("report: sql: nil db")
	}
	q := strings.TrimSpace(tableOrQuery)
	if q == "" {
		return nil, fmt.Errorf("report: sql: empty table or query")
	}
	if !sqlIsSelect(q) {
		q = "SELECT * FROM " + quote(strings.TrimSpace(tableOrQuery))
	}
	return sqlRunQuery(db, q, name)
}

// sqlQuoteIdentMSSQL brackets a SQL Server identifier.
func sqlQuoteIdentMSSQL(s string) string {
	out := ""
	for _, r := range s {
		if r == ']' {
			out += "]]"
		} else {
			out += string(r)
		}
	}
	return "[" + out + "]"
}

// sqlInspectPostgres builds a dictionary skeleton from information_schema.
func sqlInspectPostgres(db *sql.DB) (*DataDictionary, error) {
	return sqlInspectInfoSchema(db,
		`SELECT table_name FROM information_schema.tables WHERE table_schema NOT IN ('pg_catalog','information_schema') AND table_type = 'BASE TABLE' ORDER BY table_name`,
		`SELECT column_name, data_type FROM information_schema.columns WHERE table_name = $1 ORDER BY ordinal_position`)
}

// sqlInspectMSSQL builds a dictionary skeleton from INFORMATION_SCHEMA.
func sqlInspectMSSQL(db *sql.DB) (*DataDictionary, error) {
	return sqlInspectInfoSchema(db,
		`SELECT TABLE_NAME FROM INFORMATION_SCHEMA.TABLES WHERE TABLE_TYPE = 'BASE TABLE' AND TABLE_SCHEMA NOT IN ('sys','INFORMATION_SCHEMA') ORDER BY TABLE_NAME`,
		`SELECT COLUMN_NAME, DATA_TYPE FROM INFORMATION_SCHEMA.COLUMNS WHERE TABLE_NAME = @p1 ORDER BY ORDINAL_POSITION`)
}

func sqlInspectInfoSchema(db *sql.DB, tablesQ, colsQ string) (*DataDictionary, error) {
	if db == nil {
		return nil, fmt.Errorf("report: sql: nil db")
	}
	rows, err := db.Query(tablesQ)
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
		crows, err := db.Query(colsQ, tbl)
		if err != nil {
			return nil, fmt.Errorf("report: sql columns %q: %w", tbl, err)
		}
		var cols []DataColumn
		for crows.Next() {
			var name, decl string
			if err := crows.Scan(&name, &decl); err != nil {
				crows.Close()
				return nil, fmt.Errorf("report: sql column scan %q: %w", tbl, err)
			}
			cols = append(cols, DataColumn{Name: name, DataType: sqlMapType(decl)})
		}
		if err := crows.Err(); err != nil {
			crows.Close()
			return nil, fmt.Errorf("report: sql columns %q: %w", tbl, err)
		}
		crows.Close()
		d.Sources = append(d.Sources, DataSource{Name: tbl, Columns: cols})
	}
	return d, nil
}
