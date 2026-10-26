// Package pdf writes minimal PDF 1.4 documents. Base-14 fonts (Helvetica,
// Courier and their bold faces) need no embedded data but render ASCII
// only (see fontmetrics). FntEmbeddedFont embeds one subset TrueType font
// as a custom 8-bit encoding, which is how Thai script is served.
package pdf

import (
	"bytes"
	"fmt"
	"sort"
	"strings"

	"report-next/internal/fontmetrics"
)

// Face re-exports the measurable base-14 faces.
type Face = fontmetrics.Face

const (
	Helvetica     = fontmetrics.Helvetica
	HelveticaBold = fontmetrics.HelveticaBold
	Courier       = fontmetrics.Courier
	CourierBold   = fontmetrics.CourierBold
)

type textOp struct {
	x, y float64
	size float64
	face Face
	s    string
	// fntFont selects an embedded font by FntAddEmbeddedFont handle;
	// negative means the base-14 face above.
	fntFont int
}

// FntEmbeddedFont is one subset TrueType font ready to embed: the subset
// sfnt bytes plus the 8-bit code assignment (FirstChar..LastChar) with its
// /Widths, /Differences names and /ToUnicode mapping. Metrics are in
// thousandths of an em.
type FntEmbeddedFont struct {
	Tag  string // subset prefix, 6 uppercase letters
	Name string // PostScript name without the prefix
	Data []byte // standalone subset sfnt

	FirstChar int
	LastChar  int
	Widths    []int // per code FirstChar..LastChar
	Names     map[int]string
	Unicodes  map[int]rune
	ToUnicode string // /ToUnicode CMap text mapping codes back to unicode

	Ascent      float64
	Descent     float64
	CapHeight   float64
	BBox        [4]float64
	ItalicAngle float64
}

type rectOp struct {
	x, y, w, h float64
	r, g, b    float64
}

// pathSeg is one subpath segment: "m" (move, 1 point), "l" (line, 1
// point) or "c" (bezier, 3 points).
type pathSeg struct {
	op  byte
	pts [3][2]float64
}

// pathOp is one stroked/filled path: outlines, polylines, polygons and
// bezier ellipses. Fill uses the fr/fg/fb color, stroke the sr/sg/sb
// color with line width lineW.
type pathOp struct {
	segs       []pathSeg
	close      bool
	fill       bool
	stroke     bool
	lineW      float64
	fr, fg, fb float64
	sr, sg, sb float64
}

type page struct {
	w, h  float64 // points
	texts []textOp
	rects []rectOp
	paths []pathOp
}

// Doc is one PDF document. Coordinates are PDF points with the origin at
// the bottom-left of each page.
type Doc struct {
	pages  []page
	embeds []*FntEmbeddedFont
}

// New returns an empty document.
func New() *Doc { return &Doc{} }

// AddPage appends a blank page of w by h points.
func (d *Doc) AddPage(w, h float64) { d.pages = append(d.pages, page{w: w, h: h}) }

// Pages returns the page count.
func (d *Doc) Pages() int { return len(d.pages) }

// Text draws s on page p. Only ASCII is supported; anything else is an error.
func (d *Doc) Text(p int, x, y, size float64, face Face, s string) error {
	if p < 0 || p >= len(d.pages) {
		return fmt.Errorf("pdf: page %d out of range", p)
	}
	if !fontmetrics.ASCIIOnly(s) {
		return fmt.Errorf("pdf: non-ascii text unsupported: %q", s)
	}
	d.pages[p].texts = append(d.pages[p].texts, textOp{x: x, y: y, size: size, face: face, s: s, fntFont: -1})
	return nil
}

