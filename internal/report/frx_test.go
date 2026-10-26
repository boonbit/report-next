package report

import (
	"testing"
)

func loadHello(t *testing.T) Report {
	t.Helper()
	return loadFile(t, "hello.frx")
}

func TestParseHello(t *testing.T) {
	rep := loadHello(t)
	if rep.ScriptLanguage != "CSharp" {
		t.Fatalf("ScriptLanguage = %q", rep.ScriptLanguage)
	}
	if len(rep.Pages) != 1 {
		t.Fatalf("pages = %d, want 1", len(rep.Pages))
	}
	p := rep.Pages[0]
	if len(p.Bands) != 2 || p.Bands[0].Kind != "ReportTitleBand" || p.Bands[1].Kind != "PageFooterBand" {
		t.Fatalf("bands = %+v", p.Bands)
	}
	got := attrVal(p.Bands[0].Objects[0].Attrs, "Text")
	if got != "The report goes here..." {
		t.Fatalf("title text = %q", got)
	}
	if len(rep.Extra) != 1 || rep.Extra[0].Name != "DialogPage" {
		t.Fatalf("extra = %+v", rep.Extra)
	}
}

func TestRoundTrip(t *testing.T) {
	roundTrip(t, loadHello(t))
}
