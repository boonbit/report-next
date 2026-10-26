package report

import (
	"strconv"
	"strings"

	"report-next/internal/fontmetrics"
	"report-next/internal/pdf"
)

// PDF drawing for TableObject/LineObject/ShapeObject/CheckBoxObject/
// PictureObject, mirroring the HTML preview geometry. Tables use the same
// tabParseTable layout: column widths from TableColumn Width (columns
// without a width share the remainder equally), row heights from TableRow
// Height (rows without one share the table remainder, defaulting to one
// text line when the table is already full). ColSpan/RowSpan merge cells.
// Approximations, documented: RoundRectangle corners square off, table and
// cell borders draw as full outlines whenever Border.Lines is set (FastReport
// side selections are not split out), and cell text uses the same baseline
// convention as TextObject export.

// pdfVectorObject draws one non-text vector object (barcode, table, line,
// shape, checkbox, picture), returning skipped non-ASCII texts (barcode
// captions and table cell/nested texts count like TextObject export).
func pdfVectorObject(doc *pdf.Doc, pg int, pageH float64, o Object) (int, error) {
	switch o.Kind {
	case "BarcodeObject":
		return pdfBarcode(doc, pg, pageH, o)
	case "TableObject":
		return pdfTable(doc, pg, pageH, o)
	case "MatrixObject":
		return pdfMatrix(doc, pg, pageH, o)
	case "LinearGauge", "SimpleGauge", "RadialGauge", "SimpleProgressGauge":
		return pdfGauge(doc, pg, pageH, o)
	case "SubreportObject":
		return 0, pdfSubreport(doc, pg, pageH, o)
	}
	switch o.Kind {
	case "TableObject":
		return pdfTable(doc, pg, pageH, o)
	case "LineObject":
		return 0, pdfLine(doc, pg, pageH, o)
	case "ShapeObject":
		return 0, pdfShape(doc, pg, pageH, o)
	case "CheckBoxObject":
		return 0, pdfCheckBox(doc, pg, pageH, o)
	case "PictureObject":
		return 0, pdfPicture(doc, pg, pageH, o)
	default:
		return 0, nil
	}
}

// pdfColor parses a FastReport color: "r, g, b" triples or common .NET
// color names (case-insensitive). Transparent and unknown values report
// ok=false so callers skip the fill or fall back to black borders.
func pdfColor(v string) (r, g, b float64, ok bool) {
	v = strings.TrimSpace(v)
	if v == "" || strings.EqualFold(v, "Transparent") {
		return 0, 0, 0, false
	}
	if parts := strings.Split(v, ","); len(parts) == 3 {
		var c [3]float64
		for i, p := range parts {
			n, err := strconv.Atoi(strings.TrimSpace(p))
			if err != nil || n < 0 || n > 255 {
				return 0, 0, 0, false
			}
			c[i] = float64(n) / 255
		}
		return c[0], c[1], c[2], true
	}
	if c, found := map[string][3]float64{
		"black": {0, 0, 0}, "white": {1, 1, 1},
		"red": {1, 0, 0}, "green": {0, 0.50196, 0}, "blue": {0, 0, 1},
		"yellow": {1, 1, 0}, "orange": {1, 0.64706, 0},
		"silver": {0.75294, 0.75294, 0.75294},
		"gray":   {0.50196, 0.50196, 0.50196}, "grey": {0.50196, 0.50196, 0.50196},
		"lightgray": {0.82745, 0.82745, 0.82745}, "lightgrey": {0.82745, 0.82745, 0.82745},
		"darkgray": {0.66275, 0.66275, 0.66275}, "darkgrey": {0.66275, 0.66275, 0.66275},
		"maroon": {0.50196, 0, 0}, "navy": {0, 0, 0.50196},
		"teal": {0, 0.50196, 0.50196}, "purple": {0.50196, 0, 0.50196},
		"olive": {0.50196, 0.50196, 0}, "aqua": {0, 1, 1}, "cyan": {0, 1, 1},
		"fuchsia": {1, 0, 1}, "magenta": {1, 0, 1},
		"brown": {0.64706, 0.16471, 0.16471}, "pink": {1, 0.75294, 0.79608},
		"whitesmoke": {0.96078, 0.96078, 0.96078}, "gainsboro": {0.86275, 0.86275, 0.86275},
		"lightyellow": {1, 1, 0.87843}, "lightcyan": {0.87843, 1, 1},
		"lavender": {0.90196, 0.90196, 0.98039}, "beige": {0.96078, 0.96078, 0.86275},
		"ivory": {1, 1, 0.94118}, "honeydew": {0.94118, 1, 0.94118},
		"azure": {0.94118, 1, 1}, "linen": {0.98039, 0.94118, 0.90196},
		"khaki": {0.94118, 0.90196, 0.54902}, "gold": {1, 0.84314, 0},
		"salmon": {0.98039, 0.50196, 0.44706}, "violet": {0.93333, 0.5098, 0.93333},
	}[strings.ToLower(v)]; found {
		return c[0], c[1], c[2], true
	}
	return 0, 0, 0, false
}

