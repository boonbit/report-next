package report

import (
	"strings"
	"testing"
)

func exprTestMaps() (map[string]string, map[string]string) {
	fields := map[string]string{
		"FirstName":           "Ada",
		"LastName":            "Lovelace",
		"Categories.Category": "Drinks",
		"price":               "19.5",
		"qty":                 "4",
		"pad":                 "  hi  ",
		"neg":                 "-3",
		"word":                "Hello World",
	}
	params := map[string]string{
		"Title": "Ms",
		"rate":  "0.07",
	}
	return fields, params
}

func TestExprEval(t *testing.T) {
	fields, params := exprTestMaps()
	tests := []struct {
		name    string
		text    string
		want    string
		wantErr string
	}{
		{"plain", "hello", "hello", ""},
		{"empty", "", "", ""},
		{"single", "hi [FirstName]!", "hi Ada!", ""},
		{"dotted", "[Categories.Category]", "Drinks", ""},
		{"dottedSuffix", "x [Category] y", "x Drinks y", ""},
		{"param", "[Title] [LastName]", "Ms Lovelace", ""},
		{"caseFold", "[firstname]", "Ada", ""},
		{"arith", "[[price]*[qty]]", "78", ""},
		{"headline", "[[price]*[qty]]", "78", ""},
		{"bareIdent", "[[qty]]", "4", ""},
		{"upper", "[[UpperCase([FirstName])]]", "ADA", ""},
		{"lower", "[[LowerCase([LastName])]]", "lovelace", ""},
		{"trim", "[[Trim([pad])]]", "hi", ""},
		{"substr2", "[[Substring([word], 6)]]", "World", ""},
		{"substr3", "[[Substring([word], 0, 5)]]", "Hello", ""},
		{"tostring", "[[ToString(42)]]", "42", ""},
		{"iifTrue", "[[Iif([qty] > 2, 'big', 'small')]]", "big", ""},
		{"iifFalse", "[[Iif([qty] < 2, 'big', 'small')]]", "small", ""},
		{"abs", "[[Abs([neg])]]", "3", ""},
		{"round0", "[[Round(2.5)]]", "3", ""},
		{"round2", "[[Round(2.345, 2)]]", "2.35", ""},
		{"nowConst", "[[Now]]", "nonempty", ""},
		{"todayConst", "[[Today]]", "nonempty", ""},
		{"mixed", "Total: [[price]*[qty]] USD", "Total: 78 USD", ""},
		{"concat", "[[[FirstName] + ' ' + [LastName]]]", "Ada Lovelace", ""},
		{"unknownSingle", "[Nope]", "", "unknown name"},
		{"unknownInner", "[[Nope + 1]]", "", "unknown name"},
		{"unknownFunc", "[[Bogus(1)]]", "", "unknown function"},
		{"unclosedSingle", "hi [FirstName", "", "unclosed"},
		{"unclosedDouble", "hi [[price*qty", "", "unclosed"},
		{"emptyRef", "hi []", "", "empty"},
		{"divZero", "[[1/0]]", "", "division by zero"},
		{"nonNumeric", "[[[word] + 1]]", "Hello World1", ""},
		{"nonNumericMul", "[[[word] * 2]]", "", "not numeric"},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			got, err := exprEval(tc.text, fields, params)
			if tc.wantErr != "" {
				if err == nil {
					t.Fatalf("exprEval(%q) = %q, want error %q", tc.text, got, tc.wantErr)
				}
				if !strings.Contains(err.Error(), tc.wantErr) {
					t.Fatalf("exprEval(%q) error = %v, want %q", tc.text, err, tc.wantErr)
				}
				return
			}
			if err != nil {
				t.Fatalf("exprEval(%q) error = %v", tc.text, err)
			}
			if tc.want == "nonempty" {
				if got == "" {
					t.Fatalf("exprEval(%q) = empty, want non-empty", tc.text)
				}
				return
			}
			if got != tc.want {
				t.Fatalf("exprEval(%q) = %q, want %q", tc.text, got, tc.want)
			}
		})
	}
}

func TestExprAggregate(t *testing.T) {
	exprRows := []map[string]string{
		{"amount": "10", "name": "a"},
		{"amount": "20", "name": "b"},
		{"amount": "30", "name": "c"},
		{"amount": "", "name": "d"},
	}
	tests := []struct {
		name    string
		fn      string
		col     string
		rows    []map[string]string
		want    string
		wantErr string
	}{
		{"sum", "sum", "amount", exprRows, "60", ""},
		{"sumCase", "SUM", "amount", exprRows, "60", ""},
		{"count", "count", "amount", exprRows, "3", ""},
		{"countEmptyCol", "count", "", exprRows, "4", ""},
		{"avg", "avg", "amount", exprRows, "20", ""},
		{"averageAlias", "average", "amount", exprRows, "20", ""},
		{"min", "min", "amount", exprRows, "10", ""},
		{"max", "max", "amount", exprRows, "30", ""},
		{"sumEmpty", "sum", "amount", nil, "0", ""},
		{"countEmpty", "count", "amount", nil, "0", ""},
		{"avgEmpty", "avg", "amount", nil, "", "no values"},
		{"minEmpty", "min", "amount", nil, "", "no values"},
		{"maxEmpty", "max", "amount", nil, "", "no values"},
		{"badFn", "median", "amount", exprRows, "", "unknown aggregate"},
		{"nonNumeric", "sum", "name", exprRows, "", "not numeric"},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			got, err := exprAggregate(tc.fn, tc.col, tc.rows)
			if tc.wantErr != "" {
				if err == nil {
					t.Fatalf("exprAggregate(%q) = %q, want error %q", tc.fn, got, tc.wantErr)
				}
				if !strings.Contains(err.Error(), tc.wantErr) {
					t.Fatalf("exprAggregate(%q) error = %v, want %q", tc.fn, err, tc.wantErr)
				}
				return
			}
			if err != nil {
				t.Fatalf("exprAggregate(%q) error = %v", tc.fn, err)
			}
			if got != tc.want {
				t.Fatalf("exprAggregate(%q) = %q, want %q", tc.fn, got, tc.want)
			}
		})
	}
}
