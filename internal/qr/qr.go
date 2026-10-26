// Package qr is a pure-standard-library QR Code encoder (byte mode).
//
// Wiring note for the report engine (owner wires dispatch later): FastReport
// Barcode.frx uses BarcodeObject elements with Barcode="QR Code" and these
// attributes, which map to this package as follows:
//
//	BarcodeObject Barcode="QR Code" Text="http://fast-report.com"
//	  Barcode.ErrorCorrection="L"  -> QrL (also M/Q/H; Swiss QR uses "M")
//	  Barcode.Encoding="UTF8"      -> byte mode over UTF-8 bytes (this package)
//	  Barcode.QuietZone="false"    -> quiet zone handled by renderer, not modules
//	  Barcode.ShowMarker="true"     -> finder patterns always drawn
//	  Barcode.Shape="Rectangle" / Barcode.UseThinModules="false" / Barcode.Angle="0"
//	    -> plain square modules, no rotation (this package only)
//
// Example Barcode61: Text "http://fast-report.com", ErrorCorrection L.
// Example Barcode63 (Swiss QR): large SPC payload, ErrorCorrection M.
// Example Barcode38 (vCard) and Barcode46 (WIFI:...): ErrorCorrection L.
//
// No external oracle is available offline; correctness is pinned by structural
// tests plus two spec known-answer checks (format bits M/mask0 = 0x5412,
// version bits v7 = 0x07C94).
package qr

import (
	"errors"
	"fmt"
	"strings"
)

// QrErrTooLong is returned (wrapped) when the payload does not fit in
// version 40 at the requested error-correction level.
var QrErrTooLong = errors.New("qr: data too long for version 40")

// QrLevel is the QR error-correction level.
type QrLevel int

const (
	// QrL recovers ~7% of codewords.
	QrL QrLevel = iota
	// QrM recovers ~15% of codewords.
	QrM
	// QrQ recovers ~25% of codewords.
	QrQ
	// QrH recovers ~30% of codewords.
	QrH
)

// QrString returns the one-letter level name.
func (l QrLevel) QrString() string {
	switch l {
	case QrL:
		return "L"
	case QrM:
		return "M"
	case QrQ:
		return "Q"
	case QrH:
		return "H"
	}
	return "?"
}

// qrFormatECBits are the 2 EC-level bits used in format information.
func (l QrLevel) qrFormatECBits() int {
	switch l {
	case QrL:
		return 0x01
	case QrM:
		return 0x00
	case QrQ:
		return 0x03
	case QrH:
		return 0x02
	}
	return 0x00
}

// QrCode is an encoded QR symbol. The zero value is not usable;
// construct via QrEncode.
type QrCode struct {
	size    int
	version int
	level   QrLevel
	mask    int
	modules [][]bool // modules[y][x], true = dark
}

// Size returns the symbol edge length in modules: 21+4*(version-1).
func (c *QrCode) Size() int { return c.size }

// QrVersion returns the QR version (1..40).
func (c *QrCode) QrVersion() int { return c.version }

// QrMask returns the selected data mask (0..7).
func (c *QrCode) QrMask() int { return c.mask }

// QrLevelName returns the one-letter error-correction level name.
func (c *QrCode) QrLevelName() string { return c.level.QrString() }

// At reports whether module (x, y) is dark. Out-of-range reads are light.
func (c *QrCode) At(x, y int) bool {
	if x < 0 || y < 0 || x >= c.size || y >= c.size {
		return false
	}
	return c.modules[y][x]
}

// QrMatrix returns a copy of the module grid (row-major, true = dark).
func (c *QrCode) QrMatrix() [][]bool {
	out := make([][]bool, c.size)
	for y := range out {
		out[y] = make([]bool, c.size)
		copy(out[y], c.modules[y])
	}
	return out
}

