package datamatrix

import (
	"fmt"
	"strings"
	"testing"
)

func TestDmFinder(t *testing.T) {
	c, err := Encode([]byte("A"), 0)
	if err != nil {
		t.Fatal(err)
	}
	if c.Size() != 10 {
		t.Fatalf("size = %d, want 10", c.Size())
	}
	// Solid L: left column and bottom row dark.
	for y := 0; y < 10; y++ {
		if !c.At(0, y) {
			t.Fatalf("left finder gap at row %d", y)
		}
	}
	for x := 0; x < 10; x++ {
		if !c.At(x, 9) {
			t.Fatalf("bottom finder gap at col %d", x)
		}
	}
	// Timing: top row and right column alternate, starting dark.
	for x := 0; x < 10; x++ {
		if want := x%2 == 0; c.At(x, 0) != want {
			t.Fatalf("top timing col %d = %v", x, c.At(x, 0))
		}
	}
}

func TestDmCoverage(t *testing.T) {
	for _, p := range []string{"A", "Hello, World! 123", strings.Repeat("0123456789", 30)} {
		c, err := Encode([]byte(p), 0)
		if err != nil {
			t.Fatal(err)
		}
		if c.Size() < 10 {
			t.Fatalf("size %d for %q", c.Size(), p)
		}
	}
}

func TestDmOverflow(t *testing.T) {
	if _, err := Encode(make([]byte, 3000), 0); err == nil {
		t.Fatal("oversize accepted")
	}
	if _, err := Encode([]byte("x"), 11); err == nil {
		t.Fatal("bad size accepted")
	}
}

// TestDmGoldenAB locks the full 10x10 module matrix for "AB",
// verified byte-identical against the gozxing (ZXing port) oracle.
func TestDmGoldenAB(t *testing.T) {
	c, err := Encode([]byte("AB"), 0)
	if err != nil {
		t.Fatal(err)
	}
	m := c.Matrix()
	want := []string{
		"aa80",
		"a7c0",
		"8900",
		"bec0",
		"9a00",
		"b8c0",
		"ae80",
		"d040",
		"a100",
		"ffc0",
	}
	if len(m) != len(want) {
		t.Fatalf("rows = %d, want %d", len(m), len(want))
	}
	for y, row := range m {
		s := ""
		var v byte
		n := 0
		flush := func() {
			s += fmt.Sprintf("%02x", v<<(8-uint(n)))
			v, n = 0, 0
		}
		for _, b := range row {
			v <<= 1
			if b {
				v |= 1
			}
			n++
			if n == 8 {
				flush()
			}
		}
		if n > 0 {
			flush()
		}
		if s != want[y] {
			t.Fatalf("row %d = %s, want %s", y, s, want[y])
		}
	}
}

func TestDmSVG(t *testing.T) {
	c, err := Encode([]byte("AB"), 0)
	if err != nil {
		t.Fatal(err)
	}
	svg := c.SVG()
	if !strings.HasPrefix(svg, "<svg") || !strings.Contains(svg, "<rect") {
		t.Fatal("svg malformed")
	}
}