// pdfBorder reads the border width (px, default 1 like the HTML preview).
func pdfBorder(a []Attr) float64 {
	return objBorderWidth(a) * pxToPt
}

// pdfHasBorder reports whether an outline should draw: Border.Lines set to
// anything but empty or None.
func pdfHasBorder(a []Attr) bool {
	v := strings.TrimSpace(attrVal(a, "Border.Lines"))
	return v != "" && !strings.EqualFold(v, "None")
}

// pdfLine draws a LineObject: a horizontal rule for wide boxes, vertical
// for tall ones, corner-to-corner for Diagonal, mirroring objLineHTML.
func pdfLine(doc *pdf.Doc, pg int, pageH float64, o Object) error {
	l, t, w, h := objRectOf(o.Attrs)
	if w <= 0 || h <= 0 {
		return nil
	}
	x, y0 := l*pxToPt, pageH-(t+h)*pxToPt
	lw := pdfBorder(o.Attrs)
	sr, sg, sb, ok := pdfColor(attrVal(o.Attrs, "Border.Color"))
	if !ok {
		sr, sg, sb = 0, 0, 0
	}
	x1, y1, x2, y2 := x, y0, x+w*pxToPt, y0+h*pxToPt
	if strings.EqualFold(strings.TrimSpace(attrVal(o.Attrs, "Diagonal")), "true") {
		return doc.StrokeLine(pg, x1, y0, x2, y0+h*pxToPt, lw, sr, sg, sb)
	}
	if w >= h {
		y := y0 + h*pxToPt/2
		return doc.StrokeLine(pg, x1, y, x2, y, lw, sr, sg, sb)
	}
	cx := x + w*pxToPt/2
	return doc.StrokeLine(pg, cx, y1, cx, y2, lw, sr, sg, sb)
}

// pdfShape draws Rectangle/RoundRectangle/Ellipse/Diamond/Triangle
// ShapeObjects with fill and outline. Unknown shapes draw nothing, like
// the HTML labeled-box fallback draws no PDF geometry.
func pdfShape(doc *pdf.Doc, pg int, pageH float64, o Object) error {
	shape := strings.TrimSpace(attrVal(o.Attrs, "Shape"))
	if shape == "" {
		shape = "Rectangle"
	}
	switch shape {
	case "Rectangle", "RoundRectangle", "Ellipse", "Diamond", "Triangle":
	default:
		return nil
	}
	l, t, w, h := objRectOf(o.Attrs)
	return pdfShapeRect(doc, pg, l*pxToPt, pageH-(t+h)*pxToPt, w*pxToPt, h*pxToPt, o.Attrs)
}