// FntAddEmbeddedFont registers f for use with FntText and returns its
// handle. Widths must cover every code from FirstChar to LastChar.
func (d *Doc) FntAddEmbeddedFont(f *FntEmbeddedFont) (int, error) {
	if f == nil || len(f.Data) == 0 {
		return -1, fmt.Errorf("pdf: embedded font has no data")
	}
	if f.FirstChar < 0 || f.LastChar > 255 || f.FirstChar > f.LastChar {
		return -1, fmt.Errorf("pdf: embedded font range %d..%d invalid", f.FirstChar, f.LastChar)
	}
	if len(f.Widths) != f.LastChar-f.FirstChar+1 {
		return -1, fmt.Errorf("pdf: embedded font widths cover %d codes, want %d",
			len(f.Widths), f.LastChar-f.FirstChar+1)
	}
	d.embeds = append(d.embeds, f)
	return len(d.embeds) - 1, nil
}

// FntText draws already-encoded subset bytes (see FntEmbeddedFont) on page
// p with the embedded font handle from FntAddEmbeddedFont.
func (d *Doc) FntText(p, font int, x, y, size float64, encoded string) error {
	if p < 0 || p >= len(d.pages) {
		return fmt.Errorf("pdf: page %d out of range", p)
	}
	if font < 0 || font >= len(d.embeds) {
		return fmt.Errorf("pdf: embedded font %d out of range", font)
	}
	f := d.embeds[font]
	for i := 0; i < len(encoded); i++ {
		if c := int(encoded[i]); c < f.FirstChar || c > f.LastChar {
			return fmt.Errorf("pdf: byte %#02x outside embedded range", c)
		}
	}
	d.pages[p].texts = append(d.pages[p].texts, textOp{x: x, y: y, size: size, s: encoded, fntFont: font})
	return nil
}

// FillRect fills a gray rectangle (0 black .. 1 white) on page p.
func (d *Doc) FillRect(p int, x, y, w, h, gray float64) error {
	return d.FillRectRGB(p, x, y, w, h, gray, gray, gray)
}

// clamp01 pins a color channel to 0..1.
func clamp01(v float64) float64 {
	if v < 0 {
		return 0
	}
	if v > 1 {
		return 1
	}
	return v
}

// FillRectRGB fills an RGB rectangle (0..1 per channel) on page p.
func (d *Doc) FillRectRGB(p int, x, y, w, h, r, g, b float64) error {
	if p < 0 || p >= len(d.pages) {
		return fmt.Errorf("pdf: page %d out of range", p)
	}
	d.pages[p].rects = append(d.pages[p].rects, rectOp{x, y, w, h, clamp01(r), clamp01(g), clamp01(b)})
	return nil
}

// addPath validates the page and appends a path op.
func (d *Doc) addPath(p int, pa pathOp) error {
	if p < 0 || p >= len(d.pages) {
		return fmt.Errorf("pdf: page %d out of range", p)
	}
	if len(pa.segs) == 0 {
		return fmt.Errorf("pdf: empty path")
	}
	d.pages[p].paths = append(d.pages[p].paths, pa)
	return nil
}

// StrokeLine strokes one segment with line width lineW and an RGB color.
func (d *Doc) StrokeLine(p int, x1, y1, x2, y2, lineW, r, g, b float64) error {
	return d.addPath(p, pathOp{
		segs: []pathSeg{
			{op: 'm', pts: [3][2]float64{{x1, y1}}},
			{op: 'l', pts: [3][2]float64{{x2, y2}}},
		},
		stroke: true, lineW: lineW,
		sr: clamp01(r), sg: clamp01(g), sb: clamp01(b),
	})
}

// StrokeRect strokes a rectangle outline with line width lineW.
func (d *Doc) StrokeRect(p int, x, y, w, h, lineW, r, g, b float64) error {
	return d.Poly(p, [][2]float64{{x, y}, {x + w, y}, {x + w, y + h}, {x, y + h}},
		true, false, true, lineW, 0, 0, 0, r, g, b)
}

