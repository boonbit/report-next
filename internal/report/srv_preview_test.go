package report

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

// Preview paging (P1) + hyperlinks/outline (P4-Go) server tests.

// previewRows builds a one-source dictionary of Name rows.
func previewRows(names ...string) *DataDictionary {
	rows := make([]DataRow, 0, len(names))
	for _, n := range names {
		rows = append(rows, DataRow{"Name": n})
	}
	return &DataDictionary{Sources: []DataSource{{
		Name:    "Items",
		Columns: []DataColumn{{Name: "Name"}},
		Rows:    rows,
	}}}
}

// previewPagedReport builds a report that prepares exactly two pages with
// eight 18px rows on a 30mm usable height (6 rows on page 1, 2 on page 2).
func previewPagedReport() *Report {
	rep := &Report{Version: IRVersion, Pages: []Page{{
		Name:  "Page1",
		Attrs: []Attr{{Name: "PaperHeight", Value: "30"}},
	}}}
	bi := exprAddBand(rep, 0, "DataBand", "Data1")
	rep.Pages[0].Bands[bi].Attrs = append(rep.Pages[0].Bands[bi].Attrs,
		Attr{Name: "DataSource", Value: "Items"},
		Attr{Name: "Height", Value: "18"},
		Attr{Name: "OutlineExpression", Value: "Row: [Name]"})
	exprAddText(rep, 0, bi, "T1", "[Name]")
	return rep
}

func previewPost(t *testing.T, mux *http.ServeMux, target string, body []byte) *httptest.ResponseRecorder {
	t.Helper()
	res := httptest.NewRecorder()
	mux.ServeHTTP(res, httptest.NewRequest(http.MethodPost, target, bytes.NewReader(body)))
	return res
}

func previewJSON(t *testing.T, rep *Report) []byte {
	t.Helper()
	body, err := MarshalJSON(*rep)
	if err != nil {
		t.Fatal(err)
	}
	return body
}

func TestSrvRenderPageParam(t *testing.T) {
	dict := previewRows("R1", "R2", "R3", "R4", "R5", "R6", "R7", "R8")
	body := previewJSON(t, previewPagedReport())
	mux := SrvMux(dict)

	all := previewPost(t, mux, "/api/render", body)
	if all.Code != http.StatusOK {
		t.Fatalf("all pages: status = %d", all.Code)
	}
	if got := all.Header().Get("X-Total-Pages"); got != "2" {
		t.Fatalf("all pages: X-Total-Pages = %q, want 2", got)
	}
	if got := strings.Count(all.Body.String(), "<section"); got != 2 {
		t.Fatalf("all pages: want 2 sections, got %d", got)
	}

	p2 := previewPost(t, mux, "/api/render?page=2", body)
	if p2.Code != http.StatusOK {
		t.Fatalf("page=2: status = %d", p2.Code)
	}
	if got := p2.Header().Get("X-Total-Pages"); got != "2" {
		t.Fatalf("page=2: X-Total-Pages = %q, want 2", got)
	}
	got := p2.Body.String()
	if n := strings.Count(got, "<section"); n != 1 {
		t.Fatalf("page=2: want 1 section, got %d", n)
	}
	if !strings.Contains(got, `data-page="2"`) {
		t.Fatalf("page=2: missing data-page=2:\n%s", got)
	}
	for _, want := range []string{"R7", "R8"} {
		if !strings.Contains(got, want) {
			t.Errorf("page=2: missing %q:\n%s", want, got)
		}
	}
	if strings.Contains(got, "R1") {
		t.Errorf("page=2: leaked page-1 row R1:\n%s", got)
	}

	p1 := previewPost(t, mux, "/api/render?page=1", body)
	if !strings.Contains(p1.Body.String(), `data-page="1"`) || !strings.Contains(p1.Body.String(), "R1") {
		t.Fatalf("page=1: wrong section:\n%s", p1.Body.String())
	}
	if strings.Contains(p1.Body.String(), "R8") {
		t.Fatalf("page=1: leaked page-2 row R8:\n%s", p1.Body.String())
	}

	for _, target := range []string{"/api/render?page=abc", "/api/render?page=0", "/api/render?page=9"} {
		res := previewPost(t, mux, target, body)
		if res.Code != http.StatusOK {
			t.Fatalf("%s: status = %d", target, res.Code)
		}
		if got := strings.Count(res.Body.String(), "<section"); got != 2 {
			t.Fatalf("%s: want fallback to all 2 pages, got %d sections", target, got)
		}
		if got := res.Header().Get("X-Total-Pages"); got != "2" {
			t.Fatalf("%s: X-Total-Pages = %q, want 2", target, got)
		}
	}
}

