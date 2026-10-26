package report

import (
	"strings"
	"testing"
)

// TRACK viz: MatrixObject tests against real attribute names from
// FastReport/Demos/Reports/Simple Matrix.frx (Matrix2) and
// "Sparkline, Gauge.frx" (nested SimpleProgressGauge + skipped buttons).

// vizMatrixFixture mirrors Simple Matrix.frx Matrix2: one column header
// ([MatrixDemo.Year]), one row header ([MatrixDemo.Name]), one data cell
// ([MatrixDemo.Revenue]) and a 3x3 template grid.
func vizMatrixFixture() Object {
	cell := func(name, text string, kids ...Node) Node {
		a := []Attr{{"Name", name}}
		if text != "" {
			a = append(a, Attr{"Text", text})
		}
		return Node{Name: "TableCell", Attrs: a, Children: kids}
	}
	row := func(name string, cells ...Node) Node {
		return Node{Name: "TableRow", Attrs: []Attr{{"Name", name}}, Children: cells}
	}
	return Object{Kind: "MatrixObject", Name: "Matrix2",
		Attrs: objAttrs("Left", "9.45", "Top", "66.15", "Width", "321.3", "Height", "66.15",
			"FixedRows", "1", "FixedColumns", "1", "DataSource", "MatrixDemo"),
		Children: []Node{
			{Name: "MatrixColumns", Children: []Node{
				{Name: "Header", Attrs: objAttrs("Expression", "[MatrixDemo.Year]")},
			}},
			{Name: "MatrixRows", Children: []Node{
				{Name: "Header", Attrs: objAttrs("Expression", "[MatrixDemo.Name]")},
			}},
			{Name: "MatrixCells", Children: []Node{
				{Name: "Cell", Attrs: objAttrs("Expression", "[MatrixDemo.Revenue]")},
			}},
			{Name: "TableColumn", Attrs: objAttrs("Name", "Column5", "Width", "122.85")},
			{Name: "TableColumn", Attrs: objAttrs("Name", "Column6", "Width", "94.5")},
			row("Row8", cell("Cell17", "Employee"), cell("Cell18", "[Year]"), cell("Cell19", "Total")),
			row("Row9", cell("Cell20", "[Name]"), cell("Cell21", "[Revenue]"), cell("Cell22", "")),
			row("Row4", cell("Cell23", "Total"), cell("Cell24", ""), cell("Cell25", "")),
		}}
}

func TestVizMatrixCellTexts(t *testing.T) {
	got := vizMatrixHTML(vizMatrixFixture())
	for _, want := range []string{"Employee", "[Year]", "Total", "[Name]", "[Revenue]",
		"<table", `data-kind="MatrixObject"`, `data-cell="Cell17"`} {
		if !strings.Contains(got, want) {
			t.Errorf("matrix HTML missing %q:\n%s", want, got)
		}
	}
}

func TestVizMatrixDescriptors(t *testing.T) {
	got := vizMatrixHTML(vizMatrixFixture())
	for _, want := range []string{`data-col-expr="[MatrixDemo.Year]"`,
		`data-row-expr="[MatrixDemo.Name]"`, `data-cell-expr="[MatrixDemo.Revenue]"`} {
		if !strings.Contains(got, want) {
			t.Errorf("matrix HTML missing descriptor %q:\n%s", want, got)
		}
	}
}

func TestVizMatrixSkipsHighlight(t *testing.T) {
	o := vizMatrixFixture()
	// Mirror Simple Matrix.frx Cell8: a Highlight/Condition styling node that
	// must not render as a fallback box.
	o.Children[5].Children[2].Children = []Node{{Name: "Highlight", Children: []Node{
		{Name: "Condition", Attrs: objAttrs("Expression", "Value > 3000")},
	}}}
	got := vizMatrixHTML(o)
	if strings.Contains(got, "Highlight") {
		t.Errorf("matrix cell styling node should be skipped:\n%s", got)
	}
	if !strings.Contains(got, "[Revenue]") {
		t.Errorf("matrix cell text lost:\n%s", got)
	}
}

func TestVizMatrixNestedObjects(t *testing.T) {
	o := vizMatrixFixture()
	// Mirror "Sparkline, Gauge.frx" Cell6: a nested gauge plus sort/collapse
	// buttons that must be skipped.
	o.Children[5].Children[1].Children = []Node{
		{Name: "SimpleProgressGauge", Attrs: objAttrs("Name", "SimpleProgressGauge1",
			"Maximum", "1", "Value", "1", "Label.Text", "100%")},
		{Name: "MatrixSortButton", Attrs: objAttrs("Name", "SortButton2")},
	}
	got := vizMatrixHTML(o)
	if !strings.Contains(got, "<svg") || !strings.Contains(got, "100%") {
		t.Errorf("nested gauge should render via synthetic RenderBand:\n%s", got)
	}
	if strings.Contains(got, "SortButton") {
		t.Errorf("matrix sort button should be skipped:\n%s", got)
	}
}

func TestVizMatrixViaRenderBand(t *testing.T) {
	b := Band{Kind: "DataBand", Name: "Data1",
		Attrs:   objAttrs("Width", "718.2", "Height", "141.75"),
		Objects: []Object{vizMatrixFixture()}}
	if got := RenderBand(b); !strings.Contains(got, `data-kind="MatrixObject"`) {
		t.Errorf("RenderBand should dispatch MatrixObject:\n%s", got)
	}
}

func TestVizMatrixEmptyFallback(t *testing.T) {
	got := vizMatrixHTML(Object{Kind: "MatrixObject", Name: "Matrix9"})
	if !strings.Contains(got, "Matrix9") || strings.Contains(got, "<table") {
		t.Errorf("grid-less matrix should be a labeled fallback:\n%s", got)
	}
}
