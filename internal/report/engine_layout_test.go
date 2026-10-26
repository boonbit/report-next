package report

import (
	"strings"
	"testing"
)

// Layout goldens for R1 (CanGrow/KeepTogether/ShiftMode) and R5
// (ReportSummaryBand/OverlayBand routing). Reports without the new attrs
// behave exactly as before; these tests pin the new behavior only.

// engLayoutGrowReport builds a one-page report with a single static band
// holding a growing text over a second text. Courier at 10pt is uniform
// 6px per glyph, so Width=65 fits one "AAAAA" word per line: three words
// wrap to exactly 3 lines, needing 3*13.33px*1.15 = 46px over Height=15.
func engLayoutGrowReport(shift bool) (*Report, *DataDictionary) {
	src, err := DataLoadCSV(strings.NewReader("K\nv\n"), "Items")
	if err != nil {
		panic(err)
	}
	dict := &DataDictionary{Sources: []DataSource{*src}}
	rep := &Report{Version: IRVersion, Pages: []Page{{Name: "Page1"}}}
	bi := exprAddBand(rep, 0, "DataBand", "Data1")
	rep.Pages[0].Bands[bi].Attrs = append(rep.Pages[0].Bands[bi].Attrs,
		Attr{Name: "Height", Value: "40"},
		Attr{Name: "DataSource", Value: "Items"})
	if shift {
		rep.Pages[0].Bands[bi].Attrs = append(rep.Pages[0].Bands[bi].Attrs,
			Attr{Name: "ShiftMode", Value: "true"})
	}
	oi := exprAddText(rep, 0, bi, "Grow", "AAAAA AAAAA AAAAA")
	rep.Pages[0].Bands[bi].Objects[oi].Attrs = append(
		rep.Pages[0].Bands[bi].Objects[oi].Attrs,
		Attr{Name: "Top", Value: "0"},
		Attr{Name: "Width", Value: "65"},
		Attr{Name: "Height", Value: "15"},
		Attr{Name: "Font", Value: "Courier New, 10pt"},
		Attr{Name: "CanGrow", Value: "true"})
	oi2 := exprAddText(rep, 0, bi, "Below", "BELOW")
	rep.Pages[0].Bands[bi].Objects[oi2].Attrs = append(
		rep.Pages[0].Bands[bi].Objects[oi2].Attrs,
		Attr{Name: "Top", Value: "15"},
		Attr{Name: "Width", Value: "200"},
		Attr{Name: "Height", Value: "15"},
		Attr{Name: "Font", Value: "Courier New, 10pt"})
	return rep, dict
}

func TestEngLayoutCanGrowShiftDown(t *testing.T) {
	rep, dict := engLayoutGrowReport(true)
	doc, err := engPrepare(*rep, dict)
	if err != nil {
		t.Fatalf("engPrepare: %v", err)
	}
	if len(doc.Pages) != 1 {
		t.Fatalf("want 1 page, got %d", len(doc.Pages))
	}
	html := engRenderDocumentHTML(doc)
	// Grown text: 3 lines x 13.33px x 1.15 = 46px over Height=15.
	if !strings.Contains(html, "height:46.00px") {
		t.Errorf("grown text height 46.00px missing:\n%s", html)
	}
	// ShiftMode: the lower text moves down by the growth delta
	// 15 + (34.5-15) = 34.5, and the band encloses both (34.5+15).
	if !strings.Contains(html, "top:46.00px") {
		t.Errorf("shifted text top 46.0px missing:\n%s", html)
	}
	if !strings.Contains(html, "height:61.00px") {
		t.Errorf("grown band height 61.00px missing:\n%s", html)
	}
	if !strings.Contains(html, "BELOW") {
		t.Errorf("lower text lost:\n%s", html)
	}
}

