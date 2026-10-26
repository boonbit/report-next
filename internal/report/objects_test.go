package report

import (
	"strings"
	"testing"
)

func objAttrs(pairs ...string) []Attr {
	var out []Attr
	for i := 0; i+1 < len(pairs); i += 2 {
		out = append(out, Attr{pairs[i], pairs[i+1]})
	}
	return out
}

func TestObjRectOf(t *testing.T) {
	x, y, w, h := objRectOf(objAttrs("Left", "9.45", "Top", "28.35", "Width", "103.95", "Height", "132.3"))
	if x != 9.45 || y != 28.35 || w != 103.95 || h != 132.3 {
		t.Fatalf("rect = %v,%v,%v,%v", x, y, w, h)
	}
	x, y, w, h = objRectOf(objAttrs("Width", "bogus"))
	if x != 0 || y != 0 || w != 0 || h != 0 {
		t.Fatalf("bad/missing rect = %v,%v,%v,%v", x, y, w, h)
	}
}

func TestObjLineHorizontal(t *testing.T) {
	got := objLineHTML(Object{Kind: "LineObject", Name: "Line1",
		Attrs: objAttrs("Left", "28.35", "Width", "463.05", "Border.Width", "2")})
	if !strings.Contains(got, "border-top:2px solid Black") {
		t.Errorf("horizontal rule missing:\n%s", got)
	}
}

func TestObjLineVertical(t *testing.T) {
	got := objLineHTML(Object{Kind: "LineObject", Name: "Line1",
		Attrs: objAttrs("Top", "9.45", "Width", "18.9", "Height", "75.6", "Border.Color", "DarkGray")})
	if !strings.Contains(got, "border-left:1px solid DarkGray") {
		t.Errorf("vertical rule missing:\n%s", got)
	}
}

func TestObjLineDiagonal(t *testing.T) {
	got := objLineHTML(Object{Kind: "LineObject", Name: "Line1",
		Attrs: objAttrs("Left", "132.3", "Top", "113.4", "Width", "94.5", "Height", "94.5", "Diagonal", "true")})
	if !strings.Contains(got, "<line") || !strings.Contains(got, "<svg") {
		t.Errorf("diagonal line missing svg:\n%s", got)
	}
}

func TestObjShapeRectangle(t *testing.T) {
	got := objShapeHTML(Object{Kind: "ShapeObject", Name: "Shape1",
		Attrs: objAttrs("Left", "85.05", "Top", "444.15", "Width", "94.5", "Height", "94.5", "Border.Width", "2")})
	if !strings.Contains(got, "<rect") || !strings.Contains(got, "<svg") {
		t.Errorf("rectangle missing:\n%s", got)
	}
}

func TestObjShapeEllipse(t *testing.T) {
	got := objShapeHTML(Object{Kind: "ShapeObject", Name: "S",
		Attrs: objAttrs("Width", "94.5", "Height", "94.5", "Shape", "Ellipse", "Fill.Color", "Orange")})
	if !strings.Contains(got, "<ellipse") || !strings.Contains(got, "Orange") {
		t.Errorf("ellipse missing:\n%s", got)
	}
}

func TestObjShapeDiamond(t *testing.T) {
	got := objShapeHTML(Object{Kind: "ShapeObject", Name: "Shape5",
		Attrs: objAttrs("Width", "56.7", "Height", "56.7", "Shape", "Diamond", "Fill.Color", "MediumSeaGreen")})
	if !strings.Contains(got, "<polygon") || !strings.Contains(got, "MediumSeaGreen") {
		t.Errorf("diamond missing:\n%s", got)
	}
}

func TestObjShapeUnknownFallback(t *testing.T) {
	got := objShapeHTML(Object{Kind: "ShapeObject", Name: "S",
		Attrs: objAttrs("Shape", "Nope")})
	if !strings.Contains(got, "ShapeObject: S") {
		t.Errorf("unknown shape fallback missing:\n%s", got)
	}
}

