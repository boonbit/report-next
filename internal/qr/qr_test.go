package qr

import (
	"encoding/hex"
	"errors"
	"strings"
	"testing"
)

// TestQrGoldenMatrix pins the exact symbol for one payload. The matrix was
// verified byte-identical to the independent ZXing encoder (gozxing) and
// decodes back to the payload with a third-party reader, so any future
// change in framing, EC, placement, masking or format bits fails loudly
// here instead of shipping a subtly wrong code.
func TestQrGoldenMatrix(t *testing.T) {
	rows := []string{
		"fe853f80", "82fca080", "ba31ae80", "badbae80", "ba732e80",
		"8214a080", "feaabf80", "00b10000", "b7642580", "cd465200",
		"cf54d000", "1570af00", "ab9bef80", "545b7f80", "4b74eb00",
		"b0533080", "270afb80", "00a28c00", "fe86ab80", "82968800",
		"ba46f800", "baa07380", "baa27900", "821e5200", "feb12780",
	}
	c, err := QrEncode([]byte("Hello, World! 123"))
	if err != nil {
		t.Fatal(err)
	}
	if c.Size() != 25 || len(rows) != 25 {
		t.Fatalf("size = %d", c.Size())
	}
	for y, h := range rows {
		raw, err := hex.DecodeString(h)
		if err != nil {
			t.Fatal(err)
		}
		for x := 0; x < 25; x++ {
			want := raw[x/8]>>(7-uint(x%8))&1 == 1
			if got := c.At(x, y); got != want {
				t.Fatalf("golden mismatch at %d,%d", x, y)
			}
		}
	}
}

// Block table must agree with total codewords for every version and level.
func TestQrBlockTableTotals(t *testing.T) {
	for v := 1; v <= 40; v++ {
		for _, lvl := range []QrLevel{QrL, QrM, QrQ, QrH} {
			total, data := 0, 0
			for _, g := range qrBlockGroups(v, lvl) {
				if g.count <= 0 || g.total <= 0 || g.data <= 0 || g.data >= g.total {
					t.Fatalf("v%d level %s: bad group %+v", v, lvl.QrString(), g)
				}
				total += g.count * g.total
				data += g.count * g.data
			}
			if total != qrTotalCodewords[v-1] {
				t.Fatalf("v%d level %s: block total %d != %d", v, lvl.QrString(), total, qrTotalCodewords[v-1])
			}
			if data != qrDataCodewords(v, lvl) {
				t.Fatalf("v%d level %s: inconsistent data count", v, lvl.QrString())
			}
		}
	}
}

// Reed-Solomon: data+EC must be a codeword (zero remainder) for every
// version/level block shape.
func TestQrReedSolomon(t *testing.T) {
	for v := 1; v <= 40; v++ {
		for _, lvl := range []QrLevel{QrL, QrM, QrQ, QrH} {
			for _, g := range qrBlockGroups(v, lvl) {
				gen := qrRSGenerator(g.total - g.data)
				data := make([]byte, g.data)
				for i := range data {
					data[i] = byte((i*37 + v) & 0xFF)
				}
				ec := qrRSRemainder(data, gen)
				if len(ec) != g.total-g.data {
					t.Fatalf("v%d %s: ec length %d", v, lvl.QrString(), len(ec))
				}
				code := append(append([]byte(nil), data...), ec...)
				if rem := qrRSRemainder(code, gen); !isZero(rem) {
					t.Fatalf("v%d %s: nonzero syndrome", v, lvl.QrString())
				}
			}
		}
	}
}

func isZero(b []byte) bool {
	for _, x := range b {
		if x != 0 {
			return false
		}
	}
	return true
}

// Spec known answers: format M/mask0 and version 7 info bits.
func TestQrKnownAnswerCodes(t *testing.T) {
	if got := qrFormatBits(QrM, 0); got != 0x5412 {
		t.Fatalf("format M/mask0 = %04X, want 5412", got)
	}
	if got := qrVersionBits(7); got != 0x07C94 {
		t.Fatalf("version 7 bits = %05X, want 07C94", got)
	}
	// Known byte capacities: V1-L holds 17 bytes, V40-H holds 1273 bytes.
	if !qrFits(1, QrL, 17) || qrFits(1, QrL, 18) {
		t.Fatal("V1-L byte capacity must be exactly 17")
	}
	if !qrFits(40, QrH, 1273) || qrFits(40, QrH, 1274) {
		t.Fatal("V40-H byte capacity must be exactly 1273")
	}
}

