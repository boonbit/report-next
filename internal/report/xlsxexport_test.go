package report

import (
	"archive/zip"
	"bytes"
	"io"
	"net/http"
	"net/http/httptest"
	"os"
	"strings"
	"testing"
)

func loadCustomersDict(t *testing.T) *DataDictionary {
	t.Helper()
	f, err := os.Open("testdata/customers.csv")
	if err != nil {
		t.Fatal(err)
	}
	src, err := DataLoadCSV(f, "Customers")
	f.Close()
	if err != nil {
		t.Fatal(err)
	}
	return &DataDictionary{Sources: []DataSource{*src}}
}

func unzipPart(t *testing.T, data []byte, name string) string {
	t.Helper()
	zr, err := zip.NewReader(bytes.NewReader(data), int64(len(data)))
	if err != nil {
		t.Fatal(err)
	}
	f, err := zr.Open(name)
	if err != nil {
		t.Fatal(err)
	}
	defer f.Close()
	body, err := io.ReadAll(f)
	if err != nil {
		t.Fatal(err)
	}
	return string(body)
}

func TestXLSXExportSource(t *testing.T) {
	out, err := xlsxExportSource(loadCustomersDict(t), "Customers")
	if err != nil {
		t.Fatal(err)
	}
	sheet := unzipPart(t, out, "xl/worksheets/sheet1.xml")
	for _, want := range []string{"CustomerID", "ALFKI", "<v>6</v>"} {
		if !strings.Contains(sheet, want) {
			t.Errorf("xlsx sheet missing %q", want)
		}
	}
	if _, err := xlsxExportSource(loadCustomersDict(t), "Nope"); err == nil {
		t.Error("unknown source accepted")
	}
}

func TestSrvXLSXRoute(t *testing.T) {
	mux := SrvMux(loadCustomersDict(t))
	res := httptest.NewRecorder()
	mux.ServeHTTP(res, httptest.NewRequest(http.MethodPost, "/api/xlsx?source=Customers", nil))
	if res.Code != http.StatusOK {
		t.Fatalf("status = %d", res.Code)
	}
	if ct := res.Header().Get("Content-Type"); ct != "application/vnd.openxmlformats-officedocument.spreadsheetml.sheet" {
		t.Fatalf("content-type = %q", ct)
	}
	sheet := unzipPart(t, res.Body.Bytes(), "xl/worksheets/sheet1.xml")
	if !strings.Contains(sheet, "ANATR") {
		t.Fatal("served xlsx missing row")
	}
	bad := httptest.NewRecorder()
	mux.ServeHTTP(bad, httptest.NewRequest(http.MethodPost, "/api/xlsx", nil))
	if bad.Code != http.StatusBadRequest {
		t.Fatalf("missing source status = %d", bad.Code)
	}
}
