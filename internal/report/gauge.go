package report

import (
	"fmt"
	"html"
	"math"
	"strconv"
	"strings"
)

// TRACK viz (gauge render): gauge HTML renderers as inline SVG.
//
// Real attribute names sampled from FastReport/Demos/Reports/Gauge.frx and
// "Sparkline, Gauge.frx":
//   LinearGauge         Left Top Value Expression Scale.Font
//                       Scale.MajorTicks.Width Scale.MajorTicks.Color
//                       Scale.MinorTicks.Width Scale.MinorTicks.Color
//                       Pointer.Fill.Color Pointer.BorderColor
//                       Pointer.Height Pointer.Width Label.Font
//   SimpleGauge         (same shape as LinearGauge)
//   RadialGauge         Left Top Expression Type(Semicircle) Position(Top)
//                       Scale.Font Scale.MajorTicks.Length/Color
//                       Scale.MinorTicks.Length/Color Pointer.Fill.Color
//                       Pointer.BorderColor Label.Text Label.Font
//   SimpleProgressGauge Left/Width/Height Top/Dock Expression Maximum Value
//                       Scale.Font Pointer.Fill.Color Pointer.Height/Width
//                       Pointer.Type(Small) Label.Text Label.Font
//
// Value/Maximum are raw numbers (LinearGauge1 Value="15" with no Maximum
// means 15 of a default 0..100 scale; SimpleProgressGauge1 Maximum="1"
// Value="1" means full). Expression data binding is out of scope, so the
// static Value attribute drives the pointer.

// vizGaugeKinds are the object kinds vizGaugeHTML can render. Anything else
// (including a bare GaugeObject, which carries no gauge geometry in the
// sampled fixtures) keeps the labeled-box fallback.
var vizGaugeKinds = map[string]bool{
	"LinearGauge":         true,
	"SimpleGauge":         true,
	"RadialGauge":         true,
	"SimpleProgressGauge": true,
}

// vizGaugeFraction parses Value/Minimum/Maximum into a 0..1 fraction.
// Missing Minimum defaults to 0, missing Maximum to 100; the result clamps.
func vizGaugeFraction(a []Attr) float64 {
	min := 0.0
	max := 100.0
	if v := strings.TrimSpace(attrVal(a, "Minimum")); v != "" {
		if f, err := strconv.ParseFloat(v, 64); err == nil {
			min = f
		}
	}
	if v := strings.TrimSpace(attrVal(a, "Maximum")); v != "" {
		if f, err := strconv.ParseFloat(v, 64); err == nil {
			max = f
		}
	}
	val := min
	if v := strings.TrimSpace(attrVal(a, "Value")); v != "" {
		if f, err := strconv.ParseFloat(v, 64); err == nil {
			val = f
		}
	}
	if max <= min {
		return 0
	}
	frac := (val - min) / (max - min)
	if frac < 0 {
		return 0
	}
	if frac > 1 {
		return 1
	}
	return frac
}

// vizGaugeFill reads Pointer.Fill.Color ("r, g, b" triples pass through;
// Transparent falls back to gray so the pointer stays visible).
func vizGaugeFill(a []Attr) string {
	if c := strings.TrimSpace(attrVal(a, "Pointer.Fill.Color")); c != "" &&
		!strings.EqualFold(c, "Transparent") {
		return c
	}
	return "Gray"
}

// vizGaugeLabel is the caption under the gauge: Label.Text when present,
// else the raw Value.
func vizGaugeLabel(a []Attr) string {
	if t := attrVal(a, "Label.Text"); t != "" {
		return t
	}
	return strings.TrimSpace(attrVal(a, "Value"))
}

// vizGaugeBarSVG renders a horizontal bar gauge (linear, simple and simple
// progress kinds): background track, filled fraction, 5 major ticks, label.
func vizGaugeBarSVG(fill, label string, frac float64) string {
	f := func(v float64) string { return strconv.FormatFloat(v, 'f', 2, 64) }
	var sb strings.Builder
	sb.WriteString(`<svg width="100%" height="100%" viewBox="0 0 100 24" preserveAspectRatio="none">`)
	sb.WriteString(`<rect x="0" y="4" width="100" height="10" fill="Gainsboro"/>`)
	fmt.Fprintf(&sb, `<rect x="0" y="4" width="%s" height="10" fill="%s"/>`,
		f(100*frac), html.EscapeString(fill))
	for i := 0; i <= 4; i++ {
		fmt.Fprintf(&sb, `<line x1="%s" y1="4" x2="%s" y2="14" stroke="Black" stroke-width="0.5"/>`,
			f(float64(i*25)), f(float64(i*25)))
	}
	if label != "" {
		fmt.Fprintf(&sb, `<text x="50" y="22" font-size="6" text-anchor="middle">%s</text>`,
			html.EscapeString(label))
	}
	sb.WriteString(`</svg>`)
	return sb.String()
}

// vizGaugeRadialSVG renders a radial gauge as a semicircle dial: background
// arc, a needle at angle pi*(1-frac), a hub circle and the label.
func vizGaugeRadialSVG(fill, label string, frac float64) string {
	f := func(v float64) string { return strconv.FormatFloat(v, 'f', 2, 64) }
	// Needle tip on a r=38 semicircle centered at (50,52).
	ang := math.Pi * (1 - frac)
	tx := 50 + 38*math.Cos(ang)
	ty := 52 - 38*math.Sin(ang)
	var sb strings.Builder
	sb.WriteString(`<svg width="100%" height="100%" viewBox="0 0 100 60">`)
	sb.WriteString(`<path d="M 12 52 A 38 38 0 0 1 88 52" fill="none" stroke="Gainsboro" stroke-width="8"/>`)
	fmt.Fprintf(&sb, `<line x1="50" y1="52" x2="%s" y2="%s" stroke="%s" stroke-width="3"/>`,
		f(tx), f(ty), html.EscapeString(fill))
	sb.WriteString(`<circle cx="50" cy="52" r="4" fill="Black"/>`)
	if label != "" {
		fmt.Fprintf(&sb, `<text x="50" y="59" font-size="6" text-anchor="middle">%s</text>`,
			html.EscapeString(label))
	}
	sb.WriteString(`</svg>`)
	return sb.String()
}

// vizGaugeHTML renders linear/radial/simple gauges as inline SVG inside the
// object's absolute box. Unknown gauge kinds keep the labeled-box fallback.
func vizGaugeHTML(o Object) string {
	if !vizGaugeKinds[o.Kind] {
		return objFallbackHTML(o)
	}
	frac := vizGaugeFraction(o.Attrs)
	fill := vizGaugeFill(o.Attrs)
	label := vizGaugeLabel(o.Attrs)
	var svg string
	if o.Kind == "RadialGauge" {
		svg = vizGaugeRadialSVG(fill, label, frac)
	} else {
		svg = vizGaugeBarSVG(fill, label, frac)
	}
	return fmt.Sprintf(`<div class="gauge" data-kind="%s" data-name="%s" style="%s">%s</div>`,
		html.EscapeString(o.Kind), html.EscapeString(o.Name),
		objAbsStyle(o.Attrs, "overflow:hidden;"), svg)
}
