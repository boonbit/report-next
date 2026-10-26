package report

import (
	"encoding/base64"
	"fmt"
	"html"
	"strconv"
	"strings"

	"report-next/internal/aztec"
	"report-next/internal/code128"
	"report-next/internal/datamatrix"
	"report-next/internal/eanupc"
	"report-next/internal/maxicode"
	"report-next/internal/pdf417"
	"report-next/internal/qr"
)

// TRACK obj (P4 render): HTML renderers for non-text band objects.
//
// Real attribute names were sampled from FastReport/Demos/Reports:
//   LineObject    Left Top Width Height Border.Width Border.Color Diagonal
//   ShapeObject   Shape(Rectangle/RoundRectangle/Ellipse/Diamond/Triangle)
//                 Border.Color Border.Width Fill.Color
//   CheckBoxObject Checked DataColumn CheckedSymbol(Cross/Fill/Plus)
//                 CheckColor Border.Lines Fill.Color
//   PictureObject DataColumn SizeMode Angle
//   BarcodeObject Text Barcode Barcode.WideBarRatio ShowText AutoSize
//                 Barcode.QuietZone Barcode.Angle
//   TableCell     Text Font HorzAlign VertAlign Fill.Color Border.Lines
//                 (inspected only; TableObject nesting stays verbatim for P5)
//
// P5: Angle / Barcode.Angle rotation is ignored by every renderer below.

// objRectOf reads the FastReport geometry attributes. Missing or
// non-numeric values yield 0.
func objRectOf(a []Attr) (x, y, w, h float64) {
	return objFloatAttr(a, "Left"), objFloatAttr(a, "Top"),
		objFloatAttr(a, "Width"), objFloatAttr(a, "Height")
}

func objFloatAttr(a []Attr, name string) float64 {
	v := strings.TrimSpace(attrVal(a, name))
	if v == "" {
		return 0
	}
	f, err := strconv.ParseFloat(v, 64)
	if err != nil {
		return 0
	}
	return f
}

// objAbsStyle positions an object box inside the band div.
func objAbsStyle(a []Attr, extra string) string {
	x, y, w, h := objRectOf(a)
	return fmt.Sprintf("position:absolute;left:%gpx;top:%gpx;width:%gpx;height:%gpx;%s",
		x, y, w, h, extra)
}

func objBorderWidth(a []Attr) float64 {
	if v := strings.TrimSpace(attrVal(a, "Border.Width")); v != "" {
		if f, err := strconv.ParseFloat(v, 64); err == nil {
			return f
		}
	}
	return 1
}

func objStrokeColor(a []Attr) string {
	if c := strings.TrimSpace(attrVal(a, "Border.Color")); c != "" {
		if strings.EqualFold(c, "Transparent") {
			return "none"
		}
		return c
	}
	return "Black"
}

func objFillColor(a []Attr) string {
	if c := strings.TrimSpace(attrVal(a, "Fill.Color")); c != "" {
		if strings.EqualFold(c, "Transparent") {
			return "none"
		}
		return c
	}
	return "none"
}

// objFallbackHTML is the labeled-box fallback for object kinds without a
// dedicated renderer (and for 2D barcode types beyond QR/DataMatrix).
func objFallbackHTML(o Object) string {
	label := o.Kind
	if o.Name != "" {
		label += ": " + o.Name
	}
	if t := attrVal(o.Attrs, "Barcode"); t != "" {
		label += " (" + t + ")"
	}
	return fmt.Sprintf(`<div class="obj" data-kind="%s" style="%s">%s</div>`,
		html.EscapeString(o.Kind), objAbsStyle(o.Attrs, "border:1px dashed gray;overflow:hidden;"),
		html.EscapeString(label))
}

