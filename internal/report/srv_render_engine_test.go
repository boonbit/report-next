package report

import (
	"bytes"
	"net/http"
	"net/http/httptest"
	"os"
	"strings"
	"testing"
)

// TestSrvRenderUsesEngine posts a data-bound IR report and requires engine
// output: substituted field values, not the literal [CompanyName] text the
// RenderBand fallback would emit.
func TestSrvRenderUsesEngine(t *testing.T) {
	f, err := os.Open("testdata/customers.csv")
	if err != nil {
		t.Fatal(err)
	}
	src, err := DataLoadCSV(f, "Customers")
	f.Close()
	if err != nil {
		t.Fatal(err)
	}
	dict := &DataDictionary{Sources: []DataSource{*src}}

	rep := exprNewReport("Engine check")
	pi := exprAddPage(rep, "Page1")
	bi := exprAddBand(rep, pi, "DataBand", "Data1")
	rep.Pages[pi].Bands[bi].Attrs = append(rep.Pages[pi].Bands[bi].Attrs,
		Attr{Name: "DataSource", Value: "Customers"})
	exprAddText(rep, pi, bi, "T1", "[CompanyName]")
	body, err := MarshalJSON(*rep)
	if err != nil {
		t.Fatal(err)
	}

	res := httptest.NewRecorder()
	SrvMux(dict).ServeHTTP(res, httptest.NewRequest(http.MethodPost, "/api/render", bytes.NewReader(body)))
	if res.Code != http.StatusOK {
		t.Fatalf("status = %d", res.Code)
	}
	got := res.Body.String()
	if !strings.Contains(got, "Alfreds Futterkiste") {
		t.Fatalf("engine did not substitute data:\n%s", got)
	}
	if strings.Contains(got, "[CompanyName]") {
		t.Fatalf("fallback rendered literal expression:\n%s", got)
	}
}