func TestRenderHyperlinkGoldens(t *testing.T) {
	geom := []Attr{
		{Name: "Left", Value: "1"}, {Name: "Top", Value: "2"},
		{Name: "Width", Value: "3"}, {Name: "Height", Value: "4"},
	}
	bandOf := func(text, link string, extra ...Attr) Band {
		attrs := append([]Attr{{Name: "Text", Value: text}}, geom...)
		if link != "" {
			attrs = append(attrs, Attr{Name: "Hyperlink.Value", Value: link})
		}
		return Band{Objects: []Object{{Kind: "TextObject", Name: "T", Attrs: attrs}}}
	}
	for _, tc := range []struct {
		name string
		band Band
		want string
	}{
		{
			"url",
			bandOf("Visit", "https://example.com/?a=1&b=2"),
			`<div class="band" style="position:relative;width:0px;height:0px;">` +
				`<div class="text" style="position:absolute;left:1px;top:2px;width:3px;height:4px;">` +
				`<a href="https://example.com/?a=1&amp;b=2" target="_blank" rel="noopener">Visit</a></div></div>`,
		},
		{
			"url escapes text",
			bandOf("A<B", "http://example.com/"),
			`<div class="band" style="position:relative;width:0px;height:0px;">` +
				`<div class="text" style="position:absolute;left:1px;top:2px;width:3px;height:4px;">` +
				`<a href="http://example.com/" target="_blank" rel="noopener">A&lt;B</a></div></div>`,
		},
		{
			"mailto",
			bandOf("Mail", "mailto:doc@example.com"),
			`<div class="band" style="position:relative;width:0px;height:0px;">` +
				`<div class="text" style="position:absolute;left:1px;top:2px;width:3px;height:4px;">` +
				`<a href="mailto:doc@example.com" target="_blank" rel="noopener">Mail</a></div></div>`,
		},
		{
			"page jump",
			bandOf("Page two", "2"),
			`<div class="band" style="position:relative;width:0px;height:0px;">` +
				`<div class="text" style="position:absolute;left:1px;top:2px;width:3px;height:4px;">` +
				`<a data-goto-page="2">Page two</a></div></div>`,
		},
		{
			"plain value stays text",
			bandOf("Intro", "Chapter1"),
			`<div class="band" style="position:relative;width:0px;height:0px;">` +
				`<div class="text" style="position:absolute;left:1px;top:2px;width:3px;height:4px;">Intro</div></div>`,
		},
		{
			"no hyperlink stays text",
			bandOf("Just text", ""),
			`<div class="band" style="position:relative;width:0px;height:0px;">` +
				`<div class="text" style="position:absolute;left:1px;top:2px;width:3px;height:4px;">Just text</div></div>`,
		},
	} {
		if got := RenderBand(tc.band); got != tc.want {
			t.Errorf("%s:\n got %q\nwant %q", tc.name, got, tc.want)
		}
	}

	// Nested <Hyperlink Value="..."/> element form also jumps pages.
	nested := Band{Objects: []Object{{
		Kind: "TextObject", Name: "T",
		Attrs:    []Attr{{Name: "Text", Value: "Go"}},
		Children: []Node{{Name: "Hyperlink", Attrs: []Attr{{Name: "Value", Value: "3"}}}},
	}}}
	if got := RenderBand(nested); !strings.Contains(got, `<a data-goto-page="3">Go</a>`) {
		t.Errorf("nested hyperlink: got %q", got)
	}
}

