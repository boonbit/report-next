package report

import (
	"os"
	"strconv"
	"strings"

	"report-next/internal/aztec"
	"report-next/internal/code128"
	"report-next/internal/datamatrix"
	"report-next/internal/eanupc"
	"report-next/internal/fontmetrics"
	"report-next/internal/maxicode"
	"report-next/internal/pdf"
	"report-next/internal/pdf417"
	"report-next/internal/pdffont"
	"report-next/internal/qr"
)

// PDF export renders TextObjects to a PDF 1.4 document. ASCII-only text
// uses base-14 fonts with no embedded data. When REPORT_FONT_TTF points at
// a loadable TrueType font, non-ASCII text (Thai) is rendered with an
// embedded 8-bit subset of that font instead of being skipped and counted;
// without the variable the legacy skip-and-count behavior is unchanged.
// BarcodeObjects (Code39, QR Code, Datamatrix) are drawn as filled vector
// rectangles with an ASCII caption like the HTML preview;
// TableObject/LineObject/ShapeObject/CheckBoxObject/PictureObject draw with
// fills, outlines and cell text; gauges draw track/fill/ticks/needle/label,
// MatrixObject draws its static template grid (engine-synthesized pivot
// tables arrive as TableObject), and SubreportObject draws a labeled
// placeholder outline.

const (
	pxToPt = 72.0 / 96.0 // .frx geometry is pixels at 96dpi
	mmToPt = 72.0 / 25.4 // ReportPage paper sizes are millimetres
)

// pdfExportReport converts rep to PDF bytes, returning the count of
// skipped non-ASCII text objects.
func pdfExportReport(rep Report) ([]byte, int, error) {
	if path := os.Getenv(pdffont.FntEnvVar); path != "" {
		if fnt, err := pdffont.FntLoadTTF(path); err == nil {
			if out, skipped, ok := fntExportReport(rep, fnt); ok {
				return out, skipped, nil
			}
		}
		// Unloadable font: fall through to legacy skip-and-count so the
		// export still renders without error.
	}
	return fntExportLegacy(rep)
}

// fntExportLegacy is the base-14 ASCII-only export: non-ASCII text objects
// are skipped and counted.
func fntExportLegacy(rep Report) ([]byte, int, error) {
	doc := pdf.New()
	skipped := 0
	for _, p := range rep.Pages {
		w, h := pdfPageSize(p)
		doc.AddPage(w, h)
		pg := doc.Pages() - 1
		for _, b := range p.Bands {
			for _, o := range b.Objects {
				switch o.Kind {
				case "BarcodeObject", "TableObject", "LineObject", "ShapeObject", "CheckBoxObject", "PictureObject",
					"MatrixObject", "LinearGauge", "SimpleGauge", "RadialGauge", "SimpleProgressGauge", "SubreportObject":
					n, err := pdfVectorObject(doc, pg, h, o)
					if err != nil {
						return nil, skipped, err
					}
					skipped += n
					continue
				case "TextObject":
				default:
					continue
				}
				text := attrVal(o.Attrs, "Text")
				if text == "" {
					continue
				}
				if !fontmetrics.ASCIIOnly(text) {
					skipped++
					continue
				}
				size, face := pdfFont(o.Attrs)
				x := pdfNum(o.Attrs, "Left") * pxToPt
				// PDF origin is bottom-left: flip the top-down .frx Y.
				y := h - pdfNum(o.Attrs, "Top")*pxToPt - size
				if err := doc.Text(pg, x, y, size, face, text); err != nil {
					return nil, skipped, err
				}
			}
		}
	}
	if doc.Pages() == 0 {
		doc.AddPage(210*mmToPt, 297*mmToPt)
	}
	return doc.Bytes(), skipped, nil
}

