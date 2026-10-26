package report

import (
	"strings"
	"testing"
)

// TRACK viz: gauge tests against real attribute names from
// FastReport/Demos/Reports/Gauge.frx (LinearGauge1, SimpleGauge1,
// RadialGauge1, SimpleProgressGauge1) and "Sparkline, Gauge.frx".

func TestVizLinearGaugeSVG(t *testing.T) {
	got := vizGaugeHTML(Object{Kind: "LinearGauge", Name: "LinearGauge1",
		Attrs: objAttrs("Left", "37.8", "Top", "103.95", "Value", "15",
			"Expression", "[Order Details.Quantity]",
			"Pointer.Fill.Color", "200, 160, 120")})
	for _, want := range []string{"<svg", "<rect", "<line", `data-kind="LinearGauge"`, "15"} {
		if !strings.Contains(got, want) {
			t.Errorf("linear gauge HTML missing %q:\n%s", want, got)
		}
	}
}

func TestVizSimpleGaugeSVG(t *testing.T) {
	got := vizGaugeHTML(Object{Kind: "SimpleGauge", Name: "SimpleGauge1",
		Attrs: objAttrs("Value", "15", "Pointer.Fill.Color", "200, 160, 120")})
	if !strings.Contains(got, "<svg") || !strings.Contains(got, "<rect") {
		t.Errorf("simple gauge should render bar SVG:\n%s", got)
	}
}

func TestVizRadialGaugeSVG(t *testing.T) {
	got := vizGaugeHTML(Object{Kind: "RadialGauge", Name: "RadialGauge1",
		Attrs: objAttrs("Expression", "[Order Details.Quantity]",
			"Type", "Semicircle", "Position", "Top",
			"Pointer.Fill.Color", "160, 120, 120")})
	for _, want := range []string{"<svg", "<path", "<line", "<circle"} {
		if !strings.Contains(got, want) {
			t.Errorf("radial gauge HTML missing %q:\n%s", want, got)
		}
	}
}

func TestVizSimpleProgressGaugeSVG(t *testing.T) {
	got := vizGaugeHTML(Object{Kind: "SimpleProgressGauge", Name: "SimpleProgressGauge1",
		Attrs: objAttrs("Maximum", "1", "Value", "1",
			"Pointer.Fill.Color", "151, 189, 253", "Label.Text", "100%")})
	for _, want := range []string{"<svg", "<rect", "100%"} {
		if !strings.Contains(got, want) {
			t.Errorf("progress gauge HTML missing %q:\n%s", want, got)
		}
	}
}

func TestVizGaugeFractionClamp(t *testing.T) {
	if f := vizGaugeFraction(objAttrs("Value", "15")); f != 0.15 {
		t.Errorf("Value=15 of default 0..100 should be 0.15, got %v", f)
	}
	if f := vizGaugeFraction(objAttrs("Maximum", "1", "Value", "1")); f != 1 {
		t.Errorf("Value=1/Maximum=1 should be 1, got %v", f)
	}
	if f := vizGaugeFraction(objAttrs("Maximum", "1", "Value", "9")); f != 1 {
		t.Errorf("overflow should clamp to 1, got %v", f)
	}
}

func TestVizGaugeUnknownKindFallback(t *testing.T) {
	for _, kind := range []string{"GaugeObject", "FancyGauge"} {
		got := vizGaugeHTML(Object{Kind: kind, Name: "G9",
			Attrs: objAttrs("Left", "9.45", "Width", "100", "Height", "50")})
		if strings.Contains(got, "<svg") || !strings.Contains(got, `data-kind="`+kind+`"`) {
			t.Errorf("unknown gauge kind %q should be a labeled fallback:\n%s", kind, got)
		}
	}
}

func TestVizGaugesViaRenderBand(t *testing.T) {
	b := Band{Kind: "DataBand", Name: "Data1",
		Attrs: objAttrs("Width", "718.2", "Height", "642.6"),
		Objects: []Object{
			{Kind: "LinearGauge", Name: "LinearGauge1", Attrs: objAttrs("Value", "15")},
			{Kind: "RadialGauge", Name: "RadialGauge1", Attrs: objAttrs("Type", "Semicircle")},
		}}
	got := RenderBand(b)
	if strings.Count(got, "<svg") < 2 {
		t.Errorf("RenderBand should dispatch both gauge kinds:\n%s", got)
	}
}