// SVG renders the symbol as a self-contained SVG document, one unit per
// module, white background with black modules.
func (c *QrCode) SVG() string {
	var sb strings.Builder
	fmt.Fprintf(&sb, `<svg xmlns="http://www.w3.org/2000/svg" version="1.1" viewBox="0 0 %d %d" shape-rendering="crispEdges">`, c.size, c.size)
	fmt.Fprintf(&sb, `<rect width="%d" height="%d" fill="#FFFFFF"/>`, c.size, c.size)
	for y := 0; y < c.size; y++ {
		for x := 0; x < c.size; x++ {
			if c.modules[y][x] {
				fmt.Fprintf(&sb, `<rect x="%d" y="%d" width="1" height="1" fill="#000000"/>`, x, y)
			}
		}
	}
	sb.WriteString(`</svg>`)
	return sb.String()
}

// QrEncode encodes data in byte mode with level QrM, auto-selecting the
// smallest version that fits.
func QrEncode(data []byte) (*QrCode, error) {
	return QrEncodeWithLevel(data, QrM)
}

// QrEncodeWithLevel encodes data in byte mode with the given
// error-correction level, auto-selecting the smallest fitting version.
func QrEncodeWithLevel(data []byte, level QrLevel) (*QrCode, error) {
	if level < QrL || level > QrH {
		return nil, fmt.Errorf("qr: invalid level %d", int(level))
	}
	version := 0
	for v := 1; v <= 40; v++ {
		if qrFits(v, level, len(data)) {
			version = v
			break
		}
	}
	if version == 0 {
		return nil, fmt.Errorf("qr: %d bytes at level %s: %w", len(data), level.QrString(), QrErrTooLong)
	}
	raw := qrBuildDataCodewords(data, version, level)
	blocks := qrBlockGroups(version, level)
	final := qrInterleave(raw, blocks)
	return qrBuildMatrix(final, version, level), nil
}

// qrCountBits returns byte-mode character-count length for a version.
func qrCountBits(version int) int {
	if version <= 9 {
		return 8
	}
	return 16
}

// qrFits reports whether n payload bytes fit in version/ec-level in byte mode.
func qrFits(version int, level QrLevel, n int) bool {
	bits := 4 + qrCountBits(version) + 8*n
	return bits <= qrDataCodewords(version, level)*8
}

// qrDataCodewords returns total data codewords for version/level.
func qrDataCodewords(version int, level QrLevel) int {
	total := 0
	for _, g := range qrBlockGroups(version, level) {
		total += g.count * g.data
	}
	return total
}

// qrBlockGroup is one run of identical RS blocks:
// count blocks of (total bytes, data bytes); ec = total-data.
type qrBlockGroup struct {
	count int
	total int
	data  int
}

