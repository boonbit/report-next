// Package code128 encodes Code 128-B symbols as module-width sequences.
//
// Each data symbol holds 6 elements (alternating bar/space, starting with
// a bar) whose widths sum to 11 modules; the stop symbol adds a 2-module
// termination bar (7 elements, 13 modules). The encoded symbol order is
// mandatory: START-B, data symbols, Mod-103 check symbol, STOP.
//
// Mod-103 check digit: with the start value 104 and data values v1..vn
// (Code-B value = ASCII code minus 32), the check value is
// (104 + v1*1 + v2*2 + ... + vn*n) mod 103.
package code128

import (
	"fmt"
	"strings"
)

// StartB is the Code-B start symbol value (104).
const StartB = 104

// Stop is the stop symbol value (106).
const Stop = 106

// Code is an encoded Code 128-B symbol.
type Code struct {
	// Content is the encoded string.
	Content string
	// Values holds the symbol values: StartB, one per character, the
	// Mod-103 check symbol, then Stop.
	Values []int
	// Widths holds the concatenated element widths (alternating bar and
	// space, starting and ending with a bar), including the stop
	// termination bar.
	Widths []int
	// Checksum is the Mod-103 check symbol value.
	Checksum int
}

// Modules returns the total symbol width in modules.
func (c *Code) Modules() int {
	n := 0
	for _, w := range c.Widths {
		n += w
	}
	return n
}

// Bars expands Widths into a bar/space bit string ('1' = bar, '0' =
// space).
func (c *Code) Bars() string {
	var b strings.Builder
	bar := true
	for _, w := range c.Widths {
		ch := '0'
		if bar {
			ch = '1'
		}
		for i := 0; i < w; i++ {
			b.WriteByte(byte(ch))
		}
		bar = !bar
	}
	return b.String()
}

// Encode encodes content as Code 128-B: START-B, one symbol per
// character, the Mod-103 check symbol, then STOP with its termination
// bar. Every character must be in the Code-B range U+0020..U+007F
// (value = ASCII code minus 32); anything else, including an empty
// string, is an error.
func Encode(content string) (*Code, error) {
	if content == "" {
		return nil, fmt.Errorf("code128: empty content")
	}
	vals := make([]int, 0, len(content)+3)
	vals = append(vals, StartB)
	sum := StartB
	for i := 0; i < len(content); i++ {
		ch := content[i]
		if ch < 0x20 || ch > 0x7F {
			return nil, fmt.Errorf("code128: illegal character 0x%02X in %q (Code-B needs U+0020..U+007F)", ch, content)
		}
		v := int(ch - 0x20)
		vals = append(vals, v)
		sum += v * (len(vals) - 1)
	}
	check := sum % 103
	vals = append(vals, check, Stop)
	widths := make([]int, 0, 11*(len(vals)-1)+13)
	for _, v := range vals {
		widths = append(widths, table[v]...)
	}
	return &Code{Content: content, Values: vals, Widths: widths, Checksum: check}, nil
}

