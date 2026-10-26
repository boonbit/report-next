// Package xlsx writes minimal .xlsx workbooks using only the standard
// library (archive/zip + encoding/xml). Strings are stored inline
// (t="inlineStr") so no shared-strings table is needed. Numbers are stored
// as numeric cells when they parse as float.
package xlsx

import (
	"archive/zip"
	"bytes"
	"encoding/xml"
	"fmt"
	"strconv"
)

// Cell is one cell value. Num marks a preformatted numeric literal.
type Cell struct {
	Value string
	Num   bool
}

// Sheet is one worksheet: rows of cells, ragged rows allowed.
type Sheet struct {
	Name string
	rows [][]Cell
}

// Workbook is an ordered set of worksheets.
type Workbook struct {
	sheets []*Sheet
}

// New returns an empty workbook.
func New() *Workbook { return &Workbook{} }

// AddSheet appends a worksheet and returns it.
func (w *Workbook) AddSheet(name string) *Sheet {
	s := &Sheet{Name: name}
	w.sheets = append(w.sheets, s)
	return s
}

// Set stores a string cell (0-based row/col), growing the sheet.
func (s *Sheet) Set(r, c int, v string) {
	s.set(r, c, Cell{Value: v})
}

// SetNum stores a numeric cell.
func (s *Sheet) SetNum(r, c int, f float64) {
	s.set(r, c, Cell{Value: strconv.FormatFloat(f, 'f', -1, 64), Num: true})
}

func (s *Sheet) set(r, c int, cell Cell) {
	for len(s.rows) <= r {
		s.rows = append(s.rows, nil)
	}
	for len(s.rows[r]) <= c {
		s.rows[r] = append(s.rows[r], Cell{})
	}
	s.rows[r][c] = cell
}

// colName converts a 0-based column to A1 letters.
func colName(c int) string {
	name := ""
	for c >= 0 {
		name = string(rune('A'+c%26)) + name
		c = c/26 - 1
	}
	return name
}

func writePart(w *zip.Writer, name, body string) error {
	f, err := w.Create(name)
	if err != nil {
		return err
	}
	_, err = f.Write([]byte(body))
	return err
}

func sheetXML(s *Sheet) (string, error) {
	var buf bytes.Buffer
	buf.WriteString(`<?xml version="1.0" encoding="UTF-8" standalone="yes"?>` + "\n")
	enc := xml.NewEncoder(&buf)
	start := func(name string, attrs ...xml.Attr) {
		_ = enc.EncodeToken(xml.StartElement{Name: xml.Name{Local: name}, Attr: attrs})
	}
	attr := func(k, v string) xml.Attr { return xml.Attr{Name: xml.Name{Local: k}, Value: v} }
	start("worksheet", xml.Attr{Name: xml.Name{Local: "xmlns"}, Value: "http://schemas.openxmlformats.org/spreadsheetml/2006/main"})
	start("sheetData")
	for r, row := range s.rows {
		start("row", attr("r", strconv.Itoa(r+1)))
		for c, cell := range row {
			ref := colName(c) + strconv.Itoa(r+1)
			if cell.Num {
				start("c", attr("r", ref))
				start("v")
				_ = enc.EncodeToken(xml.CharData(cell.Value))
				_ = enc.EncodeToken(xml.EndElement{Name: xml.Name{Local: "v"}})
				_ = enc.EncodeToken(xml.EndElement{Name: xml.Name{Local: "c"}})
				continue
			}
			start("c", attr("r", ref), attr("t", "inlineStr"))
			start("is")
			start("t")
			_ = enc.EncodeToken(xml.CharData(cell.Value))
			_ = enc.EncodeToken(xml.EndElement{Name: xml.Name{Local: "t"}})
			_ = enc.EncodeToken(xml.EndElement{Name: xml.Name{Local: "is"}})
			_ = enc.EncodeToken(xml.EndElement{Name: xml.Name{Local: "c"}})
		}
		_ = enc.EncodeToken(xml.EndElement{Name: xml.Name{Local: "row"}})
	}
	_ = enc.EncodeToken(xml.EndElement{Name: xml.Name{Local: "sheetData"}})
	_ = enc.EncodeToken(xml.EndElement{Name: xml.Name{Local: "worksheet"}})
	if err := enc.Flush(); err != nil {
		return "", err
	}
	return buf.String(), nil
}

