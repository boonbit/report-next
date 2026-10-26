package report

import (
	"sort"
	"strconv"
	"strings"

	"report-next/internal/docx"
)

// docxExportReport converts a report to .docx bytes. Reports without
// geometry attributes keep the legacy layout (TextObjects become paragraphs
// in band order, TableObjects become tables). Reports with Left/Top/Width
// attributes use the positioned grid: single texts stay styled paragraphs in
// Top order, side-by-side texts share one table row in Left order, wide
// objects span table cells only where the minimal docx writer allows, and
// LineObject/ShapeObject render as basic text-rule and bordered-box
// drawings. Unlike the base-14 PDF path, UTF-8 scripts such as Thai pass
// through. No image embedding.
func docxExportReport(rep Report) ([]byte, error) {
	if !exportHasLayout(rep) {
		return docxExportFlat(rep)
	}
	d := docx.New()
	for _, p := range rep.Pages {
		for _, b := range p.Bands {
			docxExportBandPositioned(d, b)
		}
	}
	return d.Bytes()
}

// docxExportFlat is the legacy docx export: TextObjects become paragraphs
// (bold/size/alignment preserved), TableObjects become tables.
func docxExportFlat(rep Report) ([]byte, error) {
	d := docx.New()
	for _, p := range rep.Pages {
		for _, b := range p.Bands {
			for _, o := range b.Objects {
				switch o.Kind {
				case "TextObject":
					if para, ok := docxParaFor(o); ok {
						d.AddPara(para)
					}
				case "TableObject":
					if tab, ok := docxTableFor(o); ok {
						d.AddTable(tab)
					}
				}
			}
		}
	}
	return d.Bytes()
}

// docxExportBandPositioned lays one band out by object geometry: texts group
// into Top rows, single texts keep styled paragraphs, side-by-side texts
// share one table row in Left order, and rules/boxes draw with the text and
// bordered-table primitives the minimal docx writer allows.
func docxExportBandPositioned(d *docx.Document, b Band) {
	type textCell struct {
		left float64
		para docx.Paragraph
	}
	byRow := map[int][]textCell{}
	var rows []int
	for _, o := range b.Objects {
		switch o.Kind {
		case "TextObject":
			para, ok := docxParaFor(o)
			if !ok {
				continue
			}
			r := exportGridRowOff(objFloatAttr(o.Attrs, "Top"))
			if _, found := byRow[r]; !found {
				rows = append(rows, r)
			}
			byRow[r] = append(byRow[r], textCell{left: objFloatAttr(o.Attrs, "Left"), para: para})
		case "TableObject":
			if tab, ok := docxTableFor(o); ok {
				d.AddTable(tab)
			}
		case "LineObject":
			if para, ok := docxRuleFor(o); ok {
				d.AddPara(para)
			}
		case "ShapeObject":
			switch attrVal(o.Attrs, "Shape") {
			case "", "Rectangle", "RoundRectangle":
				// The writer always draws full table borders, so a
				// single empty cell reads as a basic rectangle outline.
				d.AddTable(docx.Table{Rows: [][]string{{""}}})
			}
		}
	}
	sort.Ints(rows)
	for _, r := range rows {
		cells := byRow[r]
		if len(cells) == 1 {
			d.AddPara(cells[0].para)
			continue
		}
		sort.SliceStable(cells, func(i, j int) bool { return cells[i].left < cells[j].left })
		row := make([]string, len(cells))
		for i, c := range cells {
			row[i] = c.para.Text
		}
		d.AddTable(docx.Table{Rows: [][]string{row}})
	}
}

// docxParaFor converts a TextObject to a styled paragraph, reporting false
// for empty texts. Bold/size/alignment mirror the legacy export.
func docxParaFor(o Object) (docx.Paragraph, bool) {
	text := attrVal(o.Attrs, "Text")
	if text == "" {
		return docx.Paragraph{}, false
	}
	para := docx.Paragraph{Text: text}
	for _, part := range strings.Split(attrVal(o.Attrs, "Font"), ",") {
		part = strings.TrimSpace(part)
		if strings.HasSuffix(part, "pt") {
			if f, err := strconv.ParseFloat(strings.TrimSuffix(part, "pt"), 64); err == nil {
				para.Size = f * 2
			}
		}
		if strings.Contains(part, "Bold") {
			para.Bold = true
		}
	}
	switch attrVal(o.Attrs, "HorzAlign") {
	case "Center":
		para.Align = "center"
	case "Right":
		para.Align = "right"
	case "Justify":
		para.Align = "justify"
	}
	return para, true
}

// docxRuleFor draws a LineObject as text rules: a horizontal run of box
// rules, or one vertical bar paragraph per grid row. It reports false for
// zero-size objects.
func docxRuleFor(o Object) (docx.Paragraph, bool) {
	_, _, w, h := objRectOf(o.Attrs)
	if h > w {
		n := exportGridSpan(h, exportRowUnit)
		if n < 1 {
			return docx.Paragraph{}, false
		}
		var sb strings.Builder
		for i := 0; i < n; i++ {
			sb.WriteString("\u2502\n")
		}
		return docx.Paragraph{Text: strings.TrimSuffix(sb.String(), "\n")}, true
	}
	n := int(w / 8)
	if n < 4 {
		n = 4
	}
	if n > 80 {
		n = 80
	}
	if w <= 0 {
		return docx.Paragraph{}, false
	}
	return docx.Paragraph{Text: strings.Repeat("\u2500", n)}, true
}

// docxTableFor converts a TableObject to a docx table: one grid row per
// table row, cell Text plus nested TextObjects concatenated.
func docxTableFor(o Object) (docx.Table, bool) {
	tab, err := tabParseTable(o)
	if err != nil {
		return docx.Table{}, false
	}
	var rows [][]string
	for _, r := range tab.rows {
		var row []string
		for _, c := range r.cells {
			cell := attrVal(c.attrs, "Text")
			for _, no := range c.objects {
				if no.Kind == "TextObject" {
					if t := attrVal(no.Attrs, "Text"); t != "" {
						cell += t
					}
				}
			}
			row = append(row, cell)
		}
		rows = append(rows, row)
	}
	return docx.Table{Rows: rows}, true
}