// pdfShapeRect draws a ShapeObject at an explicit box (PDF points, y0 at the
// box bottom), so matrix/table cells can reuse it at relative offsets.
func pdfShapeRect(doc *pdf.Doc, pg int, x, y0, pw, ph float64, a []Attr) error {
	shape := strings.TrimSpace(attrVal(a, "Shape"))
	if shape == "" {
		shape = "Rectangle"
	}
	switch shape {
	case "Rectangle", "RoundRectangle", "Ellipse", "Diamond", "Triangle":
	default:
		return nil
	}
	if pw <= 0 || ph <= 0 {
		return nil
	}
	fr, fg, fb, fill := pdfColor(attrVal(a, "Fill.Color"))
	sr, sg, sb, sok := pdfColor(attrVal(a, "Border.Color"))
	if !sok {
		sr, sg, sb = 0, 0, 0
	}
	lw := pdfBorder(a)
	stroke := lw > 0 && !strings.EqualFold(strings.TrimSpace(attrVal(a, "Border.Color")), "Transparent")
	switch shape {
	case "Ellipse":
		return doc.Ellipse(pg, x+pw/2, y0+ph/2, pw/2, ph/2,
			fill, stroke, lw, fr, fg, fb, sr, sg, sb)
	case "Diamond":
		return doc.Poly(pg, [][2]float64{
			{x + pw/2, y0 + ph}, {x + pw, y0 + ph/2}, {x + pw/2, y0}, {x, y0 + ph/2},
		}, true, fill, stroke, lw, fr, fg, fb, sr, sg, sb)
	case "Triangle":
		return doc.Poly(pg, [][2]float64{
			{x + pw/2, y0 + ph}, {x + pw, y0}, {x, y0},
		}, true, fill, stroke, lw, fr, fg, fb, sr, sg, sb)
	default:
		// Rectangle; RoundRectangle corners square off (documented).
		if fill {
			if err := doc.FillRectRGB(pg, x, y0, pw, ph, fr, fg, fb); err != nil {
				return err
			}
		}
		if stroke {
			return doc.StrokeRect(pg, x, y0, pw, ph, lw, sr, sg, sb)
		}
		return nil
	}
}

// pdfCheckBoxRect draws a checkbox at an explicit box with the checked mark
// honoring CheckedSymbol (Cross/Plus/Fill, default tick), mirroring
// objCheckBoxHTML. The state comes from objCheckedState (static attrs only:
// CheckState wins over Checked under the objTruthy rule); row-bound state
// needs objCheckedStateBound, which the PDF path cannot reach because band
// instances never carry their fields this far. Coordinates are PDF points
// with y0 at the box bottom.
func pdfCheckBoxRect(doc *pdf.Doc, pg int, x, y0, w, h float64, a []Attr) error {
	if w <= 0 || h <= 0 {
		return nil
	}
	mr, mg, mb, ok := pdfColor(attrVal(a, "CheckColor"))
	if !ok {
		mr, mg, mb = 0, 0, 0
	}
	if err := doc.StrokeRect(pg, x, y0, w, h, 0.75, 0, 0, 0); err != nil {
		return err
	}
	if !objCheckedState(a) {
		return nil
	}
	at := func(fx, fy float64) [2]float64 { return [2]float64{x + fx*w, y0 + fy*h} }
	switch strings.TrimSpace(attrVal(a, "CheckedSymbol")) {
	case "Cross":
		if err := doc.StrokeLine(pg, at(0.2, 0.2)[0], at(0.2, 0.2)[1], at(0.8, 0.8)[0], at(0.8, 0.8)[1], 1, mr, mg, mb); err != nil {
			return err
		}
		return doc.StrokeLine(pg, at(0.2, 0.8)[0], at(0.2, 0.8)[1], at(0.8, 0.2)[0], at(0.8, 0.2)[1], 1, mr, mg, mb)
	case "Plus":
		if err := doc.StrokeLine(pg, at(0.2, 0.5)[0], at(0.2, 0.5)[1], at(0.8, 0.5)[0], at(0.8, 0.5)[1], 1, mr, mg, mb); err != nil {
			return err
		}
		return doc.StrokeLine(pg, at(0.5, 0.2)[0], at(0.5, 0.2)[1], at(0.5, 0.8)[0], at(0.5, 0.8)[1], 1, mr, mg, mb)
	case "Fill":
		return doc.FillRectRGB(pg, x+w*0.2, y0+h*0.2, w*0.6, h*0.6, mr, mg, mb)
	default:
		return doc.Poly(pg, [][2]float64{at(0.2, 0.5), at(0.45, 0.25), at(0.8, 0.75)},
			false, false, true, 1, 0, 0, 0, mr, mg, mb)
	}
}