func qrMustEncode(t *testing.T, data []byte, lvl QrLevel) *QrCode {
	t.Helper()
	c, err := QrEncodeWithLevel(data, lvl)
	if err != nil {
		t.Fatalf("encode %d bytes level %s: %v", len(data), lvl.QrString(), err)
	}
	return c
}

// Finder, separator, timing and dark-module geometry.
func TestQrFunctionPatterns(t *testing.T) {
	c := qrMustEncode(t, []byte("http://fast-report.com"), QrL)
	n := c.Size()
	at := func(x, y int) bool { return c.At(x, y) }
	// Top-left finder: 7x7 border dark, inner ring light, 3x3 center dark.
	for i := 0; i < 7; i++ {
		if !at(i, 0) || !at(0, i) || !at(6, i) || !at(i, 6) {
			t.Fatalf("finder border light at edge %d", i)
		}
	}
	if at(1, 1) != false || at(1, 2) != false || at(2, 1) != false {
		t.Fatal("finder inner ring must be light")
	}
	for y := 2; y <= 4; y++ {
		for x := 2; x <= 4; x++ {
			if !at(x, y) {
				t.Fatalf("finder center light at %d,%d", x, y)
			}
		}
	}
	// Separators around top-left finder.
	for i := 0; i < 8; i++ {
		if at(7, i) {
			t.Fatalf("separator dark at 7,%d", i)
		}
		if at(i, 7) {
			t.Fatalf("separator dark at %d,7", i)
		}
	}
	// Other finders mirrored.
	if !at(n-1, 0) || !at(n-4, 3) || !at(0, n-1) || !at(3, n-4) {
		t.Fatal("top-right/bottom-left finders misplaced")
	}
	// Timing row 6 / column 6 alternate starting dark, between finders.
	for i := 8; i < n-8; i++ {
		if at(6, i) != (i%2 == 0) {
			t.Fatalf("timing column wrong at row %d", i)
		}
		if at(i, 6) != (i%2 == 0) {
			t.Fatalf("timing row wrong at col %d", i)
		}
	}
	// Dark module at (8, 4v+9).
	v := c.QrVersion()
	if !at(8, 4*v+9) {
		t.Fatal("dark module missing")
	}
}

// Size follows 21+4*(v-1).
func TestQrSize(t *testing.T) {
	for _, lvl := range []QrLevel{QrL, QrM, QrQ, QrH} {
		c := qrMustEncode(t, []byte("hi"), lvl)
		if c.Size() != 21+4*(c.QrVersion()-1) {
			t.Fatalf("level %s: size %d vs version %d", lvl.QrString(), c.Size(), c.QrVersion())
		}
	}
	if c := qrMustEncode(t, make([]byte, 200), QrL); c.Size() != 21+4*(c.QrVersion()-1) {
		t.Fatal("larger payload size mismatch")
	}
}

// Every module is covered: function cells plus exactly data+remainder data cells.
func TestQrFullCoverage(t *testing.T) {
	for v := 1; v <= 10; v++ {
		for _, lvl := range []QrLevel{QrL, QrM, QrQ, QrH} {
			// Largest payload fitting version v (capacities grow per version).
			n := 0
			for qrFits(v, lvl, n+1) {
				n++
			}
			c, err := QrEncodeWithLevel(make([]byte, n), lvl)
			if err != nil {
				t.Fatalf("v%d %s: %v", v, lvl.QrString(), err)
			}
			if c.QrVersion() != v {
				t.Fatalf("payload %d picked v%d, want v%d", n, c.QrVersion(), v)
			}
			m := newQrMatrix(v)
			qrDrawFinders(m)
			qrDrawTiming(m)
			qrDrawAlignment(m)
			qrDrawDarkModule(m)
			qrReserveFormat(m)
			if v >= 7 {
				qrReserveVersion(m)
			}
			funcCells := 0
			for y := 0; y < m.size; y++ {
				for x := 0; x < m.size; x++ {
					if m.function[y][x] {
						funcCells++
					}
				}
			}
			dataCells := m.size*m.size - funcCells
			// The final stream holds data + EC codewords; remainder bits stay light.
			want := qrTotalCodewords[v-1]*8 + qrRemainderBits[v-1]
			if dataCells != want {
				t.Fatalf("v%d %s: %d data cells, want %d", v, lvl.QrString(), dataCells, want)
			}
		}
	}
}