func TestEngLayoutCanGrowNoShiftKeepsPositions(t *testing.T) {
	rep, dict := engLayoutGrowReport(false)
	doc, err := engPrepare(*rep, dict)
	if err != nil {
		t.Fatalf("engPrepare: %v", err)
	}
	html := engRenderDocumentHTML(doc)
	// The text still grows, but without ShiftMode the lower object stays
	// at Top=15 (may overlap); the band still encloses the grown text
	// (46px) or overflow:hidden would clip it.
	if !strings.Contains(html, "height:46.00px") {
		t.Errorf("grown text height 46.00px missing:\n%s", html)
	}
	if !strings.Contains(html, "top:15px") {
		t.Errorf("lower text should keep top:15px without ShiftMode:\n%s", html)
	}
	if !strings.Contains(html, `<div class="band" style="position:relative;width:0px;height:46.00px;">`) {
		t.Errorf("band should enclose grown text at 46.00px:\n%s", html)
	}
}

func TestEngLayoutCanGrowDataRowsStayIndependent(t *testing.T) {
	csvData := "Name\nAAAAA AAAAA AAAAA\nshort\n"
	src, err := DataLoadCSV(strings.NewReader(csvData), "Items")
	if err != nil {
		t.Fatalf("DataLoadCSV: %v", err)
	}
	dict := &DataDictionary{Sources: []DataSource{*src}}
	rep := &Report{Version: IRVersion, Pages: []Page{{
		Name:  "Page1",
		Attrs: []Attr{{Name: "PaperHeight", Value: "100"}},
	}}}
	bi := exprAddBand(rep, 0, "DataBand", "Data1")
	rep.Pages[0].Bands[bi].Attrs = append(rep.Pages[0].Bands[bi].Attrs,
		Attr{Name: "Height", Value: "40"},
		Attr{Name: "ShiftMode", Value: "true"},
		Attr{Name: "DataSource", Value: "Items"})
	oi := exprAddText(rep, 0, bi, "Grow", "[Name]")
	rep.Pages[0].Bands[bi].Objects[oi].Attrs = append(
		rep.Pages[0].Bands[bi].Objects[oi].Attrs,
		Attr{Name: "Top", Value: "0"},
		Attr{Name: "Width", Value: "65"},
		Attr{Name: "Height", Value: "15"},
		Attr{Name: "Font", Value: "Courier New, 10pt"},
		Attr{Name: "CanGrow", Value: "true"})
	doc, err := engPrepare(*rep, dict)
	if err != nil {
		t.Fatalf("engPrepare: %v", err)
	}
	html := engRenderDocumentHTML(doc)
	// Only the long row grows; the short row must not inherit growth
	// through the shared band template. The grown text div carries
	// line-height, which the enclosing band div does not.
	if got := strings.Count(html, "height:46.00px;line-height"); got != 1 {
		t.Errorf("want exactly 1 grown instance, got %d:\n%s", got, html)
	}
	if !strings.Contains(html, "short") {
		t.Errorf("short row text lost:\n%s", html)
	}
}

// engLayoutPagedReport builds a report whose two 50px data instances need
// two pages on a 20mm sheet (usable room 20*3.7795 = 75.59px).
func engLayoutPagedReport(keep string) (*Report, *DataDictionary) {
	csvData := "Name\nROW1\nROW2\n"
	src, err := DataLoadCSV(strings.NewReader(csvData), "Items")
	if err != nil {
		panic(err)
	}
	dict := &DataDictionary{Sources: []DataSource{*src}}
	rep := &Report{Version: IRVersion, Pages: []Page{{
		Name:  "Page1",
		Attrs: []Attr{{Name: "PaperHeight", Value: "20"}},
	}}}
	bi := exprAddBand(rep, 0, "DataBand", "Data1")
	rep.Pages[0].Bands[bi].Attrs = append(rep.Pages[0].Bands[bi].Attrs,
		Attr{Name: "Height", Value: "50"},
		Attr{Name: "DataSource", Value: "Items"})
	if keep != "" {
		rep.Pages[0].Bands[bi].Attrs = append(rep.Pages[0].Bands[bi].Attrs,
			Attr{Name: "KeepTogether", Value: keep})
	}
	oi := exprAddText(rep, 0, bi, "T", "[Name]")
	rep.Pages[0].Bands[bi].Objects[oi].Attrs = append(
		rep.Pages[0].Bands[bi].Objects[oi].Attrs,
		Attr{Name: "Top", Value: "0"},
		Attr{Name: "Width", Value: "200"},
		Attr{Name: "Height", Value: "15"})
	return rep, dict
}

