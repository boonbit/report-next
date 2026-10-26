package report

import (
	"strings"
	"testing"
)

// tabFindTable locates a TableObject by name anywhere in the report,
// descending into verbatim children (e.g. Table2/Table3 sit under a
// ChildBand node inside the DataBand in table.frx).
func tabFindTable(t *testing.T, rep Report, name string) Object {
	t.Helper()
	var rec func(ns []Node) (Object, bool)
	rec = func(ns []Node) (Object, bool) {
		for _, n := range ns {
			if n.Name == "TableObject" && attrVal(n.Attrs, "Name") == name {
				return tabObjectOf(n), true
			}
			if o, ok := rec(n.Children); ok {
				return o, true
			}
		}
		return Object{}, false
	}
	for _, p := range rep.Pages {
		for _, b := range p.Bands {
			for _, o := range b.Objects {
				if o.Kind == "TableObject" && o.Name == name {
					return o
				}
				if found, ok := rec(o.Children); ok {
					return found
				}
			}
		}
	}
	t.Fatalf("table %q not found", name)
	return Object{}
}

func TestTabParseTableColumnsRows(t *testing.T) {
	rep := loadFile(t, "table.frx")
	tab := mustTabParse(t, tabFindTable(t, rep, "Table1"))
	if len(tab.columns) != 4 {
		t.Fatalf("columns = %d, want 4", len(tab.columns))
	}
	for i, c := range tab.columns {
		if c.width != "75.6" {
			t.Errorf("column %d width = %q, want 75.6", i, c.width)
		}
	}
	if len(tab.rows) != 3 {
		t.Fatalf("rows = %d, want 3", len(tab.rows))
	}
	if got := attrVal(tab.rows[0].cells[0].attrs, "Text"); got != "Cell with horizontal span" {
		t.Errorf("cell text = %q", got)
	}
	if tab.rows[0].cells[0].colSpan != 3 {
		t.Errorf("Cell1 colSpan = %d, want 3", tab.rows[0].cells[0].colSpan)
	}
	if tab.rows[0].cells[3].rowSpan != 3 {
		t.Errorf("Cell4 rowSpan = %d, want 3", tab.rows[0].cells[3].rowSpan)
	}
}

func mustTabParse(t *testing.T, o Object) *tabTable {
	t.Helper()
	tab, err := tabParseTable(o)
	if err != nil {
		t.Fatal(err)
	}
	return tab
}

func TestTabParseTableErrors(t *testing.T) {
	if _, err := tabParseTable(Object{Kind: "TextObject", Name: "T"}); err == nil {
		t.Error("non-table object accepted")
	}
	if _, err := tabParseTable(Object{Kind: "TableObject", Name: "Empty"}); err == nil {
		t.Error("columnless/rowless table accepted")
	}
	bad := Object{Kind: "TableObject", Name: "Bad", Children: []Node{
		{Name: "TableColumn", Attrs: []Attr{{"Name", "C"}}},
		{Name: "TableRow", Attrs: []Attr{{"Name", "R"}}, Children: []Node{
			{Name: "TableCell", Attrs: []Attr{{"Name", "C1"}, {"ColSpan", "0"}}},
		}},
	}}
	if _, err := tabParseTable(bad); err == nil {
		t.Error("ColSpan=0 accepted")
	}
}

func TestTabRenderTableCellsWidthsSpans(t *testing.T) {
	rep := loadFile(t, "table.frx")
	got := tabRenderTableHTML(tabFindTable(t, rep, "Table1"))
	for _, want := range []string{
		"Cell with horizontal span",
		"Row with auto-height",
		"Column with auto-width",
		"Cell with embedded object",
		"Cell with vertical span",
		"width:75.6px",
		`colspan="3"`,
		`rowspan="3"`,
		`data-kind="TableObject"`,
		`data-cell="Cell1"`,
	} {
		if !strings.Contains(got, want) {
			t.Errorf("table render missing %q:\n%s", want, got)
		}
	}
}

func TestTabRenderTableNestedObjects(t *testing.T) {
	rep := loadFile(t, "table.frx")
	// Table2/Cell22 embeds a PictureObject with a DataColumn.
	if got := tabRenderTableHTML(tabFindTable(t, rep, "Table2")); !strings.Contains(got, "Picture: Categories.Picture") {
		t.Errorf("nested picture placeholder missing:\n%s", got)
	}
	// Table1/Cell13 embeds an unchecked CheckBoxObject.
	if got := tabRenderTableHTML(tabFindTable(t, rep, "Table1")); !strings.Contains(got, `data-kind="CheckBoxObject"`) {
		t.Errorf("nested checkbox missing:\n%s", got)
	}
	// Table3/Cell23 carries RowSpan="4" (dynamic-columns fixture).
	if got := tabRenderTableHTML(tabFindTable(t, rep, "Table3")); !strings.Contains(got, `rowspan="4"`) {
		t.Errorf("RowSpan=4 missing:\n%s", got)
	}
}

func TestTabRenderTableViaBand(t *testing.T) {
	rep := loadFile(t, "table.frx")
	var band Band
bandLoop:
	for _, p := range rep.Pages {
		for _, b := range p.Bands {
			for _, o := range b.Objects {
				if o.Kind == "TableObject" && o.Name == "Table1" {
					band = Band{Kind: b.Kind, Name: b.Name, Attrs: b.Attrs, Objects: []Object{o}}
					break bandLoop
				}
			}
		}
	}
	got := RenderBand(band)
	for _, want := range []string{"Cell with horizontal span", `colspan="3"`, "<table"} {
		if !strings.Contains(got, want) {
			t.Errorf("band table render missing %q:\n%s", want, got)
		}
	}
}

func TestTabSubreportPlaceholder(t *testing.T) {
	got := tabSubreportHTML(Object{Kind: "SubreportObject", Name: "Subreport1",
		Attrs: objAttrs("Left", "255.15", "Top", "47.25", "Width", "444.15", "Height", "103.95",
			"ReportPage", "Page2", "PrintOnParent", "true")})
	for _, want := range []string{
		`data-kind="SubreportObject"`,
		"Subreport1",
		"Page2",
	} {
		if !strings.Contains(got, want) {
			t.Errorf("subreport placeholder missing %q:\n%s", want, got)
		}
	}
	// Dispatch: SubreportObject must not fall through to the generic fallback.
	got = RenderBand(Band{Kind: "DataBand", Name: "D", Attrs: objAttrs("Width", "718.2", "Height", "100"),
		Objects: []Object{{Kind: "SubreportObject", Name: "S1", Attrs: objAttrs("ReportPage", "Page2")}}})
	if !strings.Contains(got, "S1") || !strings.Contains(got, "Page2") {
		t.Errorf("subreport dispatch missing label:\n%s", got)
	}
}

func TestTabRenderTableFallback(t *testing.T) {
	got := tabRenderTableHTML(Object{Kind: "TableObject", Name: "Broken"})
	if !strings.Contains(got, "TableObject: Broken") {
		t.Errorf("broken table should keep labeled fallback:\n%s", got)
	}
}
