package pdf

import (
	"strings"
	"testing"
)

func TestDrawPaths(t *testing.T) {
	d := New()
	d.AddPage(100, 100)
	if err := d.FillRectRGB(0, 1, 2, 10, 5, 1, 0, 0); err != nil {
		t.Fatal(err)
	}
	if err := d.StrokeRect(0, 1, 2, 10, 5, 0.75, 0, 0, 1); err != nil {
		t.Fatal(err)
	}
	if err := d.StrokeLine(0, 0, 0, 10, 10, 1, 0, 0, 0); err != nil {
		t.Fatal(err)
	}
	if err := d.Poly(0, [][2]float64{{0, 0}, {5, 5}, {10, 0}}, true, true, true,
		1, 1, 1, 1, 0, 0, 0); err != nil {
		t.Fatal(err)
	}
	if err := d.Ellipse(0, 50, 50, 10, 5, true, false, 0, 1, 0, 0, 0, 0, 0); err != nil {
		t.Fatal(err)
	}
	out := string(d.Bytes())
	for _, want := range []string{
		"1.00 0.00 0.00 rg", // red fill
		"0.00 0.00 1.00 RG", // blue stroke
		" m\n", " l\n", " c\n", "h\n", "S\n", "B\n", "f\n", " re f\n",
	} {
		if !strings.Contains(out, want) {
			t.Errorf("pdf missing %q", want)
		}
	}
}

func TestDrawPathErrors(t *testing.T) {
	d := New()
	d.AddPage(100, 100)
	if err := d.StrokeLine(3, 0, 0, 1, 1, 1, 0, 0, 0); err == nil {
		t.Error("bad page accepted")
	}
	if err := d.Poly(0, nil, false, true, false, 1, 0, 0, 0, 0, 0, 0); err == nil {
		t.Error("empty poly accepted")
	}
	if err := d.Poly(0, [][2]float64{{0, 0}}, false, false, false, 1, 0, 0, 0, 0, 0, 0); err == nil {
		t.Error("fill-less strok-less poly accepted")
	}
	if err := d.Ellipse(0, 1, 1, 0, 1, true, false, 0, 0, 0, 0, 0, 0, 0); err == nil {
		t.Error("zero radius accepted")
	}
}

func TestFillRectGrayStillWorks(t *testing.T) {
	d := New()
	d.AddPage(100, 100)
	if err := d.FillRect(0, 1, 2, 3, 4, 0); err != nil {
		t.Fatal(err)
	}
	if out := string(d.Bytes()); !strings.Contains(out, "0.00 0.00 0.00 rg") {
		t.Errorf("black fill missing:\n%s", out)
	}
}