// qrBlockTableRaw mirrors the ISO error-correction block table as flat
// triples (block count, block total codewords, block data codewords),
// one []int per version (1..40) per level (L, M, Q, H).
var qrBlockTableRaw = [40][4][]int{
	{{1, 26, 19}, {1, 26, 16}, {1, 26, 13}, {1, 26, 9}},
	{{1, 44, 34}, {1, 44, 28}, {1, 44, 22}, {1, 44, 16}},
	{{1, 70, 55}, {1, 70, 44}, {2, 35, 17}, {2, 35, 13}},
	{{1, 100, 80}, {2, 50, 32}, {2, 50, 24}, {4, 25, 9}},
	{{1, 134, 108}, {2, 67, 43}, {2, 33, 15, 2, 34, 16}, {2, 33, 11, 2, 34, 12}},
	{{2, 86, 68}, {4, 43, 27}, {4, 43, 19}, {4, 43, 15}},
	{{2, 98, 78}, {4, 49, 31}, {2, 32, 14, 4, 33, 15}, {4, 39, 13, 1, 40, 14}},
	{{2, 121, 97}, {2, 60, 38, 2, 61, 39}, {4, 40, 18, 2, 41, 19}, {4, 40, 14, 2, 41, 15}},
	{{2, 146, 116}, {3, 58, 36, 2, 59, 37}, {4, 36, 16, 4, 37, 17}, {4, 36, 12, 4, 37, 13}},
	{{2, 86, 68, 2, 87, 69}, {4, 69, 43, 1, 70, 44}, {6, 43, 19, 2, 44, 20}, {6, 43, 15, 2, 44, 16}},
	{{4, 101, 81}, {1, 80, 50, 4, 81, 51}, {4, 50, 22, 4, 51, 23}, {3, 36, 12, 8, 37, 13}},
	{{2, 116, 92, 2, 117, 93}, {6, 58, 36, 2, 59, 37}, {4, 46, 20, 6, 47, 21}, {7, 42, 14, 4, 43, 15}},
	{{4, 133, 107}, {8, 59, 37, 1, 60, 38}, {8, 44, 20, 4, 45, 21}, {12, 33, 11, 4, 34, 12}},
	{{3, 145, 115, 1, 146, 116}, {4, 64, 40, 5, 65, 41}, {11, 36, 16, 5, 37, 17}, {11, 36, 12, 5, 37, 13}},
	{{5, 109, 87, 1, 110, 88}, {5, 65, 41, 5, 66, 42}, {5, 54, 24, 7, 55, 25}, {11, 36, 12, 7, 37, 13}},
	{{5, 122, 98, 1, 123, 99}, {7, 73, 45, 3, 74, 46}, {15, 43, 19, 2, 44, 20}, {3, 45, 15, 13, 46, 16}},
	{{1, 135, 107, 5, 136, 108}, {10, 74, 46, 1, 75, 47}, {1, 50, 22, 15, 51, 23}, {2, 42, 14, 17, 43, 15}},
	{{5, 150, 120, 1, 151, 121}, {9, 69, 43, 4, 70, 44}, {17, 50, 22, 1, 51, 23}, {2, 42, 14, 19, 43, 15}},
	{{3, 141, 113, 4, 142, 114}, {3, 70, 44, 11, 71, 45}, {17, 47, 21, 4, 48, 22}, {9, 39, 13, 16, 40, 14}},
	{{3, 135, 107, 5, 136, 108}, {3, 67, 41, 13, 68, 42}, {15, 54, 24, 5, 55, 25}, {15, 43, 15, 10, 44, 16}},
	{{4, 144, 116, 4, 145, 117}, {17, 68, 42}, {17, 50, 22, 6, 51, 23}, {19, 46, 16, 6, 47, 17}},
	{{2, 139, 111, 7, 140, 112}, {17, 74, 46}, {7, 54, 22, 16, 55, 23}, {34, 37, 13}},
	{{4, 151, 121, 5, 152, 122}, {4, 75, 47, 14, 76, 48}, {11, 54, 24, 14, 55, 25}, {16, 45, 15, 14, 46, 16}},
	{{6, 147, 117, 4, 148, 118}, {6, 73, 45, 14, 74, 46}, {11, 54, 24, 16, 55, 25}, {30, 46, 16, 2, 47, 17}},
	{{8, 132, 106, 4, 133, 107}, {8, 75, 47, 13, 76, 48}, {7, 54, 24, 22, 55, 25}, {22, 45, 15, 13, 46, 16}},
	{{10, 142, 114, 2, 143, 115}, {19, 74, 46, 4, 75, 47}, {28, 50, 22, 6, 51, 23}, {33, 46, 16, 4, 47, 17}},
	{{8, 152, 122, 4, 153, 123}, {22, 73, 45, 3, 74, 46}, {8, 53, 23, 26, 54, 24}, {12, 45, 15, 28, 46, 16}},
	{{3, 147, 115, 10, 148, 116}, {3, 73, 45, 23, 74, 46}, {4, 54, 24, 31, 55, 25}, {11, 45, 15, 31, 46, 16}},
	{{7, 146, 116, 7, 147, 117}, {21, 73, 45, 7, 74, 46}, {1, 53, 23, 37, 54, 24}, {19, 45, 15, 26, 46, 16}},
	{{5, 145, 115, 10, 146, 116}, {19, 75, 47, 10, 76, 48}, {15, 54, 24, 25, 55, 25}, {23, 45, 15, 25, 46, 16}},
	{{13, 145, 115, 3, 146, 116}, {2, 74, 46, 29, 75, 47}, {42, 54, 24, 1, 55, 25}, {23, 45, 15, 28, 46, 16}},
	{{17, 145, 115}, {10, 74, 46, 23, 75, 47}, {10, 54, 24, 35, 55, 25}, {19, 45, 15, 35, 46, 16}},
	{{17, 145, 115, 1, 146, 116}, {14, 74, 46, 21, 75, 47}, {29, 54, 24, 19, 55, 25}, {11, 45, 15, 46, 46, 16}},
	{{13, 145, 115, 6, 146, 116}, {14, 74, 46, 23, 75, 47}, {44, 54, 24, 7, 55, 25}, {59, 46, 16, 1, 47, 17}},
	{{12, 151, 121, 7, 152, 122}, {12, 75, 47, 26, 76, 48}, {39, 54, 24, 14, 55, 25}, {22, 45, 15, 41, 46, 16}},
	{{6, 151, 121, 14, 152, 122}, {6, 75, 47, 34, 76, 48}, {46, 54, 24, 10, 55, 25}, {2, 45, 15, 64, 46, 16}},
	{{17, 152, 122, 4, 153, 123}, {29, 74, 46, 14, 75, 47}, {49, 54, 24, 10, 55, 25}, {24, 45, 15, 46, 46, 16}},
	{{4, 152, 122, 18, 153, 123}, {13, 74, 46, 32, 75, 47}, {48, 54, 24, 14, 55, 25}, {42, 45, 15, 32, 46, 16}},
	{{20, 147, 117, 4, 148, 118}, {40, 75, 47, 7, 76, 48}, {43, 54, 24, 22, 55, 25}, {10, 45, 15, 67, 46, 16}},
	{{19, 148, 118, 6, 149, 119}, {18, 75, 47, 31, 76, 48}, {34, 54, 24, 34, 55, 25}, {20, 45, 15, 61, 46, 16}},
}