// fntExportReport renders rep with non-ASCII text embedded as an 8-bit
// subset of fnt. ASCII text keeps the base-14 path. Objects whose runes the
// font lacks are skipped and counted. It reports false when there is
// nothing to embed, letting the caller use the legacy export instead.
func fntExportReport(rep Report, fnt *pdffont.FntFont) ([]byte, int, bool) {
	type key struct{ p, b, o int }
	render := map[key]bool{}
	var runes []rune
	seen := map[rune]bool{}
	skipped := 0
	for pi := range rep.Pages {
		for bi := range rep.Pages[pi].Bands {
			for oi := range rep.Pages[pi].Bands[bi].Objects {
				o := &rep.Pages[pi].Bands[bi].Objects[oi]
				if o.Kind != "TextObject" {
					continue
				}
				text := attrVal(o.Attrs, "Text")
				if text == "" || fontmetrics.ASCIIOnly(text) {
					continue
				}
				if len(fnt.FntMissing(text)) > 0 {
					skipped++
					continue
				}
				render[key{pi, bi, oi}] = true
				for _, r := range text {
					if !seen[r] {
						seen[r] = true
						runes = append(runes, r)
					}
				}
			}
		}
	}
	if len(runes) == 0 {
		return nil, 0, false
	}
	sub, err := fnt.FntSubsetFor(runes)
	if err != nil {
		return nil, 0, false
	}
	doc := pdf.New()
	handle, err := doc.FntAddEmbeddedFont(&pdf.FntEmbeddedFont{
		Tag:       sub.Tag,
		Name:      sub.Name,
		Data:      sub.Data,
		FirstChar: sub.FirstChar,
		LastChar:  sub.LastChar,
		Widths:    sub.Widths,
		Names:     sub.Names,
		Unicodes:  sub.Unicodes,
		ToUnicode: sub.FntToUnicodeCMap(),
		Ascent:    sub.Ascent,
		Descent:   sub.Descent,
		CapHeight: sub.CapHeight,
		BBox:      sub.BBox,
	})
	if err != nil {
		return nil, 0, false
	}
	for pi := range rep.Pages {
		p := &rep.Pages[pi]
		w, h := pdfPageSize(*p)
		doc.AddPage(w, h)
		pg := doc.Pages() - 1
		for bi := range p.Bands {
			for oi := range p.Bands[bi].Objects {
				o := &p.Bands[bi].Objects[oi]
				switch o.Kind {
				case "BarcodeObject", "TableObject", "LineObject", "ShapeObject", "CheckBoxObject", "PictureObject",
					"MatrixObject", "LinearGauge", "SimpleGauge", "RadialGauge", "SimpleProgressGauge", "SubreportObject":
					n, err := pdfVectorObject(doc, pg, h, *o)
					if err != nil {
						return nil, skipped, false
					}
					skipped += n
					continue
				case "TextObject":
				default:
					continue
				}
				text := attrVal(o.Attrs, "Text")
				if text == "" {
					continue
				}
				size, face := pdfFont(o.Attrs)
				x := pdfNum(o.Attrs, "Left") * pxToPt
				y := h - pdfNum(o.Attrs, "Top")*pxToPt - size
				if fontmetrics.ASCIIOnly(text) {
					if err := doc.Text(pg, x, y, size, face, text); err != nil {
						return nil, skipped, false
					}
					continue
				}
				if !render[key{pi, bi, oi}] {
					continue // already counted above
				}
				enc, err := sub.FntEncodeString(text)
				if err != nil {
					skipped++
					continue
				}
				if err := doc.FntText(pg, handle, x, y, size, enc); err != nil {
					return nil, skipped, false
				}
			}
		}
		_ = w
	}
	if doc.Pages() == 0 {
		doc.AddPage(210*mmToPt, 297*mmToPt)
	}
	return doc.Bytes(), skipped, true
}

// pdfBarcodeLinear draws Code128, EAN-13 and UPC-A barcodes with the
// internal/code128 and internal/eanupc encoders, scaling the module grid
// into the x/y/w/h body box. It reports drawn=false for every other type
// (including undecodable content and EAN-8) so the caller keeps the legacy
// Code39 rendering.
func pdfBarcodeLinear(doc *pdf.Doc, pg int, x, y, w, h float64, o Object, text string) (bool, error) {
	l := strings.ToLower(strings.TrimSpace(attrVal(o.Attrs, "Barcode")))
	var widths []int
	if strings.Contains(l, "128") {
		code, err := code128.Encode(text)
		if err != nil || code.Modules() == 0 {
			return false, nil
		}
		widths = append([]int(nil), code.Widths...)
	} else {
		var bars string
		if strings.Contains(l, "upc") {
			code, err := eanupc.EncodeUPCA(text)
			if err != nil {
				return false, nil
			}
			bars = code.Bars
		} else if strings.Contains(l, "ean") {
			code, err := eanupc.EncodeEAN13(text)
			if err != nil {
				return false, nil
			}
			bars = code.Bars
		} else {
			return false, nil
		}
		if bars == "" {
			return false, nil
		}
		for i := 0; i < len(bars); {
			j := i
			for j < len(bars) && bars[j] == bars[i] {
				j++
			}
			widths = append(widths, j-i)
			i = j
		}
	}
	total := 0
	for _, m := range widths {
		total += m
	}
	if total == 0 || w <= 0 || h <= 0 {
		return false, nil
	}
	unit := w / float64(total)
	bx := x
	bar := true
	for _, m := range widths {
		bwd := unit * float64(m)
		if bar {
			if err := doc.FillRect(pg, bx, y, bwd, h, 0); err != nil {
				return true, err
			}
		}
		bar = !bar
		bx += bwd
	}
	return true, nil
}

