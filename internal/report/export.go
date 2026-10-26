package report

import (
	"bytes"
	"encoding/csv"
	"sort"
	"strconv"
)

// reportExportHeader is the single column of a report-level export: one
// row per prepared text value, in print order.
const reportExportHeader = "Text"

// exportCSVReport renders the prepared texts of rep as CSV (header +
// records, LF line endings) by reusing srvExportCSV over the same prepared
// rows the xlsx report export uses. dict supplies data binding like the
// /api/render path; both inputs are never mutated. An empty report yields
// the header alone.
func exportCSVReport(rep Report, dict *DataDictionary) string {
	rows := reportExportRows(rep, dict)
	if len(rows) == 0 {
		return reportExportHeader + "\n"
	}
	return srvExportCSV(nil, "", rows)
}

// reportExportRows flattens a report to one text value per row in print
// order: titles, headers, engine-expanded flow instances (one per bound
// data row, [references] evaluated like the render path), then footers.
// TableObject cells contribute their own rows. Neither rep nor dict is
// mutated: band object slices are copied before matrix binding, and band
// evaluation already copies attribute slices.
func reportExportRows(rep Report, dict *DataDictionary) []DataRow {
	params := engParams(dict)
	var rows []DataRow
	emit := func(band Band, fields map[string]string) {
		band = engEvalBand(band, fields, params)
		for _, o := range band.Objects {
			switch o.Kind {
			case "TextObject":
				if t := attrVal(o.Attrs, "Text"); t != "" {
					rows = append(rows, DataRow{reportExportHeader: t})
				}
			case "TableObject":
				rows = append(rows, reportExportTableCells(o, fields, params)...)
			}
		}
	}
	for _, p := range rep.Pages {
		var titles, headers, footers []Band
		var flow []engBand
		for _, b := range p.Bands {
			bc := b
			bc.Objects = append([]Object(nil), b.Objects...)
			bc = engBindMatrixBand(bc, dict)
			switch bc.Kind {
			case "ReportTitleBand":
				titles = append(titles, bc)
			case "PageHeaderBand", "ColumnHeaderBand":
				headers = append(headers, bc)
			case "PageFooterBand", "ColumnFooterBand":
				footers = append(footers, bc)
			default:
				flow = append(flow, engDecompose(bc, dict))
			}
		}
		for _, b := range titles {
			emit(b, map[string]string{})
		}
		for _, b := range headers {
			emit(b, map[string]string{})
		}
		var body []engInst
		engExpand(flow, dict, params, &body)
		for _, in := range body {
			emit(in.band, in.fields)
		}
		for _, b := range footers {
			emit(b, engScopeTotals(dict, engEvaluatorRows(b.Name, flow, dict)))
		}
	}
	return rows
}

// reportExportTableCells flattens one TableObject to one text row per
// non-empty cell (cell Text plus nested TextObjects), with [references]
// evaluated against the band instance fields.
func reportExportTableCells(o Object, fields, params map[string]string) []DataRow {
	tab, err := tabParseTable(o)
	if err != nil {
		return nil
	}
	var rows []DataRow
	for _, r := range tab.rows {
		for _, c := range r.cells {
			if t := engEvalText(attrVal(c.attrs, "Text"), fields, params); t != "" {
				rows = append(rows, DataRow{reportExportHeader: t})
			}
			for _, no := range c.objects {
				if no.Kind == "TextObject" {
					if t := engEvalText(attrVal(no.Attrs, "Text"), fields, params); t != "" {
						rows = append(rows, DataRow{reportExportHeader: t})
					}
				}
			}
		}
	}
	return rows
}

// Positioned report export grid (R6A): TextObjects are laid out by their
// FastReport Left/Top geometry instead of flattened to one text column.
// One grid column spans exportColUnit px (~64) and one grid row spans
// exportRowUnit px (~20); wide/tall objects span multiple cells and the
// exporters merge those cells. Reports whose objects carry no geometry
// attributes keep the legacy flat behavior via exportHasLayout.
const (
	exportColUnit = 64.0
	exportRowUnit = 20.0
)

// exportItem is one positioned text cell in a report-level export grid.
// Row/Col are 0-based; Row counts content rows (exporters add their own
// header offset). ColSpan/RowSpan are at least 1.
type exportItem struct {
	Text    string
	Num     bool
	FNum    float64
	Row     int
	Col     int
	ColSpan int
	RowSpan int
}

// exportRule is one LineObject reduced to a grid run: Horizontal rules fill
// ColSpan cells in one row, vertical rules fill RowSpan rows in one column.
type exportRule struct {
	Row        int
	Col        int
	ColSpan    int
	RowSpan    int
	Horizontal bool
}

// exportBox is one rectangle-like ShapeObject (Rectangle, RoundRectangle or
// empty Shape) reduced to a grid region for a basic border drawing.
type exportBox struct {
	Row     int
	Col     int
	ColSpan int
	RowSpan int
}

