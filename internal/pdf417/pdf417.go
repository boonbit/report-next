package pdf417

// PDF417 symbol assembly: dimension search, length prefix + padding, EC
// append, per-row start/left/data/right/stop codewords mapped through the
// cluster tables into 17/18-module bars.

import (
	"fmt"
	"math"
	"strings"
)

const (
	pdfMinCols = 2
	pdfMaxCols = 30
	pdfMaxRows = 30
	pdfMinRows = 2

	pdfPaddingCodeword = 900
	pdfPreferredRatio  = 3.0
)

// Code is one PDF417 symbol:.cols data columns, rows, and the full module
// grid including start/stop patterns (row-major, true = dark).
type Code struct {
	data []byte
	cols int
	rows int
	bars [][]bool
}

// Encode builds a PDF417 symbol for data at EC security level 0..8.
// Columns are chosen automatically; level defaults to 2 when negative.
func Encode(data []byte, level int) (*Code, error) {
	if level < 0 {
		level = 2
	}
	if level > 8 {
		return nil, fmt.Errorf("pdf417: security level %d out of range 0..8", level)
	}
	sl := pdfLevel(byte(level))
	words, err := pdfHighLevelEncode(data)
	if err != nil {
		return nil, err
	}
	cols, rows := pdfDimensions(len(words), sl.ecWords())
	if cols < pdfMinCols || cols > pdfMaxCols || rows < pdfMinRows || rows > pdfMaxRows {
		return nil, fmt.Errorf("pdf417: unable to fit %d bytes", len(data))
	}
	full, err := pdfAssemble(words, cols, sl)
	if err != nil {
		return nil, err
	}
	grid := [][]int{}
	for i := 0; i < len(full); i += cols {
		end := i + cols
		if end > len(full) {
			end = len(full)
		}
		grid = append(grid, full[i:end])
	}
	bars := make([][]bool, 0, rows)
	for rowNum, row := range grid {
		table := rowNum % 3
		bits := []bool{}
		emit := func(word, n int) {
			for i := n - 1; i >= 0; i-- {
				bits = append(bits, word>>uint(i)&1 == 1)
			}
		}
		emit(start_word, 17)
		emit(getCodeword(table, pdfLeftCodeword(rowNum, rows, cols, level)), 17)
		for _, w := range row {
			emit(getCodeword(table, w), 17)
		}
		emit(getCodeword(table, pdfRightCodeword(rowNum, rows, cols, level)), 17)
		emit(stop_word, 18)
		bars = append(bars, bits)
	}
	return &Code{data: append([]byte(nil), data...), cols: cols, rows: rows, bars: bars}, nil
}

// Cols reports the data column count (excluding start/stop codewords).
func (c *Code) Cols() int { return c.cols }

// Rows reports the row count.
func (c *Code) Rows() int { return c.rows }

// Width reports the full module width including start/stop patterns.
func (c *Code) Width() int {
	if len(c.bars) == 0 {
		return 0
	}
	return len(c.bars[0])
}

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
	fmt.Fprintf(&sb, `<svg viewBox="0 0 %d %d" xmlns="http://www.w3.org/2000/svg">`, c.Width(), c.rows)
	sb.WriteString(`<rect width="100%" height="100%" fill="white"/>`)
	for y := 0; y < c.rows; y++ {
		for x := 0; x < c.Width(); x++ {
			if c.bars[y][x] {
				fmt.Fprintf(&sb, `<rect x="%d" y="%d" width="1" height="1"/>`, x, y)
			}
		}
	}
	sb.WriteString(`</svg>`)
	return sb.String()
}

func pdfRowCount(m, k, c int) int {
	r := (m+1+k)/c + 1
	if c*r >= m+1+k+c {
		r--
	}
	return r
}

func pdfDimensions(dataWords, ecWords int) (cols, rows int) {
	ratio := 0.0
	for c := pdfMinCols; c <= pdfMaxCols; c++ {
		r := pdfRowCount(dataWords, ecWords, c)
		if r < pdfMinRows {
			break
		}
		if r > pdfMaxRows {
			continue
		}
		newRatio := float64(17*cols+69) / float64(rows*2)
		if rows != 0 && math.Abs(newRatio-pdfPreferredRatio) > math.Abs(ratio-pdfPreferredRatio) {
			continue
		}
		ratio = newRatio
		cols = c
		rows = r
	}
	if rows == 0 {
		if r := pdfRowCount(dataWords, ecWords, pdfMinCols); r < pdfMinRows {
			rows, cols = pdfMinRows, pdfMinCols
		}
	}
	return cols, rows
}

func pdfAssemble(dataWords []int, cols int, sl pdfLevel) ([]int, error) {
	ecCount := sl.ecWords()
	total := len(dataWords) + ecCount + 1
	pad := []int{}
	if mod := total % cols; mod > 0 {
		for i := 0; i < cols-mod; i++ {
			pad = append(pad, pdfPaddingCodeword)
		}
	}
	dataWords = append(dataWords, pad...)
	full := append([]int{len(dataWords) + 1}, dataWords...)
	return append(full, sl.compute(full)...), nil
}

func pdfLeftCodeword(rowNum, rows, cols, level int) int {
	var x int
	switch rowNum % 3 {
	case 0:
		x = (rows - 3) / 3
	case 1:
		x = level*3 + (rows-1)%3
	case 2:
		x = cols - 1
	}
	return 30*(rowNum/3) + x
}

func pdfRightCodeword(rowNum, rows, cols, level int) int {
	var x int
	switch rowNum % 3 {
	case 0:
		x = cols - 1
	case 1:
		x = (rows - 1) / 3
	case 2:
		x = level*3 + (rows-1)%3
	}
	return 30*(rowNum/3) + x
}
