package report

import (
	"bytes"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

func TestPDFExportHello(t *testing.T) {
	rep := loadHello(t)
	out, skipped, err := pdfExportReport(rep)
	if err != nil {
		t.Fatal(err)
	}
	if skipped != 0 {
		t.Fatalf("skipped = %d", skipped)
	}
	if !bytes.HasPrefix(out, []byte("%PDF-1.4")) {
		t.Fatal("missing pdf header")
	}
	if !bytes.Contains(out, []byte("The report goes here")) {
		t.Fatal("title text missing from pdf")
	}
}

func TestPDFExportSkipsNonASCII(t *testing.T) {
	rep := loadHello(t)
	rep.Pages[0].Bands[0].Objects = append(rep.Pages[0].Bands[0].Objects,
		Object{Kind: "TextObject", Name: "Thai", Attrs: []Attr{{"Text", "สวัสดี"}}})
	_, skipped, err := pdfExportReport(rep)
	if err != nil {
		t.Fatal(err)
	}
	if skipped != 1 {
		t.Fatalf("skipped = %d, want 1", skipped)
	}
}

func TestSrvPDFRoute(t *testing.T) {
	raw, err := MarshalJSON(loadHello(t))
	if err != nil {
		t.Fatal(err)
	}
	res := httptest.NewRecorder()
	SrvMux(&DataDictionary{}).ServeHTTP(res,
		httptest.NewRequest(http.MethodPost, "/api/pdf", bytes.NewReader(raw)))
	if res.Code != http.StatusOK {
		t.Fatalf("status = %d", res.Code)
	}
	if ct := res.Header().Get("Content-Type"); ct != "application/pdf" {
		t.Fatalf("content-type = %q", ct)
	}
	if !bytes.HasPrefix(res.Body.Bytes(), []byte("%PDF-1.4")) {
		t.Fatal("body is not a pdf")
	}
	if !strings.Contains(res.Body.String(), "The report goes here") {
		t.Fatal("title text missing from served pdf")
	}
}