// qrTotalCodewords is the total codewords per version (independent check
// against the block table: sum(count*total) must equal this per level).
var qrTotalCodewords = [40]int{
	26, 44, 70, 100, 134, 172, 196, 242, 292, 346,
	404, 466, 532, 581, 655, 733, 815, 901, 991, 1085,
	1156, 1258, 1364, 1474, 1588, 1706, 1828, 1921, 2051, 2185,
	2323, 2465, 2611, 2761, 2876, 3034, 3196, 3362, 3532, 3706,
}

// qrRemainderBits per version (spec remainder-bit table).
var qrRemainderBits = [40]int{
	0, 7, 7, 7, 7, 7, 0, 0, 0, 0,
	0, 0, 0, 3, 3, 3, 3, 3, 3, 3,
	4, 4, 4, 4, 4, 4, 4, 3, 3, 3,
	3, 3, 3, 3, 0, 0, 0, 0, 0, 0,
}

// qrAlignPositions are alignment-pattern centers per version (index 0 = v1).
var qrAlignPositions = [][]int{
	{},
	{6, 18}, {6, 22}, {6, 26}, {6, 30}, {6, 34},
	{6, 22, 38}, {6, 24, 42}, {6, 26, 46}, {6, 28, 50}, {6, 30, 54},
	{6, 32, 58}, {6, 34, 62}, {6, 26, 46, 66}, {6, 26, 48, 70}, {6, 26, 50, 74},
	{6, 30, 54, 78}, {6, 30, 56, 82}, {6, 30, 58, 86}, {6, 34, 62, 90},
	{6, 28, 50, 72, 94}, {6, 26, 50, 74, 98}, {6, 30, 54, 78, 102},
	{6, 28, 54, 80, 106}, {6, 32, 58, 84, 110}, {6, 30, 58, 86, 114},
	{6, 34, 62, 90, 118}, {6, 26, 50, 74, 98, 122}, {6, 30, 54, 78, 102, 126},
	{6, 26, 52, 78, 104, 130}, {6, 30, 56, 82, 108, 134},
	{6, 34, 60, 86, 112, 138}, {6, 30, 58, 86, 114, 142},
	{6, 34, 62, 90, 118, 146}, {6, 30, 54, 78, 102, 126, 150},
	{6, 24, 50, 76, 102, 128, 154}, {6, 28, 54, 80, 106, 132, 158},
	{6, 32, 58, 84, 110, 136, 162}, {6, 26, 54, 82, 110, 138, 166},
	{6, 30, 58, 86, 114, 142, 170},
}

