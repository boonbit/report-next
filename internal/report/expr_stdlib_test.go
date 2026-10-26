package report

import (
	"strings"
	"testing"
)

func TestExprStdlib(t *testing.T) {
	fields, params := exprTestMaps()
	fields["birth"] = "2024-01-31"
	fields["when"] = "2024-01-01 10:00:00"
	fields["bad"] = "not-a-date"
	fields["empty"] = ""
	tests := []struct {
		name    string
		text    string
		want    string
		wantErr string
	}{
		// Contains.
		{"containsTrue", "[[Contains('hello', 'ell')]]", "True", ""},
		{"containsFalse", "[[Contains('hello', 'z')]]", "False", ""},
		{"containsEmptyHay", "[[Contains('', 'a')]]", "False", ""},
		{"containsEmptyNeedle", "[[Contains('abc', '')]]", "True", ""},
		{"containsField", "[[Contains([word], 'World')]]", "True", ""},
		{"containsArity", "[[Contains('a')]]", "", "wants 2 arguments"},
		// Replace.
		{"replace", "[[Replace('hello', 'l', 'L')]]", "heLLo", ""},
		{"replaceNoMatch", "[[Replace('hello', 'z', 'Z')]]", "hello", ""},
		{"replaceEmptyNew", "[[Replace('hello', 'l', '')]]", "heo", ""},
		{"replaceEmptyOld", "[[Replace('hello', '', 'x')]]", "", "must not be empty"},
		{"replaceArity", "[[Replace('a', 'b')]]", "", "wants 3 arguments"},
		// Remove.
		{"removeToEnd", "[[Remove('hello', 2)]]", "he", ""},
		{"removeCount", "[[Remove('hello', 1, 3)]]", "ho", ""},
		{"removeZero", "[[Remove('hello', 0, 0)]]", "hello", ""},
		{"removeAtEnd", "[[Remove('hello', 5)]]", "hello", ""},
		{"removeNeg", "[[Remove('hello', -1)]]", "", "out of range"},
		{"removePastEnd", "[[Remove('hello', 6)]]", "", "out of range"},
		{"removeCountOverflow", "[[Remove('hello', 3, 5)]]", "", "out of range"},
		{"removeArity", "[[Remove('hello')]]", "", "wants 2 or 3 arguments"},
		// PadLeft / PadRight.
		{"padLeftSpace", "[[PadLeft('ab', 5)]]", "   ab", ""},
		{"padLeftChar", "[[PadLeft('ab', 5, '0')]]", "000ab", ""},
		{"padRightSpace", "[[PadRight('ab', 5)]]", "ab   ", ""},
		{"padRightChar", "[[PadRight('ab', 5, '0')]]", "ab000", ""},
		{"padLeftWide", "[[PadLeft('abcdef', 3)]]", "abcdef", ""},
		{"padRightExact", "[[PadRight('abc', 3)]]", "abc", ""},
		{"padLeftNeg", "[[PadLeft('ab', -1)]]", "", "out of range"},
		{"padEmptyPadChar", "[[PadLeft('ab', 4, '')]]", "  ab", ""},
		{"padLeftArity", "[[PadLeft('ab')]]", "", "wants 2 or 3 arguments"},
		// Insert.
		{"insertMid", "[[Insert('hello', 2, 'XX')]]", "heXXllo", ""},
		{"insertStart", "[[Insert('hello', 0, 'XX')]]", "XXhello", ""},
		{"insertEnd", "[[Insert('hello', 5, 'XX')]]", "helloXX", ""},
		{"insertNeg", "[[Insert('hello', -1, 'x')]]", "", "out of range"},
		{"insertPastEnd", "[[Insert('hello', 6, 'x')]]", "", "out of range"},
		// Length.
		{"length", "[[Length('hello')]]", "5", ""},
		{"lengthEmpty", "[[Length('')]]", "0", ""},
		{"lengthUnicode", "[[Length('héllo')]]", "5", ""},
		// Chr / Asc.
		{"chr", "[[Chr(65)]]", "A", ""},
		{"chrLower", "[[Chr(97)]]", "a", ""},
		{"chrNeg", "[[Chr(-1)]]", "", "out of range"},
		{"chrHuge", "[[Chr(2000000)]]", "", "out of range"},
		{"asc", "[[Asc('A')]]", "65", ""},
		{"ascFirst", "[[Asc('ABC')]]", "65", ""},
		{"ascEmpty", "[[Asc('')]]", "", "empty string"},
		{"ascDigit", "[[Asc('0')]]", "48", ""},
		// TitleCase.
		{"titleBasic", "[[TitleCase('hello world')]]", "Hello World", ""},
		{"titleUpper", "[[TitleCase('HELLO WORLD')]]", "Hello World", ""},
		{"titleMixed", "[[TitleCase('hELLO wORLD')]]", "Hello World", ""},
		{"titleEmpty", "[[TitleCase('')]]", "", ""},
		{"titlePunct", "[[TitleCase('one-two three')]]", "One-Two Three", ""},
		// AddDays.
		{"addDays", "[[AddDays('2024-01-01', 5)]]", "2024-01-06", ""},
		{"addDaysCrossMonth", "[[AddDays('2024-01-30', 5)]]", "2024-02-04", ""},
		{"addDaysNeg", "[[AddDays('2024-01-10', -10)]]", "2023-12-31", ""},
		{"addDaysFrac", "[[AddDays('2024-01-01', 1.5)]]", "2024-01-02 12:00:00", ""},
		{"addDaysField", "[[AddDays([birth], 1)]]", "2024-02-01", ""},
		{"addDaysDateTime", "[[AddDays([when], 1)]]", "2024-01-02 10:00:00", ""},
		{"addDaysBad", "[[AddDays('nope', 1)]]", "", ""},
		{"addDaysBadField", "[[AddDays([bad], 1)]]", "", ""},
		{"addDaysNonNum", "[[AddDays('2024-01-01', 'x')]]", "", "not numeric"},
		// AddMonths / AddYears.
		{"addMonths", "[[AddMonths('2024-01-15', 2)]]", "2024-03-15", ""},
		{"addMonthsClamp", "[[AddMonths([birth], 1)]]", "2024-02-29", ""},
		{"addMonthsNeg", "[[AddMonths('2024-03-31', -1)]]", "2024-02-29", ""},
		{"addMonthsBad", "[[AddMonths('nope', 1)]]", "", ""},
		{"addYears", "[[AddYears('2024-02-29', 1)]]", "2025-02-28", ""},
		{"addYearsPlain", "[[AddYears('2020-06-15', 4)]]", "2024-06-15", ""},
		{"addYearsBad", "[[AddYears([bad], 1)]]", "", ""},
		// Day / Month / Year.
		{"day", "[[Day('2024-03-05')]]", "5", ""},
		{"month", "[[Month('2024-03-05')]]", "3", ""},
		{"year", "[[Year('2024-03-05')]]", "2024", ""},
		{"dayBad", "[[Day('junk')]]", "", ""},
		{"monthBad", "[[Month([bad])]]", "", ""},
		{"yearBad", "[[Year('2024-13-01')]]", "", ""},
		// DayOfWeek.
		{"dow", "[[DayOfWeek('2024-01-01')]]", "Monday", ""},
		{"dowSunday", "[[DayOfWeek('2024-01-07')]]", "Sunday", ""},
		{"dowBad", "[[DayOfWeek('junk')]]", "", ""},
		// DaysInMonth.
		{"dimLeap", "[[DaysInMonth(2024, 2)]]", "29", ""},
		{"dimCommon", "[[DaysInMonth(2023, 2)]]", "28", ""},
		{"dim30", "[[DaysInMonth(2024, 4)]]", "30", ""},
		{"dim31", "[[DaysInMonth(2024, 1)]]", "31", ""},
		{"dimBadMonth", "[[DaysInMonth(2024, 13)]]", "", "out of range"},
		{"dimZeroMonth", "[[DaysInMonth(2024, 0)]]", "", "out of range"},
		// DateSerial.
		{"serial", "[[DateSerial(2024, 2, 29)]]", "2024-02-29", ""},
		{"serialPad", "[[DateSerial(2024, 3, 5)]]", "2024-03-05", ""},
		{"serialBadDay", "[[DateSerial(2023, 2, 29)]]", "", "out of range"},
		{"serialBadMonth", "[[DateSerial(2024, 13, 1)]]", "", "out of range"},
		{"serialZeroDay", "[[DateSerial(2024, 1, 0)]]", "", "out of range"},
		// Maximum / Minimum.
		{"max", "[[Maximum(3, 7)]]", "7", ""},
		{"min", "[[Minimum(3, 7)]]", "3", ""},
		{"maxFloat", "[[Maximum(2.5, 2.4)]]", "2.5", ""},
		{"minFloat", "[[Minimum(2.5, 2.4)]]", "2.4", ""},
		{"maxNeg", "[[Maximum(-3, -9)]]", "-3", ""},
		{"minNeg", "[[Minimum(-3, -9)]]", "-9", ""},
		{"maxField", "[[Maximum([price], [qty])]]", "19.5", ""},
		{"minField", "[[Minimum([price], [qty])]]", "4", ""},
		{"maxNonNum", "[[Maximum('x', 1)]]", "", "not numeric"},
		{"maxArity", "[[Maximum(1)]]", "", "wants 2 arguments"},
		// Switch.
		{"switchFirst", "[[Switch(1 = 2, 'a', 2 = 2, 'b')]]", "b", ""},
		{"switchFirstWins", "[[Switch(1 = 1, 'a', 2 = 2, 'b')]]", "a", ""},
		{"switchNone", "[[Switch(1 = 2, 'a', 3 = 4, 'b')]]", "", ""},
		{"switchThreePairs", "[[Switch(1 = 2, 'a', 2 = 3, 'b', 3 = 3, 'c')]]", "c", ""},
		{"switchOdd", "[[Switch(1 = 1, 'a', 2 = 2)]]", "", "even number"},
		{"switchSingle", "[[Switch(1 = 1)]]", "", "even number"},
		{"switchOddField", "[[Switch([qty] > 2, 'big', 'small')]]", "", "even number"},
		{"switchFieldCond", "[[Switch([qty] > 2, 'big', [qty] < 2, 'small')]]", "big", ""},
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
			if got != tc.want {
				t.Fatalf("exprEval(%q) = %q, want %q", tc.text, got, tc.want)
			}
		})
	}
}

func TestExprStdlibCaseInsensitive(t *testing.T) {
	fields, params := exprTestMaps()
	for _, text := range []string{
		"[[contains('hello', 'ell')]]",
		"[[titlecase('hello world')]]",
		"[[dayofweek('2024-01-01')]]",
		"[[maximum(1, 2)]]",
		"[[switch(1 = 1, 'y', 1 = 2, 'n')]]",
	} {
		if _, err := exprEval(text, fields, params); err != nil {
			t.Fatalf("exprEval(%q) error = %v", text, err)
		}
	}
}