// pdfCheckBox draws a CheckBoxObject at its own geometry.
func pdfCheckBox(doc *pdf.Doc, pg int, pageH float64, o Object) error {
	l, t, w, h := objRectOf(o.Attrs)
	return pdfCheckBoxRect(doc, pg, l*pxToPt, pageH-(t+h)*pxToPt, w*pxToPt, h*pxToPt, o.Attrs)
}

// pdfPicture draws a PictureObject as a gray placeholder outline with its
// ASCII label, mirroring the HTML placeholder. Bound raster bytes resolve
// through objPictureBytes, but internal/pdf has no image/XObject emission
// API, so the raster cannot be drawn yet and the box is kept (R2 unresolved).
func pdfPicture(doc *pdf.Doc, pg int, pageH float64, o Object) error {
	l, t, w, h := objRectOf(o.Attrs)
	return pdfPictureRect(doc, pg, l*pxToPt, pageH-(t+h)*pxToPt, w*pxToPt, h*pxToPt, o)
}

// pdfPictureRect draws a PictureObject placeholder at an explicit box (PDF
// points, y0 at the box bottom) for reuse at cell-relative offsets.
func pdfPictureRect(doc *pdf.Doc, pg int, x, y0, pw, ph float64, o Object) error {
	if pw <= 0 || ph <= 0 {
		return nil
	}
	if err := doc.StrokeRect(pg, x, y0, pw, ph, 0.75, 0.5, 0.5, 0.5); err != nil {
		return err
	}
	label := strings.TrimSpace(attrVal(o.Attrs, "DataColumn"))
	if label == "" {
		label = o.Name
	}
	label = "Picture: " + label
	if label == "" || !fontmetrics.ASCIIOnly(label) {
		return nil
	}
	return doc.Text(pg, x+2, y0+ph-10, 8, fontmetrics.Helvetica, label)
}

// pdfTable draws a TableObject: fills, outlines, cell text with alignment
// and nested TextObject/CheckBoxObject content. Unparseable tables draw
// nothing (the HTML labeled-box fallback has no PDF geometry).
func pdfTable(doc *pdf.Doc, pg int, pageH float64, o Object) (int, error) {
	t, err := tabParseTable(o)
	if err != nil {
		return 0, nil
	}
	l, top, w, h := objRectOf(o.Attrs)
	if w <= 0 || h <= 0 || len(t.rows) == 0 {
		return 0, nil
	}
	return pdfGrid(doc, pg, l*pxToPt, pageH-top*pxToPt, w, h, t.columns, t.rows, o.Attrs)
}

// pdfGrid draws a column/row grid shared by TableObject and the static
// MatrixObject template. Widths/heights are FastReport pixels; x/yTop/pw/ph
// are the same box in PDF points. borderAttrs supplies the outer outline.
func pdfGrid(doc *pdf.Doc, pg int, x, yTop, w, h float64, cols []tabColumn, rows []tabRow, borderAttrs []Attr) (int, error) {
	pw, ph := w*pxToPt, h*pxToPt
	widths := pdfColumnWidths(cols, w)
	if len(widths) == 0 {
		// No declared columns: split the table evenly across the widest row.
		maxCols := 0
		for _, r := range rows {
			n := 0
			for _, c := range r.cells {
				n += max(c.colSpan, 1)
			}
			if n > maxCols {
				maxCols = n
			}
		}
		if maxCols > 0 {
			each := w * pxToPt / float64(maxCols)
			widths = make([]float64, maxCols)
			for i := range widths {
				widths[i] = each
			}
		}
	}
	heights := pdfRowHeights(rows, h)
	skipped := 0
	// Columns beyond the declared TableColumns (real fixtures overflow:
	// Table1/Row1 holds 6 logical columns on 4 declared ones) reuse the
	// mean declared width, mirroring the HTML table that simply overflows.
	avgW := 0.0
	if len(widths) > 0 {
		for _, cw := range widths {
			avgW += cw
		}
		avgW /= float64(len(widths))
	}
	colW := func(i int) float64 {
		if i < len(widths) {
			return widths[i]
		}
		return avgW
	}
	// Occupancy tracks RowSpan merges from earlier rows.
	occ := map[[2]int]bool{}
	y := yTop
	for r := range rows {
		xc, ci := x, 0
		for _, cell := range rows[r].cells {
			for occ[[2]int{r, ci}] {
				xc += colW(ci)
				ci++
			}
			cs := cell.colSpan
			if cs < 1 {
				cs = 1
			}
			rs := cell.rowSpan
			if rs < 1 {
				rs = 1
			}
			cw := 0.0
			for k := ci; k < ci+cs; k++ {
				cw += colW(k)
			}
			ch := 0.0
			for k := r; k < r+rs && k < len(heights); k++ {
				ch += heights[k]
			}
			for k := r; k < r+rs && k < len(rows); k++ {
				for m := ci; m < ci+cs; m++ {
					occ[[2]int{k, m}] = true
				}
			}
			n, err := pdfCell(doc, pg, xc, y, cw, ch*pxToPt, cell)
			if err != nil {
				return skipped, err
			}
			skipped += n
			xc += cw
			ci += cs
		}
		y -= heights[r] * pxToPt
	}
	if pdfHasBorder(borderAttrs) {
		sr, sg, sb, ok := pdfColor(attrVal(borderAttrs, "Border.Color"))
		if !ok {
			sr, sg, sb = 0, 0, 0
		}
		if err := doc.StrokeRect(pg, x, yTop-ph, pw, ph, pdfBorder(borderAttrs), sr, sg, sb); err != nil {
			return skipped, err
		}
	}
	return skipped, nil
}

