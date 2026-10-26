// Package docx writes minimal .docx documents using only the standard
// library (archive/zip + encoding/xml). Text runs are stored directly in
// paragraph XML; complex scripts are passed through as UTF-8 (Word handles
// shaping), unlike the base-14 PDF path.
package docx

import (
	"archive/zip"
	"bytes"
	"encoding/xml"
	"fmt"
	"strings"
)

// Paragraph is one block of text with an optional style and alignment.
type Paragraph struct {
	Text  string
	Bold  bool
	Size  float64 // half-points; 0 means default
	Align string  // left, center, right, justify
}

// Table is a simple grid: first row is the header when Header is true.
type Table struct {
	Header bool
	Rows   [][]string
}

// Document is an ordered list of blocks.
type Document struct {
	Paras  []Paragraph
	Tables []Table // rendered after paragraphs, in order
}

// New returns an empty document.
func New() *Document { return &Document{} }

// AddPara appends a paragraph.
func (d *Document) AddPara(p Paragraph) { d.Paras = append(d.Paras, p) }

// AddTable appends a table.
func (d *Document) AddTable(t Table) { d.Tables = append(d.Tables, t) }

func writePart(w *zip.Writer, name, body string) error {
	f, err := w.Create(name)
	if err != nil {
		return err
	}
	_, err = f.Write([]byte(body))
	return err
}

func esc(s string) string {
	var buf bytes.Buffer
	_ = xml.EscapeText(&buf, []byte(s))
	return buf.String()
}

func paraXML(p Paragraph) string {
	var sb strings.Builder
	sb.WriteString("<w:p>")
	if p.Align != "" || p.Bold || p.Size > 0 {
		sb.WriteString("<w:pPr>")
		if p.Align != "" {
			sb.WriteString(`<w:jc w:val="` + p.Align + `"/>`)
		}
		sb.WriteString("</w:pPr>")
	}
	sb.WriteString("<w:r>")
	if p.Bold || p.Size > 0 {
		sb.WriteString("<w:rPr>")
		if p.Bold {
			sb.WriteString("<w:b/>")
		}
		if p.Size > 0 {
			sb.WriteString(fmt.Sprintf(`<w:sz w:val="%.0f"/>`, p.Size))
		}
		sb.WriteString("</w:rPr>")
	}
	sb.WriteString("<w:t xml:space=\"preserve\">" + esc(p.Text) + "</w:t>")
	sb.WriteString("</w:r></w:p>")
	return sb.String()
}

func tableXML(t Table) string {
	var sb strings.Builder
	sb.WriteString(`<w:tbl><w:tblPr><w:tblW w:w="0" w:type="auto"/>` +
		`<w:tblBorders><w:top w:val="single" w:sz="4"/><w:left w:val="single" w:sz="4"/>` +
		`<w:bottom w:val="single" w:sz="4"/><w:right w:val="single" w:sz="4"/>` +
		`<w:insideH w:val="single" w:sz="4"/><w:insideV w:val="single" w:sz="4"/>` +
		`</w:tblBorders></w:tblPr><w:tblGrid>`)
	width := 0
	for _, r := range t.Rows {
		if len(r) > width {
			width = len(r)
		}
	}
	for i := 0; i < width; i++ {
		sb.WriteString(`<w:gridCol w:w="2880"/>`)
	}
	sb.WriteString("</w:tblGrid>")
	for ri, r := range t.Rows {
		sb.WriteString("<w:tr>")
		for c := 0; c < width; c++ {
			cell := ""
			if c < len(r) {
				cell = r[c]
			}
			sb.WriteString("<w:tc><w:p><w:r>")
			if t.Header && ri == 0 {
				sb.WriteString("<w:rPr><w:b/></w:rPr>")
			}
			sb.WriteString(`<w:t xml:space="preserve">` + esc(cell) + "</w:t>")
			sb.WriteString("</w:r></w:p></w:tc>")
		}
		sb.WriteString("</w:tr>")
	}
	sb.WriteString("</w:tbl>")
	return sb.String()
}

// Bytes serializes the document to .docx bytes.
func (d *Document) Bytes() ([]byte, error) {
	var buf bytes.Buffer
	zw := zip.NewWriter(&buf)
	parts := map[string]string{
		"[Content_Types].xml": `<?xml version="1.0" encoding="UTF-8" standalone="yes"?>` + "\n" +
			`<Types xmlns="http://schemas.openxmlformats.org/package/2006/content-types">` +
			`<Default Extension="rels" ContentType="application/vnd.openxmlformats-package.relationships+xml"/>` +
			`<Default Extension="xml" ContentType="application/xml"/>` +
			`<Override PartName="/word/document.xml" ContentType="application/vnd.openxmlformats-officedocument.wordprocessingml.document.main+xml"/>` +
			`</Types>`,
		"_rels/.rels": `<?xml version="1.0" encoding="UTF-8" standalone="yes"?>` + "\n" +
			`<Relationships xmlns="http://schemas.openxmlformats.org/package/2006/relationships">` +
			`<Relationship Id="rId1" Type="http://schemas.openxmlformats.org/officeDocument/2006/relationships/officeDocument" Target="word/document.xml"/>` +
			`</Relationships>`,
		"word/_rels/document.xml.rels": `<?xml version="1.0" encoding="UTF-8" standalone="yes"?>` + "\n" +
			`<Relationships xmlns="http://schemas.openxmlformats.org/package/2006/relationships"></Relationships>`,
	}
	var body strings.Builder
	body.WriteString(`<?xml version="1.0" encoding="UTF-8" standalone="yes"?>` + "\n" +
		`<w:document xmlns:w="http://schemas.openxmlformats.org/wordprocessingml/2006/main"><w:body>`)
	for _, p := range d.Paras {
		body.WriteString(paraXML(p))
	}
	for _, t := range d.Tables {
		body.WriteString(tableXML(t))
	}
	body.WriteString("</w:body></w:document>")
	parts["word/document.xml"] = body.String()
	order := []string{"[Content_Types].xml", "_rels/.rels", "word/_rels/document.xml.rels", "word/document.xml"}
	for _, name := range order {
		if err := writePart(zw, name, parts[name]); err != nil {
			return nil, err
		}
	}
	if err := zw.Close(); err != nil {
		return nil, err
	}
	return buf.Bytes(), nil
}
