package report

import (
	"bytes"
	"fmt"
	"strings"
	"testing"
)

func pdfTestTableObject() Object {
	col := func(name, width string) Node {
		return Node{Name: "TableColumn", Attrs: objAttrs("Name", name, "Width", width)}
	}
	cell := func(attrs []Attr, objs ...Node) Node {
		return Node{Name: "TableCell", Attrs: attrs, Children: objs}
	}
	row := func(name, height string, cells ...Node) Node {
		attrs := objAttrs("Name", name)
		if height != "" {
			attrs = append(attrs, Attr{"Height", height})
		}
		return Node{Name: "TableRow", Attrs: attrs, Children: cells}
	}
	return Object{
		Kind: "TableObject", Name: "Tab1",
		Attrs: objAttrs("Left", "10", "Top", "10", "Width", "300", "Height", "100",
			"Border.Lines", "All", "Border.Color", "Black"),
		Children: []Node{
			col("C1", "100"),
			col("C2", ""), // flexible: shares the remainder
			row("R1", "",
				cell(objAttrs("Name", "A1", "Text", "red fill", "Fill.Color", "255, 0, 0",
					"Border.Lines", "All", "Border.Color", "Silver", "HorzAlign", "Center",
					"Font", "Segoe UI, 9pt")),
				cell(objAttrs("Name", "A2", "Text", "plain", "Border.Lines", "All"))),
			row("R2", "60",
				cell(objAttrs("Name", "B1", "Text", "span two", "ColSpan", "2",
					"Border.Lines", "All", "HorzAlign", "Right",
					"Font", "Segoe UI, 9pt"))),
		},
	}
}

func TestPDFExportTable(t *testing.T) {
	rep := loadHello(t)
	rep.Pages[0].Bands[0].Objects = append(rep.Pages[0].Bands[0].Objects, pdfTestTableObject())
	out, skipped, err := pdfExportReport(rep)
	if err != nil {
		t.Fatal(err)
	}
	if skipped != 0 {
		t.Fatalf("skipped = %d, want 0", skipped)
	}
	s := string(out)
	for _, want := range []string{
		"1.00 0.00 0.00 rg", // red cell fill
		"(red fill)", "(plain)", "(span two)",
		" RG", // outlines
	} {
		if !strings.Contains(s, want) {
			t.Errorf("pdf missing %q", want)
		}
	}
	// Centered "red fill": column is 100px = 75pt wide, face.Width is points.
	size, face := pdfFont(objAttrs("Font", "Segoe UI, 9pt"))
	tx := 10*pxToPt + (100*pxToPt-face.Width("red fill", size))/2
	if want := fmt.Sprintf("%.2f", tx); !strings.Contains(s, want+" ") && !strings.Contains(s, " "+want) {
		t.Errorf("pdf missing centered x %s", want)
	}
}

func TestPDFExportTableAuto(t *testing.T) {
	rep := loadHello(t)
	tab := pdfTestTableObject()
	// Drop widths and heights: equal split must still render every text.
	for i, c := range tab.Children {
		if c.Name == "TableColumn" {
			tab.Children[i].Attrs = objAttrs("Name", attrVal(c.Attrs, "Name"))
		}
		if c.Name == "TableRow" {
			var kept []Attr
			for _, a := range c.Attrs {
				if a.Name != "Height" {
					kept = append(kept, a)
				}
			}
			tab.Children[i].Attrs = kept
		}
	}
	rep.Pages[0].Bands[0].Objects = append(rep.Pages[0].Bands[0].Objects, tab)
	out, skipped, err := pdfExportReport(rep)
	if err != nil {
		t.Fatal(err)
	}
	if skipped != 0 {
		t.Fatalf("skipped = %d, want 0", skipped)
	}
	for _, want := range []string{"(red fill)", "(plain)", "(span two)"} {
		if !strings.Contains(string(out), want) {
			t.Errorf("pdf missing %q", want)
		}
	}
}

func TestPDFExportTableNonASCII(t *testing.T) {
	rep := loadHello(t)
	tab := pdfTestTableObject()
	tab.Children[3].Children[0].Attrs = objAttrs("Name", "A1", "Text", "สวัสดี")
	rep.Pages[0].Bands[0].Objects = append(rep.Pages[0].Bands[0].Objects, tab)
	_, skipped, err := pdfExportReport(rep)
	if err != nil {
		t.Fatal(err)
	}
	if skipped != 1 {
		t.Fatalf("skipped = %d, want 1", skipped)
	}
}