func TestSrvOutline(t *testing.T) {
	dict := previewRows("A", "B")
	rep := exprNewReport("Title")
	rep.Pages[0].Bands[0].Attrs = append(rep.Pages[0].Bands[0].Attrs,
		Attr{Name: "OutlineExpression", Value: "Cover"})
	bi := exprAddBand(rep, 0, "DataBand", "Data1")
	rep.Pages[0].Bands[bi].Attrs = append(rep.Pages[0].Bands[bi].Attrs,
		Attr{Name: "DataSource", Value: "Items"},
		Attr{Name: "Height", Value: "18"},
		Attr{Name: "OutlineExpression", Value: "Row: [Name]"})
	exprAddText(rep, 0, bi, "T1", "[Name]")

	res := previewPost(t, SrvMux(dict), "/api/outline", previewJSON(t, rep))
	if res.Code != http.StatusOK {
		t.Fatalf("status = %d: %s", res.Code, res.Body.String())
	}
	if ct := res.Header().Get("Content-Type"); !strings.Contains(ct, "application/json") {
		t.Fatalf("content-type = %q", ct)
	}
	var decoded struct {
		Outline []struct {
			Title string `json:"title"`
			Page  int    `json:"page"`
		} `json:"outline"`
	}
	if err := json.Unmarshal(res.Body.Bytes(), &decoded); err != nil {
		t.Fatal(err)
	}
	if decoded.Outline == nil {
		t.Fatal("outline is null, want non-nil array")
	}
	want := []struct {
		Title string
		Page  int
	}{{"Cover", 1}, {"Row: A", 1}, {"Row: B", 1}}
	if len(decoded.Outline) != len(want) {
		t.Fatalf("outline = %+v, want %+v", decoded.Outline, want)
	}
	for i, w := range want {
		if decoded.Outline[i].Title != w.Title || decoded.Outline[i].Page != w.Page {
			t.Fatalf("outline[%d] = %+v, want %+v", i, decoded.Outline[i], w)
		}
	}

	// Multi-page outline numbers follow the prepared pages.
	multi := previewPost(t, SrvMux(previewRows("R1", "R2", "R3", "R4", "R5", "R6", "R7", "R8")),
		"/api/outline", previewJSON(t, previewPagedReport()))
	var paged struct {
		Outline []struct {
			Title string `json:"title"`
			Page  int    `json:"page"`
		} `json:"outline"`
	}
	if err := json.Unmarshal(multi.Body.Bytes(), &paged); err != nil {
		t.Fatal(err)
	}
	if len(paged.Outline) != 8 {
		t.Fatalf("paged outline has %d entries, want 8", len(paged.Outline))
	}
	names := []string{"R1", "R2", "R3", "R4", "R5", "R6", "R7", "R8"}
	for i, e := range paged.Outline {
		wantPage := 1
		if i >= 6 {
			wantPage = 2
		}
		if wantTitle := "Row: " + names[i]; e.Page != wantPage || e.Title != wantTitle {
			t.Fatalf("entry %d = %+v, want {%s %d}", i, e, wantTitle, wantPage)
		}
	}

	// Empty outline encodes as [] rather than null.
	empty := previewPost(t, SrvMux(&DataDictionary{}), "/api/outline", previewJSON(t, exprNewReport("Hi")))
	if !strings.Contains(empty.Body.String(), `"outline":[]`) {
		t.Fatalf("empty outline body = %s, want \"outline\":[]", empty.Body.String())
	}

	// Same decode errors as /api/render, plus the POST-only guard.
	bad := previewPost(t, SrvMux(&DataDictionary{}), "/api/outline", []byte("nope"))
	if bad.Code != http.StatusBadRequest {
		t.Fatalf("bad body status = %d, want 400", bad.Code)
	}
	get := httptest.NewRecorder()
	SrvMux(&DataDictionary{}).ServeHTTP(get, httptest.NewRequest(http.MethodGet, "/api/outline", nil))
	if get.Code != http.StatusMethodNotAllowed {
		t.Fatalf("GET status = %d, want 405", get.Code)
	}
}
