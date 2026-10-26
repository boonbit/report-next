package aztec

import (
	"strings"
	"testing"
)

// TestAzGoldenAB locks the full 15x15 module matrix for "AB" at 33% ECC,
// verified byte-identical against the upstream boombuler writer.
func TestAzGoldenAB(t *testing.T) {
	c, err := Encode([]byte("AB"), 33, 0)
	if err != nil {
		t.Fatal(err)
	}
	if c.Size() != 15 {
		t.Fatalf("size = %d, want 15", c.Size())
	}
	want := []string{
		"073c",
		"2950",
		"700a",
		"3ffc",
		"301a",
		"d7de",
		"d454",
		"955e",
		"d45e",
		"37d0",
		"901c",
		"dffc",
		"8302",
		"4cf6",
		"b56a",
	}
	m := c.Matrix()
	if len(m) != len(want) {
		t.Fatalf("rows = %d, want %d", len(m), len(want))
	}
	for y, row := range m {
		if got := azBitsHex(row); got != want[y] {
			t.Fatalf("row %d = %s, want %s", y, got, want[y])
		}
	}
}

func azBitsHex(row []bool) string {
	out := ""
	var b byte
	k := 0
	emit := func() {
		out += "0123456789abcdef"[b>>4 : b>>4+1]
		out += "0123456789abcdef"[b&0xf : b&0xf+1]
		b, k = 0, 0
	}
	for _, bit := range row {
		b <<= 1
		if bit {
			b |= 1
		}
		k++
		if k == 8 {
			emit()
		}
	}
	if k > 0 {
		b <<= 8 - uint(k)
		emit()
	}
	return out
}

// TestAzStructure validates symbol geometry across sizes: square grids,
// legal compact/full edges, a dark bulls-eye core and a well-formed SVG.
// Payloads span compact, multi-layer and binary-shift paths.
func TestAzStructure(t *testing.T) {
	for _, p := range [][]byte{
		[]byte("A"),
		[]byte("Hello, World! 123"),
		[]byte("0123456789012345678901234567890123456789"),
		[]byte("the quick brown fox jumps over the lazy dog 0123456789 ~ four score"),
		[]byte("tel:(71) 555-4444"),
		[]byte("bin\x00\x01\x02\xffshift"),
		[]byte(""),
	} {
		c, err := Encode(p, 33, 0)
		if err != nil {
			t.Fatalf("%q: %v", p, err)
		}
		n := c.Size()
		if n < 15 || n%2 == 0 {
			t.Fatalf("%q size = %d, want odd >= 15", p, n)
		}
		// Legal edges: compact 11+4L (L=1..4) or full 15+4L+alignment.
		legal := false
		for L := 1; L <= 4; L++ {
			if n == 11+4*L {
				legal = true
			}
		}
		for L := 1; L <= 32; L++ {
			base := 14 + 4*L
			if n == base+1+2*((base/2-1)/15) {
				legal = true
			}
		}
		if !legal {
			t.Fatalf("%q size = %d, not a legal aztec edge", p, n)
		}
		m := c.Matrix()
		if len(m) != n {
			t.Fatalf("%q rows = %d, want %d", p, len(m), n)
		}
		for y, row := range m {
			if len(row) != n {
				t.Fatalf("%q row %d len = %d", p, y, len(row))
			}
		}
		// Bulls-eye core is always dark.
		if !c.At(n/2, n/2) {
			t.Fatalf("%q bulls-eye core not dark", p)
		}
		if !strings.HasPrefix(c.SVG(), "<svg") || !strings.Contains(c.SVG(), "<rect") {
			t.Fatalf("%q svg malformed", p)
		}
	}
}

func TestAzLayers(t *testing.T) {
	c, err := Encode([]byte("AB"), 33, -1)
	if err != nil {
		t.Fatal(err)
	}
	if c.Size() != 15 {
		t.Fatalf("forced compact size = %d, want 15", c.Size())
	}
	c, err = Encode([]byte("AB"), 33, 5)
	if err != nil {
		t.Fatal(err)
	}
	if want := 14 + 4*5 + 1 + 2*((14+4*5)/2-1)/15; c.Size() != want {
		t.Fatalf("forced full size = %d, want %d", c.Size(), want)
	}
	if _, err := Encode([]byte("AB"), 33, 33); err == nil {
		t.Fatal("33 layers accepted")
	}
	if _, err := Encode([]byte("AB"), 33, -5); err == nil {
		t.Fatal("compact 5 accepted")
	}
}
