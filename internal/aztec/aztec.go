package aztec

// Aztec Code symbol assembly: layer search, bit stuffing, Reed-Solomon
// check words, mode message, bulls-eye and alignment marks, following the
// published algorithm. Compact layers 1..4 use an 11+4L base grid, full
// layers 1..32 a 14+4L base grid plus alignment rows/columns.

import (
	"fmt"
	"strings"
)

const (
	azDefaultECCPercent = 33
	azMaxLayers         = 32
	azMaxCompactLayers  = 4
)

// azWordSize maps the layer index to the data word size.
var azWordSize = []int{
	4, 6, 6, 8, 8, 8, 8, 8, 8, 10, 10, 10, 10, 10, 10, 10, 10, 10, 10, 10, 10, 10, 10,
	12, 12, 12, 12, 12, 12, 12, 12, 12, 12,
}

// Code is one Aztec symbol: a square module grid (row-major, true = dark).
type Code struct {
	data []byte
	size int
	bars [][]bool
}

// Encode builds an Aztec symbol for data with at least eccPercent error
// correction (default 33 when negative). Layers selects the size: 0 means
// automatic, negative -L forces compact with L layers, positive forces full
// with L layers (at most 32).
func Encode(data []byte, eccPercent int, layers int) (*Code, error) {
	if eccPercent < 0 {
		eccPercent = azDefaultECCPercent
	}
	bits := azHighLevelEncode(data)
	eccBits := bits.Len()*eccPercent/100 + 11
	totalSizeBits := bits.Len() + eccBits
	var (
		nLayers, totalInLayer, wordSize int
		compact                         bool
		stuffed                         *azBits
	)
	if layers != 0 {
		compact = layers < 0
		if compact {
			nLayers = -layers
		} else {
			nLayers = layers
		}
		if (compact && nLayers > azMaxCompactLayers) || (!compact && nLayers > azMaxLayers) {
			return nil, fmt.Errorf("aztec: illegal layer count %d", layers)
		}
		totalInLayer = azTotalBitsInLayer(nLayers, compact)
		wordSize = azWordSize[nLayers]
		usable := totalInLayer - totalInLayer%wordSize
		stuffed = azStuffBits(bits, wordSize)
		if stuffed.Len()+eccBits > usable {
			return nil, fmt.Errorf("aztec: data too large for %d layers", layers)
		}
		if compact && stuffed.Len() > wordSize*64 {
			return nil, fmt.Errorf("aztec: data too large for compact layer")
		}
	} else {
		// Sizes in order Compact1..4, Normal4..32: Normal(i) for i < 4 is
		// skipped since Compact(i+1) matches its size with more capacity.
		for i := 0; ; i++ {
			if i > azMaxLayers {
				return nil, fmt.Errorf("aztec: data too large for an aztec code")
			}
			compact = i <= 3
			nLayers = i
			if compact {
				nLayers = i + 1
			}
			totalInLayer = azTotalBitsInLayer(nLayers, compact)
			if totalSizeBits > totalInLayer {
				continue
			}
			if wordSize != azWordSize[nLayers] {
				wordSize = azWordSize[nLayers]
				stuffed = azStuffBits(bits, wordSize)
			}
			usable := totalInLayer - totalInLayer%wordSize
			if compact && stuffed.Len() > wordSize*64 {
				continue
			}
			if stuffed.Len()+eccBits <= usable {
				break
			}
		}
	}
	message := azCheckWords(stuffed, totalInLayer, wordSize)
	words := stuffed.Len() / wordSize
	mode := azModeMessage(compact, nLayers, words)

	var base int
	if compact {
		base = 11 + nLayers*4
	} else {
		base = 14 + nLayers*4
	}
	align := make([]int, base)
	var size int
	if compact {
		// No alignment marks in compact mode: the map is identity.
		size = base
		for i := range align {
			align[i] = i
		}
	} else {
		size = base + 1 + 2*((base/2-1)/15)
		origCenter, center := base/2, size/2
		for i := 0; i < origCenter; i++ {
			off := i + i/15
			align[origCenter-i-1] = center - off - 1
			align[origCenter+i] = center + off + 1
		}
	}
	bars := make([][]bool, size)
	for i := range bars {
		bars[i] = make([]bool, size)
	}
	set := func(x, y int) { bars[y][x] = true }
	// Data bits, four sides per layer.
	for i, rowOff := 0, 0; i < nLayers; i++ {
		rowSize := (nLayers-i)*4 + 9
		if !compact {
			rowSize += 3
		}
		for j := 0; j < rowSize; j++ {
			colOff := j * 2
			for k := 0; k < 2; k++ {
				if message.Get(rowOff + colOff + k) {
					set(align[i*2+k], align[i*2+j])
				}
				if message.Get(rowOff + rowSize*2 + colOff + k) {
					set(align[i*2+j], align[base-1-i*2-k])
				}
				if message.Get(rowOff + rowSize*4 + colOff + k) {
					set(align[base-1-i*2-k], align[base-1-i*2-j])
				}
				if message.Get(rowOff + rowSize*6 + colOff + k) {
					set(align[base-1-i*2-j], align[i*2+k])
				}
			}
		}
		rowOff += rowSize * 8
	}
	azDrawModeMessage(set, compact, size, mode)
	if compact {
		azDrawBullsEye(set, size/2, 5)
	} else {
		azDrawBullsEye(set, size/2, 7)
		for i, j := 0, 0; i < base/2-1; i, j = i+15, j+16 {
			for k := (size / 2) & 1; k < size; k += 2 {
				set(size/2-j, k)
				set(size/2+j, k)
				set(k, size/2-j)
				set(k, size/2+j)
			}
		}
	}
	return &Code{data: append([]byte(nil), data...), size: size, bars: bars}, nil
}

