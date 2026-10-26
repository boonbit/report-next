// Package datamatrix encodes ECC200 DataMatrix symbols (square only) in
// ASCII encodation with Upper Shift for bytes >= 128. The symbol table,
// factor polynomials, pad randomizer, placement and finder rules mirror the
// ZXing encoder (github.com/makiuchi-d/gozxing/datamatrix/encoder), which
// also serves as the independent oracle in verification.
package datamatrix

import (
	"fmt"
	"strings"
)

// dmSymbol is one square symbol size. regions is the data-region count
// (1, 4, 16 or 36); region is one region's mapping edge. rsData/rsEC
// describe one Reed-Solomon block (blockCount = data/rsData).
type dmSymbol struct {
	size    int
	regions int
	region  int
	data    int
	ec      int
	rsData  int
	rsEC    int
}

var dmSymbols = []dmSymbol{
	{10, 1, 8, 3, 5, 3, 5},
	{12, 1, 10, 5, 7, 5, 7},
	{14, 1, 12, 8, 10, 8, 10},
	{16, 1, 14, 12, 12, 12, 12},
	{18, 1, 16, 18, 14, 18, 14},
	{20, 1, 18, 22, 18, 22, 18},
	{22, 1, 20, 30, 20, 30, 20},
	{24, 1, 22, 36, 24, 36, 24},
	{26, 1, 24, 44, 28, 44, 28},
	{32, 4, 14, 62, 36, 62, 36},
	{36, 4, 16, 86, 42, 86, 42},
	{40, 4, 18, 114, 48, 114, 48},
	{44, 4, 20, 144, 56, 144, 56},
	{48, 4, 22, 174, 68, 174, 68},
	{52, 4, 24, 204, 84, 102, 42},
	{64, 16, 14, 280, 112, 140, 56},
	{72, 16, 16, 368, 144, 92, 36},
	{80, 16, 18, 456, 192, 114, 48},
	{88, 16, 20, 576, 224, 144, 56},
	{96, 16, 22, 696, 272, 174, 68},
	{104, 16, 24, 816, 336, 136, 56},
	{120, 36, 18, 1050, 408, 175, 68},
	{132, 36, 20, 1304, 496, 163, 62},
}

// dmFactors are the precomputed ECC200 generator factors per EC count.
var dmFactors = [][]int{
	{228, 48, 15, 111, 62},
	{23, 68, 144, 134, 240, 92, 254},
	{28, 24, 185, 166, 223, 248, 116, 255, 110, 61},
	{175, 138, 205, 12, 194, 168, 39, 245, 60, 97, 120},
	{41, 153, 158, 91, 61, 42, 142, 213, 97, 178, 100, 242},
	{156, 97, 192, 252, 95, 9, 157, 119, 138, 45, 18, 186, 83, 185},
	{83, 195, 100, 39, 188, 75, 66, 61, 241, 213, 109, 129, 94, 254, 225, 48, 90, 188},
	{15, 195, 244, 9, 233, 71, 168, 2, 188, 160, 153, 145, 253, 79, 108, 82, 27, 174, 186, 172},
	{52, 190, 88, 205, 109, 39, 176, 21, 155, 197, 251, 223, 155, 21, 5, 172, 254, 124, 12, 181, 184, 96, 50, 193},
	{211, 231, 43, 97, 71, 96, 103, 174, 37, 151, 170, 53, 75, 34, 249, 121, 17, 138, 110, 213, 141, 136, 120, 151, 233, 168, 93, 255},
	{245, 127, 242, 218, 130, 250, 162, 181, 102, 120, 84, 179, 220, 251, 80, 182, 229, 18, 2, 4, 68, 33, 101, 137, 95, 119, 115, 44, 175, 184, 59, 25, 225, 98, 81, 112},
	{77, 193, 137, 31, 19, 38, 22, 153, 247, 105, 122, 2, 245, 133, 242, 8, 175, 95, 100, 9, 167, 105, 214, 111, 57, 121, 21, 1, 253, 57, 54, 101, 248, 202, 69, 50, 150, 177, 226, 5, 9, 5},
	{245, 132, 172, 223, 96, 32, 117, 22, 238, 133, 238, 231, 205, 188, 237, 87, 191, 106, 16, 147, 118, 23, 37, 90, 170, 205, 131, 88, 120, 100, 66, 138, 186, 240, 82, 44, 176, 87, 187, 147, 160, 175, 69, 213, 92, 253, 225, 19},
	{175, 9, 223, 238, 12, 17, 220, 208, 100, 29, 175, 170, 230, 192, 215, 235, 150, 159, 36, 223, 38, 200, 132, 54, 228, 146, 218, 234, 117, 203, 29, 232, 144, 238, 22, 150, 201, 117, 62, 207, 164, 13, 137, 245, 127, 67, 247, 28, 155, 43, 203, 107, 233, 53, 143, 46},
	{242, 93, 169, 50, 144, 210, 39, 118, 202, 188, 201, 189, 143, 108, 196, 37, 185, 112, 134, 230, 245, 63, 197, 190, 250, 106, 185, 221, 175, 64, 114, 71, 161, 44, 147, 6, 27, 218, 51, 63, 87, 10, 40, 130, 188, 17, 163, 31, 176, 170, 4, 107, 232, 7, 94, 166, 224, 124, 86, 47, 11, 204},
	{220, 228, 173, 89, 251, 149, 159, 56, 89, 33, 147, 244, 154, 36, 73, 127, 213, 136, 248, 180, 234, 197, 158, 177, 68, 122, 93, 213, 15, 160, 227, 236, 66, 139, 153, 185, 202, 167, 179, 25, 220, 232, 96, 210, 231, 136, 223, 239, 181, 241, 59, 52, 172, 25, 49, 232, 211, 189, 64, 54, 108, 153, 132, 63, 96, 103, 82, 186},
}