// pdfColumnWidths resolves column widths in points: numeric widths win,
// the remainder splits equally among the rest; with no flexible column the
// fixed widths scale to the table width.
func pdfColumnWidths(cols []tabColumn, tableW float64) []float64 {
	out := make([]float64, len(cols))
	fixed, flex := 0.0, 0
	for i, c := range cols {
		if f, err := strconv.ParseFloat(strings.TrimSpace(c.width), 64); err == nil && f > 0 {
			out[i] = f
			fixed += f
		} else {
			flex++
		}
	}
	switch {
	case flex > 0:
		rest := (tableW - fixed) / float64(flex)
		if rest < 0 {
			rest = 0
		}
		for i := range out {
			if out[i] == 0 {
				out[i] = rest
			}
		}
	case fixed > 0 && fixed != tableW:
		for i := range out {
			out[i] = out[i] * tableW / fixed
		}
	case fixed == 0 && len(out) > 0:
		for i := range out {
			out[i] = tableW / float64(len(out))
		}
	}
	for i := range out {
		out[i] *= pxToPt
	}
	return out
}

// pdfDefaultRowH is one auto row: 9pt text plus padding, in px.
const pdfDefaultRowH = 18.9

// pdfRowHeights resolves row heights in px: explicit TableRow Height wins,
// rows without one share the table remainder (or the default when full).
func pdfRowHeights(rows []tabRow, tableH float64) []float64 {
	out := make([]float64, len(rows))
	explicit, auto := 0.0, 0
	for i, r := range rows {
		if f, err := strconv.ParseFloat(strings.TrimSpace(attrVal(r.attrs, "Height")), 64); err == nil && f > 0 {
			out[i] = f
			explicit += f
		} else {
			auto++
		}
	}
	if auto > 0 {
		each := (tableH - explicit) / float64(auto)
		if each <= 0 {
			each = pdfDefaultRowH
		}
		for i := range out {
			if out[i] == 0 {
				out[i] = each
			}
		}
	}
	return out
}

