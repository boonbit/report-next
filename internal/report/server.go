package report

import (
	"bytes"
	"embed"
	"encoding/json"
	"io"
	"net/http"
	"os"
	"strconv"
	"strings"
)

// TRACK srv (P8-minimal designer): tiny HTTP front end over the IR.
//
// Rendering goes through the engine track (engPrepare/engRenderDocumentHTML)
// with data binding, grouping, totals and pagination. srvRenderDocumentHTML
// below remains as the fallback when the engine cannot prepare a report.
//

// Static UI and the sample report are embedded so the server works from any
// working directory.

//go:embed static/index.html static/app.js
var srvStatic embed.FS

//go:embed testdata/hello.frx
var srvHelloFRX []byte

//go:embed testdata/*.frx
var srvSamples embed.FS

// srvSampleNames lists bundled sample reports (file names without .frx,
// sorted) served by /api/samples and /api/sample?name=.
func srvSampleNames() []string {
	entries, err := srvSamples.ReadDir("testdata")
	if err != nil {
		return nil
	}
	var names []string
	for _, e := range entries {
		if e.IsDir() || !strings.HasSuffix(e.Name(), ".frx") {
			continue
		}
		names = append(names, strings.TrimSuffix(e.Name(), ".frx"))
	}
	return names
}

