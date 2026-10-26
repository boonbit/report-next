package report

import (
	"bytes"
	"net/http"
	"net/http/httptest"
	"os"
	"strings"
	"testing"
)

func srvTestMux(t *testing.T) *http.ServeMux {
	t.Helper()
	return SrvMux(&DataDictionary{})
}

func TestSrvIndex(t *testing.T) {
	res := httptest.NewRecorder()
	srvTestMux(t).ServeHTTP(res, httptest.NewRequest(http.MethodGet, "/", nil))
	if res.Code != http.StatusOK {
		t.Fatalf("status = %d", res.Code)
	}
	if ct := res.Header().Get("Content-Type"); !strings.Contains(ct, "text/html") {
		t.Fatalf("content-type = %q", ct)
	}
	if !strings.Contains(res.Body.String(), "/static/app.js") {
		t.Fatal("index does not reference app.js")
	}
}

func TestSrvSampleRoundTrip(t *testing.T) {
	res := httptest.NewRecorder()
	srvTestMux(t).ServeHTTP(res, httptest.NewRequest(http.MethodGet, "/api/sample", nil))
	if res.Code != http.StatusOK {
		t.Fatalf("status = %d", res.Code)
	}
	rep, err := UnmarshalJSON(res.Body.Bytes())
	if err != nil {
		t.Fatal(err)
	}
	if rep.Version != IRVersion || len(rep.Pages) != 1 {
		t.Fatalf("sample = %+v", rep)
	}
	if got := attrVal(rep.Pages[0].Bands[0].Objects[0].Attrs, "Text"); got != "The report goes here..." {
		t.Fatalf("sample text = %q", got)
	}
}

func TestSrvSamplesList(t *testing.T) {
	res := httptest.NewRecorder()
	srvTestMux(t).ServeHTTP(res, httptest.NewRequest(http.MethodGet, "/api/samples", nil))
	if res.Code != http.StatusOK {
		t.Fatalf("status = %d", res.Code)
	}
	body := res.Body.String()
	for _, want := range []string{"hello", "table", "Gauge", "Simple Matrix", "Subreport", "box"} {
		if !strings.Contains(body, `"`+want+`"`) {
			t.Fatalf("samples missing %q: %s", want, body)
		}
	}
}

func TestSrvSampleByName(t *testing.T) {
	mux := srvTestMux(t)
	res := httptest.NewRecorder()
	mux.ServeHTTP(res, httptest.NewRequest(http.MethodGet, "/api/sample?name=table", nil))
	if res.Code != http.StatusOK {
		t.Fatalf("status = %d: %s", res.Code, res.Body.String())
	}
	if _, err := UnmarshalJSON(res.Body.Bytes()); err != nil {
		t.Fatal(err)
	}
	for _, target := range []string{"/api/sample?name=nope", "/api/sample?name=../server"} {
		bad := httptest.NewRecorder()
		mux.ServeHTTP(bad, httptest.NewRequest(http.MethodGet, target, nil))
		if bad.Code == http.StatusOK {
			t.Fatalf("%s unexpectedly ok", target)
		}
	}
}

func TestSrvImportRoundTrip(t *testing.T) {
	raw, err := os.ReadFile("testdata/hello.frx")
	if err != nil {
		t.Fatal(err)
	}
	mux := srvTestMux(t)
	res := httptest.NewRecorder()
	mux.ServeHTTP(res, httptest.NewRequest(http.MethodPost, "/api/import", bytes.NewReader(raw)))
	if res.Code != http.StatusOK {
		t.Fatalf("status = %d: %s", res.Code, res.Body.String())
	}
	rep, err := UnmarshalJSON(res.Body.Bytes())
	if err != nil {
		t.Fatal(err)
	}
	if len(rep.Pages) != 1 || len(rep.Extra) != 1 {
		t.Fatalf("import = %+v", rep)
	}
	// Import output must equal the direct ParseFRX+MarshalJSON path.
	direct, err := ParseFRX(bytes.NewReader(raw))
	if err != nil {
		t.Fatal(err)
	}
	want, _ := MarshalJSON(direct)
	if !bytes.Equal(bytes.TrimSpace(res.Body.Bytes()), bytes.TrimSpace(want)) {
		t.Fatal("import JSON differs from ParseFRX+MarshalJSON")
	}
}

func TestSrvRenderRoundTrip(t *testing.T) {
	raw, err := os.ReadFile("testdata/hello.frx")
	if err != nil {
		t.Fatal(err)
	}
	mux := srvTestMux(t)
	for _, tc := range []struct {
		name string
		body []byte
	}{
		{"frx", raw},
		{"irjson", mustSrvJSON(t, raw)},
	} {
		res := httptest.NewRecorder()
		mux.ServeHTTP(res, httptest.NewRequest(http.MethodPost, "/api/render", bytes.NewReader(tc.body)))
		if res.Code != http.StatusOK {
			t.Fatalf("%s: status = %d: %s", tc.name, res.Code, res.Body.String())
		}
		if ct := res.Header().Get("Content-Type"); !strings.Contains(ct, "text/html") {
			t.Fatalf("%s: content-type = %q", tc.name, ct)
		}
		if !strings.Contains(res.Body.String(), "The report goes here...") {
			t.Fatalf("%s: preview missing title text", tc.name)
		}
	}
}

func mustSrvJSON(t *testing.T, frx []byte) []byte {
	t.Helper()
	rep, err := ParseFRX(bytes.NewReader(frx))
	if err != nil {
		t.Fatal(err)
	}
	out, err := MarshalJSON(rep)
	if err != nil {
		t.Fatal(err)
	}
	return out
}

func TestSrvMethodGuards(t *testing.T) {
	mux := srvTestMux(t)
	for _, target := range []string{"/api/render", "/api/import", "/api/sample"} {
		res := httptest.NewRecorder()
		mux.ServeHTTP(res, httptest.NewRequest(http.MethodGet, target, nil))
		if target == "/api/sample" {
			continue // GET is legal here
		}
		if res.Code != http.StatusMethodNotAllowed {
			t.Errorf("%s GET status = %d", target, res.Code)
		}
	}
	res := httptest.NewRecorder()
	mux.ServeHTTP(res, httptest.NewRequest(http.MethodPost, "/api/render", strings.NewReader("nope")))
	if res.Code != http.StatusBadRequest {
		t.Errorf("bad render body status = %d", res.Code)
	}
}