func TestPDFExportShapes(t *testing.T) {
	rep := loadHello(t)
	rep.Pages[0].Bands[0].Objects = append(rep.Pages[0].Bands[0].Objects,
		Object{Kind: "LineObject", Name: "L1",
			Attrs: objAttrs("Left", "10", "Top", "10", "Width", "100", "Height", "2",
				"Border.Width", "2", "Border.Color", "Red")},
		Object{Kind: "LineObject", Name: "L2",
			Attrs: objAttrs("Left", "10", "Top", "20", "Width", "100", "Height", "50",
				"Diagonal", "true")},
		Object{Kind: "ShapeObject", Name: "S1",
			Attrs: objAttrs("Left", "10", "Top", "80", "Width", "60", "Height", "40",
				"Shape", "Ellipse", "Fill.Color", "Yellow", "Border.Color", "Navy")},
		Object{Kind: "ShapeObject", Name: "S2",
			Attrs: objAttrs("Left", "80", "Top", "80", "Width", "60", "Height", "40",
				"Shape", "Diamond", "Fill.Color", "0, 128, 0")},
		Object{Kind: "ShapeObject", Name: "S3",
			Attrs: objAttrs("Left", "150", "Top", "80", "Width", "60", "Height", "40",
				"Shape", "Triangle", "Border.Color", "Black")},
		Object{Kind: "CheckBoxObject", Name: "CB1",
			Attrs: objAttrs("Left", "10", "Top", "130", "Width", "20", "Height", "20",
				"Checked", "true", "CheckedSymbol", "Cross", "CheckColor", "Green")},
		Object{Kind: "CheckBoxObject", Name: "CB2",
			Attrs: objAttrs("Left", "40", "Top", "130", "Width", "20", "Height", "20")},
		Object{Kind: "PictureObject", Name: "P1",
			Attrs: objAttrs("Left", "70", "Top", "130", "Width", "80", "Height", "40",
				"DataColumn", "Products.Photo")},
	)
	out, skipped, err := pdfExportReport(rep)
	if err != nil {
		t.Fatal(err)
	}
	if skipped != 0 {
		t.Fatalf("skipped = %d, want 0", skipped)
	}
	s := string(out)
	for _, want := range []string{
		"1.00 0.00 0.00 RG",    // red line
		" m\n", " l\n", " c\n", // diagonal, diamond/triangle, ellipse
		"1.00 1.00 0.00 rg",       // yellow ellipse fill
		"Picture: Products.Photo", // placeholder label
	} {
		if !strings.Contains(s, want) {
			t.Errorf("pdf missing %q", want)
		}
	}
}

func TestPDFExportTableFixture(t *testing.T) {
	rep := loadFile(t, "table.frx")
	out, skipped, err := pdfExportReport(rep)
	if err != nil {
		t.Fatal(err)
	}
	if skipped != 0 {
		t.Fatalf("skipped = %d, want 0", skipped)
	}
	s := string(out)
	// Table1 lives directly in Data1; Table2/Table3 sit inside ChildBands,
	// which stay opaque labeled boxes like the HTML preview.
	for _, want := range []string{
		"(STATIC TABLE)", "(Cell with horizontal span)", "(Cell with vertical span)",
		"(Cell with embedded object)", "(Column with auto-width)", "(Row with auto-height)",
		" RG",
	} {
		if !strings.Contains(s, want) {
			t.Errorf("table.frx pdf missing %q", want)
		}
	}
	if strings.Contains(s, "(Category Name)") {
		t.Error("child-band Table2 must stay out of the pdf like the HTML preview")
	}
	if !bytes.Contains(out, []byte("%PDF-1.4")) {
		t.Fatal("missing pdf header")
	}
}

func TestPDFColor(t *testing.T) {
	if r, g, b, ok := pdfColor("110, 145, 190"); !ok || r == 0 || g == 0 || b == 0 {
		t.Errorf("triple = %v,%v,%v,%v", r, g, b, ok)
	}
	if _, _, _, ok := pdfColor("Transparent"); ok {
		t.Error("transparent should not parse")
	}
	if _, _, _, ok := pdfColor("NotAColor"); ok {
		t.Error("unknown name should not parse")
	}
	if r, g, b, ok := pdfColor("Silver"); !ok || r == 0 {
		t.Errorf("silver = %v,%v,%v,%v", r, g, b, ok)
	}
	if r, g, b, ok := pdfColor("WhiteSmoke"); !ok || r < 0.95 || g < 0.95 || b < 0.95 {
		t.Errorf("whitesmoke = %v,%v,%v,%v", r, g, b, ok)
	}
}