// dmLog/dmAlog are GF(256) tables with primitive 0x12D.
var dmLog [256]int
var dmAlog [255]int

func init() {
	p := 1
	for i := 0; i < 255; i++ {
		dmAlog[i] = p
		dmLog[p] = i
		p *= 2
		if p >= 256 {
			p ^= 0x12D
		}
	}
}

// dmEncodeASCII encodes data in ASCII encodation: digit pairs pack two
// digits per codeword, bytes < 128 shift by one, bytes >= 128 use the
// Upper Shift latch. Output is unpadded.
func dmEncodeASCII(data []byte) []byte {
	var out []byte
	for i := 0; i < len(data); {
		if i+1 < len(data) && data[i] >= '0' && data[i] <= '9' && data[i+1] >= '0' && data[i+1] <= '9' {
			out = append(out, byte(130+int(data[i]-'0')*10+int(data[i+1]-'0')))
			i += 2
			continue
		}
		if data[i] > 127 {
			out = append(out, 235, data[i]-127)
		} else {
			out = append(out, data[i]+1)
		}
		i++
	}
	return out
}

// dmRandomize253State pads position (1-based) per the ECC200 rule.
func dmRandomize253State(pos int) byte {
	pseudo := ((149 * pos) % 253) + 1
	temp := 129 + pseudo
	if temp <= 254 {
		return byte(temp)
	}
	return byte(temp - 254)
}

// dmFactorsFor returns the generator factors for an EC word count.
func dmFactorsFor(ec int) ([]int, error) {
	for _, f := range dmFactors {
		if len(f) == ec {
			return f, nil
		}
	}
	return nil, fmt.Errorf("datamatrix: no factors for %d ec codewords", ec)
}

// dmECCBlock computes ec codewords for one block.
func dmECCBlock(data []byte, ec int) ([]byte, error) {
	poly, err := dmFactorsFor(ec)
	if err != nil {
		return nil, err
	}
	ecc := make([]int, ec)
	for _, d := range data {
		m := ecc[ec-1] ^ int(d)
		for k := ec - 1; k > 0; k-- {
			if m != 0 && poly[k] != 0 {
				ecc[k] = ecc[k-1] ^ dmAlog[(dmLog[m]+dmLog[poly[k]])%255]
			} else {
				ecc[k] = ecc[k-1]
			}
		}
		ecc[0] = 0
		if m != 0 && poly[0] != 0 {
			ecc[0] = dmAlog[(dmLog[m]+dmLog[poly[0]])%255]
		}
	}
	// The shift register accumulates lowest-first; codewords are
	// transmitted highest-first.
	out := make([]byte, ec)
	for i, v := range ecc {
		out[ec-1-i] = byte(v)
	}
	return out, nil
}

