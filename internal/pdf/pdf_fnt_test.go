package pdf

import (
	"bytes"
	"strings"
	"testing"
)

func fntTestFont() *FntEmbeddedFont {
	return &FntEmbeddedFont{
		Tag:       "FNTAAA",
		Name:      "Test",
		Data:      []byte{0, 1, 0, 0, 0, 0},
		FirstChar: 0x41,
		LastChar:  0x42,
		Widths:    []int{600, 550},
		Names:     map[int]string{0x41: "uni0041", 0x42: "uni0042"},
		Unicodes:  map[int]rune{0x41: 'A', 0x42: 0xE01},
		ToUnicode: "1 beginbfchar\n<42> <0E01>\nendbfchar\n",
		Ascent:    800,
		Descent:   -200,
		CapHeight: 700,
		BBox:      [4]float64{0, -200, 600, 800},
	}
}

func TestFntEmbedRoundTrip(t *testing.T) {
	d := New()
	d.AddPage(200, 200)
	h, err := d.FntAddEmbeddedFont(fntTestFont())
	if err != nil {
		t.Fatal(err)
	}
	if err := d.FntText(0, h, 10, 100, 12, "AB"); err != nil {
		t.Fatal(err)
	}
	if err := d.Text(0, 10, 50, 12, Helvetica, "plain"); err != nil {
		t.Fatal(err)
	}
	out := d.Bytes()
	for _, want := range []string{
		"%PDF-1.4",
		"/Subtype /TrueType",
		"/BaseFont /FNTAAA+Test",
		"/FirstChar 65 /LastChar 66",
		"/Widths [600 550]",
		"/Differences [65 /uni0041 /uni0042]",
		"/FontDescriptor",
		"/FontFile2",
		"/ToUnicode",
		"<4142>",
		"(plain)",
		"/Ascent 800.00 /Descent -200.00 /CapHeight 700.00",
		"beginbfchar",
	} {
		if !strings.Contains(string(out), want) {
			t.Fatalf("output missing %q", want)
		}
	}
	if strings.Contains(string(out), "/WinAnsiEncoding") == false {
		t.Fatal("base font encoding lost")
	}
}

func TestFntEmbedValidation(t *testing.T) {
	d := New()
	d.AddPage(100, 100)
	if _, err := d.FntAddEmbeddedFont(nil); err == nil {
		t.Fatal("accepted nil font")
	}
	bad := fntTestFont()
	bad.Data = nil
	if _, err := d.FntAddEmbeddedFont(bad); err == nil {
		t.Fatal("accepted empty data")
	}
	bad = fntTestFont()
	bad.Widths = []int{600}
	if _, err := d.FntAddEmbeddedFont(bad); err == nil {
		t.Fatal("accepted short widths")
	}
	bad = fntTestFont()
	bad.FirstChar, bad.LastChar = 0x42, 0x41
	if _, err := d.FntAddEmbeddedFont(bad); err == nil {
		t.Fatal("accepted inverted range")
	}
	h, err := d.FntAddEmbeddedFont(fntTestFont())
	if err != nil {
		t.Fatal(err)
	}
	if err := d.FntText(9, h, 0, 0, 12, "A"); err == nil {
		t.Fatal("accepted bad page")
	}
	if err := d.FntText(0, h+1, 0, 0, 12, "A"); err == nil {
		t.Fatal("accepted bad handle")
	}
	if err := d.FntText(0, h, 0, 0, 12, " "); err == nil {
		t.Fatal("accepted out-of-range byte")
	}
	if !bytes.HasPrefix(d.Bytes(), []byte("%PDF-1.4")) {
		t.Fatal("missing header")
	}
}