// exportGridCol maps a Left value to a 0-based grid column.
func exportGridCol(left float64) int {
	c := int(left / exportColUnit)
	if c < 0 {
		c = 0
	}
	return c
}

// exportGridRowOff maps a Top value to a 0-based row offset inside a band.
func exportGridRowOff(top float64) int {
	r := int(top / exportRowUnit)
	if r < 0 {
		r = 0
	}
	return r
}

// exportGridSpan maps a Width/Height extent to a cell span of at least 1.
func exportGridSpan(size, unit float64) int {
	n := int((size + unit/2) / unit)
	if n < 1 {
		n = 1
	}
	return n
}

// exportColName converts a 0-based column to A1 letters.
func exportColName(c int) string {
	name := ""
	for c >= 0 {
		name = string(rune('A'+c%26)) + name
		c = c/26 - 1
	}
	return name
}

// exportMergeRef returns the A1 range of a grid item on a sheet whose
// content starts at 0-based sheet row base (exporters offset past headers).
func exportMergeRef(col, row, cs, rs, base int) string {
	return exportColName(col) + strconv.Itoa(row+base+1) + ":" +
		exportColName(col+cs-1) + strconv.Itoa(row+rs+base+1)
}

// exportHasLayout reports whether any band object carries FastReport
// geometry attributes. Reports without them keep legacy flat exports.
func exportHasLayout(rep Report) bool {
	for _, p := range rep.Pages {
		for _, b := range p.Bands {
			for _, o := range b.Objects {
				switch o.Kind {
				case "TextObject", "LineObject", "ShapeObject", "TableObject":
				default:
					continue
				}
				for _, a := range o.Attrs {
					switch a.Name {
					case "Left", "Top", "Width", "Height":
						if a.Value != "" {
							return true
						}
					}
				}
			}
		}
	}
	return false
}

// exportPlacer hands out collision-free grid cells: when the target area is
// taken it shifts down until the whole span is free, so positioned exports
// never lose text to overlaps.
type exportPlacer struct {
	taken map[[2]int]bool
}

func (p *exportPlacer) place(row, col, cs, rs int) int {
	if p.taken == nil {
		p.taken = map[[2]int]bool{}
	}
	for {
		free := true
		for r := row; r < row+rs && free; r++ {
			for c := col; c < col+cs && free; c++ {
				if p.taken[[2]int{r, c}] {
					free = false
				}
			}
		}
		if free {
			for r := row; r < row+rs; r++ {
				for c := col; c < col+cs; c++ {
					p.taken[[2]int{r, c}] = true
				}
			}
			return row
		}
		row++
	}
}

