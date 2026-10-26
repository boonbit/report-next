package code128

import (
	"reflect"
	"testing"
)

// TestTableIntegrity guards the 107-entry code table: every data/start
// symbol has 6 elements summing to 11 modules, the stop symbol has 7
// elements summing to 13 modules.
func TestTableIntegrity(t *testing.T) {
	if len(table) != 107 {
		t.Fatalf("want 107 symbols, got %d", len(table))
	}
	for i, w := range table {
		n, sum := len(w), 0
		for _, e := range w {
			if e < 1 || e > 4 {
				t.Fatalf("symbol %d: element width %d out of range", i, e)
			}
			sum += e
		}
		if i < 106 && (n != 6 || sum != 11) {
			t.Fatalf("symbol %d: want 6 elements / 11 modules, got %d / %d", i, n, sum)
		}
		if i == 106 && (n != 7 || sum != 13) {
			t.Fatalf("stop: want 7 elements / 13 modules, got %d / %d", n, sum)
		}
	}
}

// TestEncodeA is a published-vector golden: "A" is Code-B value 33, so
// the Mod-103 check is (104+33*1)%103 = 34. The bit string matches the
// Wikipedia Code 128 reference patterns for START-B (11010010000),
// 'A' (10100011000), value 34 (10001011000) and STOP (1100011101011),
// and the independent reportlab oracle.
func TestEncodeA(t *testing.T) {
	c, err := Encode("A")
	if err != nil {
		t.Fatal(err)
	}
	if c.Checksum != 34 {
		t.Fatalf("checksum = %d, want 34", c.Checksum)
	}
	wantVals := []int{StartB, 33, 34, Stop}
	if !reflect.DeepEqual(c.Values, wantVals) {
		t.Fatalf("values = %v, want %v", c.Values, wantVals)
	}
	wantWidths := []int{2, 1, 1, 2, 1, 4, 1, 1, 1, 3, 2, 3, 1, 3, 1, 1, 2, 3, 2, 3, 3, 1, 1, 1, 2}
	if !reflect.DeepEqual(c.Widths, wantWidths) {
		t.Fatalf("widths = %v, want %v", c.Widths, wantWidths)
	}
	if c.Modules() != 46 {
		t.Fatalf("modules = %d, want 46", c.Modules())
	}
	wantBars := "11010010000" + "10100011000" + "10001011000" + "1100011101011"
	if c.Bars() != wantBars {
		t.Fatalf("bars = %s, want %s", c.Bars(), wantBars)
	}
}

// TestEncodeHi golden from the reportlab oracle: H=40, i=73, check =
// (104+40*1+73*2)%103 = 290%103 = 84, 57 modules total.
func TestEncodeHi(t *testing.T) {
	c, err := Encode("Hi")
	if err != nil {
		t.Fatal(err)
	}
	if c.Checksum != 84 {
		t.Fatalf("checksum = %d, want 84", c.Checksum)
	}
	if !reflect.DeepEqual(c.Values, []int{StartB, 40, 73, 84, Stop}) {
		t.Fatalf("values = %v", c.Values)
	}
	if c.Modules() != 57 {
		t.Fatalf("modules = %d, want 57", c.Modules())
	}
	wantBars := "110100100001100010100010000110100100111101001100011101011"
	if c.Bars() != wantBars {
		t.Fatalf("bars = %s, want %s", c.Bars(), wantBars)
	}
}

// TestEncodeSentence golden from the reportlab oracle, exercising the
// full printable ASCII range, space and punctuation.
func TestEncodeSentence(t *testing.T) {
	c, err := Encode("Hello, World!")
	if err != nil {
		t.Fatal(err)
	}
	if c.Checksum != 76 {
		t.Fatalf("checksum = %d, want 76", c.Checksum)
	}
	wantBars := "1101001000011000101000101100100001100101000011001010000100011110101011001110011011001100111010001101000111101010010011110110010100001000010011011001101100110010100001100011101011"
	if c.Bars() != wantBars {
		t.Fatalf("bars = %s, want %s", c.Bars(), wantBars)
	}
}

// TestEncodeStartStop verifies the mandatory framing symbols around the
// payload for a longer input.
func TestEncodeStartStop(t *testing.T) {
	c, err := Encode("0123")
	if err != nil {
		t.Fatal(err)
	}
	if c.Values[0] != StartB || c.Values[len(c.Values)-1] != Stop {
		t.Fatalf("values = %v, want START-B ... STOP", c.Values)
	}
	if !reflect.DeepEqual(c.Widths[:6], []int{2, 1, 1, 2, 1, 4}) {
		t.Fatalf("first widths = %v, want START-B 211214", c.Widths[:6])
	}
	if !reflect.DeepEqual(c.Widths[len(c.Widths)-7:], []int{2, 3, 3, 1, 1, 1, 2}) {
		t.Fatalf("last widths = %v, want STOP 2331112", c.Widths[len(c.Widths)-7:])
	}
}

// TestEncodeCharset rejects every byte outside Code-B (U+0020..U+007F)
// and the empty string, while accepting the boundary characters.
func TestEncodeCharset(t *testing.T) {
	for _, content := range []string{"", "A\nB", "caf\xc3\xa9", "\x1f", "\x80", "tab\there"} {
		if _, err := Encode(content); err == nil {
			t.Fatalf("Encode(%q): want error, got nil", content)
		}
	}
	for _, content := range []string{" ", "~", "\x7f", "AZaz09 !"} {
		c, err := Encode(content)
		if err != nil {
			t.Fatalf("Encode(%q): unexpected error: %v", content, err)
		}
		if c.Content != content {
			t.Fatalf("Content = %q, want %q", c.Content, content)
		}
	}
	// DEL (U+007F) is Code-B value 95.
	c, err := Encode("\x7f")
	if err != nil {
		t.Fatal(err)
	}
	if c.Values[1] != 95 {
		t.Fatalf("DEL value = %d, want 95", c.Values[1])
	}
}
