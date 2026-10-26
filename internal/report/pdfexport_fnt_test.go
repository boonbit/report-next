package report

import (
	"bytes"
	"net/http"
	"net/http/httptest"
	"os"
	"strings"
	"testing"

	"report-next/internal/pdffont"
)

// TestFntPDFExportFallbackBadEnv pins the legacy path: an unloadable font
// behaves exactly like no variable (skip-and-count, valid PDF).
func TestFntPDFExportFallbackBadEnv(t *testing.T) {
	t.Setenv(pdffont.FntEnvVar, "/nonexistent/font.ttf")
	rep := loadHello(t)
	rep.Pages[0].Bands[0].Objects = append(rep.Pages[0].Bands[0].Objects,
		Object{Kind: "TextObject", Name: "Thai", Attrs: []Attr{{"Text", "สวัสดี"}}})
	out, skipped, err := pdfExportReport(rep)
	if err != nil {
		t.Fatal(err)
	}
	if skipped != 1 {
		t.Fatalf("skipped = %d, want 1", skipped)
	}
	if !bytes.HasPrefix(out, []byte("%PDF-1.4")) {
		t.Fatal("missing pdf header")
	}

	res := httptest.NewRecorder()
	raw, err := MarshalJSON(loadHello(t))
	if err != nil {
		t.Fatal(err)
	}
	SrvMux(&DataDictionary{}).ServeHTTP(res,
		httptest.NewRequest(http.MethodPost, "/api/pdf", bytes.NewReader(raw)))
	if res.Code != http.StatusOK {
		t.Fatalf("status = %d", res.Code)
	}
	if !strings.Contains(res.Body.String(), "The report goes here") {
		t.Fatal("title text missing from served pdf")
	}
}

// TestFntPDFExportThai is the full Thai end-to-end: with REPORT_FONT_TTF
// set to a loadable TTF, Thai text embeds (nothing skipped) and ASCII stays
// literal; runes the font lacks still skip without error.
func TestFntPDFExportThai(t *testing.T) {
	path := os.Getenv(pdffont.FntEnvVar)
	if path == "" {
		t.Skip("REPORT_FONT_TTF not set")
	}
	fnt, err := pdffont.FntLoadTTF(path)
	if err != nil {
		t.Skipf("font not loadable: %v", err)
	}
	rep := loadHello(t)
	thai := "สวัสดี"
	if len(fnt.FntMissing(thai)) > 0 {
		t.Skip("font lacks Thai glyphs")
	}
	rep.Pages[0].Bands[0].Objects = append(rep.Pages[0].Bands[0].Objects,
		Object{Kind: "TextObject", Name: "Thai", Attrs: []Attr{{"Text", thai}}})
	out, skipped, err := pdfExportReport(rep)
	if err != nil {
		t.Fatal(err)
	}
	if skipped != 0 {
		t.Fatalf("skipped = %d, want 0", skipped)
	}
	s := string(out)
	for _, want := range []string{
		"%PDF-1.4", "/Subtype /TrueType", "/FontFile2", "/ToUnicode",
		"The report goes here", "beginbfchar",
	} {
		if !strings.Contains(s, want) {
			t.Fatalf("output missing %q", want)
		}
	}
	if strings.Contains(s, thai) {
		t.Fatal("thai text embedded raw instead of subset-encoded")
	}
	// A rune the font lacks must skip without error, not fail the export.
	missing := ""
	for _, r := range []rune{0x1F600, 0x4E00, 0xE000, 0x0370} {
		if len(fnt.FntMissing(string(r))) > 0 {
			missing = string(r)
			break
		}
	}
	if missing == "" {
		t.Skip("font covers every probe rune")
	}
	rep.Pages[0].Bands[0].Objects = append(rep.Pages[0].Bands[0].Objects,
		Object{Kind: "TextObject", Name: "Missing", Attrs: []Attr{{"Text", missing}}})
	if _, skipped, err := pdfExportReport(rep); err != nil {
		t.Fatal(err)
	} else if skipped != 1 {
		t.Fatalf("skipped = %d, want 1", skipped)
	}
}

// TestFntPDFExportASCIIOnlyWithEnv: with a loadable font but no non-ASCII
// text, the export matches the legacy bytes exactly.
func TestFntPDFExportASCIIOnlyWithEnv(t *testing.T) {
	path := os.Getenv(pdffont.FntEnvVar)
	if path == "" {
		t.Skip("REPORT_FONT_TTF not set")
	}
	if _, err := pdffont.FntLoadTTF(path); err != nil {
		t.Skipf("font not loadable: %v", err)
	}
	rep := loadHello(t)
	got, skipped, err := pdfExportReport(rep)
	if err != nil {
		t.Fatal(err)
	}
	if skipped != 0 {
		t.Fatalf("skipped = %d", skipped)
	}
	want, _, err := fntExportLegacy(rep)
	if err != nil {
		t.Fatal(err)
	}
	if !bytes.Equal(got, want) {
		t.Fatal("ascii-only export with env differs from legacy")
	}
}