// Poly draws a polyline/polygon through pts. Close joins the last point
// back to the first. Fill and stroke may be combined (with one color
// each); both false is an error.
func (d *Doc) Poly(p int, pts [][2]float64, close, fill, stroke bool,
	lineW, fr, fg, fb, sr, sg, sb float64) error {
	if len(pts) == 0 {
		return fmt.Errorf("pdf: empty poly")
	}
	if !fill && !stroke {
		return fmt.Errorf("pdf: poly needs fill or stroke")
	}
	segs := make([]pathSeg, 0, len(pts))
	for i, pt := range pts {
		op := byte('l')
		if i == 0 {
			op = 'm'
		}
		segs = append(segs, pathSeg{op: op, pts: [3][2]float64{pt}})
	}
	return d.addPath(p, pathOp{
		segs: segs, close: close, fill: fill, stroke: stroke, lineW: lineW,
		fr: clamp01(fr), fg: clamp01(fg), fb: clamp01(fb),
		sr: clamp01(sr), sg: clamp01(sg), sb: clamp01(sb),
	})
}

// Ellipse draws an axis-aligned ellipse from four bezier curves (kappa
// 0.5523). Fill and stroke may be combined with one color each.
func (d *Doc) Ellipse(p int, cx, cy, rx, ry float64, fill, stroke bool,
	lineW, fr, fg, fb, sr, sg, sb float64) error {
	if rx <= 0 || ry <= 0 {
		return fmt.Errorf("pdf: ellipse needs positive radii")
	}
	if !fill && !stroke {
		return fmt.Errorf("pdf: ellipse needs fill or stroke")
	}
	const k = 0.5522847498
	kx, ky := k*rx, k*ry
	segs := []pathSeg{
		{op: 'm', pts: [3][2]float64{{cx + rx, cy}}},
		{op: 'c', pts: [3][2]float64{{cx + rx, cy + ky}, {cx + kx, cy + ry}, {cx, cy + ry}}},
		{op: 'c', pts: [3][2]float64{{cx - kx, cy + ry}, {cx - rx, cy + ky}, {cx - rx, cy}}},
		{op: 'c', pts: [3][2]float64{{cx - rx, cy - ky}, {cx - kx, cy - ry}, {cx, cy - ry}}},
		{op: 'c', pts: [3][2]float64{{cx + kx, cy - ry}, {cx + rx, cy - ky}, {cx + rx, cy}}},
	}
	return d.addPath(p, pathOp{
		segs: segs, close: true, fill: fill, stroke: stroke, lineW: lineW,
		fr: clamp01(fr), fg: clamp01(fg), fb: clamp01(fb),
		sr: clamp01(sr), sg: clamp01(sg), sb: clamp01(sb),
	})
}

func escapeWinAnsi(s string) string {
	var sb strings.Builder
	for i := 0; i < len(s); i++ {
		switch s[i] {
		case '(', ')', '\\':
			sb.WriteByte('\\')
			sb.WriteByte(s[i])
		default:
			sb.WriteByte(s[i])
		}
	}
	return sb.String()
}

