// Package eanupc encodes EAN-13 and UPC-A symbols as bar/space bit strings.
//
// EAN-13 check digit (Mod-10): number the first 12 digits d1..d12 from the
// left; s = sum(d[i] for odd i) + 3*sum(d[i] for even i); the check digit
// is (10 - s%10) % 10. UPC-A uses the same Mod-10 scheme with the weights
// mirrored (odd positions from the left count triple, the number system
// digit included), which is identical to encoding the 12 UPC-A digits as
// an EAN-13 with a leading zero; hence EncodeUPCA delegates to the EAN-13
// core with a "0" prefix.
package eanupc

import (
	"fmt"
	"strings"
)

// tableL is the odd-parity (L) left-half digit encoding, index by digit.
var tableL = [10]string{
	"0001101", "0011001", "0010011", "0111101", "0100011",
	"0110001", "0101111", "0111011", "0110111", "0001011",
}

// tableG is the even-parity (G) left-half digit encoding, index by digit.
var tableG = [10]string{
	"0100111", "0110011", "0011011", "0100001", "0011101",
	"0111001", "0000101", "0010001", "0001001", "0010111",
}

// tableR is the right-half digit encoding, index by digit.
var tableR = [10]string{
	"1110010", "1100110", "1101100", "1000010", "1011100",
	"1001110", "1010000", "1000100", "1001000", "1110100",
}

// firstParity maps the leading EAN-13 digit to the parity pattern of the
// six left-half digits ('L' = odd, 'G' = even).
var firstParity = [10]string{
	"LLLLLL", "LLGLGG", "LLGGLG", "LLGGGL", "LGLLGG",
	"LGGLLG", "LGGGLL", "LGLGLG", "LGLGGL", "LGGLGL",
}

// Kind identifies the symbology of an encoded Code.
type Kind string

const (
	// KindEAN13 is a 13-digit EAN symbol.
	KindEAN13 Kind = "EAN-13"
	// KindUPCA is a 12-digit UPC symbol.
	KindUPCA Kind = "UPC-A"
)

// Code is an encoded EAN-13 or UPC-A symbol.
type Code struct {
	// Kind is the symbology (EAN-13 or UPC-A).
	Kind Kind
	// Digits holds the full digit string including the check digit
	// (13 digits for EAN-13, 12 for UPC-A).
	Digits string
	// Check is the Mod-10 check digit value (0-9).
	Check int
	// Bars is the 95-module bar/space pattern ('1' = bar, '0' = space),
	// including the left/center/right guards.
	Bars string
}

// Modules returns the total module count (always 95).
func (c *Code) Modules() int { return len(c.Bars) }

// allDigits reports whether s consists only of ASCII digits.
func allDigits(s string) bool {
	if s == "" {
		return false
	}
	for i := 0; i < len(s); i++ {
		if s[i] < '0' || s[i] > '9' {
			return false
		}
	}
	return true
}

// checkEAN13 returns the Mod-10 check digit for the first 12 digits of an
// EAN-13 symbol: s = sum of digits in odd positions (1-based from the
// left) + 3 * sum of digits in even positions; check = (10 - s%10) % 10.
func checkEAN13(d12 string) int {
	s := 0
	for i := 0; i < 12; i++ {
		d := int(d12[i] - '0')
		if i%2 == 0 {
			s += d
		} else {
			s += 3 * d
		}
	}
	return (10 - s%10) % 10
}

// encodeEAN13Digits builds the 95-module pattern for 13 validated digits.
func encodeEAN13Digits(d13 string) *Code {
	var b strings.Builder
	b.Grow(95)
	b.WriteString("101") // left guard
	par := firstParity[d13[0]-'0']
	for i := 1; i <= 6; i++ {
		d := d13[i] - '0'
		if par[i-1] == 'L' {
			b.WriteString(tableL[d])
		} else {
			b.WriteString(tableG[d])
		}
	}
	b.WriteString("01010") // center guard
	for i := 7; i <= 12; i++ {
		b.WriteString(tableR[d13[i]-'0'])
	}
	b.WriteString("101") // right guard
	return &Code{
		Kind:   KindEAN13,
		Digits: d13,
		Check:  int(d13[12] - '0'),
		Bars:   b.String(),
	}
}

// EncodeEAN13 encodes an EAN-13 symbol. content must hold 12 digits (the
// check digit is computed) or 13 digits (the check digit is verified and
// an error is returned on mismatch). Anything else, including non-digit
// characters, is an error.
func EncodeEAN13(content string) (*Code, error) {
	if !allDigits(content) || (len(content) != 12 && len(content) != 13) {
		return nil, fmt.Errorf("eanupc: EAN-13 needs 12 or 13 digits, got %q", content)
	}
	d12 := content
	if len(content) == 13 {
		d12 = content[:12]
	}
	check := checkEAN13(d12)
	if len(content) == 13 && int(content[12]-'0') != check {
		return nil, fmt.Errorf("eanupc: bad EAN-13 check digit in %q (want %d)", content, check)
	}
	return encodeEAN13Digits(d12 + string(rune('0'+check))), nil
}

// EncodeUPCA encodes a UPC-A symbol. content must hold 11 digits (the
// check digit is computed) or 12 digits (the check digit is verified and
// an error is returned on mismatch). A UPC-A is encoded exactly as an
// EAN-13 with a leading zero digit, so the returned Code carries the
// 12-digit UPC-A digit string while Bars equals the EAN-13 pattern of
// "0"+Digits.
func EncodeUPCA(content string) (*Code, error) {
	if !allDigits(content) || (len(content) != 11 && len(content) != 12) {
		return nil, fmt.Errorf("eanupc: UPC-A needs 11 or 12 digits, got %q", content)
	}
	ean, err := EncodeEAN13("0" + content[:11])
	if err != nil {
		return nil, err
	}
	if len(content) == 12 && int(content[11]-'0') != ean.Check {
		return nil, fmt.Errorf("eanupc: bad UPC-A check digit in %q (want %d)", content, ean.Check)
	}
	return &Code{
		Kind:   KindUPCA,
		Digits: content[:11] + string(rune('0'+ean.Check)),
		Check:  ean.Check,
		Bars:   ean.Bars,
	}, nil
}

// Encode encodes a 1D retail symbol from a digit string, dispatching on
// length: 11 or 12 digits encode as UPC-A, 12 or 13 digits as EAN-13.
// (A 12-digit input is read as UPC-A with its check digit, matching the
// retail convention that a bare 12-digit code is a UPC-A.) The check digit
// is computed when omitted and verified when present.
func Encode(content string) (*Code, error) {
	if allDigits(content) {
		switch len(content) {
		case 13:
			return EncodeEAN13(content)
		case 11, 12:
			return EncodeUPCA(content)
		default:
			return nil, fmt.Errorf("eanupc: need 11-13 digits, got %d digits", len(content))
		}
	}
	return nil, fmt.Errorf("eanupc: illegal charset in %q (digits only)", content)
}
