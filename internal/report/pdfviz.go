package report

import (
	"math"
	"strings"

	"report-next/internal/fontmetrics"
	"report-next/internal/pdf"
)

// PDF drawing for Gauge/MatrixObject/SubreportObject, mirroring the HTML
// preview geometry. Bar gauges draw track + filled fraction + 5 ticks;
// radial gauges draw a thick semicircle dial (chord segments), a needle at
// angle pi*(1-frac), a hub and the label. MatrixObject draws its static
// template grid through pdfGrid (dynamic pivot aggregation stays in the
// engine, whose synthesized TableObject also lands here). SubreportObject
// draws a gray placeholder outline with its ASCII label.

// pdfGauge draws a gauge object at its own geometry.
func pdfGauge(doc *pdf.Doc, pg int, pageH float64, o Object) (int, error) {
	l, t, w, h := objRectOf(o.Attrs)
	return pdfGaugeRect(doc, pg, l*pxToPt, pageH-(t+h)*pxToPt, w*pxToPt, h*pxToPt, o)
}

// pdfGaugeRect draws a gauge at an explicit box (PDF points, y0 at the box
// bottom). Unknown gauge kinds draw nothing, like the HTML labeled-box
// fallback draws no PDF geometry. It returns the skipped-label count.
func pdfGaugeRect(doc *pdf.Doc, pg int, x, y0, pw, ph float64, o Object) (int, error) {
	if !vizGaugeKinds[o.Kind] {
		return 0, nil
	}
	if pw <= 0 || ph <= 0 {
		return 0, nil
	}
	frac := vizGaugeFraction(o.Attrs)
	fr, fg, fb, ok := pdfColor(vizGaugeFill(o.Attrs))
	if !ok {
		fr, fg, fb = 0.5, 0.5, 0.5
	}
	label := vizGaugeLabel(o.Attrs)
	size, lface := pdfLabelFont(o.Attrs)
	capH := 0.0
	if label != "" && fontmetrics.ASCIIOnly(label) && ph > size {
		capH = size
	}
	skipped := 0
	if label != "" && !fontmetrics.ASCIIOnly(label) {
		skipped = 1
	}
	if o.Kind == "RadialGauge" {
		return skipped, pdfGaugeRadial(doc, pg, x, y0, pw, ph-capH, capH, size, lface, label, frac, fr, fg, fb)
	}
	return skipped, pdfGaugeBar(doc, pg, x, y0, pw, ph-capH, capH, size, lface, label, frac, fr, fg, fb)
}

// pdfLabelFont reads Label.Font ("Name, 9pt, ..."), defaulting like text.
func pdfLabelFont(a []Attr) (float64, fontmetrics.Face) {
	if v := strings.TrimSpace(attrVal(a, "Label.Font")); v != "" {
		return pdfFont([]Attr{{"Font", v}})
	}
	return pdfFont(a)
}

// pdfGaugeBar draws a horizontal bar gauge: track, filled fraction, 5 major
// ticks and a centered label below the track.
func pdfGaugeBar(doc *pdf.Doc, pg int, x, y0, pw, ph, capH, size float64, face fontmetrics.Face, label string, frac, fr, fg, fb float64) error {
	if ph <= 0 {
		return nil
	}
	gy := y0 + capH
	if err := doc.FillRectRGB(pg, x, gy, pw, ph, 0.82745, 0.82745, 0.82745); err != nil {
		return err
	}
	if frac > 0 {
		if err := doc.FillRectRGB(pg, x, gy, pw*frac, ph, fr, fg, fb); err != nil {
			return err
		}
	}
	for i := 0; i <= 4; i++ {
		tx := x + pw*float64(i)/4
		if err := doc.StrokeLine(pg, tx, gy, tx, gy+ph, 0.5, 0, 0, 0); err != nil {
			return err
		}
	}
	if label != "" && capH > 0 {
		if w := face.Width(label, size); w < pw {
			x += (pw - w) / 2
		}
		return doc.Text(pg, x, y0, size, face, label)
	}
	return nil
}

