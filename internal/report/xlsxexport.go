package report

import (
	"archive/zip"
	"bytes"
	"fmt"
	"io"
	"sort"
	"strconv"
	"strings"

	"report-next/internal/xlsx"
)

// xlsxExportSource renders one dictionary source as .xlsx bytes: header row
// from declared columns (or sorted keys when undeclared), numbers stored as
// numeric cells when they parse as float.
func xlsxExportSource(dict *DataDictionary, source string) ([]byte, error) {
	if dict == nil {
		return nil, fmt.Errorf("xlsx: no dictionary")
	}
	var src *DataSource
	for i := range dict.Sources {
		if dict.Sources[i].Name == source {
			src = &dict.Sources[i]
			break
		}
	}
	if src == nil {
		return nil, fmt.Errorf("xlsx: unknown source %q", source)
	}
	cols := make([]string, 0, len(src.Columns))
	for _, c := range src.Columns {
		cols = append(cols, c.Name)
	}
	if len(cols) == 0 && len(src.Rows) > 0 {
		for k := range src.Rows[0] {
			cols = append(cols, k)
		}
		sort.Strings(cols)
	}
	w := xlsx.New()
	s := w.AddSheet(source)
	for c, name := range cols {
		s.Set(0, c, name)
	}
	for r, row := range src.Rows {
		for c, name := range cols {
			v := row[name]
			if f, err := strconv.ParseFloat(v, 64); err == nil && v != "" {
				s.SetNum(r+1, c, f)
			} else {
				s.Set(r+1, c, v)
			}
		}
	}
	return w.Bytes()
}

// xlsxExportReport renders the prepared texts of rep as a one-sheet .xlsx
// workbook, reusing the xlsx helpers with the numeric-cell handling of
// xlsxExportSource. dict supplies data binding like the /api/render path;
// both inputs are never mutated. An empty report yields the header alone.
//
// Reports without geometry attributes keep the legacy flat layout (header +
// one data row per text). Reports with Left/Top/Width/Height use the
// positioned grid from exportLayout: TextObjects land in their grid cells,
// wide/tall objects merge cells, and LineObject/ShapeObject render as basic
// box-drawing runs and outlines (the minimal xlsx writer has no style or
// drawing primitives, so rules cannot become real borders). No image
// embedding: PictureObjects are skipped like before.
func xlsxExportReport(rep Report, dict *DataDictionary) ([]byte, error) {
	if !exportHasLayout(rep) {
		return xlsxExportReportFlat(rep, dict)
	}
	const base = 1 // content starts past the header row
	items, rules, boxes := exportLayout(rep, dict)
	w := xlsx.New()
	s := w.AddSheet("Report")
	s.Set(0, 0, reportExportHeader)
	var merges []string
	putText := func(row, col int, text string, num bool, fnum float64) {
		if num {
			s.SetNum(row+base, col, fnum)
		} else {
			s.Set(row+base, col, text)
		}
	}
	for _, it := range items {
		putText(it.Row, it.Col, it.Text, it.Num, it.FNum)
		if it.ColSpan > 1 || it.RowSpan > 1 {
			merges = append(merges, exportMergeRef(it.Col, it.Row, it.ColSpan, it.RowSpan, base))
		}
	}
	for _, ru := range rules {
		if ru.Horizontal {
			n := ru.ColSpan * 8
			if n < 4 {
				n = 4
			}
			putText(ru.Row, ru.Col, strings.Repeat("\u2500", n), false, 0)
			if ru.ColSpan > 1 {
				merges = append(merges, exportMergeRef(ru.Col, ru.Row, ru.ColSpan, 1, base))
			}
		} else {
			for i := 0; i < ru.RowSpan; i++ {
				putText(ru.Row+i, ru.Col, "\u2502", false, 0)
			}
		}
	}
	for _, bx := range boxes {
		if bx.ColSpan < 2 || bx.RowSpan < 2 {
			continue
		}
		inner := bx.ColSpan*8 - 2
		if inner < 1 {
			inner = 1
		}
		putText(bx.Row, bx.Col, "\u250c"+strings.Repeat("\u2500", inner)+"\u2510", false, 0)
		merges = append(merges, exportMergeRef(bx.Col, bx.Row, bx.ColSpan, 1, base))
		for i := 1; i < bx.RowSpan-1; i++ {
			putText(bx.Row+i, bx.Col, "\u2502"+strings.Repeat(" ", inner)+"\u2502", false, 0)
			merges = append(merges, exportMergeRef(bx.Col, bx.Row+i, bx.ColSpan, 1, base))
		}
		putText(bx.Row+bx.RowSpan-1, bx.Col, "\u2514"+strings.Repeat("\u2500", inner)+"\u2518", false, 0)
		merges = append(merges, exportMergeRef(bx.Col, bx.Row+bx.RowSpan-1, bx.ColSpan, 1, base))
	}
	data, err := w.Bytes()
	if err != nil {
		return nil, err
	}
	return xlsxAddMerges(data, merges)
}

