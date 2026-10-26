package report

import (
	"strings"
	"testing"
)

// Parent wiring tests for R2 (bound checkbox/picture in engEvalBand) and R3
// (Code128/EAN-13/UPC-A dispatch in HTML and PDF renderers).

func TestWireBarcodeLinearDispatch(t *testing.T) {
	mk := func(kind, text string) Object {
		return Object{Kind: "BarcodeObject", Name: "B",
			Attrs: []Attr{{Name: "Barcode", Value: kind}, {Name: "Text", Value: text},
				{Name: "Left", Value: "0"}, {Name: "Top", Value: "0"},
				{Name: "Width", Value: "200"}, {Name: "Height", Value: "60"}}}
	}
	svg128, ok := objBarcodeLinearSVG(mk("Code128", "HELLO128"), "HELLO128")
	if !ok || !strings.Contains(svg128, "<rect") {
		t.Fatalf("Code128 not dispatched: ok=%v", ok)
	}
	svgEan, ok := objBarcodeLinearSVG(mk("EAN13", "5901234123457"), "5901234123457")
	if !ok || !strings.Contains(svgEan, `viewBox="0 0 95 1"`) {
		t.Fatalf("EAN13 modules != 95: ok=%v", ok)
	}
	svgUpc, ok := objBarcodeLinearSVG(mk("UPC-A", "036000291452"), "036000291452")
	if !ok || !strings.Contains(svgUpc, `viewBox="0 0 95 1"`) {
		t.Fatalf("UPC-A modules != 95: ok=%v", ok)
	}
	if _, ok := objBarcodeLinearSVG(mk("Code93", "ABC"), "ABC"); ok {
		t.Fatalf("Code93 should keep the legacy path")
	}
	if _, ok := objBarcodeLinearSVG(mk("Code128", "\x01"), "\x01"); ok {
		t.Fatalf("undecodable Code128 content should keep the legacy path")
	}
	html := objBarcodeHTML(mk("EAN13", "5901234123457"))
	if !strings.Contains(html, `viewBox="0 0 95 1"`) || !strings.Contains(html, "5901234123457") {
		t.Fatalf("EAN13 HTML missing symbol or caption: %s", html)
	}
}

func TestWireBoundCheckBoxAndPicture(t *testing.T) {
	fields := map[string]string{"Country": "Germany"}
	b := Band{Kind: "DataBand", Name: "Data1",
		Attrs: []Attr{{Name: "Width", Value: "718.2"}, {Name: "Height", Value: "60"}},
		Objects: []Object{
			{Kind: "CheckBoxObject", Name: "C1",
				Attrs: []Attr{{Name: "DataColumn", Value: "Customers.Country"}}},
			{Kind: "CheckBoxObject", Name: "C2",
				Attrs: []Attr{{Name: "Checked", Value: "false"}}},
		}}
	got := engEvalBand(b, fields, nil)
	if attrVal(got.Objects[0].Attrs, "Checked") != "true" {
		t.Fatalf("bound checkbox not checked: %+v", got.Objects[0].Attrs)
	}
	if attrVal(got.Objects[1].Attrs, "Checked") != "false" {
		t.Fatalf("static checkbox changed: %+v", got.Objects[1].Attrs)
	}
	if html := RenderBand(got); !strings.Contains(html, "&#10003;") {
		t.Fatalf("checked box missing tick: %s", html)
	}
	// 1x1 PNG resolves to a data URI; junk stays a placeholder.
	png1 := "iVBORw0KGgoAAAANSUhEUgAAAAEAAAABCAYAAAAfFcSJAAAADUlEQVR42mP8z8BQDwAEhQGAhKmMIQAAAABJRU5ErkJggg=="
	pb := Band{Kind: "DataBand", Name: "Data1",
		Objects: []Object{{Kind: "PictureObject", Name: "P1",
			Attrs: []Attr{{Name: "DataColumn", Value: "Photo"}}}}}
	pg := engEvalBand(pb, map[string]string{"Photo": png1}, nil)
	if !strings.HasPrefix(attrVal(pg.Objects[0].Attrs, "DataURI"), "data:image/png;base64,") {
		t.Fatalf("picture DataURI not resolved")
	}
	pj := engEvalBand(pb, map[string]string{"Photo": "not-an-image"}, nil)
	if attrVal(pj.Objects[0].Attrs, "DataURI") != "" {
		t.Fatalf("junk picture should not resolve")
	}
	if html := RenderBand(pg); !strings.Contains(html, "<img") {
		t.Fatalf("bound picture missing img: %s", html)
	}
}