func TestObjCheckBoxUncheckedDefault(t *testing.T) {
	for _, attrs := range [][]Attr{
		objAttrs("Left", "28.35", "Top", "66.15", "Width", "18.9", "Height", "18.9", "Border.Lines", "All"),
		objAttrs("Width", "18.9", "Height", "18.9", "Checked", "false", "DataColumn", "Products.Discontinued"),
	} {
		got := objCheckBoxHTML(Object{Kind: "CheckBoxObject", Name: "CheckBox1", Attrs: attrs})
		if strings.Contains(got, "10003") || strings.Contains(got, "10005") {
			t.Errorf("unchecked box shows a mark:\n%s", got)
		}
		if !strings.Contains(got, `data-kind="CheckBoxObject"`) {
			t.Errorf("checkbox kind missing:\n%s", got)
		}
	}
}

func TestObjCheckBoxChecked(t *testing.T) {
	got := objCheckBoxHTML(Object{Kind: "CheckBoxObject", Name: "C",
		Attrs: objAttrs("Width", "18.9", "Height", "18.9", "Checked", "true")})
	if !strings.Contains(got, "&#10003;") {
		t.Errorf("checked box missing tick:\n%s", got)
	}
	got = objCheckBoxHTML(Object{Kind: "CheckBoxObject", Name: "C",
		Attrs: objAttrs("Width", "18.9", "Height", "18.9", "Checked", "true", "CheckedSymbol", "Cross", "CheckColor", "Chocolate")})
	if !strings.Contains(got, "&#10005;") || !strings.Contains(got, "Chocolate") {
		t.Errorf("cross box missing mark/color:\n%s", got)
	}
	got = objCheckBoxHTML(Object{Kind: "CheckBoxObject", Name: "C",
		Attrs: objAttrs("Width", "18.9", "Height", "18.9", "CheckState", "Checked")})
	if !strings.Contains(got, "&#10003;") {
		t.Errorf("CheckState box missing tick:\n%s", got)
	}
}

func TestObjPictureDataColumn(t *testing.T) {
	got := objPictureHTML(Object{Kind: "PictureObject", Name: "Picture1",
		Attrs: objAttrs("Left", "9.45", "Top", "28.35", "Width", "103.95", "Height", "132.3",
			"DataColumn", "Order Details.Orders.Employees.Photo")})
	if !strings.Contains(got, "Picture: Order Details.Orders.Employees.Photo") {
		t.Errorf("picture placeholder missing DataColumn:\n%s", got)
	}
	if !strings.Contains(got, "no data binding yet") {
		t.Errorf("picture placeholder not honest:\n%s", got)
	}
}

func TestObjCode39SentinelAndLength(t *testing.T) {
	p := objCode39Pattern("A1")
	if !strings.HasPrefix(p, objCode39Sentinel) || !strings.HasSuffix(p, objCode39Sentinel) {
		t.Fatalf("pattern missing start/stop sentinel: %q", p)
	}
	// 10 chars per encodable char (9 + gap) plus 10-char start and 9-char stop.
	if want := 10*2 + 19; len(p) != want {
		t.Fatalf("pattern len = %d, want %d (%q)", len(p), want, p)
	}
	// Lowercase and punctuation outside the table are skipped like FastReport.
	if got := len(objCode39Pattern("a!")); got != 19 {
		t.Fatalf("skipped-char pattern len = %d, want 19", got)
	}
}

func TestObjCode39BarCount(t *testing.T) {
	// Every character (text + 2 sentinels) contributes exactly 5 bars.
	for _, text := range []string{"", "1", "12", "ABC-123"} {
		n := 0
		for _, r := range text {
			if _, ok := objCode39Data[string(r)]; ok {
				n++
			}
		}
		svg := objCode39SVG(objCode39Pattern(text), 2)
		if got := strings.Count(svg, "<rect"); got != 5*(n+2) {
			t.Errorf("text %q: %d rects, want %d", text, got, 5*(n+2))
		}
	}
}

func TestObjBarcodeRendersSVG(t *testing.T) {
	got := objBarcodeHTML(Object{Kind: "BarcodeObject", Name: "Barcode23",
		Attrs: objAttrs("Left", "18.9", "Top", "170.1", "Width", "145", "Height", "56.7",
			"Fill.Color", "White", "Text", "123456")})
	if !strings.Contains(got, "<svg") || !strings.Contains(got, "123456") {
		t.Errorf("barcode missing svg/caption:\n%s", got)
	}
	if want := 5 * (6 + 2); strings.Count(got, "<rect") != want {
		t.Errorf("barcode rects = %d, want %d", strings.Count(got, "<rect"), want)
	}
}