// objLineHTML renders a LineObject as a div rule. Orientation comes from the
// box shape: wide boxes are horizontal rules, tall boxes vertical rules.
// Diagonal="true" draws a corner-to-corner SVG line instead.
func objLineHTML(o Object) string {
	_, _, w, h := objRectOf(o.Attrs)
	bw := objBorderWidth(o.Attrs)
	bc := objStrokeColor(o.Attrs)
	pos := objAbsStyle(o.Attrs, "")
	if strings.EqualFold(strings.TrimSpace(attrVal(o.Attrs, "Diagonal")), "true") {
		return fmt.Sprintf(`<div class="line" data-kind="LineObject" style="%s">`+
			`<svg width="100%%" height="100%%" viewBox="0 0 100 100" preserveAspectRatio="none">`+
			`<line x1="0" y1="0" x2="100" y2="100" stroke="%s" stroke-width="%g"/></svg></div>`,
			pos, html.EscapeString(bc), bw)
	}
	if w >= h {
		return fmt.Sprintf(`<div class="line" data-kind="LineObject" style="%sborder-top:%gpx solid %s;"></div>`,
			pos, bw, html.EscapeString(bc))
	}
	return fmt.Sprintf(`<div class="line" data-kind="LineObject" style="%sborder-left:%gpx solid %s;"></div>`,
		pos, bw, html.EscapeString(bc))
}

// objShapeHTML renders Rectangle (default), RoundRectangle, Ellipse, Diamond
// and Triangle ShapeObjects as inline SVG. Other Shape values fall back to a
// labeled box.
func objShapeHTML(o Object) string {
	shape := strings.TrimSpace(attrVal(o.Attrs, "Shape"))
	switch shape {
	case "", "Rectangle", "RoundRectangle", "Ellipse", "Diamond", "Triangle":
	default:
		return objFallbackHTML(o)
	}
	if shape == "" {
		shape = "Rectangle"
	}
	_, _, w, h := objRectOf(o.Attrs)
	if w <= 0 {
		w = 100
	}
	if h <= 0 {
		h = 100
	}
	bw := objBorderWidth(o.Attrs)
	stroke := objStrokeColor(o.Attrs)
	fill := objFillColor(o.Attrs)
	strokeAttr := fmt.Sprintf(`stroke="%s" stroke-width="%g"`, html.EscapeString(stroke), bw)
	if stroke == "none" || bw == 0 {
		strokeAttr = `stroke="none"`
	}
	f := func(v float64) string { return strconv.FormatFloat(v, 'f', 2, 64) }
	var body string
	switch shape {
	case "Rectangle":
		body = fmt.Sprintf(`<rect x="%s" y="%s" width="%s" height="%s" fill="%s" %s/>`,
			f(bw/2), f(bw/2), f(w-bw), f(h-bw), html.EscapeString(fill), strokeAttr)
	case "RoundRectangle":
		r := f(minFloat(w, h) * 0.15)
		body = fmt.Sprintf(`<rect x="%s" y="%s" width="%s" height="%s" rx="%s" fill="%s" %s/>`,
			f(bw/2), f(bw/2), f(w-bw), f(h-bw), r, html.EscapeString(fill), strokeAttr)
	case "Ellipse":
		body = fmt.Sprintf(`<ellipse cx="%s" cy="%s" rx="%s" ry="%s" fill="%s" %s/>`,
			f(w/2), f(h/2), f(w/2-bw/2), f(h/2-bw/2), html.EscapeString(fill), strokeAttr)
	case "Diamond":
		body = fmt.Sprintf(`<polygon points="%s,0 %s,%s %s,%s 0,%s" fill="%s" %s/>`,
			f(w/2), f(w), f(h/2), f(w/2), f(h), f(h/2), html.EscapeString(fill), strokeAttr)
	case "Triangle":
		body = fmt.Sprintf(`<polygon points="%s,0 %s,%s 0,%s" fill="%s" %s/>`,
			f(w/2), f(w), f(h), f(h), html.EscapeString(fill), strokeAttr)
	}
	return fmt.Sprintf(`<div class="shape" data-kind="ShapeObject" data-shape="%s" style="%s">`+
		`<svg width="100%%" height="100%%" viewBox="0 0 %s %s">%s</svg></div>`,
		html.EscapeString(shape), objAbsStyle(o.Attrs, ""), f(w), f(h), body)
}

func minFloat(a, b float64) float64 {
	if a < b {
		return a
	}
	return b
}

// objTruthy reports whether a bound checkbox value counts as checked.
//
// Truthiness rule (R2 data binding): empty means unchecked; the words
// "true", "checked", "yes" (plus "y" and "on", case-insensitive) mean
// checked; the words "false", "unchecked", "no" (plus "n" and "off",
// case-insensitive) mean unchecked; any other numeric value is checked
// when it is non-zero ("1" checked, "0" unchecked); any other non-empty
// value is checked. In particular "false" is unchecked even though it is
// non-empty.
func objTruthy(v string) bool {
	s := strings.TrimSpace(v)
	if s == "" {
		return false
	}
	switch strings.ToLower(s) {
	case "true", "checked", "yes", "y", "on":
		return true
	case "false", "unchecked", "no", "n", "off":
		return false
	}
	if f, err := strconv.ParseFloat(s, 64); err == nil {
		return f != 0
	}
	return true
}

