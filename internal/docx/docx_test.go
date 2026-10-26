package docx

import (
	"archive/zip"
	"bytes"
	"encoding/xml"
	"io"
	"strings"
	"testing"
)

func openDoc(t *testing.T, data []byte) string {
	t.Helper()
	zr, err := zip.NewReader(bytes.NewReader(data), int64(len(data)))
	if err != nil {
		t.Fatal(err)
	}
	names := map[string]bool{}
	for _, f := range zr.File {
		names[f.Name] = true
	}
	for _, want := range []string{"[Content_Types].xml", "_rels/.rels", "word/document.xml", "word/_rels/document.xml.rels"} {
		if !names[want] {
			t.Fatalf("missing part %s", want)
		}
	}
	f, err := zr.Open("word/document.xml")
	if err != nil {
		t.Fatal(err)
	}
	defer f.Close()
	body, err := io.ReadAll(f)
	if err != nil {
		t.Fatal(err)
	}
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

func TestDocumentRoundTrip(t *testing.T) {
	d := New()
	d.AddPara(Paragraph{Text: "Hello <world> & สวัสดี", Bold: true, Size: 28, Align: "center"})
	d.AddTable(Table{Header: true, Rows: [][]string{{"ID", "Name"}, {"ALFKI", "Alfreds"}, {"X"}}})
	data, err := d.Bytes()
	if err != nil {
		t.Fatal(err)
	}
	body := openDoc(t, data)
	for _, want := range []string{
		"Hello &lt;world&gt; &amp; สวัสดี",
		"<w:b/>",
		"<w:sz",
		`w:val="center"`,
		"Alfreds",
	} {
		if !strings.Contains(body, want) {
			t.Errorf("document missing %q", want)
		}
	}
}
