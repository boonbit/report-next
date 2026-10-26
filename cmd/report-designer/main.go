package main

// Minimal designer server: serves the report package's SrvMux on :8080.
// Run from the report-next module root so ./testdata-style seed data
// resolves: `go run ./cmd/report-designer` (cwd = report-next/).
// The demo dictionary is seeded from the package testdata copy of
// customers.csv when present; otherwise the server starts with an
// empty dictionary and the API still works.

import (
	"log"
	"net/http"
	"os"

	"report-next/internal/report"
)

func main() {
	dict := &report.DataDictionary{}
	seeded := false
	for _, path := range []string{
		"internal/report/testdata/customers.csv",
		"testdata/customers.csv",
	} {
		if f, err := os.Open(path); err == nil {
			src, err := report.DataLoadCSV(f, "Customers")
			f.Close()
			if err == nil {
				dict.Sources = append(dict.Sources, *src)
				seeded = true
				break
			}
		}
	}
	log.Printf("report-designer: listening on :8080 (seeded=%v)", seeded)
	log.Fatal(http.ListenAndServe(":8080", report.SrvMux(dict)))
}