// table holds the element widths of symbols 0-105 (6 elements each) and
// the stop symbol 106 (7 elements, including the termination bar).
// Values match ISO/IEC 15417; cross-checked against the independent
// reportlab code128 pattern table and Wikipedia bit patterns.
var table = [107][]int{
	{2, 1, 2, 2, 2, 2}, {2, 2, 2, 1, 2, 2}, {2, 2, 2, 2, 2, 1},
	{1, 2, 1, 2, 2, 3}, {1, 2, 1, 3, 2, 2}, {1, 3, 1, 2, 2, 2},
	{1, 2, 2, 2, 1, 3}, {1, 2, 2, 3, 1, 2}, {1, 3, 2, 2, 1, 2},
	{2, 2, 1, 2, 1, 3}, {2, 2, 1, 3, 1, 2}, {2, 3, 1, 2, 1, 2},
	{1, 1, 2, 2, 3, 2}, {1, 2, 2, 1, 3, 2}, {1, 2, 2, 2, 3, 1},
	{1, 1, 3, 2, 2, 2}, {1, 2, 3, 1, 2, 2}, {1, 2, 3, 2, 2, 1},
	{2, 2, 3, 2, 1, 1}, {2, 2, 1, 1, 3, 2}, {2, 2, 1, 2, 3, 1},
	{2, 1, 3, 2, 1, 2}, {2, 2, 3, 1, 1, 2}, {3, 1, 2, 1, 3, 1},
	{3, 1, 1, 2, 2, 2}, {3, 2, 1, 1, 2, 2}, {3, 2, 1, 2, 2, 1},
	{3, 1, 2, 2, 1, 2}, {3, 2, 2, 1, 1, 2}, {3, 2, 2, 2, 1, 1},
	{2, 1, 2, 1, 2, 3}, {2, 1, 2, 3, 2, 1}, {2, 3, 2, 1, 2, 1},
	{1, 1, 1, 3, 2, 3}, {1, 3, 1, 1, 2, 3}, {1, 3, 1, 3, 2, 1},
	{1, 1, 2, 3, 1, 3}, {1, 3, 2, 1, 1, 3}, {1, 3, 2, 3, 1, 1},
	{2, 1, 1, 3, 1, 3}, {2, 3, 1, 1, 1, 3}, {2, 3, 1, 3, 1, 1},
	{1, 1, 2, 1, 3, 3}, {1, 1, 2, 3, 3, 1}, {1, 3, 2, 1, 3, 1},
	{1, 1, 3, 1, 2, 3}, {1, 1, 3, 3, 2, 1}, {1, 3, 3, 1, 2, 1},
	{3, 1, 3, 1, 2, 1}, {2, 1, 1, 3, 3, 1}, {2, 3, 1, 1, 3, 1},
	{2, 1, 3, 1, 1, 3}, {2, 1, 3, 3, 1, 1}, {2, 1, 3, 1, 3, 1},
	{3, 1, 1, 1, 2, 3}, {3, 1, 1, 3, 2, 1}, {3, 3, 1, 1, 2, 1},
	{3, 1, 2, 1, 1, 3}, {3, 1, 2, 3, 1, 1}, {3, 3, 2, 1, 1, 1},
	{3, 1, 4, 1, 1, 1}, {2, 2, 1, 4, 1, 1}, {4, 3, 1, 1, 1, 1},
	{1, 1, 1, 2, 2, 4}, {1, 1, 1, 4, 2, 2}, {1, 2, 1, 1, 2, 4},
	{1, 2, 1, 4, 2, 1}, {1, 4, 1, 1, 2, 2}, {1, 4, 1, 2, 2, 1},
	{1, 1, 2, 2, 1, 4}, {1, 1, 2, 4, 1, 2}, {1, 2, 2, 1, 1, 4},
	{1, 2, 2, 4, 1, 1}, {1, 4, 2, 1, 1, 2}, {1, 4, 2, 2, 1, 1},
	{2, 4, 1, 2, 1, 1}, {2, 2, 1, 1, 1, 4}, {4, 1, 3, 1, 1, 1},
	{2, 4, 1, 1, 1, 2}, {1, 3, 4, 1, 1, 1}, {1, 1, 1, 2, 4, 2},
	{1, 2, 1, 1, 4, 2}, {1, 2, 1, 2, 4, 1}, {1, 1, 4, 2, 1, 2},
	{1, 2, 4, 1, 1, 2}, {1, 2, 4, 2, 1, 1}, {4, 1, 1, 2, 1, 2},
	{4, 2, 1, 1, 1, 2}, {4, 2, 1, 2, 1, 1}, {2, 1, 2, 1, 4, 1},
	{2, 1, 4, 1, 2, 1}, {4, 1, 2, 1, 2, 1}, {1, 1, 1, 1, 4, 3},
	{1, 1, 1, 3, 4, 1}, {1, 3, 1, 1, 4, 1}, {1, 1, 4, 1, 1, 3},
	{1, 1, 4, 3, 1, 1}, {4, 1, 1, 1, 1, 3}, {4, 1, 1, 3, 1, 1},
	{1, 1, 3, 1, 4, 1}, {1, 1, 4, 1, 3, 1}, {3, 1, 1, 1, 4, 1},
	{4, 1, 1, 1, 3, 1}, {2, 1, 1, 4, 1, 2}, {2, 1, 1, 2, 1, 4},
	{2, 1, 1, 2, 3, 2}, {2, 3, 3, 1, 1, 1, 2},
}