// qrBlockGroups parses the raw triple list for version/level.
func qrBlockGroups(version int, level QrLevel) []qrBlockGroup {
	raw := qrBlockTableRaw[version-1][level]
	groups := make([]qrBlockGroup, 0, 2)
	for i := 0; i+2 <= len(raw); i += 3 {
		groups = append(groups, qrBlockGroup{count: raw[i], total: raw[i+1], data: raw[i+2]})
	}
	return groups
}

// --- GF(256) with primitive polynomial 0x11D ---

var qrExpTable [512]byte
var qrLogTable [256]byte

func init() {
	x := 1
	for i := 0; i < 255; i++ {
		qrExpTable[i] = byte(x)
		qrLogTable[x] = byte(i)
		x <<= 1
		if x&0x100 != 0 {
			x ^= 0x11D
		}
	}
	for i := 255; i < 512; i++ {
		qrExpTable[i] = qrExpTable[i-255]
	}
}

func qrGFMul(a, b byte) byte {
	if a == 0 || b == 0 {
		return 0
	}
	return qrExpTable[int(qrLogTable[a])+int(qrLogTable[b])]
}

// qrRSGenerator returns the Reed-Solomon generator polynomial of the given
// degree, coefficients highest-first.
func qrRSGenerator(degree int) []byte {
	g := []byte{1}
	for i := 0; i < degree; i++ {
		ng := make([]byte, len(g)+1)
		ai := qrExpTable[i]
		for j := range g {
			ng[j] ^= qrGFMul(g[j], ai)
			ng[j+1] ^= g[j]
		}
		g = ng
	}
	// The loop builds lowest-first; the shift-register division in
	// qrRSRemainder consumes highest-first.
	for i, j := 0, len(g)-1; i < j; i, j = i+1, j-1 {
		g[i], g[j] = g[j], g[i]
	}
	return g
}

// qrRSRemainder divides data (highest-first) by gen and returns the
// (len(gen)-1)-byte remainder: the error-correction codewords.
func qrRSRemainder(data []byte, gen []byte) []byte {
	ec := make([]byte, len(gen)-1)
	for _, d := range data {
		f := d ^ ec[0]
		copy(ec, ec[1:])
		ec[len(ec)-1] = 0
		if f != 0 {
			for i := range ec {
				ec[i] ^= qrGFMul(gen[i+1], f)
			}
		}
	}
	return ec
}

// --- data codewords + interleave ---

// qrBuildDataCodewords packs payload into byte mode: mode indicator 0100,
// character count, data, terminator, byte alignment, 0xEC/0x11 pad.
func qrBuildDataCodewords(data []byte, version int, level QrLevel) []byte {
	n := qrDataCodewords(version, level)
	bits := make([]int, 0, n*8)
	push := func(val, count int) {
		for i := count - 1; i >= 0; i-- {
			bits = append(bits, (val>>uint(i))&1)
		}
	}
	push(0x4, 4) // byte mode
	push(len(data), qrCountBits(version))
	for _, b := range data {
		push(int(b), 8)
	}
	capBits := n * 8
	term := capBits - len(bits)
	if term > 4 {
		term = 4
	}
	for i := 0; i < term; i++ {
		bits = append(bits, 0)
	}
	for len(bits)%8 != 0 {
		bits = append(bits, 0)
	}
	out := make([]byte, 0, n)
	for i := 0; i < len(bits); i += 8 {
		var b byte
		for j := 0; j < 8; j++ {
			b = (b << 1) | byte(bits[i+j])
		}
		out = append(out, b)
	}
	for pad := byte(0xEC); len(out) < n; {
		out = append(out, pad)
		if pad == 0xEC {
			pad = 0x11
		} else {
			pad = 0xEC
		}
	}
	return out
}