// pdfGaugeRadial draws a semicircle dial: a thick gray chord-arc, a needle
// at angle pi*(1-frac) on the r=38 circle of the 100x60 HTML viewBox, a hub
// and the label below the dial.
func pdfGaugeRadial(doc *pdf.Doc, pg int, x, y0, pw, dialH, capH, size float64, face fontmetrics.Face, label string, frac, fr, fg, fb float64) error {
	if dialH <= 0 {
		return nil
	}
	s := math.Min(pw/100, dialH/60)
	ox := x + (pw-100*s)/2
	// Y maps the SVG y-down coordinate into the dial strip above the label.
	Y := func(ty float64) float64 { return y0 + capH + (dialH - ty*s) }
	X := func(tx float64) float64 { return ox + tx*s }
	cx, cy := X(50), Y(52)
	// Background dial arc as 24 chord segments, 8 units thick like HTML.
	var pts [][2]float64
	for i := 0; i <= 24; i++ {
		a := math.Pi * float64(i) / 24
		pts = append(pts, [2]float64{cx + 38*s*math.Cos(a), cy + 38*s*math.Sin(a)})
	}
	if err := doc.Poly(pg, pts, false, false, true, 8*s,
		0, 0, 0, 0.82745, 0.82745, 0.82745); err != nil {
		return err
	}
	ang := math.Pi * (1 - frac)
	if err := doc.StrokeLine(pg, cx, cy, cx+38*s*math.Cos(ang), cy+38*s*math.Sin(ang),
		3*s, fr, fg, fb); err != nil {
		return err
	}
	if err := doc.Ellipse(pg, cx, cy, 4*s, 4*s, true, false, 0, 0, 0, 0, 0, 0, 0); err != nil {
		return err
	}
	if label != "" && capH > 0 {
		lx := x
		if w := face.Width(label, size); w < pw {
			lx = x + (pw-w)/2
		}
		return doc.Text(pg, lx, y0, size, face, label)
	}
	return nil
}

// pdfMatrix draws a MatrixObject's static template grid through pdfGrid.
// Matrices without a template grid draw nothing (the HTML labeled-box
// fallback has no PDF geometry either).
func pdfMatrix(doc *pdf.Doc, pg int, pageH float64, o Object) (int, error) {
	m, err := vizParseMatrix(o)
	if err != nil {
		return 0, nil
	}
	l, top, w, h := objRectOf(o.Attrs)
	if w <= 0 || h <= 0 || len(m.rows) == 0 {
		return 0, nil
	}
	return pdfGrid(doc, pg, l*pxToPt, pageH-top*pxToPt, w, h, m.columns, m.rows, o.Attrs)
}

// pdfSubreport draws a SubreportObject as a gray placeholder outline with
// its ASCII label, mirroring the HTML placeholder (ReportPage rides along).
func pdfSubreport(doc *pdf.Doc, pg int, pageH float64, o Object) error {
	l, t, w, h := objRectOf(o.Attrs)
	if w <= 0 || h <= 0 {
		return nil
	}
	x, y0, pw, ph := l*pxToPt, pageH-(t+h)*pxToPt, w*pxToPt, h*pxToPt
	if err := doc.StrokeRect(pg, x, y0, pw, ph, 0.75, 0.5, 0.5, 0.5); err != nil {
		return err
	}
	label := "Subreport: " + o.Name
	if p := strings.TrimSpace(attrVal(o.Attrs, "ReportPage")); p != "" {
		label += " (Page: " + p + ")"
	}
	if label == "" || !fontmetrics.ASCIIOnly(label) {
		return nil
	}
	size, face := pdfFont(o.Attrs)
	if ph <= size {
		return nil
	}
	return doc.Text(pg, x+2, y0+ph-size-2, size, face, label)
}