// objFieldLookup reads one reference from a row/param map, accepting the
// dotted "Table.Column" form, the short "Column" form and case-insensitive
// matches, mirroring engFieldOf but reporting whether the key exists so
// callers can tell "no such column" apart from "empty value".
func objFieldLookup(name string, m map[string]string) (string, bool) {
	if m == nil {
		return "", false
	}
	if v, ok := m[name]; ok {
		return v, true
	}
	short := name
	if i := strings.LastIndexByte(name, '.'); i >= 0 {
		short = strings.TrimSpace(name[i+1:])
	}
	if v, ok := m[short]; ok {
		return v, true
	}
	for k, v := range m {
		if strings.EqualFold(k, name) || strings.EqualFold(k, short) {
			return v, true
		}
	}
	return "", false
}

// objResolveRef resolves a DataColumn/CheckState reference against the
// current row fields and params. Bracketed expressions ("[Products.Photo]",
// "[Photo]") evaluate through engEvalText; a ref the evaluator leaves
// untouched counts as unresolvable. Bare names ("Products.Photo", "Photo")
// look the column up in fields first, then params. It reports false when
// nothing resolves so callers fall back to the static attributes.
func objResolveRef(ref string, fields, params map[string]string) (string, bool) {
	r := strings.TrimSpace(ref)
	if r == "" {
		return "", false
	}
	if strings.Contains(r, "[") {
		v := engEvalText(r, fields, params)
		if v == r {
			return "", false
		}
		return v, true
	}
	if v, ok := objFieldLookup(r, fields); ok {
		return v, true
	}
	if v, ok := objFieldLookup(r, params); ok {
		return v, true
	}
	return "", false
}

// objCheckedState reports whether a CheckBoxObject renders checked from its
// static attrs alone (no row context). Real .frx files carry
// Checked="false" (often with a DataColumn, which has no row context here,
// so it renders from the static attrs); a static CheckState wins over
// Checked, and both use the objTruthy rule.
func objCheckedState(a []Attr) bool {
	return objCheckedStateBound(a, nil, nil)
}

// objCheckedStateBound resolves the checked state against the current row
// fields and params: a resolvable DataColumn wins, then a resolvable (or
// literal) CheckState, then the static Checked attr.
func objCheckedStateBound(a []Attr, fields, params map[string]string) bool {
	if ref := strings.TrimSpace(attrVal(a, "DataColumn")); ref != "" {
		if v, ok := objResolveRef(ref, fields, params); ok {
			return objTruthy(v)
		}
	}
	if v := strings.TrimSpace(attrVal(a, "CheckState")); v != "" {
		if rv, ok := objResolveRef(v, fields, params); ok {
			return objTruthy(rv)
		}
		return objTruthy(v)
	}
	if v := strings.TrimSpace(attrVal(a, "Checked")); v != "" {
		return objTruthy(v)
	}
	return false
}

// objCheckBoxHTML renders a CheckBoxObject from its static attrs: an
// (unchecked by default) box, with a mark honoring CheckedSymbol
// (Cross/Plus/Fill, default tick).
func objCheckBoxHTML(o Object) string {
	return objCheckBoxHTMLChecked(o, objCheckedState(o.Attrs))
}

// objCheckBoxBoundHTML renders a CheckBoxObject against the current row
// fields and params, preferring DataColumn/CheckState binding over the
// static Checked attr. Unresolvable refs fall back to the static rendering.
func objCheckBoxBoundHTML(o Object, fields, params map[string]string) string {
	return objCheckBoxHTMLChecked(o, objCheckedStateBound(o.Attrs, fields, params))
}

