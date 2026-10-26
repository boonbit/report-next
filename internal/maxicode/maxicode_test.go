package maxicode

import (
	"strings"
	"testing"
)

// TestMxGoldenAB locks the full 30x33 module grid for "AB" in mode 4,
// verified byte-identical against the upstream ingridhq writer.
func TestMxGoldenAB(t *testing.T) {
	c, err := Encode(4, []byte("AB"))
	if err != nil {
		t.Fatal(err)
	}
	if c.Mode() != 4 {
		t.Fatalf("mode = %d, want 4", c.Mode())
	}
	want := []string{
		"5555555c",
		"00000000",
		"aaaaaaa8",
		"55555558",
		"00000004",
		"aaaaaaa0",
		"55555558",
		"00000008",
		"aaaaaaa4",
		"55b10150",
		"00508c00",
		"aa602aa8",
		"54600558",
		"03000400",
		"ab8012a0",
		"57800958",
		"00800808",
		"a8000ea0",
		"57400d5c",
		"00000408",
		"aa8036ac",
		"55802d50",
		"00a2f80c",
		"aae152a8",
		"55555f5c",
		"00000f58",
		"aaaaa008",
		"ff0f0500",
		"5af0f550",
		"aa055fa8",
		"f5a0f5a8",
		"5af5aaf0",
		"5005aa50",
	}
	m := c.Matrix()
	if len(m) != len(want) || len(m[0]) != 30 {
		t.Fatalf("geometry = %dx%d, want 30x33", len(m[0]), len(m))
	}
	for y, row := range m {
		if got := mxBitsHex(row); got != want[y] {
			t.Fatalf("row %d = %s, want %s", y, got, want[y])
		}
	}
}

func mxBitsHex(row []bool) string {
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

// TestMxOrientation validates the fixed orientation marks shared by every
// symbol plus the finder overlay in the SVG.
func TestMxOrientation(t *testing.T) {
	for _, p := range []struct {
		mode int
		data string
	}{
		{4, "AB"},
		{5, "SHORT"},
		{6, "MODE6-DATA.123"},
	} {
		c, err := Encode(p.mode, []byte(p.data))
		if err != nil {
			t.Fatalf("mode %d: %v", p.mode, err)
		}
		for _, m := range [][2]int{
			{0, 28}, {0, 29},
			{9, 10}, {9, 11}, {10, 11},
			{15, 7}, {16, 8}, {16, 20}, {17, 20},
			{22, 10}, {23, 10}, {22, 17}, {23, 17},
		} {
			if !c.At(m[1], m[0]) {
				t.Fatalf("mode %d: orientation mark (%d,%d) not dark", p.mode, m[1], m[0])
			}
		}
		svg := c.SVG()
		if !strings.HasPrefix(svg, "<svg") || !strings.Contains(svg, "<circle") {
			t.Fatalf("mode %d: svg must carry the finder overlay", p.mode)
		}
	}
}

func TestMxErrors(t *testing.T) {
	if _, err := Encode(1, []byte("x")); err == nil {
		t.Error("mode 1 accepted")
	}
	if _, err := Encode(7, []byte("x")); err == nil {
		t.Error("mode 7 accepted")
	}
	if _, err := Encode(4, make([]byte, 139)); err == nil {
		t.Error("oversize accepted")
	}
	if _, err := Encode(2, []byte("not a carrier message")); err == nil {
		t.Error("bare mode 2 accepted")
	}
	if _, err := Encode(2, []byte("[)>\x1e01\x1dshort")); err == nil {
		t.Error("short mode 2 accepted")
	}
}
