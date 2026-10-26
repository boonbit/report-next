package report

import (
	"strconv"
	"strings"
	"time"
)

// Format kinds accepted by exprFormatValue.
const (
	exprKindNumber   = "Number"
	exprKindCurrency = "Currency"
	exprKindPercent  = "Percent"
	exprKindDate     = "Date"
	exprKindBoolean  = "Boolean"
)

// exprFormatValue renders value per kind, mirroring the FastReport TextObject
// Format attribute (Format=Currency with Format.DecimalDigits, etc.).
// digits is the number of decimals for the numeric kinds; a negative digits
// falls back to 2. Unparseable values and unknown kinds pass through
// unchanged.
func exprFormatValue(kind, value string, digits int) string {
	switch {
	case strings.EqualFold(kind, exprKindCurrency):
		f, err := strconv.ParseFloat(strings.TrimSpace(value), 64)
		if err != nil {
			return value
		}
		return exprMoney(f, digits)
	case strings.EqualFold(kind, exprKindNumber):
		f, err := strconv.ParseFloat(strings.TrimSpace(value), 64)
		if err != nil {
			return value
		}
		return exprGrouped(f, digits)
	case strings.EqualFold(kind, exprKindPercent):
		f, err := strconv.ParseFloat(strings.TrimSpace(value), 64)
		if err != nil {
			return value
		}
		return exprGrouped(f*100, digits) + "%"
	case strings.EqualFold(kind, exprKindDate):
		t, ok := exprParseDate(strings.TrimSpace(value))
		if !ok {
			return value
		}
		return t.Format("01/02/2006")
	case strings.EqualFold(kind, exprKindBoolean):
		switch strings.ToLower(strings.TrimSpace(value)) {
		case "true", "1", "yes", "y", "-1":
			return "True"
		case "false", "0", "no", "n", "":
			return "False"
		}
		return value
	}
	return value
}

func exprDigits(digits int) int {
	if digits < 0 {
		return 2
	}
	return digits
}

// exprGrouped formats f with exactly digits decimals and commas grouping
// the integer part: 1234567.891 with 2 digits -> "1,234,567.89".
func exprGrouped(f float64, digits int) string {
	d := exprDigits(digits)
	s := strconv.FormatFloat(f, 'f', d, 64)
	neg := ""
	if strings.HasPrefix(s, "-") {
		neg = "-"
		s = s[1:]
	}
	intPart := s
	frac := ""
	if i := strings.IndexByte(s, '.'); i >= 0 {
		intPart, frac = s[:i], s[i+1:]
	}
	var b strings.Builder
	b.WriteString(neg)
	for i, c := range intPart {
		if i > 0 && (len(intPart)-i)%3 == 0 {
			b.WriteByte(',')
		}
		b.WriteRune(c)
	}
	if d > 0 {
		b.WriteByte('.')
		b.WriteString(frac)
	}
	return b.String()
}

func exprMoney(f float64, digits int) string {
	if f < 0 {
		return "-$" + exprGrouped(-f, digits)
	}
	return "$" + exprGrouped(f, digits)
}

func exprParseDate(s string) (time.Time, bool) {
	for _, layout := range []string{
		"2006-01-02",
		"2006-01-02 15:04:05",
		"2006-01-02T15:04:05",
		time.RFC3339,
		"01/02/2006",
		"01/02/2006 15:04:05",
		"1/2/2006",
		"02.01.2006",
		"2006/01/02",
	} {
		if t, err := time.Parse(layout, s); err == nil {
			return t, true
		}
	}
	return time.Time{}, false
}