func TestEngLayoutKeepTogetherPageJump(t *testing.T) {
	rep, dict := engLayoutPagedReport("true")
	doc, err := engPrepare(*rep, dict)
	if err != nil {
		t.Fatalf("engPrepare: %v", err)
	}
	html := engRenderDocumentHTML(doc)
	if got := strings.Count(html, "<section"); got != 2 {
		t.Fatalf("want 2 data-page sections, got %d:\n%s", got, html)
	}
	pages := strings.Split(html, "<section")
	// The whole second instance moves to the next sheet: page 1 holds
	// ROW1 only, page 2 holds ROW2.
	if !strings.Contains(pages[1], "ROW1") || strings.Contains(pages[1], "ROW2") {
		t.Errorf("page 1 should hold ROW1 only:\n%s", pages[1])
	}
	if !strings.Contains(pages[2], "ROW2") || strings.Contains(pages[2], "ROW1") {
		t.Errorf("page 2 should hold ROW2 only:\n%s", pages[2])
	}
}

func TestEngLayoutKeepTogetherFalseOverflows(t *testing.T) {
	rep, dict := engLayoutPagedReport("false")
	doc, err := engPrepare(*rep, dict)
	if err != nil {
		t.Fatalf("engPrepare: %v", err)
	}
	html := engRenderDocumentHTML(doc)
	if got := strings.Count(html, "<section"); got != 1 {
		t.Errorf("KeepTogether=false should overflow in place (1 section), got %d:\n%s", got, html)
	}
}

func TestEngLayoutSummaryLastOverlayEveryPage(t *testing.T) {
	rep, dict := engLayoutPagedReport("")
	si := exprAddBand(rep, 0, "ReportSummaryBand", "Summary1")
	rep.Pages[0].Bands[si].Attrs = append(rep.Pages[0].Bands[si].Attrs,
		Attr{Name: "Height", Value: "20"})
	soi := exprAddText(rep, 0, si, "ST", "SUMMARY")
	rep.Pages[0].Bands[si].Objects[soi].Attrs = append(
		rep.Pages[0].Bands[si].Objects[soi].Attrs,
		Attr{Name: "Top", Value: "0"},
		Attr{Name: "Width", Value: "200"},
		Attr{Name: "Height", Value: "15"})
	oi := exprAddBand(rep, 0, "OverlayBand", "Overlay1")
	rep.Pages[0].Bands[oi].Attrs = append(rep.Pages[0].Bands[oi].Attrs,
		Attr{Name: "Height", Value: "10"})
	ooi := exprAddText(rep, 0, oi, "OT", "WATERMARK")
	rep.Pages[0].Bands[oi].Objects[ooi].Attrs = append(
		rep.Pages[0].Bands[oi].Objects[ooi].Attrs,
		Attr{Name: "Top", Value: "0"},
		Attr{Name: "Width", Value: "200"},
		Attr{Name: "Height", Value: "10"})
	doc, err := engPrepare(*rep, dict)
	if err != nil {
		t.Fatalf("engPrepare: %v", err)
	}
	html := engRenderDocumentHTML(doc)
	sections := strings.Count(html, "<section")
	if sections != 2 {
		t.Fatalf("want 2 data-page sections, got %d:\n%s", sections, html)
	}
	// Summary renders once, after all flow instances.
	if got := strings.Count(html, "SUMMARY"); got != 1 {
		t.Errorf("want SUMMARY exactly once, got %d:\n%s", got, html)
	}
	if strings.Index(html, "ROW2") > strings.Index(html, "SUMMARY") {
		t.Errorf("SUMMARY should render after the last flow instance:\n%s", html)
	}
	if strings.Index(html, "ROW1") > strings.Index(html, "SUMMARY") {
		t.Errorf("SUMMARY should render after the first flow instance:\n%s", html)
	}
	// Overlay content repeats on every prepared page.
	if got := strings.Count(html, "WATERMARK"); got != sections {
		t.Errorf("want WATERMARK on every page (%d), got %d:\n%s", sections, got, html)
	}
}
