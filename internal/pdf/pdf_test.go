package pdf

import (
	"bytes"
	"fmt"
	"strings"
	"testing"
)

func buildSample(t *testing.T) []byte {
	t.Helper()
	d := New()
	d.AddPage(595.28, 841.89)
	if err := d.Text(0, 72, 750, 12, Helvetica, "Hello (world) \\ test"); err != nil {
		t.Fatal(err)
	}
	if err := d.FillRect(0, 72, 700, 100, 20, 0.9); err != nil {
		t.Fatal(err)
	}
	return d.Bytes()
}

func TestHeaderAndEOF(t *testing.T) {
	b := buildSample(t)
	if !bytes.HasPrefix(b, []byte("%PDF-1.4\n")) {
		t.Fatal("missing header")
	}
	if !bytes.HasSuffix(bytes.TrimRight(b, "\n"), []byte("%%EOF")) {
		t.Fatal("missing EOF")
	}
}

func TestEscaping(t *testing.T) {
	b := buildSample(t)
	if !bytes.Contains(b, []byte(`(Hello \(world\) \\ test)`)) {
		t.Fatal("parentheses not escaped")
	}
}

func TestXrefOffsets(t *testing.T) {
	b := buildSample(t)
	s := string(b)
	xrefAt := strings.LastIndex(s, "startxref")
	var xref int
	if _, err := fmt.Sscanf(s[xrefAt:], "startxref\n%d", &xref); err != nil {
		t.Fatal(err)
	}
	if !strings.HasPrefix(s[xref:], "xref\n") {
		t.Fatalf("xref offset %d does not point at xref table", xref)
	}
	lines := strings.Split(s[xref:], "\n")
	var total int
	if _, err := fmt.Sscanf(lines[1], "0 %d", &total); err != nil || total < 2 {
		t.Fatalf("unexpected xref head: %q", lines[1])
	}
	// Every "n" entry must point at "<num> 0 obj".
	n := 0
	for _, ln := range lines[2:] {
		var off int
		var flag string
		if _, err := fmt.Sscanf(ln, "%d 00000 %1s", &off, &flag); err != nil || flag != "n" {
			continue
		}
		n++
		if !strings.HasPrefix(s[off:], fmt.Sprintf("%d 0 obj", n)) {
			t.Fatalf("entry %d offset %d does not point at object", n, off)
		}
	}
	if n != total-1 {
		t.Fatalf("checked %d entries, want %d", n, total-1)
	}
}

func TestPageCount(t *testing.T) {
	d := New()
	d.AddPage(100, 100)
	d.AddPage(100, 100)
	if d.Pages() != 2 {
		t.Fatalf("pages = %d", d.Pages())
	}
	if !bytes.Contains(d.Bytes(), []byte("/Count 2")) {
		t.Fatal("missing /Count 2")
	}
}

func TestNonASCIIRejected(t *testing.T) {
	d := New()
	d.AddPage(100, 100)
	if err := d.Text(0, 0, 0, 12, Helvetica, "สวัสดี"); err == nil {
		t.Fatal("thai text accepted")
	}
	if err := d.Text(9, 0, 0, 12, Helvetica, "x"); err == nil {
		t.Fatal("bad page accepted")
	}
}
