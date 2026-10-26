package report

import "testing"

func TestExprFormatValue(t *testing.T) {
	tests := []struct {
		name   string
		kind   string
		value  string
		digits int
		want   string
	}{
		{"number2", "Number", "1234567.891", 2, "1,234,567.89"},
		{"number0", "Number", "19.5", 0, "20"},
		{"numberNeg", "Number", "-1234.5", 2, "-1,234.50"},
		{"numberBad", "Number", "abc", 2, "abc"},
		{"currency", "Currency", "19.5", 2, "$19.50"},
		{"currencyGrouped", "Currency", "1234567.8", 2, "$1,234,567.80"},
		{"currencyNeg", "Currency", "-5", 2, "-$5.00"},
		{"currencyBad", "Currency", "n/a", 2, "n/a"},
		{"percent", "Percent", "0.05", 2, "5.00%"},
		{"percentWhole", "Percent", "0.5", 0, "50%"},
		{"percentBad", "Percent", "x", 2, "x"},
		{"dateISO", "Date", "2009-06-03", 0, "06/03/2009"},
		{"dateUS", "Date", "06/03/2009 02:45:18", 0, "06/03/2009"},
		{"dateBad", "Date", "not a date", 0, "not a date"},
		{"boolTrue", "Boolean", "true", 0, "True"},
		{"boolOne", "Boolean", "1", 0, "True"},
		{"boolFalse", "Boolean", "false", 0, "False"},
		{"boolZero", "Boolean", "0", 0, "False"},
		{"boolOther", "Boolean", "maybe", 0, "maybe"},
		{"kindCase", "currency", "2", 2, "$2.00"},
		{"unknownKind", "Fancy", "7", 2, "7"},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			if got := exprFormatValue(tc.kind, tc.value, tc.digits); got != tc.want {
				t.Fatalf("exprFormatValue(%q,%q,%d) = %q, want %q",
					tc.kind, tc.value, tc.digits, got, tc.want)
			}
		})
	}
}
