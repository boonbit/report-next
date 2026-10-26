package report

import (
	"bytes"
	"os"
	"reflect"
	"testing"
)

func loadFile(t *testing.T, name string) Report {
	t.Helper()
	f, err := os.Open("testdata/" + name)
	if err != nil {
		t.Fatal(err)
	}
	defer f.Close()
	rep, err := ParseFRX(f)
	if err != nil {
		t.Fatal(err)
	}
	return rep
}

func roundTrip(t *testing.T, rep Report) Report {
	t.Helper()
	var buf bytes.Buffer
	if err := WriteFRX(&buf, rep); err != nil {
		t.Fatal(err)
	}
	rep2, err := ParseFRX(&buf)
	if err != nil {
		t.Fatal(err)
	}
	if !reflect.DeepEqual(rep, rep2) {
		t.Fatal("reparsed report differs from original parse")
	}
	return rep2
}

func TestParseBoxStyles(t *testing.T) {
	rep := loadFile(t, "box.frx")
	if len(rep.Styles) != 3 {
		t.Fatalf("styles = %d, want 3", len(rep.Styles))
	}
	if rep.Styles[0].Name != "Style1" {
		t.Fatalf("styles[0] = %q", rep.Styles[0].Name)
	}
	if !Valid(rep) {
		t.Fatalf("box.frx invalid: %v", Validate(rep))
	}
	roundTrip(t, rep)
}

func TestValidateFailures(t *testing.T) {
	rep := loadFile(t, "hello.frx")
	if !Valid(rep) {
		t.Fatalf("hello.frx invalid: %v", Validate(rep))
	}
	bad := rep
	bad.Version = "report/v0"
	if Valid(bad) {
		t.Error("bad version accepted")
	}
	dup := rep
	dup.Pages = append(dup.Pages, dup.Pages[0])
	if Valid(dup) {
		t.Error("duplicate page accepted")
	}
	dangle := rep
	dangle.Pages[0].Bands[0].Objects[0].Attrs = append(
		dangle.Pages[0].Bands[0].Objects[0].Attrs, Attr{"Style", "Nope"})
	if Valid(dangle) {
		t.Error("dangling style accepted")
	}
}

func TestJSONRoundTrip(t *testing.T) {
	rep := loadFile(t, "box.frx")
	a, err := MarshalJSON(rep)
	if err != nil {
		t.Fatal(err)
	}
	b, err := MarshalJSON(rep)
	if err != nil {
		t.Fatal(err)
	}
	if !bytes.Equal(a, b) {
		t.Fatal("JSON output not deterministic")
	}
	back, err := UnmarshalJSON(a)
	if err != nil {
		t.Fatal(err)
	}
	if !reflect.DeepEqual(rep, back) {
		t.Fatal("JSON round-trip differs")
	}
}

func TestNestedTablePreserved(t *testing.T) {
	rep := loadFile(t, "table.frx")
	found := false
	for _, p := range rep.Pages {
		for _, b := range p.Bands {
			for _, o := range b.Objects {
				if o.Kind == "TableObject" && len(o.Children) > 0 {
					found = true
				}
			}
		}
	}
	if !found {
		t.Fatal("no nested TableObject children preserved")
	}
	if !Valid(rep) {
		t.Fatalf("table.frx invalid: %v", Validate(rep))
	}
	roundTrip(t, rep)
}

func TestWatermarkPreserved(t *testing.T) {
	rep := loadFile(t, "hello.frx")
	got := attrVal(rep.Pages[0].Attrs, "Watermark.Font")
	if got != "Arial, 60pt" {
		t.Fatalf("watermark = %q", got)
	}
	roundTrip(t, rep)
}