// pdfCell draws one cell: fill, outline, text with HorzAlign/VertAlign and
// nested TextObject/CheckBoxObject content. All coordinates are PDF points;
// yTop is the cell top, ch the cell height.
func pdfCell(doc *pdf.Doc, pg int, cx, yTop, cw, ch float64, cell tabCell) (int, error) {
	if cw <= 0 || ch <= 0 {
		return 0, nil
	}
	skipped := 0
	if fr, fg, fb, ok := pdfColor(attrVal(cell.attrs, "Fill.Color")); ok {
		if err := doc.FillRectRGB(pg, cx, yTop-ch, cw, ch, fr, fg, fb); err != nil {
			return skipped, err
		}
	}
	if pdfHasBorder(cell.attrs) {
		sr, sg, sb, ok := pdfColor(attrVal(cell.attrs, "Border.Color"))
		if !ok {
			sr, sg, sb = 0, 0, 0
		}
		if err := doc.StrokeRect(pg, cx, yTop-ch, cw, ch, pdfBorder(cell.attrs), sr, sg, sb); err != nil {
			return skipped, err
		}
	}
	text := attrVal(cell.attrs, "Text")
	if text != "" {
		n, err := pdfCellText(doc, pg, cx, yTop, cw, ch, cell.attrs, text)
		if err != nil {
			return skipped, err
		}
		skipped += n
	}
	for _, no := range cell.objects {
		switch no.Kind {
		case "TextObject":
			if text := attrVal(no.Attrs, "Text"); text != "" {
				nx := cx + objFloatAttr(no.Attrs, "Left")*pxToPt
				ny := yTop - objFloatAttr(no.Attrs, "Top")*pxToPt
				n, err := pdfCellText(doc, pg, nx, ny, cw, ch, no.Attrs, text)
				if err != nil {
					return skipped, err
				}
				skipped += n
			}
		case "CheckBoxObject":
			nx := cx + objFloatAttr(no.Attrs, "Left")*pxToPt
			ny := yTop - (objFloatAttr(no.Attrs, "Top")+objFloatAttr(no.Attrs, "Height"))*pxToPt
			if err := pdfCheckBoxRect(doc, pg, nx, ny,
				objFloatAttr(no.Attrs, "Width")*pxToPt,
				objFloatAttr(no.Attrs, "Height")*pxToPt, no.Attrs); err != nil {
				return skipped, err
			}
		case "ShapeObject":
			nx := cx + objFloatAttr(no.Attrs, "Left")*pxToPt
			ny := yTop - (objFloatAttr(no.Attrs, "Top")+objFloatAttr(no.Attrs, "Height"))*pxToPt
			if err := pdfShapeRect(doc, pg, nx, ny,
				objFloatAttr(no.Attrs, "Width")*pxToPt,
				objFloatAttr(no.Attrs, "Height")*pxToPt, no.Attrs); err != nil {
				return skipped, err
			}
		case "PictureObject":
			nx := cx + objFloatAttr(no.Attrs, "Left")*pxToPt
			ny := yTop - (objFloatAttr(no.Attrs, "Top")+objFloatAttr(no.Attrs, "Height"))*pxToPt
			if err := pdfPictureRect(doc, pg, nx, ny,
				objFloatAttr(no.Attrs, "Width")*pxToPt,
				objFloatAttr(no.Attrs, "Height")*pxToPt, no); err != nil {
				return skipped, err
			}
		case "LinearGauge", "SimpleGauge", "RadialGauge", "SimpleProgressGauge":
			nx := cx + objFloatAttr(no.Attrs, "Left")*pxToPt
			ny := yTop - (objFloatAttr(no.Attrs, "Top")+objFloatAttr(no.Attrs, "Height"))*pxToPt
			n, err := pdfGaugeRect(doc, pg, nx, ny,
				objFloatAttr(no.Attrs, "Width")*pxToPt,
				objFloatAttr(no.Attrs, "Height")*pxToPt, no)
			if err != nil {
				return skipped, err
			}
			skipped += n
		}
	}
	return skipped, nil
}

// pdfCellText draws one left/center/right aligned text line in a cell,
// honoring VertAlign (Top/Center/Bottom, default Top). Non-ASCII text is
// skipped and counted. yTop is the line's reference top in PDF points.
func pdfCellText(doc *pdf.Doc, pg int, cx, yTop, cw, ch float64, a []Attr, text string) (int, error) {
	if !fontmetrics.ASCIIOnly(text) {
		return 1, nil
	}
	size, face := pdfFont(a)
	pad := 2 * pxToPt
	tx := cx + pad
	if w := face.Width(text, size); w < cw {
		switch attrVal(a, "HorzAlign") {
		case "Center":
			tx = cx + (cw-w)/2
		case "Right", "Justify":
			tx = cx + cw - w - pad
		}
	}
	ty := yTop - pad - size
	switch attrVal(a, "VertAlign") {
	case "Center":
		ty = yTop - ch + (ch-size)/2
	case "Bottom":
		ty = yTop - ch + pad
	}
	return 0, doc.Text(pg, tx, ty, size, face, text)
}
