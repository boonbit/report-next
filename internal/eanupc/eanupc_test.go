package eanupc

import (
	"strings"
	"testing"
)

// ean5901234123457 is the reportlab-oracle pattern for the published
// vector 5901234123457 (check digit 7).
const ean5901234123457 = "101" +
	"0001011" + "0100111" + "0110011" + "0010011" + "0111101" + "0011101" +
	"01010" +
	"1100110" + "1101100" + "1000010" + "1011100" + "1001110" + "1000100" +
	"101"

// upca036000291452 is the reportlab-oracle pattern for UPC-A
// 036000291452 (check digit 2), identical to EAN-13 0036000291452.
const upca036000291452 = "101" +
	"0001101" + "0111101" + "0101111" + "0001101" + "0001101" + "0001101" +
	"01010" +
	"1101100" + "1110100" + "1100110" + "1011100" + "1001110" + "1101100" +
	"101"

// TestEAN13Golden encodes the published vector 5901234123457: check =
// (10 - (17+3*22)%10)%10 = 7, verified against the reportlab oracle.
func TestEAN13Golden(t *testing.T) {
	c, err := EncodeEAN13("590123412345")
	if err != nil {
		t.Fatal(err)
	}
	if c.Kind != KindEAN13 {
		t.Fatalf("kind = %s, want EAN-13", c.Kind)
	}
	if c.Digits != "5901234123457" {
		t.Fatalf("digits = %s, want 5901234123457", c.Digits)
	}
	if c.Check != 7 {
		t.Fatalf("check = %d, want 7", c.Check)
	}
	if c.Modules() != 95 || len(c.Bars) != 95 {
		t.Fatalf("modules = %d, want 95", c.Modules())
	}
	if c.Bars != ean5901234123457 {
		t.Fatalf("bars = %s, want %s", c.Bars, ean5901234123457)
	}
	// Guards: left "101", center "01010", right "101".
	if !strings.HasPrefix(c.Bars, "101") || c.Bars[45:50] != "01010" || !strings.HasSuffix(c.Bars, "101") {
		t.Fatalf("bad guards in %s", c.Bars)
	}
}

// TestEAN13CheckVerify recomputes a second vector by hand: for
// 400638133393, odd-position sum = 4+0+3+1+3+9 = 20, even-position sum =
// 0+6+8+3+3+3 = 23, check = (10-(20+69)%10)%10 = 1 (matches the
// reportlab oracle check digit).
func TestEAN13CheckVerify(t *testing.T) {
	c, err := EncodeEAN13("400638133393")
	if err != nil {
		t.Fatal(err)
	}
	if c.Check != 1 || c.Digits != "4006381333931" {
		t.Fatalf("got digits %s check %d, want 4006381333931 check 1", c.Digits, c.Check)
	}
	if _, err := EncodeEAN13("4006381333931"); err != nil {
		t.Fatalf("valid 13-digit input rejected: %v", err)
	}
	if _, err := EncodeEAN13("4006381333932"); err == nil {
		t.Fatal("wrong check digit accepted, want error")
	}
}

// TestUPCAGolden encodes the published vector 036000291452 (check 2)
// and requires bit-identity with EAN-13 0036000291452.
func TestUPCAGolden(t *testing.T) {
	c, err := EncodeUPCA("03600029145")
	if err != nil {
		t.Fatal(err)
	}
	if c.Kind != KindUPCA {
		t.Fatalf("kind = %s, want UPC-A", c.Kind)
	}
	if c.Digits != "036000291452" {
		t.Fatalf("digits = %s, want 036000291452", c.Digits)
	}
	if c.Check != 2 {
		t.Fatalf("check = %d, want 2", c.Check)
	}
	if c.Modules() != 95 {
		t.Fatalf("modules = %d, want 95", c.Modules())
	}
	if c.Bars != upca036000291452 {
		t.Fatalf("bars = %s, want %s", c.Bars, upca036000291452)
	}
	ean, err := EncodeEAN13("0036000291452")
	if err != nil {
		t.Fatal(err)
	}
	if c.Bars != ean.Bars {
		t.Fatal("UPC-A pattern differs from EAN-13 with leading zero")
	}
	if _, err := EncodeUPCA("036000291452"); err != nil {
		t.Fatalf("valid 12-digit UPC-A rejected: %v", err)
	}
	if _, err := EncodeUPCA("036000291453"); err == nil {
		t.Fatal("wrong UPC-A check digit accepted, want error")
	}
}

// TestEncodeDispatch routes 13-digit input to EAN-13 and 11/12-digit
// input to UPC-A.
func TestEncodeDispatch(t *testing.T) {
	c, err := Encode("5901234123457")
	if err != nil {
		t.Fatal(err)
	}
	if c.Kind != KindEAN13 || c.Check != 7 {
		t.Fatalf("got %+v, want EAN-13 check 7", c)
	}
	c, err = Encode("036000291452")
	if err != nil {
		t.Fatal(err)
	}
	if c.Kind != KindUPCA || c.Check != 2 {
		t.Fatalf("got %+v, want UPC-A check 2", c)
	}
	c, err = Encode("03600029145")
	if err != nil {
		t.Fatal(err)
	}
	if c.Kind != KindUPCA || c.Digits != "036000291452" {
		t.Fatalf("got %+v, want UPC-A 036000291452", c)
	}
}

// TestIllegalCharset rejects non-digits, wrong lengths and empties.
func TestIllegalCharset(t *testing.T) {
	for _, content := range []string{
		"", "59012341234A", "590123412345 ", "590-123412345",
		"03600029145X", "123", "12345678901234", "12 34", "１２３",
	} {
		if _, err := Encode(content); err == nil {
			t.Fatalf("Encode(%q): want error, got nil", content)
		}
	}
	if _, err := EncodeEAN13("59012341234"); err == nil {
		t.Fatal("11-digit EAN-13 accepted, want error")
	}
	if _, err := EncodeUPCA("0360002914"); err == nil {
		t.Fatal("10-digit UPC-A accepted, want error")
	}
}