// pdfBarcode draws a BarcodeObject as filled black rectangles, mirroring the
// HTML preview dispatch: QR Code, Datamatrix, PDF417 and Aztec use the
// verified 2D encoders, every other type uses the Code39 bar layout, and
// unsupported 2D types draw nothing. An ASCII caption is drawn below the symbol unless
// ShowText is false; a non-ASCII caption is skipped and counted like
// non-ASCII text. It returns the skipped-caption count (0 or 1).
func pdfBarcode(doc *pdf.Doc, pg int, pageH float64, o Object) (int, error) {
	x := pdfNum(o.Attrs, "Left") * pxToPt
	w := pdfNum(o.Attrs, "Width") * pxToPt
	top := pageH - pdfNum(o.Attrs, "Top")*pxToPt
	h := pdfNum(o.Attrs, "Height") * pxToPt
	if w <= 0 || h <= 0 {
		return 0, nil
	}
	text := attrVal(o.Attrs, "Text")
	caption := ""
	if text != "" && !strings.EqualFold(strings.TrimSpace(attrVal(o.Attrs, "ShowText")), "false") {
		caption = text
	}
	size, face := pdfFont(o.Attrs)
	capH := 0.0
	if caption != "" && fontmetrics.ASCIIOnly(caption) && h > size {
		capH = size
	}
	// y0 is the object bottom; bars/modules sit above the caption strip.
	y0 := top - h
	bodyH := h - capH
	if t := strings.TrimSpace(attrVal(o.Attrs, "Barcode")); objBarcodeIs2D(t) {
		l := strings.ToLower(t)
		var grid [][]bool
		finder := [2]int{-1, -1}
		if strings.Contains(l, "datamatrix") || strings.Contains(l, "data matrix") {
			code, err := datamatrix.Encode([]byte(text), objDataMatrixSize(attrVal(o.Attrs, "Barcode.SymbolSize")))
			if err != nil {
				return 0, nil
			}
			grid = code.Matrix()
		} else if strings.Contains(l, "pdf417") || strings.Contains(l, "pdf-417") {
			lvl, ok := objPdf417Level(o)
			if !ok {
				return 0, nil
			}
			code, err := pdf417.Encode([]byte(text), lvl)
			if err != nil {
				return 0, nil
			}
			grid = code.Matrix()
		} else if strings.Contains(l, "aztec") {
			ecc, ok := objAztecECC(o)
			if !ok {
				return 0, nil
			}
			code, err := aztec.Encode([]byte(text), ecc, 0)
			if err != nil {
				return 0, nil
			}
			grid = code.Matrix()
		} else if strings.Contains(l, "maxicode") || strings.Contains(l, "maxi-code") {
			mode, ok := objMaxiMode(o)
			if !ok {
				return 0, nil
			}
			code, err := maxicode.Encode(mode, []byte(text))
			if err != nil {
				return 0, nil
			}
			grid = code.Matrix()
			finder[0], finder[1] = code.Finder()
		} else if strings.Contains(l, "qr") {
			code, err := qr.QrEncodeWithLevel([]byte(text), objQrLevel(o))
			if err != nil {
				return 0, nil
			}
			grid = code.QrMatrix()
		} else {
			return 0, nil
		}
		if len(grid) == 0 || len(grid[0]) == 0 {
			return 0, nil
		}
		// Rectangular grids (MaxiCode is 30x33) scale independently per axis.
		nw, nh := float64(len(grid[0])), float64(len(grid))
		cell := w / nw
		if c := bodyH / nh; c < cell {
			cell = c
		}
		ox := x + (w-cell*nw)/2
		oy := y0 + capH + (bodyH-cell*nh)/2
		for r := range grid {
			for c := range grid[r] {
				if !grid[r][c] {
					continue
				}
				if err := doc.FillRect(pg, ox+float64(c)*cell, oy+(nh-1-float64(r))*cell, cell, cell, 0); err != nil {
					return 0, err
				}
			}
		}
		if finder[0] >= 0 {
			// Bulls-eye overlay: solid core module plus a square ring at
			// radius 3 (the SVG renderer draws real circles).
			fx, fy := ox+float64(finder[0])*cell, oy+(nh-1-float64(finder[1]))*cell
			if err := doc.FillRect(pg, fx, fy, cell, cell, 0); err != nil {
				return 0, err
			}
			if err := doc.StrokeRect(pg, fx-3*cell, fy-3*cell, 7*cell, 7*cell, cell/4, 0, 0, 0); err != nil {
				return 0, err
			}
		}
	} else if drawn, err := pdfBarcodeLinear(doc, pg, x, y0+capH, w, bodyH, o, text); drawn || err != nil {
		if err != nil {
			return 0, err
		}
	} else {
		pattern := objCode39Pattern(text)
		ratio := 2.0
		if v := strings.TrimSpace(attrVal(o.Attrs, "Barcode.WideBarRatio")); v != "" {
			if f, err := strconv.ParseFloat(v, 64); err == nil {
				ratio = minFloat(3, maxFloat(2, f))
			}
		}
		total := 0.0
		widthOf := func(c byte) float64 {
			if c == '1' || c == '6' {
				return ratio
			}
			return 1
		}
		for i := 0; i < len(pattern); i++ {
			total += widthOf(pattern[i])
		}
		if total == 0 {
			return 0, nil
		}
		unit := w / total
		bx := x
		for i := 0; i < len(pattern); i++ {
			bwd := unit * widthOf(pattern[i])
			if pattern[i] == '5' || pattern[i] == '6' {
				if err := doc.FillRect(pg, bx, y0+capH, bwd, bodyH, 0); err != nil {
					return 0, err
				}
			}
			bx += bwd
		}
	}
	if caption == "" {
		return 0, nil
	}
	if !fontmetrics.ASCIIOnly(caption) {
		return 1, nil
	}
	if capH == 0 {
		return 0, nil
	}
	if err := doc.Text(pg, x, y0, size, face, caption); err != nil {
		return 0, err
	}
	return 0, nil
}