func TestObjBarcodeQRCodeRenders(t *testing.T) {
	got := objBarcodeHTML(Object{Kind: "BarcodeObject", Name: "Barcode61",
		Attrs: objAttrs("Width", "113.4", "Height", "113.4", "Text", "http://fast-report.com",
			"Barcode", "QR Code", "Barcode.ErrorCorrection", "L", "ShowText", "false")})
	if !strings.Contains(got, "<svg") {
		t.Errorf("QR code should render svg:\n%s", got)
	}
	if strings.Contains(got, "barcode-text") {
		t.Errorf("ShowText=false must hide caption:\n%s", got)
	}
}

func TestObjBarcodePDF417Renders(t *testing.T) {
	got := objBarcodeHTML(Object{Kind: "BarcodeObject", Name: "Barcode59",
		Attrs: objAttrs("Left", "37.8", "Top", "245.7", "Width", "170.1", "Height", "113.4",
			"AutoSize", "false", "ShowText", "false", "Barcode", "PDF417",
			"Barcode.ErrorCorrection", "Auto", "Text", "123456")})
	if !strings.Contains(got, "<svg") {
		t.Errorf("pdf417 should render svg:\n%s", got)
	}
	if strings.Contains(got, "barcode-text") {
		t.Errorf("ShowText=false must hide caption:\n%s", got)
	}
	// Default caption + numeric level.
	got = objBarcodeHTML(Object{Kind: "BarcodeObject", Name: "PDF2",
		Attrs: objAttrs("Width", "170.1", "Height", "113.4", "Text", "AB",
			"Barcode", "PDF417", "Barcode.ErrorCorrection", "5")})
	if !strings.Contains(got, "<svg") || !strings.Contains(got, "barcode-text") {
		t.Errorf("pdf417 default should render svg with caption:\n%s", got)
	}
	// Bad level keeps the labeled fallback.
	got = objBarcodeHTML(Object{Kind: "BarcodeObject", Name: "PDF3",
		Attrs: objAttrs("Width", "170.1", "Height", "113.4", "Text", "AB",
			"Barcode", "PDF417", "Barcode.ErrorCorrection", "99")})
	if strings.Contains(got, "<svg") || !strings.Contains(got, "PDF417") {
		t.Errorf("bad pdf417 level should fall back:\n%s", got)
	}
}

func TestObjBarcodeAztecRenders(t *testing.T) {
	got := objBarcodeHTML(Object{Kind: "BarcodeObject", Name: "BarcodeAztec",
		Attrs: objAttrs("Left", "538.65", "Top", "415.8", "Width", "113.4", "Height", "113.4",
			"AutoSize", "false", "ShowText", "false", "Barcode", "Aztec",
			"Barcode.ErrorCorrection", "33", "Text", "tel:(71) 555-4444")})
	if !strings.Contains(got, "<svg") {
		t.Errorf("aztec should render svg:\n%s", got)
	}
	if strings.Contains(got, "barcode-text") {
		t.Errorf("ShowText=false must hide caption:\n%s", got)
	}
	// Default ECC + caption.
	got = objBarcodeHTML(Object{Kind: "BarcodeObject", Name: "Aztec2",
		Attrs: objAttrs("Width", "113.4", "Height", "113.4", "Text", "AB",
			"Barcode", "Aztec")})
	if !strings.Contains(got, "<svg") || !strings.Contains(got, "barcode-text") {
		t.Errorf("aztec default should render svg with caption:\n%s", got)
	}
	// Bad percent keeps the labeled fallback.
	got = objBarcodeHTML(Object{Kind: "BarcodeObject", Name: "Aztec3",
		Attrs: objAttrs("Width", "113.4", "Height", "113.4", "Text", "AB",
			"Barcode", "Aztec", "Barcode.ErrorCorrection", "0")})
	if strings.Contains(got, "<svg") || !strings.Contains(got, "Aztec") {
		t.Errorf("bad aztec percent should fall back:\n%s", got)
	}
	// MaxiCode renders with the finder overlay.
	got = objBarcodeHTML(Object{Kind: "BarcodeObject", Name: "BarcodeMaxi",
		Attrs: objAttrs("Left", "10", "Top", "10", "Width", "113.4", "Height", "113.4",
			"ShowText", "false", "Text", "HELLO WORLD 123", "Barcode", "MaxiCode",
			"Barcode.Mode", "4")})
	if !strings.Contains(got, "<svg") || !strings.Contains(got, "<circle") {
		t.Errorf("maxicode should render svg with finder:\n%s", got)
	}
	if strings.Contains(got, "barcode-text") {
		t.Errorf("ShowText=false must hide caption:\n%s", got)
	}
	// Default mode + caption.
	got = objBarcodeHTML(Object{Kind: "BarcodeObject", Name: "Maxi2",
		Attrs: objAttrs("Width", "113.4", "Height", "113.4", "Text", "AB",
			"Barcode", "MaxiCode")})
	if !strings.Contains(got, "<svg") || !strings.Contains(got, "barcode-text") {
		t.Errorf("maxicode default should render svg with caption:\n%s", got)
	}
	// Bad mode keeps the labeled fallback.
	got = objBarcodeHTML(Object{Kind: "BarcodeObject", Name: "Maxi3",
		Attrs: objAttrs("Width", "113.4", "Height", "113.4", "Text", "AB",
			"Barcode", "MaxiCode", "Barcode.Mode", "9")})
	if strings.Contains(got, "<svg") || !strings.Contains(got, "MaxiCode") {
		t.Errorf("bad maxicode mode should fall back:\n%s", got)
	}
}