// qrInterleave splits raw data codewords into RS blocks, computes EC
// codewords per block, and interleaves data then EC codewords.
func qrInterleave(raw []byte, groups []qrBlockGroup) []byte {
	type block struct {
		data []byte
		ec   []byte
	}
	var blocks []block
	off := 0
	maxData, maxEC := 0, 0
	for _, g := range groups {
		gen := qrRSGenerator(g.total - g.data)
		for k := 0; k < g.count; k++ {
			d := raw[off : off+g.data]
			off += g.data
			e := qrRSRemainder(d, gen)
			blocks = append(blocks, block{data: append([]byte(nil), d...), ec: e})
			if g.data > maxData {
				maxData = g.data
			}
			if len(e) > maxEC {
				maxEC = len(e)
			}
		}
	}
	final := make([]byte, 0, off+len(blocks)*maxEC)
	for i := 0; i < maxData; i++ {
		for _, b := range blocks {
			if i < len(b.data) {
				final = append(final, b.data[i])
			}
		}
	}
	for i := 0; i < maxEC; i++ {
		for _, b := range blocks {
			if i < len(b.ec) {
				final = append(final, b.ec[i])
			}
		}
	}
	return final
}

// --- matrix construction ---

// qrMatrix is a build-time symbol: modules plus the function-module map.
type qrMatrix struct {
	size     int
	version  int
	modules  [][]bool
	function [][]bool
}

func newQrMatrix(version int) *qrMatrix {
	size := 21 + 4*(version-1)
	m := &qrMatrix{size: size, version: version}
	m.modules = make([][]bool, size)
	m.function = make([][]bool, size)
	for i := range m.modules {
		m.modules[i] = make([]bool, size)
		m.function[i] = make([]bool, size)
	}
	return m
}

func (m *qrMatrix) setFunction(x, y int, dark bool) {
	m.modules[y][x] = dark
	m.function[y][x] = true
}

// qrBuildMatrix places function patterns, data, format/version info and
// selects the best mask by the four penalty rules.
func qrBuildMatrix(final []byte, version int, level QrLevel) *QrCode {
	m := newQrMatrix(version)
	size := m.size
	qrDrawFinders(m)
	qrDrawTiming(m)
	qrDrawAlignment(m)
	qrDrawDarkModule(m)
	qrReserveFormat(m)
	if version >= 7 {
		qrReserveVersion(m)
	}
	qrPlaceData(m, final, version)

	bestMask, bestModules := 0, m.modules
	bestScore := -1
	for mask := 0; mask < 8; mask++ {
		cand := qrApplyMask(m, mask)
		qrDrawFormat(cand, version, level, mask)
		if version >= 7 {
			qrDrawVersion(cand, version)
		}
		if s := qrPenalty(cand.modules); bestScore < 0 || s < bestScore {
			bestScore = s
			bestMask = mask
			bestModules = cand.modules
		}
	}
	return &QrCode{size: size, version: version, level: level, mask: bestMask, modules: bestModules}
}

func qrDrawFinders(m *qrMatrix) {
	centers := [][2]int{{3, 3}, {m.size - 4, 3}, {3, m.size - 4}}
	for _, c := range centers {
		cx, cy := c[0], c[1]
		for dy := -4; dy <= 4; dy++ {
			for dx := -4; dx <= 4; dx++ {
				x, y := cx+dx, cy+dy
				if x < 0 || y < 0 || x >= m.size || y >= m.size {
					continue
				}
				ax := dx
				if ax < 0 {
					ax = -ax
				}
				ay := dy
				if ay < 0 {
					ay = -ay
				}
				// 7x7 finder + 1-module light separator ring.
				m.setFunction(x, y, qrFinderBit(ax, ay))
			}
		}
	}
}

// qrFinderBit returns the finder+separator color for |dx|,|dy| in 0..4.
func qrFinderBit(ax, ay int) bool {
	maxd := ax
	if ay > maxd {
		maxd = ay
	}
	switch {
	case maxd == 4:
		return false // separator ring
	case maxd == 3:
		return true // outer dark border
	case maxd == 2:
		return false // inner light ring
	default:
		return true // 3x3 dark center
	}
}

func qrDrawTiming(m *qrMatrix) {
	for i := 8; i < m.size-8; i++ {
		v := i%2 == 0
		m.setFunction(6, i, v)
		m.setFunction(i, 6, v)
	}
}

