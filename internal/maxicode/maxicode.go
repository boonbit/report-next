package maxicode

// MaxiCode symbol assembly: structured (modes 2/3) or plain (modes 4/5/6)
// message layout, Appendix A character sets with shift/latch/number
// compression, Reed-Solomon EC, module-sequence placement and orientation
// marks, following the published algorithm. ECI is fixed at 0.
//
// The module grid matches the data layout exactly; the central bulls-eye
// finder is overlaid at render time (SVG circles, PDF rings) the way viewer
// renderers draw it.

import (
	"errors"
	"fmt"
	"strconv"
	"strings"
)

const (
	mxCols = 30
	mxRows = 33
)

// Code is one MaxiCode symbol: a 30-wide, 33-tall module grid (row-major,
// true = dark). Finder rings are a render-time overlay, not grid data.
type Code struct {
	data []byte
	mode int
	bars [][]bool
}

// Encode builds a MaxiCode symbol for data in the given mode (2..6).
// Modes 2 and 3 need a structured carrier message starting with the
// "[)>\x1e01\x1d" header and ending with EOT; modes 4-6 take free data
// (93 codewords max, 77 for mode 5).
func Encode(mode int, data []byte) (*Code, error) {
	if mode < 2 || mode > 6 {
		return nil, fmt.Errorf("maxicode: unsupported mode %d", mode)
	}
	input := string(data)
	cw := make([]int, 144)
	secondary := ""
	if mode == 2 || mode == 3 {
		const header = "[)>\x1e01\x1d"
		if !strings.HasPrefix(input, header) {
			return nil, errors.New("maxicode: mode 2/3 needs a structured carrier message header")
		}
		want := 41
		if mode == 2 {
			want = 44
		}
		if len(input) < want {
			return nil, errors.New("maxicode: input shorter than mandatory requirements")
		}
		if !strings.HasSuffix(input, "\x04") {
			return nil, errors.New("maxicode: input should end with EOT marker")
		}
		groups := strings.Split(input[9:], "\x1d")
		if len(groups) < 3 {
			return nil, errors.New("maxicode: input should contain postcode, country code and service class")
		}
		postcode, ccStr, scStr := groups[0], groups[1], groups[2]
		if l := len(postcode); (mode == 2 && l != 9) || (mode == 3 && l != 6) {
			return nil, errors.New("maxicode: invalid postcode length")
		}
		if len(ccStr) != 3 {
			return nil, errors.New("maxicode: invalid country code length")
		}
		if len(scStr) != 3 {
			return nil, errors.New("maxicode: invalid service class length")
		}
		cc, err := strconv.Atoi(ccStr)
		if err != nil {
			return nil, errors.New("maxicode: country code must be numeric")
		}
		sc, err := strconv.Atoi(scStr)
		if err != nil {
			return nil, errors.New("maxicode: service class must be numeric")
		}
		if mode == 2 {
			if cc != 840 {
				return nil, errors.New("maxicode: mode 2 requires US country code 840")
			}
			pc, err := strconv.Atoi(postcode)
			if err != nil {
				return nil, errors.New("maxicode: postcode must be numeric in mode 2")
			}
			mxPrimaryMode2(cw, pc, cc, sc)
		} else {
			mxPrimaryMode3(cw, postcode, cc, sc)
		}
		secondary = input[:9]
		if len(groups) > 3 {
			secondary += strings.Join(groups[3:], "\x1d")
		}
	} else {
		cw[0] = mode
		secondary = input
	}
	if err := mxSecondary(cw, mode, secondary); err != nil {
		return nil, err
	}
	mxPrimaryCheck(cw)
	eccLen := 40 // 84 data codewords, 40 check codewords
	if mode == 5 {
		eccLen = 56 // 68 data codewords, 56 check codewords
	}
	mxSecondaryCheckEven(cw, eccLen/2)
	mxSecondaryCheckOdd(cw, eccLen/2)

	bars := make([][]bool, mxRows)
	for i := range bars {
		bars[i] = make([]bool, mxCols)
	}
	for row := 0; row < mxRows; row++ {
		for col := 0; col < mxCols; col++ {
			g := mxGrid[row*mxCols+col] + 5
			block, bit := g/6, g%6
			if block == 0 {
				continue
			}
			if cw[block-1]&(32>>bit)>>(5-bit) != 0 {
				bars[row][col] = true
			}
		}
	}
	// Orientation markings.
	for _, m := range [][2]int{
		{0, 28}, {0, 29}, // top right filler
		{9, 10}, {9, 11}, {10, 11}, // top left marker
		{15, 7},            // left hand marker
		{16, 8},            // left hand marker
		{16, 20}, {17, 20}, // right hand marker
		{22, 10}, {23, 10}, // bottom left marker
		{22, 17}, {23, 17}, // bottom right marker
	} {
		bars[m[0]][m[1]] = true
	}
	return &Code{data: append([]byte(nil), data...), mode: mode, bars: bars}, nil
}