// Determinism: same input gives identical modules, mask included.
func TestQrDeterministic(t *testing.T) {
	payloads := [][]byte{
		[]byte("http://fast-report.com"),
		[]byte("BEGIN:VCARD\nFN:A\nEND:VCARD"),
		make([]byte, 100),
	}
	for _, p := range payloads {
		a := qrMustEncode(t, p, QrM)
		b := qrMustEncode(t, p, QrM)
		if a.QrMask() != b.QrMask() || a.Size() != b.Size() {
			t.Fatal("nondeterministic geometry")
		}
		for y := 0; y < a.Size(); y++ {
			for x := 0; x < a.Size(); x++ {
				if a.At(x, y) != b.At(x, y) {
					t.Fatalf("nondeterministic module at %d,%d", x, y)
				}
			}
		}
	}
}

// Overflow errors at every level.
func TestQrOverflow(t *testing.T) {
	for _, lvl := range []QrLevel{QrL, QrM, QrQ, QrH} {
		_, err := QrEncodeWithLevel(make([]byte, 3000), lvl)
		if !errors.Is(err, QrErrTooLong) {
			t.Fatalf("level %s: want QrErrTooLong, got %v", lvl.QrString(), err)
		}
	}
	// Boundary: max V40-L payload fits (2953 bytes), one more fails.
	if _, err := QrEncodeWithLevel(make([]byte, 2953), QrL); err != nil {
		t.Fatalf("2953 bytes at L must fit: %v", err)
	}
	if _, err := QrEncodeWithLevel(make([]byte, 2954), QrL); !errors.Is(err, QrErrTooLong) {
		t.Fatalf("2954 bytes at L must overflow, got %v", err)
	}
}

// Version grows monotonically with payload; mask selection always completes.
func TestQrVersionGrowsAndMask(t *testing.T) {
	prev := 0
	for _, n := range []int{0, 1, 10, 17, 50, 100, 200, 500, 1000} {
		c := qrMustEncode(t, make([]byte, n), QrM)
		if c.QrVersion() < prev {
			t.Fatalf("version shrank at %d bytes", n)
		}
		prev = c.QrVersion()
		if c.QrMask() < 0 || c.QrMask() > 7 {
			t.Fatalf("mask %d out of range", c.QrMask())
		}
	}
	small := qrMustEncode(t, []byte("A"), QrL)
	big := qrMustEncode(t, make([]byte, 500), QrL)
	if big.QrVersion() <= small.QrVersion() {
		t.Fatal("500 bytes must pick a larger version than 1 byte")
	}
	if big.Size() != 21+4*(big.QrVersion()-1) {
		t.Fatal("size/version mismatch on large payload")
	}
}

// SVG output is well-formed and matches dark-module count.
func TestQrSVG(t *testing.T) {
	c := qrMustEncode(t, []byte("http://fast-report.com"), QrL)
	svg := c.SVG()
	if !strings.HasPrefix(svg, "<svg ") || !strings.HasSuffix(svg, "</svg>") {
		t.Fatal("SVG must be a self-contained <svg> document")
	}
	dark := 0
	for y := 0; y < c.Size(); y++ {
		for x := 0; x < c.Size(); x++ {
			if c.At(x, y) {
				dark++
			}
		}
	}
	if got := strings.Count(svg, `<rect x=`); got != dark {
		t.Fatalf("SVG has %d dark rects, want %d", got, dark)
	}
	// QrMatrix returns an equal copy.
	mx := c.QrMatrix()
	if len(mx) != c.Size() || len(mx[0]) != c.Size() {
		t.Fatal("QrMatrix wrong dimensions")
	}
	for y := range mx {
		for x := range mx[y] {
			if mx[y][x] != c.At(x, y) {
				t.Fatalf("QrMatrix mismatch at %d,%d", x, y)
			}
		}
	}
}
