package report

import "testing"

func TestExprBuilder(t *testing.T) {
	rep := exprNewReport("Sales")
	if !Valid(*rep) {
		t.Fatalf("fresh builder report invalid: %v", Validate(*rep))
	}
	pi := exprAddPage(rep, "Data")
	if pi != 1 {
		t.Fatalf("exprAddPage = %d, want 1", pi)
	}
	bi := exprAddBand(rep, pi, "DataBand", "Data1")
	if bi != 0 {
		t.Fatalf("exprAddBand = %d, want 0", bi)
	}
	ti := exprAddText(rep, pi, bi, "TextTotal", "[[price]*[qty]]")
	if ti != 0 {
		t.Fatalf("exprAddText = %d, want 0", ti)
	}
	if !Valid(*rep) {
		t.Fatalf("built report invalid: %v", Validate(*rep))
	}
	got, err := exprEval("Total: [[price]*[qty]]",
		map[string]string{"price": "19.5", "qty": "4"}, nil)
	if err != nil {
		t.Fatal(err)
	}
	if got != "Total: 78" {
		t.Fatalf("end-to-end eval = %q, want %q", got, "Total: 78")
	}
	if got := exprFormatValue("Currency", got[7:], 2); got != "$78.00" {
		t.Fatalf("end-to-end format = %q, want %q", got, "$78.00")
	}
}

func TestExprBuilderEdge(t *testing.T) {
	tests := []struct {
		name string
		run  func() bool
		want bool
	}{
		{"emptyTitleValid", func() bool {
			return Valid(*exprNewReport(""))
		}, true},
		{"dupPageRenamed", func() bool {
			rep := exprNewReport("t")
			exprAddPage(rep, "Page1")
			return Valid(*rep)
		}, true},
		{"emptyNamesValid", func() bool {
			rep := exprNewReport("t")
			pi := exprAddPage(rep, "")
			bi := exprAddBand(rep, pi, "", "")
			exprAddText(rep, pi, bi, "", "x")
			return Valid(*rep)
		}, true},
		{"dupBandRenamed", func() bool {
			rep := exprNewReport("t")
			exprAddBand(rep, 0, "DataBand", "ReportTitle1")
			return Valid(*rep)
		}, true},
		{"badPageIdx", func() bool {
			rep := exprNewReport("t")
			return exprAddBand(rep, 9, "DataBand", "B") == -1
		}, true},
		{"badBandIdx", func() bool {
			rep := exprNewReport("t")
			return exprAddText(rep, 0, 9, "T", "x") == -1
		}, true},
		{"nilReport", func() bool {
			return exprAddPage(nil, "P") == -1 &&
				exprAddBand(nil, 0, "K", "B") == -1 &&
				exprAddText(nil, 0, 0, "T", "x") == -1
		}, true},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			if got := tc.run(); got != tc.want {
				t.Fatalf("%s = %v, want %v", tc.name, got, tc.want)
			}
		})
	}
}