// Size reports the symbol edge in modules.
func (c *Code) Size() int { return c.size }

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

// SVG renders the symbol as dark unit rects on a white background.
func (c *Code) SVG() string {
	var sb strings.Builder
	fmt.Fprintf(&sb, `<svg viewBox="0 0 %d %d" xmlns="http://www.w3.org/2000/svg">`, c.size, c.size)
	sb.WriteString(`<rect width="100%" height="100%" fill="white"/>`)
	for y := 0; y < c.size; y++ {
		for x := 0; x < c.size; x++ {
			if c.bars[y][x] {
				fmt.Fprintf(&sb, `<rect x="%d" y="%d" width="1" height="1"/>`, x, y)
			}
		}
	}
	sb.WriteString(`</svg>`)
	return sb.String()
}

func azTotalBitsInLayer(layers int, compact bool) int {
	tmp := 112
	if compact {
		tmp = 88
	}
	return (tmp + 16*layers) * layers
}

// azStuffBits pads words of all-ones/all-zeros with an extra bit so no
// word is degenerate.
func azStuffBits(bits *azBits, wordSize int) *azBits {
	out := new(azBits)
	n := bits.Len()
	mask := (1 << uint(wordSize)) - 2
	for i := 0; i < n; i += wordSize {
		word := 0
		for j := 0; j < wordSize; j++ {
			if i+j >= n || bits.Get(i+j) {
				word |= 1 << uint(wordSize-1-j)
			}
		}
		switch word & mask {
		case mask:
			out.AddBits(word&mask, byte(wordSize))
			i--
		case 0:
			out.AddBits(word|1, byte(wordSize))
			i--
		default:
			out.AddBits(word, byte(wordSize))
		}
	}
	return out
}

func azGF(wordSize int) *azField {
	switch wordSize {
	case 4:
		return azNewField(0x13, 16, 1)
	case 6:
		return azNewField(0x43, 64, 1)
	case 8:
		return azNewField(0x012D, 256, 1)
	case 10:
		return azNewField(0x409, 1024, 1)
	case 12:
		return azNewField(0x1069, 4096, 1)
	}
	return nil
}

func azWords(bits *azBits, wordSize, count int) []int {
	out := make([]int, count)
	for i := 0; i < count; i++ {
		v := 0
		for j := 0; j < wordSize; j++ {
			if bits.Get(i*wordSize + j) {
				v |= 1 << uint(wordSize-j-1)
			}
		}
		out[i] = v
	}
	return out
}

// azCheckWords appends start padding plus the Reed-Solomon check words.
func azCheckWords(bits *azBits, totalBits, wordSize int) *azBits {
	rs := azNewRS(azGF(wordSize))
	// bits is a multiple of wordSize, so no end padding is needed.
	msgCount := bits.Len() / wordSize
	eccCount := totalBits/wordSize - msgCount
	words := azWords(bits, wordSize, msgCount)
	ecc := rs.encode(words, eccCount)
	out := new(azBits)
	out.AddBits(0, byte(totalBits%wordSize))
	for _, w := range words {
		out.AddBits(w, byte(wordSize))
	}
	for _, w := range ecc {
		out.AddBits(w, byte(wordSize))
	}
	return out
}

func azModeMessage(compact bool, layers, words int) *azBits {
	mode := new(azBits)
	if compact {
		mode.AddBits(layers-1, 2)
		mode.AddBits(words-1, 6)
		return azCheckWords(mode, 28, 4)
	}
	mode.AddBits(layers-1, 5)
	mode.AddBits(words-1, 11)
	return azCheckWords(mode, 40, 4)
}

func azDrawModeMessage(set func(x, y int), compact bool, size int, mode *azBits) {
	center := size / 2
	if compact {
		for i := 0; i < 7; i++ {
			off := center - 3 + i
			if mode.Get(i) {
				set(off, center-5)
			}
			if mode.Get(i + 7) {
				set(center+5, off)
			}
			if mode.Get(20 - i) {
				set(off, center+5)
			}
			if mode.Get(27 - i) {
				set(center-5, off)
			}
		}
		return
	}
	for i := 0; i < 10; i++ {
		off := center - 5 + i + i/5
		if mode.Get(i) {
			set(off, center-7)
		}
		if mode.Get(i + 10) {
			set(center+7, off)
		}
		if mode.Get(29 - i) {
			set(off, center+7)
		}
		if mode.Get(39 - i) {
			set(center-7, off)
		}
	}
}

func azDrawBullsEye(set func(x, y int), center, size int) {
	for i := 0; i < size; i += 2 {
		for j := center - i; j <= center+i; j++ {
			set(j, center-i)
			set(j, center+i)
			set(center-i, j)
			set(center+i, j)
		}
	}
	set(center-size, center-size)
	set(center-size+1, center-size)
	set(center-size, center-size+1)
	set(center+size, center-size)
	set(center+size, center-size+1)
	set(center+size, center+size-1)
}
