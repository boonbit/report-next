package fontmetrics

import "testing"

func TestFamousInvariants(t *testing.T) {
	for _, f := range []Face{Helvetica, HelveticaBold} {
		if got := f.Width(" ", 1000); got != 278 {
			t.Errorf("%v space = %v, want 278", f, got)
		}
		if got := f.Width("0123456789", 1000); got != 5560 {
			t.Errorf("%v digits = %v, want 5560", f, got)
		}
	}
	for _, f := range []Face{Courier, CourierBold} {
		if got := f.Width("M i~", 1000); got != 2400 {
			t.Errorf("%v uniform = %v, want 2400", f, got)
		}
	}
	if got := Helvetica.Width("AMW", 1000); got != 667+833+944 {
		t.Errorf("Helvetica AMW = %v", got)
	}
	if got := Helvetica.Width("A", 12); got != 667*12.0/1000 {
		t.Errorf("Helvetica A@12 = %v", got)
	}
	for r := 32; r <= 126; r++ {
		if helv[r-32] <= 0 || helvBold[r-32] <= 0 {
			t.Fatalf("non-positive width at %d", r)
		}
	}
}

func TestASCIIOnly(t *testing.T) {
	if !ASCIIOnly("Hello, 123!") {
		t.Error("ascii rejected")
	}
	if ASCIIOnly("สวัสดี") || ASCIIOnly("caf\u00e9") {
		t.Error("non-ascii accepted")
	}
}