// exportLayout flattens rep like reportExportRows (same band order, same
// field evaluation, same text values) but positions every TextObject by its
// Left/Top geometry and reduces LineObject/ShapeObject to grid runs/boxes.
// Neither rep nor dict is mutated.
func exportLayout(rep Report, dict *DataDictionary) (items []exportItem, rules []exportRule, boxes []exportBox) {
	params := engParams(dict)
	pl := &exportPlacer{}
	cursor := 0
	emit := func(band Band, fields map[string]string) {
		band = engEvalBand(band, fields, params)
		base := cursor
		bottom := base
		for _, o := range band.Objects {
			switch o.Kind {
			case "TextObject":
				text := attrVal(o.Attrs, "Text")
				if text == "" {
					continue
				}
				col := exportGridCol(objFloatAttr(o.Attrs, "Left"))
				off := exportGridRowOff(objFloatAttr(o.Attrs, "Top"))
				cs := exportGridSpan(objFloatAttr(o.Attrs, "Width"), exportColUnit)
				rs := exportGridSpan(objFloatAttr(o.Attrs, "Height"), exportRowUnit)
				row := pl.place(base+off, col, cs, rs)
				it := exportItem{Text: text, Row: row, Col: col, ColSpan: cs, RowSpan: rs}
				if f, err := strconv.ParseFloat(text, 64); err == nil && text != "" {
					it.Num = true
					it.FNum = f
				}
				items = append(items, it)
				if row+rs > bottom {
					bottom = row + rs
				}
			case "TableObject":
				tab, err := tabParseTable(o)
				if err != nil {
					continue
				}
				start := exportGridCol(objFloatAttr(o.Attrs, "Left"))
				for ri, tr := range tab.rows {
					cc := start
					for _, cell := range tr.cells {
						cs := cell.colSpan
						if cs < 1 {
							cs = 1
						}
						rs := cell.rowSpan
						if rs < 1 {
							rs = 1
						}
						text := engEvalText(attrVal(cell.attrs, "Text"), fields, params)
						for _, no := range cell.objects {
							if no.Kind == "TextObject" {
								text += engEvalText(attrVal(no.Attrs, "Text"), fields, params)
							}
						}
						if text != "" {
							row := pl.place(base+ri, cc, cs, rs)
							it := exportItem{Text: text, Row: row, Col: cc, ColSpan: cs, RowSpan: rs}
							if f, err := strconv.ParseFloat(text, 64); err == nil {
								it.Num = true
								it.FNum = f
							}
							items = append(items, it)
							if row+rs > bottom {
								bottom = row + rs
							}
						}
						cc += cs
					}
				}
				if n := len(tab.rows); base+n > bottom {
					bottom = base + n
				}
			case "LineObject":
				_, _, w, h := objRectOf(o.Attrs)
				horiz := true
				if h > w {
					horiz = false
				}
				col := exportGridCol(objFloatAttr(o.Attrs, "Left"))
				off := exportGridRowOff(objFloatAttr(o.Attrs, "Top"))
				cs := exportGridSpan(w, exportColUnit)
				rs := exportGridSpan(h, exportRowUnit)
				if horiz {
					rs = 1
				} else {
					cs = 1
				}
				row := pl.place(base+off, col, cs, rs)
				rules = append(rules, exportRule{Row: row, Col: col, ColSpan: cs, RowSpan: rs, Horizontal: horiz})
				if row+rs > bottom {
					bottom = row + rs
				}
			case "ShapeObject":
				switch attrVal(o.Attrs, "Shape") {
				case "", "Rectangle", "RoundRectangle":
				default:
					continue
				}
				_, _, w, h := objRectOf(o.Attrs)
				col := exportGridCol(objFloatAttr(o.Attrs, "Left"))
				off := exportGridRowOff(objFloatAttr(o.Attrs, "Top"))
				cs := exportGridSpan(w, exportColUnit)
				rs := exportGridSpan(h, exportRowUnit)
				row := pl.place(base+off, col, cs, rs)
				boxes = append(boxes, exportBox{Row: row, Col: col, ColSpan: cs, RowSpan: rs})
				if row+rs > bottom {
					bottom = row + rs
				}
			}
		}
		if n := int((engHeight(band) + exportRowUnit/2) / exportRowUnit); base+n > bottom {
			bottom = base + n
		}
		cursor = bottom
	}
	for _, p := range rep.Pages {
		var titles, headers, footers []Band
		var flow []engBand
		for _, b := range p.Bands {
			bc := b
			bc.Objects = append([]Object(nil), b.Objects...)
			bc = engBindMatrixBand(bc, dict)
			switch bc.Kind {
			case "ReportTitleBand":
				titles = append(titles, bc)
			case "PageHeaderBand", "ColumnHeaderBand":
				headers = append(headers, bc)
			case "PageFooterBand", "ColumnFooterBand":
				footers = append(footers, bc)
			default:
				flow = append(flow, engDecompose(bc, dict))
			}
		}
		for _, b := range titles {
			emit(b, map[string]string{})
		}
		for _, b := range headers {
			emit(b, map[string]string{})
		}
		var body []engInst
		engExpand(flow, dict, params, &body)
		for _, in := range body {
			emit(in.band, in.fields)
		}
		for _, b := range footers {
			emit(b, engScopeTotals(dict, engEvaluatorRows(b.Name, flow, dict)))
		}
	}
	return items, rules, boxes
}

// TRACK srv (P7-export): CSV export of one data source.
//
// srvExportCSV renders rows of the named source as CSV text (header +
// records, LF line endings). Columns come from dict's declared columns for
// source when present; otherwise from the sorted union of row keys. When
// rows is nil, the stored rows of the dict source are used. A nil dict, an
// unknown source and empty rows yield "".
func srvExportCSV(dict *DataDictionary, source string, rows []DataRow) string {
	var cols []string
	if dict != nil {
		for _, s := range dict.Sources {
			if s.Name == source {
				for _, c := range s.Columns {
					cols = append(cols, c.Name)
				}
				if rows == nil {
					rows = s.Rows
				}
				break
			}
		}
	}
	if cols == nil {
		seen := map[string]bool{}
		for _, r := range rows {
			for k := range r {
				if !seen[k] {
					seen[k] = true
					cols = append(cols, k)
				}
			}
		}
		sort.Strings(cols)
	}
	if len(cols) == 0 {
		return ""
	}
	var buf bytes.Buffer
	w := csv.NewWriter(&buf)
	w.UseCRLF = false
	if err := w.Write(cols); err != nil {
		return ""
	}
	for _, r := range rows {
		rec := make([]string, len(cols))
		for i, c := range cols {
			rec[i] = r[c]
		}
		if err := w.Write(rec); err != nil {
			return ""
		}
	}
	w.Flush()
	if err := w.Error(); err != nil {
		return ""
	}
	return buf.String()
}