// Mode reports the symbol mode.
func (c *Code) Mode() int { return c.mode }

// At reports whether module (x, y) is dark.
func (c *Code) At(x, y int) bool { return c.bars[y][x] }

// Matrix returns a copy of the module grid (row-major, true = dark).
func (c *Code) Matrix() [][]bool {
	out := make([][]bool, len(c.bars))
	for i, row := range c.bars {
		out[i] = append([]bool(nil), row...)
	}
	return out
}

// mxFinder is the bulls-eye center in module coordinates.
const mxFinderCX, mxFinderCY = 15, 16

// Finder reports the bulls-eye center in module coordinates for renderers
// that overlay the finder (SVG circles, PDF rings).
func (c *Code) Finder() (x, y int) { return mxFinderCX, mxFinderCY }

// SVG renders the symbol as dark unit rects plus the central bulls-eye
// finder (two rings and a solid core) on a white background.
func (c *Code) SVG() string {
	var sb strings.Builder
	fmt.Fprintf(&sb, `<svg viewBox="0 0 %d %d" xmlns="http://www.w3.org/2000/svg">`, mxCols, mxRows)
	sb.WriteString(`<rect width="100%" height="100%" fill="white"/>`)
	for y := 0; y < mxRows; y++ {
		for x := 0; x < mxCols; x++ {
			if c.bars[y][x] {
				fmt.Fprintf(&sb, `<rect x="%d" y="%d" width="1" height="1"/>`, x, y)
			}
		}
	}
	fmt.Fprintf(&sb, `<circle cx="%d" cy="%d" r="3" fill="none" stroke="black"/>`, mxFinderCX, mxFinderCY)
	fmt.Fprintf(&sb, `<circle cx="%d" cy="%d" r="1" fill="black"/>`, mxFinderCX, mxFinderCY)
	sb.WriteString(`</svg>`)
	return sb.String()
}

func mxPrimaryMode2(cw []int, postcode, countryCode, serviceClass int) {
	const postcodeLen = 9
	cw[0] = ((postcode & 0x03) << 4) | 2
	cw[1] = (postcode & 0xfc) >> 2
	cw[2] = (postcode & 0x3f00) >> 8
	cw[3] = (postcode & 0xfc000) >> 14
	cw[4] = (postcode & 0x3f00000) >> 20
	cw[5] = ((postcode & 0x3c000000) >> 26) | ((postcodeLen & 0x3) << 4)
	cw[6] = ((postcodeLen & 0x3c) >> 2) | ((countryCode & 0x3) << 4)
	cw[7] = (countryCode & 0xfc) >> 2
	cw[8] = ((countryCode & 0x300) >> 8) | ((serviceClass & 0xf) << 2)
	cw[9] = (serviceClass & 0x3f0) >> 4
}

func mxPrimaryMode3(cw []int, postcode string, countryCode, serviceClass int) {
	pc := make([]int, 6)
	for i := 0; i < 6; i++ {
		c := postcode[i]
		switch {
		case c >= 'A' && c <= 'Z':
			pc[i] = int(c - 64)
		case c == 27 || c == 31 || c == 33 || c >= 59:
			pc[i] = int(' ')
		default:
			pc[i] = int(c)
		}
	}
	cw[0] = ((pc[5] & 0x03) << 4) | 3
	cw[1] = ((pc[4] & 0x03) << 4) | ((pc[5] & 0x3c) >> 2)
	cw[2] = ((pc[3] & 0x03) << 4) | ((pc[4] & 0x3c) >> 2)
	cw[3] = ((pc[2] & 0x03) << 4) | ((pc[3] & 0x3c) >> 2)
	cw[4] = ((pc[1] & 0x03) << 4) | ((pc[2] & 0x3c) >> 2)
	cw[5] = ((pc[0] & 0x03) << 4) | ((pc[1] & 0x3c) >> 2)
	cw[6] = ((pc[0] & 0x3c) >> 2) | ((countryCode & 0x3) << 4)
	cw[7] = (countryCode & 0xfc) >> 2
	cw[8] = ((countryCode & 0x300) >> 8) | ((serviceClass & 0xf) << 2)
	cw[9] = (serviceClass & 0x3f0) >> 4
}