// pdfPageSize reads PaperWidth/PaperHeight (mm), defaulting to A4.
func pdfPageSize(p Page) (w, h float64) {
	w, h = 210*mmToPt, 297*mmToPt
	if v := pdfNum(p.Attrs, "PaperWidth"); v > 0 {
		w = v * mmToPt
	}
	if v := pdfNum(p.Attrs, "PaperHeight"); v > 0 {
		h = v * mmToPt
	}
	if strings.EqualFold(attrVal(p.Attrs, "Landscape"), "true") {
		w, h = h, w
	}
	return w, h
}

// pdfFont maps the .frx Font attribute to a base-14 face. The family name is
// best-effort (viewers substitute their own Helvetica/Courier); only the
// bold flag changes metrics.
func pdfFont(a []Attr) (size float64, face fontmetrics.Face) {
	size = 10
	face = fontmetrics.Helvetica
	bold := false
	for _, part := range strings.Split(attrVal(a, "Font"), ",") {
		p := strings.TrimSpace(part)
		if strings.HasSuffix(p, "pt") {
			if f, err := strconv.ParseFloat(strings.TrimSuffix(p, "pt"), 64); err == nil && f > 0 {
				size = f
			}
		}
		if strings.Contains(p, "Bold") {
			bold = true
		}
		if strings.Contains(strings.ToLower(p), "courier") {
			face = fontmetrics.Courier
		}
	}
	if bold && face == fontmetrics.Courier {
		face = fontmetrics.CourierBold
	} else if bold {
		face = fontmetrics.HelveticaBold
	}
	return size, face
}

func pdfNum(a []Attr, name string) float64 {
	f, _ := strconv.ParseFloat(attrVal(a, name), 64)
	return f
}