// xlsxExportReportFlat is the legacy report export: header plus one data
// row per prepared text in print order.
func xlsxExportReportFlat(rep Report, dict *DataDictionary) ([]byte, error) {
	rows := reportExportRows(rep, dict)
	w := xlsx.New()
	s := w.AddSheet("Report")
	s.Set(0, 0, reportExportHeader)
	for r, row := range rows {
		v := row[reportExportHeader]
		if f, err := strconv.ParseFloat(v, 64); err == nil && v != "" {
			s.SetNum(r+1, 0, f)
		} else {
			s.Set(r+1, 0, v)
		}
	}
	return w.Bytes()
}

// xlsxAddMerges injects a mergeCells element into every worksheet part of an
// xlsx workbook produced by the minimal internal/xlsx writer (which has no
// merge primitive). Empty refs return the input unchanged.
func xlsxAddMerges(data []byte, refs []string) ([]byte, error) {
	seen := map[string]bool{}
	var uniq []string
	for _, r := range refs {
		if r == "" || seen[r] {
			continue
		}
		seen[r] = true
		uniq = append(uniq, r)
	}
	if len(uniq) == 0 {
		return data, nil
	}
	sort.Strings(uniq)
	zr, err := zip.NewReader(bytes.NewReader(data), int64(len(data)))
	if err != nil {
		return nil, err
	}
	parts := make([]struct {
		name string
		body string
	}, 0, len(zr.File))
	for _, f := range zr.File {
		rc, err := f.Open()
		if err != nil {
			return nil, err
		}
		body, err := io.ReadAll(rc)
		rc.Close()
		if err != nil {
			return nil, err
		}
		parts = append(parts, struct {
			name string
			body string
		}{f.Name, string(body)})
	}
	var mc strings.Builder
	mc.WriteString(`<mergeCells count="`)
	mc.WriteString(strconv.Itoa(len(uniq)))
	mc.WriteString(`">`)
	for _, r := range uniq {
		mc.WriteString(`<mergeCell ref="` + r + `"/>`)
	}
	mc.WriteString(`</mergeCells>`)
	for i, p := range parts {
		if strings.HasPrefix(p.name, "xl/worksheets/sheet") && strings.HasSuffix(p.name, ".xml") {
			if strings.Contains(p.body, "</worksheet>") && !strings.Contains(p.body, "<mergeCells") {
				parts[i].body = strings.Replace(p.body, "</worksheet>", mc.String()+"</worksheet>", 1)
			}
		}
	}
	var buf bytes.Buffer
	zw := zip.NewWriter(&buf)
	for _, p := range parts {
		f, err := zw.Create(p.name)
		if err != nil {
			return nil, err
		}
		if _, err := f.Write([]byte(p.body)); err != nil {
			return nil, err
		}
	}
	if err := zw.Close(); err != nil {
		return nil, err
	}
	return buf.Bytes(), nil
}