func mxPickSet(set []int, i int, allowed ...int) int {
	opt1, opt2 := set[i-1], set[i+1]
	a1, a2 := false, false
	for _, a := range allowed {
		if a == opt1 {
			a1 = true
		}
		if a == opt2 {
			a2 = true
		}
	}
	switch {
	case a1 && a2:
		if opt1 < opt2 {
			return opt1
		}
		return opt2
	case a1:
		return opt1
	case a2:
		return opt2
	}
	return allowed[0]
}

func mxSecondary(cw []int, mode int, secondary string) error {
	if len(secondary) > 138 {
		return errors.New("maxicode: input data is too long")
	}
	set := make([]int, 144)
	character := make([]int, 144)
	for i := 0; i < len(secondary); i++ {
		set[i] = mxCodeSet[secondary[i]]
		character[i] = mxSymbolChar[secondary[i]]
	}
	if set[0] == 0 {
		if character[0] == 13 {
			character[0] = 0
		}
		set[0] = 1
	}
	dataLen := len(secondary)
	for i := 1; i < dataLen; i++ {
		if set[i] != 0 {
			continue
		}
		switch character[i] {
		case 13: // carriage return
			set[i] = mxPickSet(set, i, 1, 5)
			if set[i] != 5 {
				character[i] = 0
			}
		case 28: // FS
			set[i] = mxPickSet(set, i, 1, 2, 3, 4, 5)
			if set[i] == 5 {
				character[i] = 32
			}
		case 29: // GS
			set[i] = mxPickSet(set, i, 1, 2, 3, 4, 5)
			if set[i] == 5 {
				character[i] = 33
			}
		case 30: // RS
			set[i] = mxPickSet(set, i, 1, 2, 3, 4, 5)
			if set[i] == 5 {
				character[i] = 34
			}
		case 32: // space
			set[i] = mxPickSet(set, i, 1, 2, 3, 4, 5)
			switch set[i] {
			case 1:
				character[i] = 32
			case 2:
				character[i] = 47
			default:
				character[i] = 59
			}
		case 44: // comma
			set[i] = mxPickSet(set, i, 1, 2)
			if set[i] == 2 {
				character[i] = 48
			}
		case 46: // full stop
			set[i] = mxPickSet(set, i, 1, 2)
			if set[i] == 2 {
				character[i] = 49
			}
		case 47: // slash
			set[i] = mxPickSet(set, i, 1, 2)
			if set[i] == 2 {
				character[i] = 50
			}
		case 58: // colon
			set[i] = mxPickSet(set, i, 1, 2)
			if set[i] == 2 {
				character[i] = 51
			}
		}
	}
	for i := dataLen; i < 144; i++ {
		if set[dataLen-1] == 2 {
			set[i] = 2
		} else {
			set[i] = 1
		}
		character[i] = 33
	}
	// Nine digits in a row compress; not on primary data.
	start := 9
	if mode == 2 || mode == 3 {
		start = 0
	}
	count := 0
	for i := start; i < 144; i++ {
		if set[i] == 1 && character[i] >= 48 && character[i] <= 57 {
			count++
		} else {
			count = 0
		}
		if count == 9 {
			for j := i - 8; j <= i; j++ {
				set[j] = 6
			}
			count = 0
		}
	}
	// Shifts and latches.
	currSet := 1
	idx := 0
	for {
		if set[idx] != currSet && set[idx] != 6 {
			switch set[idx] {
			case 1:
				if currSet == 2 {
					if idx+1 < 144 && set[idx+1] == 1 {
						if idx+2 < 144 && set[idx+2] == 1 {
							if idx+3 < 144 && set[idx+3] == 1 {
								mxInsert(set, character, idx, &dataLen)
								character[idx] = 63 // latch A
								currSet = 1
								idx += 3
							} else {
								mxInsert(set, character, idx, &dataLen)
								character[idx] = 57 // triple shift A
								idx += 2
							}
						} else {
							mxInsert(set, character, idx, &dataLen)
							character[idx] = 56 // double shift A
							idx++
						}
					} else {
						mxInsert(set, character, idx, &dataLen)
						character[idx] = 59 // shift A
					}
				} else {
					mxInsert(set, character, idx, &dataLen)
					character[idx] = 58 // other sets latch A
					currSet = 1
				}
			case 2:
				if currSet != 1 || (idx+1 < 144 && set[idx+1] == 2) {
					mxInsert(set, character, idx, &dataLen)
					character[idx] = 63 // latch B
					currSet = 2
				} else {
					mxInsert(set, character, idx, &dataLen)
					character[idx] = 59 // shift B
				}
			case 3, 4, 5:
				if (idx == 0 && idx+3 < 144 && set[idx+1] == set[idx] && set[idx+2] == set[idx] && set[idx+3] == set[idx]) ||
					(idx > 0 && set[idx-1] == set[idx] && idx+2 < 144 && set[idx+1] == set[idx] && set[idx+2] == set[idx]) {
					if idx == 0 {
						mxInsert(set, character, idx, &dataLen)
						character[idx] = 60 + set[idx] - 3
						idx++
						mxInsert(set, character, idx, &dataLen)
						character[idx] = 60 + set[idx] - 3
						idx += 3
					} else {
						mxInsert(set, character, idx, &dataLen)
						character[idx-1] = 60 + set[idx] - 3
						idx += 2
					}
					currSet = set[idx]
				} else {
					mxInsert(set, character, idx, &dataLen)
					character[idx] = 60 + set[idx] - 3
				}
			}
			idx++ // allow for bump
		}
		idx++
		if idx >= 144 {
			break
		}
	}
	// Number compression.
	idx = 0
	for {
		if set[idx] == 6 {
			compressed := ""
			for j := 0; j < 9; j++ {
				compressed += string(rune(character[idx+j]))
			}
			value, err := strconv.Atoi(compressed)
			if err != nil {
				return fmt.Errorf("maxicode: bad compressed number: %w", err)
			}
			character[idx] = 31 // NS
			character[idx+1] = (value & 0x3f000000) >> 24
			character[idx+2] = (value & 0xfc0000) >> 18
			character[idx+3] = (value & 0x3f000) >> 12
			character[idx+4] = (value & 0xfc0) >> 6
			character[idx+5] = value & 0x3f
			idx += 6
			for j := idx; j < 140; j++ {
				set[j] = set[j+3]
				character[j] = character[j+3]
			}
			dataLen -= 3
		} else {
			idx++
		}
		if idx >= 144 {
			break
		}
	}
	switch mode {
	case 2, 3:
		if dataLen > 84 {
			return errors.New("maxicode: input data is too long")
		}
		for i := 0; i < 84; i++ {
			cw[i+20] = character[i]
		}
	case 4, 6:
		if dataLen > 93 {
			return errors.New("maxicode: input data is too long")
		}
		for i := 0; i < 9; i++ {
			cw[i+1] = character[i]
		}
		for i := 0; i < 84; i++ {
			cw[i+20] = character[i+9]
		}
	case 5:
		if dataLen > 77 {
			return errors.New("maxicode: input data is too long")
		}
		for i := 0; i < 9; i++ {
			cw[i+1] = character[i]
		}
		for i := 0; i < 68; i++ {
			cw[i+20] = character[i+9]
		}
	}
	return nil
}