func objCheckBoxHTMLChecked(o Object, checked bool) string {
	mark := ""
	if checked {
		switch strings.TrimSpace(attrVal(o.Attrs, "CheckedSymbol")) {
		case "Cross":
			mark = "&#10005;"
		case "Plus":
			mark = "+"
		case "Fill":
			mark = `<span style="display:block;width:60%;height:60%;background:Black;"></span>`
		default:
			mark = "&#10003;"
		}
	}
	check := html.EscapeString(strings.TrimSpace(attrVal(o.Attrs, "CheckColor")))
	if check == "" {
		check = "Black"
	}
	return fmt.Sprintf(`<div class="checkbox" data-kind="CheckBoxObject" style="%sborder:1px solid Black;display:flex;align-items:center;justify-content:center;color:%s;">%s</div>`,
		objAbsStyle(o.Attrs, ""), check, mark)
}

// objPictureMIME sniffs image magic: the PNG signature vs the JPEG SOI
// marker. Anything else reports "" so the caller keeps the labeled
// placeholder.
func objPictureMIME(b []byte) string {
	if len(b) >= 8 && b[0] == 0x89 && b[1] == 0x50 && b[2] == 0x4E && b[3] == 0x47 &&
		b[4] == 0x0D && b[5] == 0x0A && b[6] == 0x1A && b[7] == 0x0A {
		return "image/png"
	}
	if len(b) >= 3 && b[0] == 0xFF && b[1] == 0xD8 && b[2] == 0xFF {
		return "image/jpeg"
	}
	return ""
}

// objPictureBytes resolves the PictureObject DataColumn against the current
// row fields and params and returns the sniffed raster bytes plus MIME.
// Values may be base64 text (optionally a data: URI) or raw binary;
// anything unresolvable, empty or non-image reports ok=false.
func objPictureBytes(o Object, fields, params map[string]string) ([]byte, string, bool) {
	ref := strings.TrimSpace(attrVal(o.Attrs, "DataColumn"))
	if ref == "" {
		return nil, "", false
	}
	raw, ok := objResolveRef(ref, fields, params)
	if !ok {
		return nil, "", false
	}
	s := strings.TrimSpace(raw)
	if s == "" {
		return nil, "", false
	}
	if strings.HasPrefix(strings.ToLower(s), "data:") {
		if i := strings.Index(s, ";base64,"); i >= 0 {
			s = strings.TrimSpace(s[i+len(";base64,"):])
		}
	}
	compact := s
	for _, w := range []string{" ", "\n", "\r", "\t"} {
		compact = strings.ReplaceAll(compact, w, "")
	}
	var b []byte
	if dec, err := base64.StdEncoding.DecodeString(compact); err == nil {
		b = dec
	} else if dec, err := base64.RawStdEncoding.DecodeString(compact); err == nil {
		b = dec
	} else {
		b = []byte(s)
	}
	mime := objPictureMIME(b)
	if mime == "" {
		return nil, "", false
	}
	return b, mime, true
}

// objPictureBoundHTML renders a PictureObject against the current row fields
// and params: resolvable image bytes become an <img> with a base64 data URI
// (MIME sniffed as PNG vs JPEG); non-image or unresolvable values fall back
// to the labeled placeholder below.
func objPictureBoundHTML(o Object, fields, params map[string]string) string {
	if b, mime, ok := objPictureBytes(o, fields, params); ok {
		return fmt.Sprintf(`<div class="picture" data-kind="PictureObject" style="%s"><img src="data:%s;base64,%s" alt="%s" style="width:100%%;height:100%%;"/></div>`,
			objAbsStyle(o.Attrs, "border:1px solid gray;overflow:hidden;"),
			mime, base64.StdEncoding.EncodeToString(b), html.EscapeString(o.Name))
	}
	return objPictureHTML(o)
}

// objPictureDataURI resolves the DataColumn and returns a data: URI.
// Unresolvable or non-image values report false.
func objPictureDataURI(o Object, fields, params map[string]string) (string, bool) {
	b, mime, ok := objPictureBytes(o, fields, params)
	if !ok {
		return "", false
	}
	return "data:" + mime + ";base64," + base64.StdEncoding.EncodeToString(b), true
}

