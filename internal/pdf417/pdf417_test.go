package pdf417

import (
	"strings"
	"testing"
)

// TestPdfGoldenAB locks the full module matrix for "AB" at security level
// 2, verified byte-identical against the upstream boombuler writer.
func TestPdfGoldenAB(t *testing.T) {
	c, err := Encode([]byte("AB"), 2)
	if err != nil {
		t.Fatal(err)
	}
	if c.Cols() != 3 || c.Rows() != 4 || c.Width() != 120 {
		t.Fatalf("geometry = %dx%d w=%d, want 3x4 w=120", c.Cols(), c.Rows(), c.Width())
	}
	want := []string{
		"ff5475703d4f1eaf086327d5f3fa29",
		"ff547d462fd613e2ef0a17eae3fa29",
		"ff5454f0278858a3ecf856a3e3fa29",
		"ff546bdf24e719a20ef346bcfbfa29",
	}
	m := c.Matrix()
	if len(m) != len(want) {
		t.Fatalf("rows = %d, want %d", len(m), len(want))
	}
	for y, row := range m {
		if got := pdfBitsHex(row); got != want[y] {
			t.Fatalf("row %d = %s, want %s", y, got, want[y])
		}
	}
}

func pdfBitsHex(row []bool) string {
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

// TestPdfStartStop validates the fixed frame of every row: the start word
// opens with 8 dark modules + light, the row ends with the stop word, and
// the left/right indicator codewords sit in slots 1 and -2 decoding back
// to the row-indicator values.
func TestPdfStartStop(t *testing.T) {
	for _, p := range []struct {
		data string
		lvl  int
	}{{"A", 0}, {"AB", 2}, {"Hello, World! 123", 2}, {"01234567890123456789", 5}} {
		c, err := Encode([]byte(p.data), p.lvl)
		if err != nil {
			t.Fatalf("%q: %v", p.data, err)
		}
		if want := (c.Cols()+4)*17 + 1; c.Width() != want {
			t.Fatalf("%q width = %d, want %d", p.data, c.Width(), want)
		}
		for y := 0; y < c.Rows(); y++ {
			row := c.Matrix()[y]
			if len(row) != c.Width() {
				t.Fatalf("%q row %d len = %d", p.data, y, len(row))
			}
			for x := 0; x < 8; x++ {
				if !row[x] {
					t.Fatalf("%q row %d: start bit %d not dark", p.data, y, x)
				}
			}
			if row[8] {
				t.Fatalf("%q row %d: start bit 8 not light", p.data, y)
			}
			for i := 0; i < 18; i++ {
				if want := stop_word>>(17-uint(i))&1 == 1; row[c.Width()-18+i] != want {
					t.Fatalf("%q row %d: stop bit %d mismatch", p.data, y, i)
				}
			}
			table := y % 3
			left := pdfRowWord(row, 1)
			if pdfDecode(table, left) != pdfLeftCodeword(y, c.Rows(), c.Cols(), p.lvl) {
				t.Fatalf("%q row %d: left indicator = %d", p.data, y, pdfDecode(table, left))
			}
			right := pdfRowWord(row, c.Cols()+2)
			if pdfDecode(table, right) != pdfRightCodeword(y, c.Rows(), c.Cols(), p.lvl) {
				t.Fatalf("%q row %d: right indicator = %d", p.data, y, pdfDecode(table, right))
			}
		}
		if !strings.HasPrefix(c.SVG(), "<svg") || !strings.Contains(c.SVG(), "<rect") {
			t.Fatalf("%q svg malformed", p.data)
		}
	}
}

// pdfRowWord reads the 17-bit codeword at 1-based slot s (slot 0 = start).
func pdfRowWord(row []bool, s int) int {
	w := 0
	for i := 0; i < 17; i++ {
		w <<= 1
		if row[s*17+i] {
			w |= 1
		}
	}
	return w
}

// pdfDecode inverts the cluster table.
func pdfDecode(table, word int) int {
	for i, cw := range codewords[table] {
		if cw == word {
			return i
		}
	}
	return -1
}

func TestPdfLevels(t *testing.T) {
	for lvl := 0; lvl <= 8; lvl++ {
		c, err := Encode([]byte("level test"), lvl)
		if err != nil {
			t.Fatalf("level %d: %v", lvl, err)
		}
		if c.Rows() < 2 {
			t.Fatalf("level %d: rows = %d", lvl, c.Rows())
		}
	}
	if _, err := Encode([]byte("x"), 9); err == nil {
		t.Fatal("level 9 accepted")
	}
	if c, err := Encode([]byte(""), 2); err != nil || c.Rows() == 0 {
		t.Fatalf("empty: %v rows=%d", err, c.Rows())
	}
}