// Bytes serializes the document with a valid cross-reference table.
func (d *Doc) Bytes() []byte {
	var buf bytes.Buffer
	offsets := []int{}
	buf.WriteString("%PDF-1.4\n")

	// Object numbering: 1 catalog, 2 pages, then per page (page, content),
	// then one font object per used face in first-use order.
	next := 3
	pageObjs := make([]int, len(d.pages))
	contentObjs := make([]int, len(d.pages))
	for i := range d.pages {
		pageObjs[i] = next
		next++
		contentObjs[i] = next
		next++
	}
	faceNum := map[Face]int{}
	var faceOrder []Face
	embUsed := map[int]bool{}
	for _, pg := range d.pages {
		for _, t := range pg.texts {
			if t.fntFont >= 0 {
				embUsed[t.fntFont] = true
				continue
			}
			if _, ok := faceNum[t.face]; !ok {
				faceNum[t.face] = next
				next++
				faceOrder = append(faceOrder, t.face)
			}
		}
	}
	var embOrder []int
	for h := range embUsed {
		embOrder = append(embOrder, h)
	}
	sort.Ints(embOrder)
	embNum := map[int]int{}
	for i, h := range embOrder {
		embNum[h] = i + 1 // /Fe1, /Fe2, ...
	}
	embObj := map[int][4]int{}
	for _, h := range embOrder {
		embObj[h] = [4]int{next, next + 1, next + 2, next + 3}
		next += 4
	}
	total := next - 1

	emit := func(num int, body string) {
		offsets = append(offsets, buf.Len())
		fmt.Fprintf(&buf, "%d 0 obj\n%s\nendobj\n", num, body)
	}
	emit(1, "<< /Type /Catalog /Pages 2 0 R >>")
	kids := make([]string, len(d.pages))
	for i := range d.pages {
		kids[i] = fmt.Sprintf("%d 0 R", pageObjs[i])
	}
	emit(2, fmt.Sprintf("<< /Type /Pages /Kids [%s] /Count %d >>", strings.Join(kids, " "), len(d.pages)))
	for i, pg := range d.pages {
		var fonts []string
		for _, f := range faceOrder {
			fonts = append(fonts, fmt.Sprintf("/F%d %d 0 R", faceIndex(faceOrder, f), faceNum[f]))
		}
		for _, h := range embOrder {
			if fntPageUsesEmbed(pg, h) {
				fonts = append(fonts, fmt.Sprintf("/Fe%d %d 0 R", embNum[h], embObj[h][0]))
			}
		}
		emit(pageObjs[i], fmt.Sprintf("<< /Type /Page /Parent 2 0 R /MediaBox [0 0 %.2f %.2f] /Resources << /Font << %s >> >> /Contents %d 0 R >>",
			pg.w, pg.h, strings.Join(fonts, " "), contentObjs[i]))
		emit(contentObjs[i], sprintfStream(pageContent(pg, faceOrder, embNum)))
	}
	for _, f := range faceOrder {
		emit(faceNum[f], fmt.Sprintf("<< /Type /Font /Subtype /Type1 /BaseFont /%s /Encoding /WinAnsiEncoding >>", f.BaseName()))
	}
	for _, h := range embOrder {
		f := d.embeds[h]
		o := embObj[h]
		base := fmt.Sprintf("%s+%s", f.Tag, f.Name)
		emit(o[0], fmt.Sprintf("<< /Type /Font /Subtype /TrueType /BaseFont /%s /FirstChar %d /LastChar %d /Widths [%s] /Encoding << /Type /Encoding /Differences [%d %s] >> /FontDescriptor %d 0 R /ToUnicode %d 0 R >>",
			base, f.FirstChar, f.LastChar, fntWidths(f.Widths), f.FirstChar, fntDifferences(f), o[1], o[3]))
		flags := 32
		if f.ItalicAngle != 0 {
			flags |= 64
		}
		emit(o[1], fmt.Sprintf("<< /Type /FontDescriptor /FontName /%s /Flags %d /FontBBox [%.2f %.2f %.2f %.2f] /ItalicAngle %.2f /Ascent %.2f /Descent %.2f /CapHeight %.2f /StemV 80 /MissingWidth 0 /FontFile2 %d 0 R >>",
			base, flags, f.BBox[0], f.BBox[1], f.BBox[2], f.BBox[3], f.ItalicAngle, f.Ascent, f.Descent, f.CapHeight, o[2]))
		emit(o[2], fntStream(f.Data))
		emit(o[3], fntStream([]byte(f.ToUnicode)))
	}

	xref := buf.Len()
	fmt.Fprintf(&buf, "xref\n0 %d\n", total+1)
	fmt.Fprintf(&buf, "0000000000 65535 f \n")
	for n := 1; n <= total; n++ {
		fmt.Fprintf(&buf, "%010d 00000 n \n", offsets[n-1])
	}
	fmt.Fprintf(&buf, "trailer\n<< /Size %d /Root 1 0 R >>\nstartxref\n%d\n%%%%EOF\n", total+1, xref)
	return buf.Bytes()
}

func faceIndex(order []Face, f Face) int {
	for i, g := range order {
		if g == f {
			return i + 1
		}
	}
	return 0
}