// objPictureHTML renders a PictureObject without row context as a labeled
// placeholder box, unless engEvalBand pre-resolved a DataURI attr (bound
// image bytes), which renders as an <img> like the bound path below.
// Bound rendering with a dict row goes through objPictureBoundHTML;
// anything it cannot resolve as image bytes lands here.
func objPictureHTML(o Object) string {
	if uri := strings.TrimSpace(attrVal(o.Attrs, "DataURI")); uri != "" {
		return fmt.Sprintf(`<div class="picture" data-kind="PictureObject" style="%s"><img src="%s" alt="%s" style="width:100%%;height:100%%;"/></div>`,
			objAbsStyle(o.Attrs, "border:1px solid gray;overflow:hidden;"),
			html.EscapeString(uri), html.EscapeString(o.Name))
	}
	label := strings.TrimSpace(attrVal(o.Attrs, "DataColumn"))
	if label == "" {
		label = o.Name
	}
	return fmt.Sprintf(`<div class="picture" data-kind="PictureObject" style="%sborder:1px solid gray;overflow:hidden;" title="no data binding yet">%s</div>`,
		objAbsStyle(o.Attrs, ""),
		html.EscapeString("Picture: "+label))
}

// objBarcode2DTypes are Barcode type names needing a 2D renderer, so they
// route to the QR/DataMatrix renderers (or the labeled-box fallback) instead
// of a Code39 rendering.
var objBarcode2DTypes = []string{"qr", "datamatrix", "pdf417", "aztec", "maxicode"}

func objBarcodeIs2D(t string) bool {
	l := strings.ToLower(t)
	for _, s := range objBarcode2DTypes {
		if strings.Contains(l, s) {
			return true
		}
	}
	return false
}

// objBarcodeLinearSVG renders Code128, EAN-13 and UPC-A barcodes with the
// internal/code128 and internal/eanupc encoders. It reports false for every
// other type (including undecodable content and EAN-8) so the caller keeps
// the legacy Code39 rendering.
func objBarcodeLinearSVG(o Object, text string) (string, bool) {
	l := strings.ToLower(strings.TrimSpace(attrVal(o.Attrs, "Barcode")))
	if strings.Contains(l, "128") {
		code, err := code128.Encode(text)
		if err != nil {
			return "", false
		}
		total := code.Modules()
		if total == 0 {
			return "", false
		}
		var sb strings.Builder
		fmt.Fprintf(&sb, `<svg width="100%%" height="100%%" viewBox="0 0 %d 1" preserveAspectRatio="none">`, total)
		x := 0
		for i, w := range code.Widths {
			if i%2 == 0 {
				fmt.Fprintf(&sb, `<rect x="%d" y="0" width="%d" height="1" fill="Black"/>`, x, w)
			}
			x += w
		}
		return sb.String() + `</svg>`, true
	}
	var bars string
	if strings.Contains(l, "upc") {
		code, err := eanupc.EncodeUPCA(text)
		if err != nil {
			return "", false
		}
		bars = code.Bars
	} else if strings.Contains(l, "ean") {
		code, err := eanupc.EncodeEAN13(text)
		if err != nil {
			return "", false
		}
		bars = code.Bars
	} else {
		return "", false
	}
	if bars == "" {
		return "", false
	}
	var sb strings.Builder
	fmt.Fprintf(&sb, `<svg width="100%%" height="100%%" viewBox="0 0 %d 1" preserveAspectRatio="none">`, len(bars))
	run := 0
	flush := func(x int) {
		if run > 0 {
			fmt.Fprintf(&sb, `<rect x="%d" y="0" width="%d" height="1" fill="Black"/>`, x-run, run)
		}
		run = 0
	}
	for i := 0; i < len(bars); i++ {
		if bars[i] == '1' {
			run++
			continue
		}
		flush(i)
	}
	flush(len(bars))
	return sb.String() + `</svg>`, true
}

// objBarcodeHTML renders 1D BarcodeObjects with the built-in Code39 encoder
// below. QR Code and Datamatrix render via their 2D encoders; other 2D types
// keep the labeled-box fallback. ShowText="false" hides the human-readable
// caption.
func objBarcodeHTML(o Object) string {
	if t := strings.TrimSpace(attrVal(o.Attrs, "Barcode")); objBarcodeIs2D(t) {
		return objBarcodeQRHTML(o, t)
	}
	// P5: Barcode.Angle rotation is ignored.
	text := attrVal(o.Attrs, "Text")
	svg, ok := objBarcodeLinearSVG(o, text)
	if !ok {
		pattern := objCode39Pattern(text)
		ratio := 2.0
		if v := strings.TrimSpace(attrVal(o.Attrs, "Barcode.WideBarRatio")); v != "" {
			if f, err := strconv.ParseFloat(v, 64); err == nil {
				ratio = minFloat(3, maxFloat(2, f))
			}
		}
		svg = objCode39SVG(pattern, ratio)
	}
	out := fmt.Sprintf(`<div class="barcode" data-kind="BarcodeObject" style="%s">%s`,
		objAbsStyle(o.Attrs, ""), svg)
	if !strings.EqualFold(strings.TrimSpace(attrVal(o.Attrs, "ShowText")), "false") && text != "" {
		out += fmt.Sprintf(`<div class="barcode-text">%s</div>`, html.EscapeString(text))
	}
	return out + `</div>`
}

