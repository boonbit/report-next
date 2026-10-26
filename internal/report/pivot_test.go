package report

import (
	"strings"
	"testing"
)

func pivotFixture() (Object, *DataDictionary) {
	header := func(expr string) Node {
		return Node{Name: "Header", Attrs: []Attr{{"Expression", expr}}}
	}
	o := Object{Kind: "MatrixObject", Name: "Matrix1",
		Attrs: []Attr{{"DataSource", "Sales"}}}
	o.Children = append(o.Children,
		Node{Name: "MatrixColumns", Children: []Node{header("[Sales.Year]")}},
		Node{Name: "MatrixRows", Children: []Node{header("[Sales.Name]")}},
		Node{Name: "MatrixCells", Children: []Node{
			{Name: "Cell", Attrs: []Attr{{"Expression", "[Sales.Revenue]"}}}}})
	dict := &DataDictionary{Sources: []DataSource{{
		Name:    "Sales",
		Columns: []DataColumn{{Name: "Year"}, {Name: "Name"}, {Name: "Revenue"}},
		Rows: []DataRow{
			{"Year": "2023", "Name": "Alice", "Revenue": "100"},
			{"Year": "2023", "Name": "Bob", "Revenue": "200"},
			{"Year": "2024", "Name": "Alice", "Revenue": "150"},
			{"Year": "2024", "Name": "Bob", "Revenue": "50"},
			{"Year": "2024", "Name": "Bob", "Revenue": "25"},
		},
	}}}
	return o, dict
}

func TestVizPivotSums(t *testing.T) {
	o, dict := pivotFixture()
	g, err := vizPivotGrid(o, dict)
	if err != nil {
		t.Fatal(err)
	}
	if len(g.cols) != 2 || g.cols[0] != "2023" || g.cols[1] != "2024" {
		t.Fatalf("cols = %q", g.cols)
	}
	if len(g.rows) != 2 {
		t.Fatalf("rows = %v", g.rows)
	}
	want := map[string]map[string]float64{
		"Alice": {"2023": 100, "2024": 150},
		"Bob":   {"2023": 200, "2024": 75},
	}
	for rk, cols := range want {
		for ck, v := range cols {
			if g.cells[rk][ck] != v {
				t.Errorf("cell %s/%s = %v, want %v", rk, ck, g.cells[rk][ck], v)
			}
		}
	}
	if g.rowTotal["Alice"] != 250 || g.rowTotal["Bob"] != 275 {
		t.Errorf("row totals = %v", g.rowTotal)
	}
	if g.colTotal["2023"] != 300 || g.colTotal["2024"] != 225 || g.grand != 525 {
		t.Errorf("col totals = %v grand = %v", g.colTotal, g.grand)
	}
	if g.counted {
		t.Error("numeric values reported as counted")
	}
}

func TestVizPivotCountsNonNumeric(t *testing.T) {
	o, dict := pivotFixture()
	dict.Sources[0].Columns = append(dict.Sources[0].Columns, DataColumn{Name: "Note"})
	for _, r := range dict.Sources[0].Rows {
		r["Note"] = "x"
	}
	o.Children[2].Children[0].Attrs = []Attr{{"Expression", "[Sales.Note]"}}
	g, err := vizPivotGrid(o, dict)
	if err != nil {
		t.Fatal(err)
	}
	if !g.counted || g.cells["Alice"]["2023"] != 1 || g.grand != 5 {
		t.Errorf("count pivot wrong: %+v", g)
	}
}

func TestVizPivotErrors(t *testing.T) {
	o, dict := pivotFixture()
	if _, err := vizPivotGrid(o, &DataDictionary{}); err == nil {
		t.Error("unknown source accepted")
	}
	bad := o
	bad.Children[0].Children[0].Attrs = []Attr{{"Expression", "Sum([Sales.Year])"}}
	if _, err := vizPivotGrid(bad, dict); err == nil {
		t.Error("function expression accepted")
	}
}

func TestVizPivotEngineRendersSums(t *testing.T) {
	o, dict := pivotFixture()
	rep := exprNewReport("Pivot engine")
	pi := exprAddPage(rep, "Page1")
	bi := exprAddBand(rep, pi, "ReportTitleBand", "Title1")
	rep.Pages[pi].Bands[bi].Objects = append(rep.Pages[pi].Bands[bi].Objects, o)
	doc, err := engPrepare(*rep, dict)
	if err != nil {
		t.Fatal(err)
	}
	html := engRenderDocumentHTML(doc)
	for _, want := range []string{"Alice", "Bob", "2023", "2024", "250", "275", "525"} {
		if !strings.Contains(html, want) {
			t.Errorf("engine pivot html missing %q", want)
		}
	}
}