// Bytes serializes the workbook to .xlsx bytes.
func (w *Workbook) Bytes() ([]byte, error) {
	sheets := w.sheets
	if len(sheets) == 0 {
		sheets = []*Sheet{{Name: "Sheet1"}}
	}
	var buf bytes.Buffer
	zw := zip.NewWriter(&buf)
	types := `<?xml version="1.0" encoding="UTF-8" standalone="yes"?>` + "\n" +
		`<Types xmlns="http://schemas.openxmlformats.org/package/2006/content-types">` +
		`<Default Extension="rels" ContentType="application/vnd.openxmlformats-package.relationships+xml"/>` +
		`<Default Extension="xml" ContentType="application/xml"/>` +
		`<Override PartName="/xl/workbook.xml" ContentType="application/vnd.openxmlformats-officedocument.spreadsheetml.sheet.main+xml"/>`
	for i := range sheets {
		types += fmt.Sprintf(`<Override PartName="/xl/worksheets/sheet%d.xml" ContentType="application/vnd.openxmlformats-officedocument.spreadsheetml.worksheet+xml"/>`, i+1)
	}
	types += `</Types>`
	if err := writePart(zw, "[Content_Types].xml", types); err != nil {
		return nil, err
	}
	if err := writePart(zw, "_rels/.rels", `<?xml version="1.0" encoding="UTF-8" standalone="yes"?>`+"\n"+
		`<Relationships xmlns="http://schemas.openxmlformats.org/package/2006/relationships">`+
		`<Relationship Id="rId1" Type="http://schemas.openxmlformats.org/officeDocument/2006/relationships/officeDocument" Target="xl/workbook.xml"/>`+
		`</Relationships>`); err != nil {
		return nil, err
	}
	wb := `<?xml version="1.0" encoding="UTF-8" standalone="yes"?>` + "\n" +
		`<workbook xmlns="http://schemas.openxmlformats.org/spreadsheetml/2006/main" xmlns:r="http://schemas.openxmlformats.org/officeDocument/2006/relationships"><sheets>`
	for i, s := range sheets {
		wb += fmt.Sprintf(`<sheet name="%s" sheetId="%d" r:id="rId%d"/>`, xmlEscape(s.Name), i+1, i+1)
	}
	wb += `</sheets></workbook>`
	if err := writePart(zw, "xl/workbook.xml", wb); err != nil {
		return nil, err
	}
	rels := `<?xml version="1.0" encoding="UTF-8" standalone="yes"?>` + "\n" +
		`<Relationships xmlns="http://schemas.openxmlformats.org/package/2006/relationships">`
	for i := range sheets {
		rels += fmt.Sprintf(`<Relationship Id="rId%d" Type="http://schemas.openxmlformats.org/officeDocument/2006/relationships/worksheet" Target="worksheets/sheet%d.xml"/>`, i+1, i+1)
	}
	rels += `</Relationships>`
	if err := writePart(zw, "xl/_rels/workbook.xml.rels", rels); err != nil {
		return nil, err
	}
	for i, s := range sheets {
		body, err := sheetXML(s)
		if err != nil {
			return nil, err
		}
		if err := writePart(zw, fmt.Sprintf("xl/worksheets/sheet%d.xml", i+1), body); err != nil {
			return nil, err
		}
	}
	if err := zw.Close(); err != nil {
		return nil, err
	}
	return buf.Bytes(), nil
}

func xmlEscape(s string) string {
	var buf bytes.Buffer
	_ = xml.EscapeText(&buf, []byte(s))
	return buf.String()
}