func maxFloat(a, b float64) float64 {
	if a > b {
		return a
	}
	return b
}

// objBarcodeQRHTML renders QR Code, Datamatrix, PDF417 and Aztec barcodes
// with the internal/qr, internal/datamatrix, internal/pdf417 and
// internal/aztec encoders. QR honors Barcode.ErrorCorrection (L/M/Q/H,
// default L like FastReport); Datamatrix honors Barcode.SymbolSize (e.g.
// "Size24x24", default smallest that fits); PDF417 reads ErrorCorrection as
// a 0..8 level ("Auto" means 2); Aztec reads it as a minimum ECC percent
// ("Auto" means 33). Other 2D types keep the labeled-box fallback.
func objBarcodeQRHTML(o Object, kind string) string {
	l := strings.ToLower(kind)
	if strings.Contains(l, "datamatrix") || strings.Contains(l, "data matrix") {
		return objBarcodeDataMatrixHTML(o)
	}
	if strings.Contains(l, "pdf417") || strings.Contains(l, "pdf-417") {
		return objBarcodePDF417HTML(o)
	}
	if strings.Contains(l, "aztec") {
		return objBarcodeAztecHTML(o)
	}
	if strings.Contains(l, "maxicode") || strings.Contains(l, "maxi-code") {
		return objBarcodeMaxiHTML(o)
	}
	if !strings.Contains(l, "qr") {
		return objFallbackHTML(o)
	}
	code, err := qr.QrEncodeWithLevel([]byte(attrVal(o.Attrs, "Text")), objQrLevel(o))
	if err != nil {
		return objFallbackHTML(o)
	}
	out := fmt.Sprintf(`<div class="barcode" data-kind="BarcodeObject" style="%s">%s`,
		objAbsStyle(o.Attrs, ""), code.SVG())
	if !strings.EqualFold(strings.TrimSpace(attrVal(o.Attrs, "ShowText")), "false") {
		if text := attrVal(o.Attrs, "Text"); text != "" {
			out += fmt.Sprintf(`<div class="barcode-text">%s</div>`, html.EscapeString(text))
		}
	}
	return out + `</div>`
}

// objPdf417Level reads Barcode.ErrorCorrection as a 0..8 security level;
// "Auto", empty or unparseable means the default level 2. Out-of-range or
// non-numeric values report ok=false so the caller keeps the fallback.
func objPdf417Level(o Object) (int, bool) {
	v := strings.TrimSpace(attrVal(o.Attrs, "Barcode.ErrorCorrection"))
	if v == "" || strings.EqualFold(v, "Auto") {
		return 2, true
	}
	n, err := strconv.Atoi(v)
	if err != nil || n < 0 || n > 8 {
		return 0, false
	}
	return n, true
}

// objBarcodePDF417HTML renders PDF417 barcodes with the internal/pdf417
// encoder. Oversize data or a bad level keeps the labeled-box fallback.
func objBarcodePDF417HTML(o Object) string {
	lvl, ok := objPdf417Level(o)
	if !ok {
		return objFallbackHTML(o)
	}
	code, err := pdf417.Encode([]byte(attrVal(o.Attrs, "Text")), lvl)
	if err != nil {
		return objFallbackHTML(o)
	}
	out := fmt.Sprintf(`<div class="barcode" data-kind="BarcodeObject" style="%s">%s`,
		objAbsStyle(o.Attrs, ""), code.SVG())
	if !strings.EqualFold(strings.TrimSpace(attrVal(o.Attrs, "ShowText")), "false") {
		if text := attrVal(o.Attrs, "Text"); text != "" {
			out += fmt.Sprintf(`<div class="barcode-text">%s</div>`, html.EscapeString(text))
		}
	}
	return out + `</div>`
}