// Code is one encoded symbol: modules include the finder pattern.
type Code struct {
	size    int
	modules [][]bool
}

// Size returns the symbol edge in modules, finder included.
func (c *Code) Size() int { return c.size }

// At reports whether module (x, y) is dark.
func (c *Code) At(x, y int) bool { return c.modules[y][x] }

// Matrix returns a copy of the modules.
func (c *Code) Matrix() [][]bool {
	out := make([][]bool, len(c.modules))
	for i, r := range c.modules {
		out[i] = append([]bool(nil), r...)
	}
	return out
}

// dmPlacement places codewords into the data mapping (Utah + corners).
func dmPlacement(codewords []byte, numcols, numrows int) []int8 {
	bits := make([]int8, numcols*numrows)
	for i := range bits {
		bits[i] = -1
	}
	set := func(col, row int, v bool) {
		if v {
			bits[row*numcols+col] = 1
		} else {
			bits[row*numcols+col] = 0
		}
	}
	has := func(col, row int) bool { return bits[row*numcols+col] >= 0 }
	module := func(row, col, pos, bit int) {
		if row < 0 {
			row += numrows
			col += 4 - ((numrows + 4) % 8)
		}
		if col < 0 {
			col += numcols
			row += 4 - ((numcols + 4) % 8)
		}
		v := codewords[pos]
		set(col, row, v&(1<<uint(8-bit)) != 0)
	}
	utah := func(row, col, pos int) {
		module(row-2, col-2, pos, 1)
		module(row-2, col-1, pos, 2)
		module(row-1, col-2, pos, 3)
		module(row-1, col-1, pos, 4)
		module(row-1, col, pos, 5)
		module(row, col-2, pos, 6)
		module(row, col-1, pos, 7)
		module(row, col, pos, 8)
	}
	corner := func(cells [][2]int, pos int) {
		for i, c := range cells {
			module(c[0], c[1], pos, i+1)
		}
	}
	c1 := [][2]int{{numrows - 1, 0}, {numrows - 1, 1}, {numrows - 1, 2}, {0, numcols - 2}, {0, numcols - 1}, {1, numcols - 1}, {2, numcols - 1}, {3, numcols - 1}}
	c2 := [][2]int{{numrows - 3, 0}, {numrows - 2, 0}, {numrows - 1, 0}, {0, numcols - 4}, {0, numcols - 3}, {0, numcols - 2}, {0, numcols - 1}, {1, numcols - 1}}
	c3 := [][2]int{{numrows - 3, 0}, {numrows - 2, 0}, {numrows - 1, 0}, {0, numcols - 2}, {0, numcols - 1}, {1, numcols - 1}, {2, numcols - 1}, {3, numcols - 1}}
	c4 := [][2]int{{numrows - 1, 0}, {numrows - 1, numcols - 1}, {0, numcols - 3}, {0, numcols - 2}, {0, numcols - 1}, {1, numcols - 3}, {1, numcols - 2}, {1, numcols - 1}}
	pos, row, col := 0, 4, 0
	for {
		if row == numrows && col == 0 {
			corner(c1, pos)
			pos++
		}
		if row == numrows-2 && col == 0 && numcols%4 != 0 {
			corner(c2, pos)
			pos++
		}
		if row == numrows-2 && col == 0 && numcols%8 == 4 {
			corner(c3, pos)
			pos++
		}
		if row == numrows+4 && col == 2 && numcols%8 == 0 {
			corner(c4, pos)
			pos++
		}
		for {
			if row < numrows && col >= 0 && !has(col, row) {
				utah(row, col, pos)
				pos++
			}
			row -= 2
			col += 2
			if row < 0 || col >= numcols {
				break
			}
		}
		row++
		col += 3
		for {
			if row >= 0 && col < numcols && !has(col, row) {
				utah(row, col, pos)
				pos++
			}
			row += 2
			col -= 2
			if row >= numrows || col < 0 {
				break
			}
		}
		row += 3
		col++
		if row >= numrows && col >= numcols {
			break
		}
	}
	if !has(numcols-1, numrows-1) {
		set(numcols-1, numrows-1, true)
		set(numcols-2, numrows-2, true)
	}
	return bits
}