func pageContent(pg page, order []Face, embNum map[int]int) string {
	var sb strings.Builder
	for _, r := range pg.rects {
		fmt.Fprintf(&sb, "%.2f %.2f %.2f rg %.2f %.2f %.2f %.2f re f\n",
			r.r, r.g, r.b, r.x, r.y, r.w, r.h)
	}
	for _, pa := range pg.paths {
		if pa.lineW > 0 {
			fmt.Fprintf(&sb, "%.2f w\n", pa.lineW)
		}
		if pa.fill {
			fmt.Fprintf(&sb, "%.2f %.2f %.2f rg\n", pa.fr, pa.fg, pa.fb)
		}
		if pa.stroke {
			fmt.Fprintf(&sb, "%.2f %.2f %.2f RG\n", pa.sr, pa.sg, pa.sb)
		}
		for _, s := range pa.segs {
			switch s.op {
			case 'm':
				fmt.Fprintf(&sb, "%.2f %.2f m\n", s.pts[0][0], s.pts[0][1])
			case 'l':
				fmt.Fprintf(&sb, "%.2f %.2f l\n", s.pts[0][0], s.pts[0][1])
			case 'c':
				fmt.Fprintf(&sb, "%.2f %.2f %.2f %.2f %.2f %.2f c\n",
					s.pts[0][0], s.pts[0][1], s.pts[1][0], s.pts[1][1], s.pts[2][0], s.pts[2][1])
			}
		}
		if pa.close {
			sb.WriteString("h\n")
		}
		switch {
		case pa.fill && pa.stroke:
			sb.WriteString("B\n")
		case pa.fill:
			sb.WriteString("f\n")
		default:
			sb.WriteString("S\n")
		}
	}
	for _, t := range pg.texts {
		if t.fntFont >= 0 {
			fmt.Fprintf(&sb, "BT /Fe%d %.2f Tf %.2f %.2f Td <%s> Tj ET\n",
				embNum[t.fntFont], t.size, t.x, t.y, fntHex(t.s))
			continue
		}
		fmt.Fprintf(&sb, "BT /F%d %.2f Tf %.2f %.2f Td (%s) Tj ET\n",
			faceIndex(order, t.face), t.size, t.x, t.y, escapeWinAnsi(t.s))
	}
	return sb.String()
}

// fntPageUsesEmbed reports whether page pg draws with embedded font handle h.
func fntPageUsesEmbed(pg page, h int) bool {
	for _, t := range pg.texts {
		if t.fntFont == h {
			return true
		}
	}
	return false
}

// fntHex renders bytes as uppercase PDF hex.
func fntHex(s string) string {
	const digits = "0123456789ABCDEF"
	out := make([]byte, 2*len(s))
	for i := 0; i < len(s); i++ {
		out[2*i] = digits[s[i]>>4]
		out[2*i+1] = digits[s[i]&0x0F]
	}
	return string(out)
}

// fntWidths renders the /Widths array.
func fntWidths(w []int) string {
	parts := make([]string, len(w))
	for i, v := range w {
		parts[i] = fmt.Sprintf("%d", v)
	}
	return strings.Join(parts, " ")
}

// fntDifferences renders the glyph names for codes FirstChar..LastChar,
// falling back to /.notdef for codes with no name.
func fntDifferences(f *FntEmbeddedFont) string {
	parts := make([]string, 0, f.LastChar-f.FirstChar+1)
	for c := f.FirstChar; c <= f.LastChar; c++ {
		name := f.Names[c]
		if name == "" {
			name = ".notdef"
		}
		parts = append(parts, "/"+name)
	}
	return strings.Join(parts, " ")
}

// fntStream wraps binary data in a stream object body with a trailing
// newline so a data byte run can never merge with "endstream".
func fntStream(data []byte) string {
	return fmt.Sprintf("<< /Length %d >>\nstream\n%s\nendstream", len(data)+1, string(data))
}

func sprintfStream(content string) string {
	return fmt.Sprintf("<< /Length %d >>\nstream\n%sendstream", len(content), content)
}