// objAztecECC reads Barcode.ErrorCorrection as a minimum ECC percent;
// "Auto", empty or unparseable means the default 33. Values outside 1..100
// report ok=false so the caller keeps the fallback.
func objAztecECC(o Object) (int, bool) {
	v := strings.TrimSpace(attrVal(o.Attrs, "Barcode.ErrorCorrection"))
	if v == "" || strings.EqualFold(v, "Auto") {
		return 33, true
	}
	n, err := strconv.Atoi(v)
	if err != nil || n < 1 || n > 100 {
		return 0, false
	}
	return n, true
}

// objBarcodeAztecHTML renders Aztec barcodes with the internal/aztec
// encoder. Oversize data or a bad percent keeps the labeled-box fallback.
func objBarcodeAztecHTML(o Object) string {
	ecc, ok := objAztecECC(o)
	if !ok {
		return objFallbackHTML(o)
	}
	code, err := aztec.Encode([]byte(attrVal(o.Attrs, "Text")), ecc, 0)
	if err != nil {
		return objFallbackHTML(o)
	}
	out := fmt.Sprintf(`<div class="barcode" data-kind="BarcodeObject" style="%s">%s`,
		objAbsStyle(o.Attrs, ""), code.SVG())
	if !strings.EqualFold(strings.TrimSpace(attrVal(o.Attrs, "ShowText")), "false") {
		if text := attrVal(o.Attrs, "Text"); text != "" {
			out += fmt.Sprintf(`<div class="barcode-text">%s</div>`, html.EscapeString(text))
		}
	}
	return out + `</div>`
}

// objMaxiMode reads Barcode.Mode as a 2..6 symbol mode, defaulting to 4
// (general data) when empty or "Auto". Out-of-range or non-numeric values
// report ok=false so the caller keeps the fallback.
func objMaxiMode(o Object) (int, bool) {
	v := strings.TrimSpace(attrVal(o.Attrs, "Barcode.Mode"))
	if v == "" || strings.EqualFold(v, "Auto") {
		return 4, true
	}
	n, err := strconv.Atoi(v)
	if err != nil || n < 2 || n > 6 {
		return 0, false
	}
	return n, true
}

// objBarcodeMaxiHTML renders MaxiCode barcodes with the internal/maxicode
// encoder. Oversize data or a bad mode keeps the labeled-box fallback.
func objBarcodeMaxiHTML(o Object) string {
	mode, ok := objMaxiMode(o)
	if !ok {
		return objFallbackHTML(o)
	}
	code, err := maxicode.Encode(mode, []byte(attrVal(o.Attrs, "Text")))
	if err != nil {
		return objFallbackHTML(o)
	}
	out := fmt.Sprintf(`<div class="barcode" data-kind="BarcodeObject" style="%s">%s`,
		objAbsStyle(o.Attrs, ""), code.SVG())
	if !strings.EqualFold(strings.TrimSpace(attrVal(o.Attrs, "ShowText")), "false") {
		if text := attrVal(o.Attrs, "Text"); text != "" {
			out += fmt.Sprintf(`<div class="barcode-text">%s</div>`, html.EscapeString(text))
		}
	}
	return out + `</div>`
}

// objBarcodeDataMatrixHTML renders Datamatrix barcodes with the
// internal/datamatrix ECC200 encoder. Barcode.SymbolSize accepts FastReport
// names like "Size24x24" (edge length); empty or unparseable means the
// smallest symbol that fits. Oversize data or an unknown size keeps the
// labeled-box fallback instead of an empty rendering.
func objBarcodeDataMatrixHTML(o Object) string {
	code, err := datamatrix.Encode([]byte(attrVal(o.Attrs, "Text")), objDataMatrixSize(attrVal(o.Attrs, "Barcode.SymbolSize")))
	if err != nil {
		return objFallbackHTML(o)
	}
	out := fmt.Sprintf(`<div class="barcode" data-kind="BarcodeObject" style="%s">%s`,
		objAbsStyle(o.Attrs, ""), code.SVG())
	if !strings.EqualFold(strings.TrimSpace(attrVal(o.Attrs, "ShowText")), "false") {
		if text := attrVal(o.Attrs, "Text"); text != "" {
			out += fmt.Sprintf(`<div class="barcode-text">%s</div>`, html.EscapeString(text))
		}
	}
	return out + `</div>`
}