func TestObjBarcodeDataMatrixRenders(t *testing.T) {
	got := objBarcodeHTML(Object{Kind: "BarcodeObject", Name: "Barcode60",
		Attrs: objAttrs("Left", "538.65", "Top", "75.6", "Width", "113.4", "Height", "113.4",
			"AutoSize", "false", "ShowText", "false", "Barcode", "Datamatrix",
			"Barcode.SymbolSize", "Size24x24", "Text", "123456")})
	if !strings.Contains(got, `<svg viewBox="0 0 24 24"`) {
		t.Errorf("datamatrix should render a 24x24 svg:\n%s", got)
	}
	if strings.Contains(got, "barcode-text") {
		t.Errorf("ShowText=false must hide caption:\n%s", got)
	}
	// Default: caption shown, smallest symbol that fits.
	got = objBarcodeHTML(Object{Kind: "BarcodeObject", Name: "BarcodeDM2",
		Attrs: objAttrs("Width", "113.4", "Height", "113.4", "Text", "AB",
			"Barcode", "Datamatrix")})
	if !strings.Contains(got, `<svg viewBox="0 0 10 10"`) || !strings.Contains(got, "barcode-text") {
		t.Errorf("datamatrix default should render 10x10 svg with caption:\n%s", got)
	}
	// Oversize data keeps the labeled fallback instead of erroring.
	// (10x10 holds 3 data codewords; 5 digit-pairs need a bigger symbol.)
	got = objBarcodeHTML(Object{Kind: "BarcodeObject", Name: "BarcodeDM3",
		Attrs: objAttrs("Width", "113.4", "Height", "113.4", "Text", "1234567890",
			"Barcode", "Datamatrix", "Barcode.SymbolSize", "Size10x10")})
	if strings.Contains(got, "<svg") || !strings.Contains(got, "Datamatrix") {
		t.Errorf("oversize datamatrix should fall back:\n%s", got)
	}
}

func TestObjDataMatrixSize(t *testing.T) {
	for in, want := range map[string]int{
		"Size24x24": 24, "size10x10": 10, "": 0, "Auto": 0, "Size24x26": 0, "24": 0,
	} {
		if got := objDataMatrixSize(in); got != want {
			t.Errorf("objDataMatrixSize(%q) = %d, want %d", in, got, want)
		}
	}
}

func TestObjUnknownKindFallback(t *testing.T) {
	b := Band{Kind: "DataBand", Name: "Data1", Attrs: objAttrs("Width", "718.2", "Height", "100"),
		Objects: []Object{{Kind: "GaugeObject", Name: "Gauge1",
			Attrs: objAttrs("Left", "9.45", "Width", "100", "Height", "50")}}}
	got := RenderBand(b)
	if !strings.Contains(got, "GaugeObject: Gauge1") || !strings.Contains(got, `data-kind="GaugeObject"`) {
		t.Errorf("unknown-kind fallback missing:\n%s", got)
	}
}