func qrDrawAlignment(m *qrMatrix) {
	pos := qrAlignPositions[m.version-1]
	n := len(pos)
	for i, x := range pos {
		for j, y := range pos {
			// Skip overlaps with the three finder patterns.
			if (i == 0 && j == 0) || (i == 0 && j == n-1) || (i == n-1 && j == 0) {
				continue
			}
			for dy := -2; dy <= 2; dy++ {
				for dx := -2; dx <= 2; dx++ {
					ax := dx
					if ax < 0 {
						ax = -ax
					}
					ay := dy
					if ay < 0 {
						ay = -ay
					}
					maxd := ax
					if ay > maxd {
						maxd = ay
					}
					m.setFunction(x+dx, y+dy, maxd != 1)
				}
			}
		}
	}
}

func qrDrawDarkModule(m *qrMatrix) {
	m.setFunction(8, 4*m.version+9, true)
}

// qrFormatPositions lists the 15 (x, y) cells of each format-info copy,
// index = bit index (bit 0 = LSB of the 15-bit code).
func qrFormatPositions(size int) (first, second [15][2]int) {
	for i := 0; i < 6; i++ {
		first[i] = [2]int{8, i}
		second[i] = [2]int{size - 1 - i, 8}
	}
	first[6] = [2]int{8, 7}
	second[6] = [2]int{size - 7, 8}
	first[7] = [2]int{8, 8}
	second[7] = [2]int{size - 8, 8}
	first[8] = [2]int{7, 8}
	for i := 9; i < 15; i++ {
		first[i] = [2]int{14 - i, 8}
	}
	for i := 8; i < 15; i++ {
		second[i] = [2]int{8, size - 15 + i}
	}
	return first, second
}

func qrReserveFormat(m *qrMatrix) {
	first, second := qrFormatPositions(m.size)
	for _, p := range first {
		m.setFunction(p[0], p[1], false)
	}
	for _, p := range second {
		m.setFunction(p[0], p[1], false)
	}
}

// qrFormatBits returns the 15-bit format code for level/mask (BCH + XOR).
func qrFormatBits(level QrLevel, mask int) int {
	data := level.qrFormatECBits()<<3 | mask
	rem := data
	for i := 0; i < 10; i++ {
		rem = (rem << 1) ^ ((rem >> 9) * 0x537)
	}
	return ((data << 10) | (rem & 0x3FF)) ^ 0x5412
}

func qrDrawFormat(m *qrMatrix, _ int, level QrLevel, mask int) {
	bits := qrFormatBits(level, mask)
	first, second := qrFormatPositions(m.size)
	for i := 0; i < 15; i++ {
		v := ((bits >> uint(i)) & 1) == 1
		m.modules[first[i][1]][first[i][0]] = v
		m.modules[second[i][1]][second[i][0]] = v
	}
}

// qrVersionBits returns the 18-bit version code (BCH).
func qrVersionBits(version int) int {
	rem := version
	for i := 0; i < 12; i++ {
		rem = (rem << 1) ^ ((rem >> 11) * 0x1F25)
	}
	return (version << 12) | (rem & 0xFFF)
}

func qrReserveVersion(m *qrMatrix) {
	size := m.size
	for i := 0; i < 18; i++ {
		x := size - 11 + i%3
		y := i / 3
		m.setFunction(x, y, false)
		m.setFunction(y, x, false)
	}
}

func qrDrawVersion(m *qrMatrix, version int) {
	bits := qrVersionBits(version)
	size := m.size
	for i := 0; i < 18; i++ {
		v := ((bits >> uint(i)) & 1) == 1
		x := size - 11 + i%3
		y := i / 3
		m.modules[y][x] = v
		m.modules[x][y] = v
	}
}

// qrPlaceData writes final codewords in the standard zigzag scan,
// skipping function modules; remainder bits stay light.
func qrPlaceData(m *qrMatrix, final []byte, _ int) {
	size := m.size
	i := 0
	n := len(final) * 8
	bit := func() bool {
		b := (final[i>>3] >> uint(7-(i&7))) & 1
		i++
		return b == 1
	}
	for right := size - 1; right >= 1; right -= 2 {
		if right == 6 {
			right = 5
		}
		for vert := 0; vert < size; vert++ {
			for j := 0; j < 2; j++ {
				x := right - j
				upward := ((right + 1) & 2) == 0
				y := vert
				if upward {
					y = size - 1 - vert
				}
				if m.function[y][x] {
					continue
				}
				if i < n {
					m.modules[y][x] = bit()
				}
			}
		}
	}
}