// objQrLevel reads Barcode.ErrorCorrection (L/M/Q/H, default L like FastReport).
func objQrLevel(o Object) qr.QrLevel {
	switch strings.ToUpper(strings.TrimSpace(attrVal(o.Attrs, "Barcode.ErrorCorrection"))) {
	case "M":
		return qr.QrM
	case "Q":
		return qr.QrQ
	case "H":
		return qr.QrH
	default:
		return qr.QrL
	}
}

// objDataMatrixSize parses FastReport Barcode.SymbolSize names ("Size24x24")
// into a symbol edge length. Anything else yields 0 (smallest that fits).
func objDataMatrixSize(v string) int {
	s := strings.TrimSpace(v)
	s = strings.TrimPrefix(strings.ToLower(s), "size")
	parts := strings.SplitN(s, "x", 2)
	if len(parts) != 2 {
		return 0
	}
	n, err := strconv.Atoi(strings.TrimSpace(parts[0]))
	if err != nil || n <= 0 {
		return 0
	}
	m, err := strconv.Atoi(strings.TrimSpace(parts[1]))
	if err != nil || m != n {
		return 0
	}
	return n
}

// objCode39Data maps each encodable character to its 9-element bar/space
// pattern. Digit semantics (from FastReport LinearBarcodeBase.OneBarProps):
// 0 = narrow space, 1 = wide space, 5 = narrow bar, 6 = wide bar.
// Table ported from FastReport.Base/Barcode/Barcode39.cs tabelle_39.
var objCode39Data = map[string]string{
	"0": "505160605", "1": "605150506", "2": "506150506", "3": "606150505",
	"4": "505160506", "5": "605160505", "6": "506160505", "7": "505150606",
	"8": "605150605", "9": "506150605",
	"A": "605051506", "B": "506051506", "C": "606051505", "D": "505061506",
	"E": "605061505", "F": "506061505", "G": "505051606", "H": "605051605",
	"I": "506051605", "J": "505061605", "K": "605050516", "L": "506050516",
	"M": "606050515", "N": "505060516", "O": "605060515", "P": "506060515",
	"Q": "505050616", "R": "605050615", "S": "506050615", "T": "505060615",
	"U": "615050506", "V": "516050506", "W": "616050505", "X": "515060506",
	"Y": "615060505", "Z": "516060505",
	"-": "515050606", ".": "615050605", " ": "516050605", "*": "515060605",
	"$": "515151505", "/": "515150515", "+": "515051515", "%": "505151515",
}

// objCode39Sentinel is the start/stop "*" character pattern.
const objCode39Sentinel = "515060605"

// objCode39Pattern encodes text the way FastReport Barcode39.GetPattern
// does: start sentinel + gap, one 9-element group + gap per encodable
// character, stop sentinel. Characters outside the table are skipped, and no
// Mod43 checksum is appended (CalcCheckSum is opt-in in FastReport too).
// Pattern length is 10 per encodable character plus 19 (10 start, 9 stop).
func objCode39Pattern(text string) string {
	var sb strings.Builder
	sb.WriteString(objCode39Sentinel + "0")
	for _, r := range text {
		if d, ok := objCode39Data[string(r)]; ok {
			sb.WriteString(d + "0")
		}
	}
	sb.WriteString(objCode39Sentinel)
	return sb.String()
}

// objCode39SVG lays the digit pattern out as SVG bars: narrow elements are 1
// module wide, wide elements ratio modules. Every character contributes 5
// bars, so the rect count is 5 per encodable character plus 10 sentinels.
func objCode39SVG(pattern string, ratio float64) string {
	widthOf := func(c byte) float64 {
		if c == '1' || c == '6' {
			return ratio
		}
		return 1
	}
	total := 0.0
	for i := 0; i < len(pattern); i++ {
		total += widthOf(pattern[i])
	}
	var sb strings.Builder
	fmt.Fprintf(&sb, `<svg width="100%%" height="100%%" viewBox="0 0 %g 1" preserveAspectRatio="none">`,
		total)
	x := 0.0
	for i := 0; i < len(pattern); i++ {
		w := widthOf(pattern[i])
		if pattern[i] == '5' || pattern[i] == '6' {
			fmt.Fprintf(&sb, `<rect x="%g" y="0" width="%g" height="1" fill="Black"/>`, x, w)
		}
		x += w
	}
	sb.WriteString(`</svg>`)
	return sb.String()
}
