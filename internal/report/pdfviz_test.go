package report

import (
	"strings"
	"testing"
)

func pdfTestBarGauge() Object {
	return Object{Kind: "LinearGauge", Name: "G1",
		Attrs: objAttrs("Left", "10", "Top", "10", "Width", "200", "Height", "40",
			"Value", "15", "Pointer.Fill.Color", "200, 160, 120",
			"Label.Text", "speed", "Label.Font", "Segoe UI, 9pt")}
}

func TestPDFExportBarGauge(t *testing.T) {
	rep := loadHello(t)
	rep.Pages[0].Bands[0].Objects = append(rep.Pages[0].Bands[0].Objects, pdfTestBarGauge())
	out, skipped, err := pdfExportReport(rep)
	if err != nil {
		t.Fatal(err)
	}
	if skipped != 0 {
		t.Fatalf("skipped = %d, want 0", skipped)
	}
	s := string(out)
	// Track, 15%-wide red fill, tick strokes, label.
	for _, want := range []string{
		"0.83 0.83 0.83 rg", "0.78 0.63 0.47 rg", " l\n", "(speed)",
	} {
		if !strings.Contains(s, want) {
			t.Errorf("pdf missing %q", want)
		}
	}
}

func TestPDFExportRadialGauge(t *testing.T) {
	rep := loadHello(t)
	rep.Pages[0].Bands[0].Objects = append(rep.Pages[0].Bands[0].Objects,
		Object{Kind: "RadialGauge", Name: "G2",
			Attrs: objAttrs("Left", "10", "Top", "10", "Width", "120", "Height", "80",
				"Value", "50", "Pointer.Fill.Color", "160, 120, 120",
				"Label.Text", "qty", "Label.Font", "Segoe UI, 9pt")})
	out, skipped, err := pdfExportReport(rep)
	if err != nil {
		t.Fatal(err)
	}
	if skipped != 0 {
		t.Fatalf("skipped = %d, want 0", skipped)
	}
	s := string(out)
	// Dial arc, needle, hub circle, label.
	for _, want := range []string{" l\n", " c\n", "0.63 0.47 0.47 RG", "(qty)"} {
		if !strings.Contains(s, want) {
			t.Errorf("pdf missing %q", want)
		}
	}
}

func TestPDFExportGaugeNonASCIILabel(t *testing.T) {
	rep := loadHello(t)
	g := pdfTestBarGauge()
	for i, a := range g.Attrs {
		if a.Name == "Label.Text" {
			g.Attrs[i].Value = "ความเร็ว"
		}
	}
	rep.Pages[0].Bands[0].Objects = append(rep.Pages[0].Bands[0].Objects, g)
	out, skipped, err := pdfExportReport(rep)
	if err != nil {
		t.Fatal(err)
	}
	if skipped != 1 {
		t.Fatalf("skipped = %d, want 1 for the non-ASCII gauge label", skipped)
	}
	if !strings.Contains(string(out), "0.78 0.63 0.47 rg") {
		t.Fatal("gauge fill should still be drawn")
	}
}

func TestPDFExportGaugeFallback(t *testing.T) {
	rep := loadHello(t)
	rep.Pages[0].Bands[0].Objects = append(rep.Pages[0].Bands[0].Objects,
		Object{Kind: "GaugeObject", Name: "G0",
			Attrs: objAttrs("Left", "10", "Top", "10", "Width", "100", "Height", "40")},
		Object{Kind: "LinearGauge", Name: "G00",
			Attrs: objAttrs("Left", "10", "Top", "60", "Width", "0", "Height", "40",
				"Value", "15")})
	out, skipped, err := pdfExportReport(rep)
	if err != nil {
		t.Fatal(err)
	}
	if skipped != 0 {
		t.Fatalf("skipped = %d, want 0", skipped)
	}
	if strings.Contains(string(out), "0.83 0.83 0.83 rg") {
		t.Error("unknown/zero-size gauges must draw nothing")
	}
}