func mxInsert(set, character []int, position int, dataLen *int) {
	for i := 143; i > position; i-- {
		set[i] = set[i-1]
		character[i] = character[i-1]
	}
	*dataLen++
}

func mxPrimaryCheck(cw []int) {
	data := make([]byte, 10)
	for j := 0; j < 10; j++ {
		data[j] = byte(cw[j])
	}
	result := mxRSFor(10).encode(data)
	for j := 0; j < 10; j++ {
		cw[10+j] = int(result[10-1-j])
	}
}

func mxSecondaryHalf(cw []int, dataLen, eccLen int, odd bool) {
	data := make([]byte, dataLen)
	if !odd {
		for j := 0; j < dataLen+1; j += 2 {
			data[j/2] = byte(cw[j+20])
		}
	} else {
		for j := 1; j < dataLen+1; j += 2 {
			data[(j-1)/2] = byte(cw[j+20])
		}
	}
	result := mxRSFor(eccLen).encode(data[:dataLen/2])
	for j := 0; j < eccLen; j++ {
		if !odd {
			cw[dataLen+2*j+20] = int(result[eccLen-1-j])
		} else {
			cw[dataLen+2*j+1+20] = int(result[eccLen-1-j])
		}
	}
}

func mxSecondaryCheckEven(cw []int, eccLen int) {
	dataLen := 68
	if eccLen == 20 {
		dataLen = 84
	}
	mxSecondaryHalf(cw, dataLen, eccLen, false)
}

func mxSecondaryCheckOdd(cw []int, eccLen int) {
	dataLen := 68
	if eccLen == 20 {
		dataLen = 84
	}
	mxSecondaryHalf(cw, dataLen, eccLen, true)
}