// qrMaskBit reports whether the mask inverts module (x, y).
func qrMaskBit(mask, x, y int) bool {
	switch mask {
	case 0:
		return (x+y)%2 == 0
	case 1:
		return y%2 == 0
	case 2:
		return x%3 == 0
	case 3:
		return (x+y)%3 == 0
	case 4:
		return (y/2+x/3)%2 == 0
	case 5:
		return (x*y)%2+(x*y)%3 == 0
	case 6:
		return ((x*y)%2+(x*y)%3)%2 == 0
	case 7:
		return ((x+y)%2+(x*y)%3)%2 == 0
	}
	return false
}

// qrApplyMask returns a copy with the mask XORed onto data modules only.
func qrApplyMask(m *qrMatrix, mask int) *qrMatrix {
	out := newQrMatrix(m.version)
	for y := 0; y < m.size; y++ {
		for x := 0; x < m.size; x++ {
			v := m.modules[y][x]
			if !m.function[y][x] && qrMaskBit(mask, x, y) {
				v = !v
			}
			out.modules[y][x] = v
			out.function[y][x] = m.function[y][x]
		}
	}
	return out
}

// qrPenalty implements the four QR penalty rules N1..N4.
func qrPenalty(modules [][]bool) int {
	size := len(modules)
	score := 0
	// N1: runs of 5+ in rows and columns.
	for y := 0; y < size; y++ {
		run, prev := 1, modules[y][0]
		for x := 1; x < size; x++ {
			if modules[y][x] == prev {
				run++
			} else {
				if run >= 5 {
					score += 3 + (run - 5)
				}
				run, prev = 1, modules[y][x]
			}
		}
		if run >= 5 {
			score += 3 + (run - 5)
		}
	}
	for x := 0; x < size; x++ {
		run, prev := 1, modules[0][x]
		for y := 1; y < size; y++ {
			if modules[y][x] == prev {
				run++
			} else {
				if run >= 5 {
					score += 3 + (run - 5)
				}
				run, prev = 1, modules[y][x]
			}
		}
		if run >= 5 {
			score += 3 + (run - 5)
		}
	}
	// N2: 2x2 blocks of same color.
	for y := 0; y+1 < size; y++ {
		for x := 0; x+1 < size; x++ {
			v := modules[y][x]
			if modules[y][x+1] == v && modules[y+1][x] == v && modules[y+1][x+1] == v {
				score += 3
			}
		}
	}
	// N3: finder-like patterns 10111010000 / 00001011101 in rows/columns.
	pat1 := [11]bool{true, false, true, true, true, false, true, false, false, false, false}
	pat2 := [11]bool{false, false, false, false, true, false, true, true, true, false, true}
	checkLine := func(line []bool) {
		for i := 0; i+11 <= len(line); i++ {
			match1, match2 := true, true
			for k := 0; k < 11; k++ {
				if line[i+k] != pat1[k] {
					match1 = false
				}
				if line[i+k] != pat2[k] {
					match2 = false
				}
			}
			if match1 || match2 {
				score += 40
			}
		}
	}
	for y := 0; y < size; y++ {
		checkLine(modules[y])
	}
	col := make([]bool, size)
	for x := 0; x < size; x++ {
		for y := 0; y < size; y++ {
			col[y] = modules[y][x]
		}
		checkLine(col)
	}
	// N4: dark-module ratio.
	dark := 0
	for y := 0; y < size; y++ {
		for x := 0; x < size; x++ {
			if modules[y][x] {
				dark++
			}
		}
	}
	total := size * size
	prev := (dark * 20 / total) * 5
	d1 := prev - 50
	if d1 < 0 {
		d1 = -d1
	}
	d2 := prev + 5 - 50
	if d2 < 0 {
		d2 = -d2
	}
	lo := d1
	if d2 < lo {
		lo = d2
	}
	score += lo / 5 * 10
	return score
}
