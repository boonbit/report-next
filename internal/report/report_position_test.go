package report

import (
	"archive/zip"
	"bytes"
	"encoding/xml"
	"io"
	"strings"
	"testing"
	"unicode"
)

// positionTwoColReport builds a report with a wide title, two side-by-side
// texts and a horizontal rule: all geometry attributes present so the
// positioned exporters lay it out by Left/Top.
func positionTwoColReport() Report {
	textObj := func(name, text, left, top, width, height string) Object {
		return Object{Kind: "TextObject", Name: name, Attrs: []Attr{
			{Name: "Text", Value: text},
			{Name: "Left", Value: left},
			{Name: "Top", Value: top},
			{Name: "Width", Value: width},
			{Name: "Height", Value: height},
		}}
	}
	return Report{Version: IRVersion, Pages: []Page{{Name: "Page1", Bands: []Band{
		{Kind: "ReportTitleBand", Name: "Title1", Objects: []Object{
			textObj("Title", "Two Column Demo", "0", "0", "200", "20"),
		}},
		{Kind: "PageHeaderBand", Name: "Header1", Objects: []Object{
			textObj("Left", "LeftCell", "0", "0", "60", "20"),
			textObj("Right", "RightCell", "128", "0", "60", "20"),
			{Kind: "LineObject", Name: "Rule1", Attrs: []Attr{
				{Name: "Left", Value: "0"},
				{Name: "Top", Value: "20"},
				{Name: "Width", Value: "192"},
				{Name: "Height", Value: "1"},
			}},
		}},
	}}}}
}

// posSheet is the minimal read-back view of a produced worksheet.
type posSheet struct {
	Rows   []posRow   `xml:"sheetData>row"`
	Merges []posMerge `xml:"mergeCells>mergeCell"`
}

type posRow struct {
	Cells []posCell `xml:"c"`
}

type posCell struct {
	Ref string `xml:"r,attr"`
	V   string `xml:"v"`
	IS  string `xml:"is>t"`
}

type posMerge struct {
	Ref string `xml:"ref,attr"`
}

func (c posCell) text() string {
	if c.IS != "" {
		return c.IS
	}
	return c.V
}

// posCellCol splits an A1 ref into its column letters and row digits.
func posCellCol(ref string) string {
	i := 0
	for i < len(ref) && unicode.IsLetter(rune(ref[i])) {
		i++
	}
	return ref[:i]
}

func readPosSheet(t *testing.T, data []byte) (posSheet, string) {
	t.Helper()
	zr, err := zip.NewReader(bytes.NewReader(data), int64(len(data)))
	if err != nil {
		t.Fatal(err)
	}
	var raw string
	for _, f := range zr.File {
		if f.Name != "xl/worksheets/sheet1.xml" {
			continue
		}
		rc, err := f.Open()
		if err != nil {
			t.Fatal(err)
		}
		body, err := io.ReadAll(rc)
		rc.Close()
		if err != nil {
			t.Fatal(err)
		}
		raw = string(body)
	}
	if raw == "" {
		t.Fatal("sheet1.xml missing from produced xlsx")
	}
	var sh posSheet
	if err := xml.Unmarshal([]byte(raw), &sh); err != nil {
		t.Fatal(err)
	}
	return sh, raw
}

// TestPositionXlsxTwoColumn round-trips a two-column report through the
// positioned xlsx export: both texts must survive, the right text must sit
// in a later column on the same row, and a wide object must merge cells.
func TestPositionXlsxTwoColumn(t *testing.T) {
	out, err := xlsxExportReport(positionTwoColReport(), nil)
	if err != nil {
		t.Fatal(err)
	}
	sh, raw := readPosSheet(t, out)
	refs := map[string]string{}
	for _, r := range sh.Rows {
		for _, c := range r.Cells {
			refs[c.text()] = c.Ref
		}
	}
	for _, want := range []string{"Two Column Demo", "LeftCell", "RightCell"} {
		if _, ok := refs[want]; !ok {
			t.Fatalf("positioned xlsx lost text %q:\n%s", want, raw)
		}
	}
	left, right := refs["LeftCell"], refs["RightCell"]
	if left == "" || right == "" {
		t.Fatalf("missing cell refs: left=%q right=%q", left, right)
	}
	if posCellCol(left) >= posCellCol(right) {
		t.Fatalf("right cell not right of left cell: %q vs %q", left, right)
	}
	if strings.TrimLeft(left, "ABCDEFGHIJKLMNOPQRSTUVWXYZ") != strings.TrimLeft(right, "ABCDEFGHIJKLMNOPQRSTUVWXYZ") {
		t.Fatalf("two-column texts not on one row: %q vs %q", left, right)
	}
	spanning := false
	for _, m := range sh.Merges {
		parts := strings.SplitN(m.Ref, ":", 2)
		if len(parts) != 2 {
			continue
		}
		if posCellCol(parts[0]) != posCellCol(parts[1]) {
			spanning = true
		}
	}
	if !spanning {
		t.Fatalf("no multi-column merge in positioned xlsx:\n%s", raw)
	}
	if !strings.Contains(raw, "\u2500") {
		t.Fatalf("horizontal rule left no drawing in positioned xlsx:\n%s", raw)
	}
}

// TestPositionDocxTwoColumn round-trips a two-column report through the
// positioned docx export: both texts must survive in Left order inside one
// table row, and the rule must draw with text primitives.
func TestPositionDocxTwoColumn(t *testing.T) {
	out, err := docxExportReport(positionTwoColReport())
	if err != nil {
		t.Fatal(err)
	}
	body := unzipDoc(t, out)
	for _, want := range []string{"Two Column Demo", "LeftCell", "RightCell"} {
		if !strings.Contains(body, want) {
			t.Fatalf("positioned docx lost text %q:\n%s", want, body)
		}
	}
	li, ri := strings.Index(body, "LeftCell"), strings.Index(body, "RightCell")
	if li > ri {
		t.Fatalf("right cell precedes left cell in positioned docx:\n%s", body)
	}
	if strings.Contains(body[li:ri], "<w:tr>") {
		t.Fatalf("two-column texts split across table rows:\n%s", body)
	}
	if !strings.Contains(body, "<w:tbl>") {
		t.Fatalf("side-by-side texts produced no docx table:\n%s", body)
	}
	if !strings.Contains(body, "\u2500") {
		t.Fatalf("horizontal rule left no drawing in positioned docx:\n%s", body)
	}
}