func pdfTestMatrix() Object {
	col := func(name, width string) Node {
		return Node{Name: "TableColumn", Attrs: objAttrs("Name", name, "Width", width)}
	}
	cell := func(attrs ...string) Node {
		return Node{Name: "TableCell", Attrs: objAttrs(attrs...)}
	}
	return Object{
		Kind: "MatrixObject", Name: "M1",
		Attrs: objAttrs("Left", "10", "Top", "10", "Width", "200", "Height", "80",
			"DataSource", "MatrixDemo"),
		Children: []Node{
			col("C1", "100"),
			{Name: "MatrixColumns", Children: []Node{
				{Name: "Header", Attrs: objAttrs("Expression", "[Year]")},
			}},
			{Name: "TableRow", Attrs: objAttrs("Name", "R1"), Children: []Node{
				cell("Name", "A1", "Text", "Employee", "Fill.Color", "170, 150, 130",
					"Border.Lines", "All", "HorzAlign", "Center", "Font", "Segoe UI, 9pt"),
				cell("Name", "A2", "Text", "[Revenue]", "Fill.Color", "WhiteSmoke",
					"Border.Lines", "All", "HorzAlign", "Right", "Font", "Segoe UI, 9pt"),
			}},
		},
	}
}

func TestPDFExportMatrix(t *testing.T) {
	rep := loadHello(t)
	rep.Pages[0].Bands[0].Objects = append(rep.Pages[0].Bands[0].Objects, pdfTestMatrix())
	out, skipped, err := pdfExportReport(rep)
	if err != nil {
		t.Fatal(err)
	}
	if skipped != 0 {
		t.Fatalf("skipped = %d, want 0", skipped)
	}
	s := string(out)
	for _, want := range []string{
		"0.67 0.59 0.51 rg", "0.96 0.96 0.96 rg", "(Employee)", "([Revenue])", " RG",
	} {
		if !strings.Contains(s, want) {
			t.Errorf("pdf missing %q", want)
		}
	}
}

func TestPDFExportMatrixNoGrid(t *testing.T) {
	rep := loadHello(t)
	rep.Pages[0].Bands[0].Objects = append(rep.Pages[0].Bands[0].Objects,
		Object{Kind: "MatrixObject", Name: "M0",
			Attrs: objAttrs("Left", "10", "Top", "10", "Width", "200", "Height", "80")})
	out, skipped, err := pdfExportReport(rep)
	if err != nil {
		t.Fatal(err)
	}
	if skipped != 0 {
		t.Fatalf("skipped = %d, want 0", skipped)
	}
	if strings.Contains(string(out), " RG") {
		t.Error("gridless matrix must draw nothing")
	}
}

func TestPDFExportSubreport(t *testing.T) {
	rep := loadHello(t)
	rep.Pages[0].Bands[0].Objects = append(rep.Pages[0].Bands[0].Objects,
		Object{Kind: "SubreportObject", Name: "Sub1",
			Attrs: objAttrs("Left", "10", "Top", "10", "Width", "200", "Height", "60",
				"ReportPage", "Page2")})
	out, skipped, err := pdfExportReport(rep)
	if err != nil {
		t.Fatal(err)
	}
	if skipped != 0 {
		t.Fatalf("skipped = %d, want 0", skipped)
	}
	s := string(out)
	for _, want := range []string{"0.50 0.50 0.50 RG", "Subreport: Sub1", "Page: Page2"} {
		if !strings.Contains(s, want) {
			t.Errorf("pdf missing %q", want)
		}
	}
}

func TestPDFExportGaugeFixture(t *testing.T) {
	// Real demo gauges carry no Width/Height (HTML also renders them
	// 0-size); the export must stay green and skip-free on the file.
	rep := loadFile(t, "Gauge.frx")
	_, skipped, err := pdfExportReport(rep)
	if err != nil {
		t.Fatal(err)
	}
	if skipped != 0 {
		t.Fatalf("skipped = %d, want 0", skipped)
	}
}

func TestPDFExportMatrixFixture(t *testing.T) {
	rep := loadFile(t, "Simple Matrix.frx")
	out, skipped, err := pdfExportReport(rep)
	if err != nil {
		t.Fatal(err)
	}
	if skipped != 0 {
		t.Fatalf("skipped = %d, want 0", skipped)
	}
	s := string(out)
	for _, want := range []string{
		"(Employee)", "(Total)", "0.67 0.59 0.51 rg", "0.96 0.96 0.96 rg", " RG",
	} {
		if !strings.Contains(s, want) {
			t.Errorf("simple matrix pdf missing %q", want)
		}
	}
}

func TestPDFExportSubreportFixture(t *testing.T) {
	rep := loadFile(t, "Subreport.frx")
	out, skipped, err := pdfExportReport(rep)
	if err != nil {
		t.Fatal(err)
	}
	if skipped != 0 {
		t.Fatalf("skipped = %d, want 0", skipped)
	}
	if !strings.Contains(string(out), "Subreport: Subreport1") {
		t.Error("subreport placeholder missing from pdf")
	}
}