// dmRegionsPerSide maps a region count to the grid width.
func dmRegionsPerSide(regions int) int {
	switch regions {
	case 1:
		return 1
	case 2, 4:
		return 2
	case 16:
		return 4
	default:
		return 6
	}
}

// Encode builds a square ECC200 symbol for data. A positive size forces the
// exact symbol edge (e.g. 24 for "Size24x24"); zero or negative picks the
// smallest symbol that fits.
func Encode(data []byte, size int) (*Code, error) {
	raw := dmEncodeASCII(data)
	var sym *dmSymbol
	if size > 0 {
		for i := range dmSymbols {
			if dmSymbols[i].size == size {
				sym = &dmSymbols[i]
				break
			}
		}
		if sym == nil {
			return nil, fmt.Errorf("datamatrix: unsupported symbol size %d", size)
		}
		if len(raw) > sym.data {
			return nil, fmt.Errorf("datamatrix: %d bytes need a bigger symbol than %dx%d", len(data), size, size)
		}
	} else {
		for i := range dmSymbols {
			if len(raw) <= dmSymbols[i].data {
				sym = &dmSymbols[i]
				break
			}
		}
		if sym == nil {
			return nil, fmt.Errorf("datamatrix: data too long (%d bytes)", len(data))
		}
	}
	codewords := make([]byte, 0, sym.data)
	codewords = append(codewords, raw...)
	padded := false
	for len(codewords) < sym.data {
		if !padded {
			codewords = append(codewords, 129)
			padded = true
			continue
		}
		codewords = append(codewords, dmRandomize253State(len(codewords)+1))
	}
	// Split into RS blocks round-robin and interleave EC the ZXing way.
	blocks := sym.data / sym.rsData
	full := make([]byte, 0, sym.data+sym.ec)
	full = append(full, codewords...)
	if blocks == 1 {
		ecc, err := dmECCBlock(codewords, sym.ec)
		if err != nil {
			return nil, err
		}
		full = append(full, ecc...)
	} else {
		full = full[:cap(full)]
		for b := 0; b < blocks; b++ {
			var tmp []byte
			for d := b; d < sym.data; d += blocks {
				tmp = append(tmp, codewords[d])
			}
			ecc, err := dmECCBlock(tmp, sym.rsEC)
			if err != nil {
				return nil, err
			}
			pos := 0
			for e := b; e < sym.rsEC*blocks; e += blocks {
				full[sym.data+e] = ecc[pos]
				pos++
			}
		}
	}
	// Data-region grid and low-level finder assembly.
	per := dmRegionsPerSide(sym.regions)
	mapW := per * sym.region
	bits := dmPlacement(full, mapW, mapW)
	mod := make([][]bool, sym.size)
	for i := range mod {
		mod[i] = make([]bool, sym.size)
	}
	my := 0
	for y := 0; y < mapW; y++ {
		if y%sym.region == 0 {
			for x := 0; x < sym.size; x++ {
				mod[my][x] = x%2 == 0
			}
			my++
		}
		mx := 0
		for x := 0; x < mapW; x++ {
			if x%sym.region == 0 {
				mod[my][mx] = true
				mx++
			}
			mod[my][mx] = bits[y*mapW+x] == 1
			mx++
			if x%sym.region == sym.region-1 {
				mod[my][mx] = (y%2 == 0)
				mx++
			}
		}
		my++
		if y%sym.region == sym.region-1 {
			for x := 0; x < sym.size; x++ {
				mod[my][x] = true
			}
			my++
		}
	}
	return &Code{size: sym.size, modules: mod}, nil
}

// SVG renders the symbol as vector rects.
func (c *Code) SVG() string {
	var sb strings.Builder
	sb.WriteString(fmt.Sprintf(`<svg viewBox="0 0 %d %d" xmlns="http://www.w3.org/2000/svg">`, c.size, c.size))
	sb.WriteString(`<rect width="100%" height="100%" fill="white"/>`)
	for y := 0; y < c.size; y++ {
		for x := 0; x < c.size; x++ {
			if c.modules[y][x] {
				sb.WriteString(fmt.Sprintf(`<rect x="%d" y="%d" width="1" height="1"/>`, x, y))
			}
		}
	}
	sb.WriteString(`</svg>`)
	return sb.String()
}
