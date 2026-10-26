package report

import (
	"strings"
	"testing"
)

func TestRenderTitleBand(t *testing.T) {
	rep := loadHello(t)
	got := RenderBand(rep.Pages[0].Bands[0])
	for _, want := range []string{
		"The report goes here...",
		"color:Firebrick",
		"font-family:'Segoe UI'",
		"font-size:13.33px",
		"font-weight:bold",
	} {
		if !strings.Contains(got, want) {
			t.Errorf("rendered band missing %q:\n%s", want, got)
		}
	}
}
