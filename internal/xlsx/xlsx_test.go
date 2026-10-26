package xlsx

import (
	"archive/zip"
	"bytes"
	"encoding/xml"
	"io"
	"strings"
	"testing"
)

func openSheet(t *testing.T, data []byte, name string) string {
	t.Helper()
	zr, err := zip.NewReader(bytes.NewReader(data), int64(len(data)))
	if err != nil {
		t.Fatal(err)
	}
	names := map[string]bool{}
	for _, f := range zr.File {
		names[f.Name] = true
	}
	for _, want := range []string{"[Content_Types].xml", "_rels/.rels", "xl/workbook.xml", "xl/_rels/workbook.xml.rels", name} {
		if !names[want] {
			t.Fatalf("missing part %s", want)
		}
	}
	f, err := zr.Open(name)
	if err != nil {
		t.Fatal(err)
	}
	defer f.Close()
	body, err := io.ReadAll(f)
	if err != nil {
		t.Fatal(err)
	}
	// Must be well-formed XML.
	dec := xml.NewDecoder(bytes.NewReader(body))
	for {
		if _, err := dec.Token(); err != nil {
			if err == io.EOF {
				break
			}
			t.Fatal(err)
		}
	}
	return string(body)
}

func TestWorkbookRoundTrip(t *testing.T) {
	w := New()
	s := w.AddSheet("Customers & Co")
	s.Set(0, 0, "Name <x>")
	s.SetNum(0, 1, 6)
	s.Set(1, 0, "ANATR")
	s.Set(2, 5, "far")
	data, err := w.Bytes()
	if err != nil {
		t.Fatal(err)
	}
	body := openSheet(t, data, "xl/worksheets/sheet1.xml")
	for _, want := range []string{"Name &lt;x&gt;", "<v>6</v>", "ANATR", "F3"} {
		if !strings.Contains(body, want) {
			t.Errorf("sheet missing %q:\n%s", want, body)
		}
	}
	wb := openSheet(t, data, "xl/workbook.xml")
	if !strings.Contains(wb, "Customers &amp; Co") {
		t.Errorf("sheet name not escaped:\n%s", wb)
	}
}

func TestEmptyWorkbook(t *testing.T) {
	data, err := New().Bytes()
	if err != nil {
		t.Fatal(err)
	}
	openSheet(t, data, "xl/worksheets/sheet1.xml")
}