// SrvMux builds the minimal designer HTTP API. dict carries preview data;
// it may be nil (data-bound expressions render empty until wired).
// Routes:
//
//	GET  /            designer shell (static/index.html)
//	GET  /api/samples bundled sample names (JSON array)
//	POST /api/render  .frx (or IR JSON) bytes -> full HTML document
//	                  (?page=N selects one 1-based prepared page, still
//	                  reporting the full count in X-Total-Pages)
//	POST /api/outline .frx (or IR JSON) bytes -> {"outline":[{"title","page"}]}
//	POST /api/pdf     .frx (or IR JSON) bytes -> PDF document (text only)
//	POST /api/xlsx    ?source=Name -> .xlsx of one dictionary source
//	POST /api/schema  JSON {kind,dsn,content} -> schema tables/columns
//	POST /api/docx    .frx (or IR JSON) bytes -> .docx document
//	POST /api/csv     .frx (or IR JSON) bytes -> .csv of report text rows
//	POST /api/xlsx-report .frx (or IR JSON) bytes -> .xlsx of report text rows
//	POST /api/import  .frx bytes -> IR JSON (via ParseFRX+MarshalJSON)
//	GET  /api/sample  testdata/hello.frx as IR JSON
func SrvMux(dict *DataDictionary) *http.ServeMux {
	mux := http.NewServeMux()
	// Prefer the React designer bundle (web/dist, built via web/README.md)
	// when the server runs from the module root; otherwise fall back to the
	// embedded static UI so `go run ./cmd/report-designer` works anywhere.
	var dist http.Handler
	if _, err := os.Stat("web/dist/index.html"); err == nil {
		dist = http.FileServer(http.Dir("web/dist"))
	}
	mux.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodGet {
			http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
			return
		}
		if dist != nil && (r.URL.Path == "/" || strings.HasPrefix(r.URL.Path, "/assets/")) {
			dist.ServeHTTP(w, r)
			return
		}
		if r.URL.Path != "/" {
			http.NotFound(w, r)
			return
		}
		body, err := srvStatic.ReadFile("static/index.html")
		if err != nil {
			http.Error(w, "index missing", http.StatusInternalServerError)
			return
		}
		w.Header().Set("Content-Type", "text/html; charset=utf-8")
		_, _ = w.Write(body)
	})
	mux.HandleFunc("/api/render", func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
			return
		}
		raw, err := io.ReadAll(io.LimitReader(r.Body, 32<<20))
		if err != nil {
			http.Error(w, "bad body", http.StatusBadRequest)
			return
		}
		rep, err := srvDecodeReport(raw)
		if err != nil {
			http.Error(w, "parse: "+err.Error(), http.StatusBadRequest)
			return
		}
		html := srvRenderDocumentHTML(rep)
		total := len(rep.Pages)
		if doc, err := engPrepare(rep, dict); err == nil {
			total = len(doc.Pages)
			if pg := srvPageParam(r, total); pg > 0 {
				html = engRenderPageHTML(doc, pg)
			} else {
				html = engRenderDocumentHTML(doc)
			}
		}
		w.Header().Set("X-Total-Pages", strconv.Itoa(total))
		w.Header().Set("Content-Type", "text/html; charset=utf-8")
		_, _ = io.WriteString(w, html)
	})
	mux.HandleFunc("/api/outline", func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
			return
		}
		raw, err := io.ReadAll(io.LimitReader(r.Body, 32<<20))
		if err != nil {
			http.Error(w, "bad body", http.StatusBadRequest)
			return
		}
		rep, err := srvDecodeReport(raw)
		if err != nil {
			http.Error(w, "parse: "+err.Error(), http.StatusBadRequest)
			return
		}
		entries := engOutline(rep, dict)
		out := srvOutlineResponse{Outline: make([]srvOutlineEntry, 0, len(entries))}
		for _, e := range entries {
			out.Outline = append(out.Outline, srvOutlineEntry{Title: e.Title, Page: e.Page})
		}
		body, err := json.Marshal(out)
		if err != nil {
			http.Error(w, "encode: "+err.Error(), http.StatusInternalServerError)
			return
		}
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write(body)
	})
	mux.HandleFunc("/api/pdf", func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
			return
		}
		raw, err := io.ReadAll(io.LimitReader(r.Body, 32<<20))
		if err != nil {
			http.Error(w, "bad body", http.StatusBadRequest)
			return
		}
		rep, err := srvDecodeReport(raw)
		if err != nil {
			http.Error(w, "parse: "+err.Error(), http.StatusBadRequest)
			return
		}
		out, _, err := pdfExportReport(rep)
		if err != nil {
			http.Error(w, "export: "+err.Error(), http.StatusInternalServerError)
			return
		}
		w.Header().Set("Content-Type", "application/pdf")
		_, _ = w.Write(out)
	})
	mux.HandleFunc("/api/xlsx", func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
			return
		}
		source := r.URL.Query().Get("source")
		if source == "" {
			http.Error(w, "missing ?source=", http.StatusBadRequest)
			return
		}
		out, err := xlsxExportSource(dict, source)
		if err != nil {
			http.Error(w, "export: "+err.Error(), http.StatusBadRequest)
			return
		}
		w.Header().Set("Content-Type", "application/vnd.openxmlformats-officedocument.spreadsheetml.sheet")
		_, _ = w.Write(out)
	})
	mux.HandleFunc("/api/schema", func(w http.ResponseWriter, r *http.Request) {
		srvHandleSchema(w, r)
	})
	mux.HandleFunc("/api/docx", func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
			return
		}
		raw, err := io.ReadAll(io.LimitReader(r.Body, 32<<20))
		if err != nil {
			http.Error(w, "bad body", http.StatusBadRequest)
			return
		}
		rep, err := srvDecodeReport(raw)
		if err != nil {
			http.Error(w, "parse: "+err.Error(), http.StatusBadRequest)
			return
		}
		out, err := docxExportReport(rep)
		if err != nil {
			http.Error(w, "export: "+err.Error(), http.StatusInternalServerError)
			return
		}
		w.Header().Set("Content-Type", "application/vnd.openxmlformats-officedocument.wordprocessingml.document")
		_, _ = w.Write(out)
	})
	mux.HandleFunc("/api/csv", func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
			return
		}
		raw, err := io.ReadAll(io.LimitReader(r.Body, 32<<20))
		if err != nil {
			http.Error(w, "bad body", http.StatusBadRequest)
			return
		}
		rep, err := srvDecodeReport(raw)
		if err != nil {
			http.Error(w, "parse: "+err.Error(), http.StatusBadRequest)
			return
		}
		w.Header().Set("Content-Type", "text/csv; charset=utf-8")
		_, _ = io.WriteString(w, exportCSVReport(rep, dict))
	})
	mux.HandleFunc("/api/xlsx-report", func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
			return
		}
		raw, err := io.ReadAll(io.LimitReader(r.Body, 32<<20))
		if err != nil {
			http.Error(w, "bad body", http.StatusBadRequest)
			return
		}
		rep, err := srvDecodeReport(raw)
		if err != nil {
			http.Error(w, "parse: "+err.Error(), http.StatusBadRequest)
			return
		}
		out, err := xlsxExportReport(rep, dict)
		if err != nil {
			http.Error(w, "export: "+err.Error(), http.StatusInternalServerError)
			return
		}
		w.Header().Set("Content-Type", "application/vnd.openxmlformats-officedocument.spreadsheetml.sheet")
		_, _ = w.Write(out)
	})
	mux.HandleFunc("/api/import", func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
			return
		}
		raw, err := io.ReadAll(io.LimitReader(r.Body, 32<<20))
		if err != nil {
			http.Error(w, "bad body", http.StatusBadRequest)
			return
		}
		rep, err := ParseFRX(bytes.NewReader(raw))
		if err != nil {
			http.Error(w, "parse: "+err.Error(), http.StatusBadRequest)
			return
		}
		out, err := MarshalJSON(rep)
		if err != nil {
			http.Error(w, "encode: "+err.Error(), http.StatusInternalServerError)
			return
		}
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write(out)
	})
	mux.HandleFunc("/api/samples", func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodGet {
			http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
			return
		}
		out, err := json.Marshal(srvSampleNames())
		if err != nil {
			http.Error(w, "encode: "+err.Error(), http.StatusInternalServerError)
			return
		}
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write(out)
	})
	mux.HandleFunc("/api/sample", func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodGet {
			http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
			return
		}
		name := r.URL.Query().Get("name")
		if name == "" {
			name = "hello"
		}
		var raw []byte
		if name == "hello" {
			raw = srvHelloFRX
		} else {
			if strings.ContainsAny(name, `/\`) || strings.Contains(name, "..") {
				http.Error(w, "bad sample name", http.StatusBadRequest)
				return
			}
			var err error
			raw, err = srvSamples.ReadFile("testdata/" + name + ".frx")
			if err != nil {
				http.Error(w, "no such sample: "+name, http.StatusNotFound)
				return
			}
		}
		rep, err := ParseFRX(bytes.NewReader(raw))
		if err != nil {
			http.Error(w, "sample broken: "+err.Error(), http.StatusInternalServerError)
			return
		}
		out, err := MarshalJSON(rep)
		if err != nil {
			http.Error(w, "encode: "+err.Error(), http.StatusInternalServerError)
			return
		}
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write(out)
	})
	mux.HandleFunc("/static/app.js", func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodGet {
			http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
			return
		}
		body, err := srvStatic.ReadFile("static/app.js")
		if err != nil {
			http.Error(w, "app.js missing", http.StatusInternalServerError)
			return
		}
		w.Header().Set("Content-Type", "text/javascript; charset=utf-8")
		_, _ = w.Write(body)
	})
	return mux
}

// srvOutlineEntry is one outline item in /api/outline responses.
type srvOutlineEntry struct {
	Title string `json:"title"`
	Page  int    `json:"page"`
}

// srvOutlineResponse is the /api/outline response body; Outline stays
// non-nil so empty outlines encode as [] rather than null.
type srvOutlineResponse struct {
	Outline []srvOutlineEntry `json:"outline"`
}

// srvPageParam parses ?page=N as a 1-based prepared page number. Missing,
// non-numeric, or out-of-range values yield 0 (render all pages).
func srvPageParam(r *http.Request, total int) int {
	s := strings.TrimSpace(r.URL.Query().Get("page"))
	if s == "" {
		return 0
	}
	n, err := strconv.Atoi(s)
	if err != nil || n < 1 || n > total {
		return 0
	}
	return n
}

// srvDecodeReport accepts .frx XML first, then IR JSON (the designer editor
// posts back edited IR JSON, so /api/render takes both).
func srvDecodeReport(raw []byte) (Report, error) {
	if rep, err := ParseFRX(bytes.NewReader(raw)); err == nil {
		return rep, nil
	}
	return UnmarshalJSON(raw)
}

// srvRenderDocumentHTML is the fallback used when the engine cannot prepare
// a report: every band of every page becomes one fragment in document order.
func srvRenderDocumentHTML(rep Report) string {
	var sb strings.Builder
	sb.WriteString("<!DOCTYPE html>\n<html><head><meta charset=\"utf-8\">" +
		"<title>Report preview</title></head><body>\n")
	for _, p := range rep.Pages {
		sb.WriteString("<section class=\"page\">\n")
		for _, b := range p.Bands {
			sb.WriteString(RenderBand(b))
			sb.WriteString("\n")
		}
		sb.WriteString("</section>\n")
	}
	sb.WriteString("</body></html>\n")
	return sb.String()
}
