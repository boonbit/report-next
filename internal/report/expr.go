package report

import (
	"fmt"
	"math"
	"strconv"
	"strings"
	"time"
	"unicode"
)

// exprEval substitutes [Name] references and [[expression]] blocks in text.
//
// Single brackets look up Name in fields, then params (a dotted name such as
// "Table.Column" also matches its last segment). Double brackets evaluate the
// enclosed expression: [refs], the Now/Today constants and the UpperCase,
// LowerCase, Trim, Substring, ToString, Iif, Abs, Round, Contains, Replace,
// Remove, PadLeft, PadRight, Insert, Length, Chr, Asc, TitleCase, AddDays,
// AddMonths, AddYears, Day, Month, Year, DayOfWeek, DaysInMonth, DateSerial,
// Maximum, Minimum and Switch functions.
//
// It returns an error for unknown names and for unclosed brackets.
func exprEval(text string, fields, params map[string]string) (string, error) {
	var sb strings.Builder
	i := 0
	for i < len(text) {
		if text[i] == '[' {
			// Depth-match so [[a]*[b]] is one unit with inner "[a]*[b]".
			inner, next, err := exprMatch(text, i)
			if err != nil {
				return "", err
			}
			v, err := exprCalc(inner, fields, params)
			if err != nil {
				return "", err
			}
			sb.WriteString(v)
			i = next
			continue
		}
		sb.WriteByte(text[i])
		i++
	}
	return sb.String(), nil
}

// exprMatch returns the content between text[at] and its depth-matching
// ']' plus the offset just past that ']'.
// An unclosed bracket is an error.
func exprMatch(text string, at int) (string, int, error) {
	depth := 0
	for i := at; i < len(text); i++ {
		switch text[i] {
		case '[':
			depth++
		case ']':
			depth--
			if depth == 0 {
				return text[at+1 : i], i + 1, nil
			}
		}
	}
	return "", 0, fmt.Errorf("report: unclosed [ in %q", text)
}

// exprComplex reports whether s looks like an expression rather than a
// plain reference name.
func exprComplex(s string) bool {
	return strings.ContainsAny(s, "[]()+-*/,<>=!\"'")
}

// exprLookup resolves a reference name against fields first, then params.
// Dotted names also match their last segment ("Table.Column" finds "Column").
func exprLookup(name string, fields, params map[string]string) (string, error) {
	if name == "" {
		return "", fmt.Errorf("report: empty [] reference")
	}
	if v, ok := fields[name]; ok {
		return v, nil
	}
	if v, ok := params[name]; ok {
		return v, nil
	}
	if k := exprLastSegment(name); k != name {
		if v, ok := fields[k]; ok {
			return v, nil
		}
		if v, ok := params[k]; ok {
			return v, nil
		}
	}
	if v, ok := exprFoldGet(fields, name); ok {
		return v, nil
	}
	if v, ok := exprFoldGet(params, name); ok {
		return v, nil
	}
	return "", fmt.Errorf("report: unknown name %q", name)
}

func exprLastSegment(name string) string {
	if i := strings.LastIndexByte(name, '.'); i >= 0 {
		return strings.TrimSpace(name[i+1:])
	}
	return name
}

func exprFoldGet(m map[string]string, name string) (string, bool) {
	for k, v := range m {
		if strings.EqualFold(k, name) {
			return v, true
		}
		if tail := exprLastSegment(k); strings.EqualFold(tail, name) {
			return v, true
		}
	}
	return "", false
}

// exprAggregate folds one column over data rows.
// fn is one of sum, count, avg (average), min, max (case-insensitive).
// count is the number of rows with a non-empty col value, or len(rows)
// when col is empty. sum skips empty cells; avg, min and max skip them too
// and fail on empty input. Non-numeric non-empty cells fail the numeric
// aggregates.
func exprAggregate(fn, col string, rows []map[string]string) (string, error) {
	switch strings.ToLower(fn) {
	case "count":
		if col == "" {
			return strconv.Itoa(len(rows)), nil
		}
		n := 0
		for _, r := range rows {
			if r[col] != "" {
				n++
			}
		}
		return strconv.Itoa(n), nil
	case "sum", "avg", "average", "min", "max":
		var vals []float64
		for _, r := range rows {
			s := r[col]
			if s == "" {
				continue
			}
			f, err := strconv.ParseFloat(strings.TrimSpace(s), 64)
			if err != nil {
				return "", fmt.Errorf("report: aggregate %s: value %q is not numeric", fn, s)
			}
			vals = append(vals, f)
		}
		switch strings.ToLower(fn) {
		case "sum":
			sum := 0.0
			for _, v := range vals {
				sum += v
			}
			return exprNumStr(sum), nil
		case "avg", "average":
			if len(vals) == 0 {
				return "", fmt.Errorf("report: aggregate avg: no values")
			}
			sum := 0.0
			for _, v := range vals {
				sum += v
			}
			return exprNumStr(sum / float64(len(vals))), nil
		case "min":
			if len(vals) == 0 {
				return "", fmt.Errorf("report: aggregate min: no values")
			}
			m := vals[0]
			for _, v := range vals[1:] {
				if v < m {
					m = v
				}
			}
			return exprNumStr(m), nil
		case "max":
			if len(vals) == 0 {
				return "", fmt.Errorf("report: aggregate max: no values")
			}
			m := vals[0]
			for _, v := range vals[1:] {
				if v > m {
					m = v
				}
			}
			return exprNumStr(m), nil
		}
	}
	return "", fmt.Errorf("report: unknown aggregate %q", fn)
}

func exprNumStr(f float64) string {
	if math.IsNaN(f) || math.IsInf(f, 0) {
		return strconv.FormatFloat(f, 'g', -1, 64)
	}
	return strconv.FormatFloat(f, 'f', -1, 64)
}

// ---- expression calculator for [[...]] ----

type exprVal struct {
	s     string
	n     float64
	isNum bool
}

func exprStr(s string) exprVal { return exprVal{s: s} }

func exprNum(f float64) exprVal { return exprVal{s: exprNumStr(f), n: f, isNum: true} }

func exprBool(b bool) exprVal {
	if b {
		return exprVal{s: "True", n: 1, isNum: true}
	}
	return exprVal{s: "False", n: 0, isNum: true}
}

func exprTruth(v exprVal) bool {
	if v.isNum {
		return v.n != 0
	}
	s := strings.TrimSpace(v.s)
	if s == "" {
		return false
	}
	if strings.EqualFold(s, "true") {
		return true
	}
	if strings.EqualFold(s, "false") {
		return false
	}
	if f, err := strconv.ParseFloat(s, 64); err == nil {
		return f != 0
	}
	return true
}

func exprToNum(v exprVal, ctx string) (float64, error) {
	if v.isNum {
		return v.n, nil
	}
	f, err := strconv.ParseFloat(strings.TrimSpace(v.s), 64)
	if err != nil {
		return 0, fmt.Errorf("report: %s: value %q is not numeric", ctx, v.s)
	}
	return f, nil
}

// exprCalc evaluates one [...] or [[...]] body.
func exprCalc(body string, fields, params map[string]string) (string, error) {
	if strings.TrimSpace(body) == "" {
		return "", fmt.Errorf("report: empty [] reference")
	}
	p := &exprParser{s: body, fields: fields, params: params}
	v, err := p.parseCompare()
	if err != nil {
		return "", err
	}
	p.skipSpace()
	if p.pos != len(p.s) {
		return "", fmt.Errorf("report: unexpected %q in expression %q", p.s[p.pos:], body)
	}
	return v.s, nil
}

type exprParser struct {
	s      string
	pos    int
	fields map[string]string
	params map[string]string
}

func (p *exprParser) skipSpace() {
	for p.pos < len(p.s) && (p.s[p.pos] == ' ' || p.s[p.pos] == '\t' || p.s[p.pos] == '\n' || p.s[p.pos] == '\r') {
		p.pos++
	}
}

func (p *exprParser) parseCompare() (exprVal, error) {
	left, err := p.parseAdd()
	if err != nil {
		return exprVal{}, err
	}
	for {
		p.skipSpace()
		op := ""
		rest := p.s[p.pos:]
		for _, cand := range []string{"==", "!=", "<>", "<=", ">=", "=", "<", ">"} {
			if strings.HasPrefix(rest, cand) {
				op = cand
				break
			}
		}
		if op == "" {
			return left, nil
		}
		p.pos += len(op)
		right, err := p.parseAdd()
		if err != nil {
			return exprVal{}, err
		}
		var eq bool
		if left.isNum && right.isNum {
			eq = left.n == right.n
		} else {
			eq = left.s == right.s
		}
		switch op {
		case "=", "==":
			left = exprBool(eq)
		case "!=", "<>":
			left = exprBool(!eq)
		default:
			var less bool
			if left.isNum && right.isNum {
				less = left.n < right.n
			} else {
				less = left.s < right.s
			}
			switch op {
			case "<":
				left = exprBool(less)
			case ">":
				left = exprBool(!less && !eq)
			case "<=":
				left = exprBool(less || eq)
			case ">=":
				left = exprBool(!less || eq)
			}
		}
	}
}

func (p *exprParser) parseAdd() (exprVal, error) {
	left, err := p.parseMul()
	if err != nil {
		return exprVal{}, err
	}
	for {
		p.skipSpace()
		if p.pos >= len(p.s) || (p.s[p.pos] != '+' && p.s[p.pos] != '-') {
			return left, nil
		}
		op := p.s[p.pos]
		p.pos++
		right, err := p.parseMul()
		if err != nil {
			return exprVal{}, err
		}
		if op == '+' && (!left.isNum || !right.isNum) {
			left = exprStr(left.s + right.s)
			continue
		}
		lf, err := exprToNum(left, "expression")
		if err != nil {
			return exprVal{}, err
		}
		rf, err := exprToNum(right, "expression")
		if err != nil {
			return exprVal{}, err
		}
		if op == '+' {
			left = exprNum(lf + rf)
		} else {
			left = exprNum(lf - rf)
		}
	}
}

func (p *exprParser) parseMul() (exprVal, error) {
	left, err := p.parseUnary()
	if err != nil {
		return exprVal{}, err
	}
	for {
		p.skipSpace()
		if p.pos >= len(p.s) || (p.s[p.pos] != '*' && p.s[p.pos] != '/') {
			return left, nil
		}
		op := p.s[p.pos]
		p.pos++
		right, err := p.parseUnary()
		if err != nil {
			return exprVal{}, err
		}
		lf, err := exprToNum(left, "expression")
		if err != nil {
			return exprVal{}, err
		}
		rf, err := exprToNum(right, "expression")
		if err != nil {
			return exprVal{}, err
		}
		if op == '*' {
			left = exprNum(lf * rf)
		} else {
			if rf == 0 {
				return exprVal{}, fmt.Errorf("report: division by zero")
			}
			left = exprNum(lf / rf)
		}
	}
}

func (p *exprParser) parseUnary() (exprVal, error) {
	p.skipSpace()
	if p.pos < len(p.s) && (p.s[p.pos] == '-' || p.s[p.pos] == '+') {
		neg := p.s[p.pos] == '-'
		p.pos++
		v, err := p.parseUnary()
		if err != nil {
			return exprVal{}, err
		}
		f, err := exprToNum(v, "expression")
		if err != nil {
			return exprVal{}, err
		}
		if neg {
			return exprNum(-f), nil
		}
		return exprNum(f), nil
	}
	return p.parsePrimary()
}

func (p *exprParser) parsePrimary() (exprVal, error) {
	p.skipSpace()
	if p.pos >= len(p.s) {
		return exprVal{}, fmt.Errorf("report: unexpected end of expression")
	}
	c := p.s[p.pos]
	switch {
	case c == '(':
		p.pos++
		v, err := p.parseCompare()
		if err != nil {
			return exprVal{}, err
		}
		p.skipSpace()
		if p.pos >= len(p.s) || p.s[p.pos] != ')' {
			return exprVal{}, fmt.Errorf("report: unclosed ( in expression")
		}
		p.pos++
		return v, nil
	case c == '\'' || c == '"':
		return p.parseString()
	case c == '[':
		content, next, err := exprMatch(p.s, p.pos)
		if err != nil {
			return exprVal{}, fmt.Errorf("report: unclosed [ in expression")
		}
		p.pos = next
		content = strings.TrimSpace(content)
		if v, ok := exprAtom(content, p.fields, p.params); ok {
			return v, nil
		}
		if _, lerr := exprLookup(content, p.fields, p.params); !exprComplex(content) {
			return exprVal{}, lerr
		}
		sub := &exprParser{s: content, fields: p.fields, params: p.params}
		v, err := sub.parseCompare()
		if err != nil {
			return exprVal{}, err
		}
		sub.skipSpace()
		if sub.pos != len(sub.s) {
			return exprVal{}, fmt.Errorf("report: unexpected %q in expression %q", sub.s[sub.pos:], content)
		}
		return v, nil
	case c >= '0' && c <= '9' || c == '.':
		return p.parseNumber()
	case c == '_' || c >= 'a' && c <= 'z' || c >= 'A' && c <= 'Z':
		return p.parseIdent()
	}
	return exprVal{}, fmt.Errorf("report: unexpected character %q in expression", string(c))
}

func (p *exprParser) parseString() (exprVal, error) {
	q := p.s[p.pos]
	p.pos++
	var sb strings.Builder
	for {
		if p.pos >= len(p.s) {
			return exprVal{}, fmt.Errorf("report: unclosed string in expression")
		}
		if p.s[p.pos] == q {
			p.pos++
			return exprStr(sb.String()), nil
		}
		sb.WriteByte(p.s[p.pos])
		p.pos++
	}
}

func (p *exprParser) parseNumber() (exprVal, error) {
	start := p.pos
	dot := false
	for p.pos < len(p.s) {
		c := p.s[p.pos]
		if c >= '0' && c <= '9' {
			p.pos++
		} else if c == '.' && !dot {
			dot = true
			p.pos++
		} else {
			break
		}
	}
	f, err := strconv.ParseFloat(p.s[start:p.pos], 64)
	if err != nil {
		return exprVal{}, fmt.Errorf("report: bad number %q", p.s[start:p.pos])
	}
	return exprNum(f), nil
}

func (p *exprParser) parseIdent() (exprVal, error) {
	start := p.pos
	for p.pos < len(p.s) {
		c := p.s[p.pos]
		if c == '_' || c >= 'a' && c <= 'z' || c >= 'A' && c <= 'Z' || c >= '0' && c <= '9' || c == '.' {
			p.pos++
		} else {
			break
		}
	}
	name := p.s[start:p.pos]
	p.skipSpace()
	if p.pos < len(p.s) && p.s[p.pos] == '(' {
		return p.parseCall(name)
	}
	if v, ok := exprAtom(name, p.fields, p.params); ok {
		return v, nil
	}
	return exprVal{}, fmt.Errorf("report: unknown name %q", name)
}

// exprAtom resolves one bare or bracketed operand: the Now/Today
// constants, true/false, then fields and params. It reports false when
// name is none of those.
func exprAtom(name string, fields, params map[string]string) (exprVal, bool) {
	switch {
	case strings.EqualFold(name, "now"):
		return exprStr(time.Now().Format("2006-01-02 15:04:05")), true
	case strings.EqualFold(name, "today"):
		return exprStr(time.Now().Format("2006-01-02")), true
	case strings.EqualFold(name, "true"):
		return exprBool(true), true
	case strings.EqualFold(name, "false"):
		return exprBool(false), true
	}
	s, err := exprLookup(name, fields, params)
	if err != nil {
		return exprVal{}, false
	}
	if f, ferr := strconv.ParseFloat(strings.TrimSpace(s), 64); ferr == nil {
		return exprNum(f), true
	}
	return exprStr(s), true
}

func (p *exprParser) parseCall(name string) (exprVal, error) {
	p.pos++ // consume (
	var args []exprVal
	p.skipSpace()
	if p.pos < len(p.s) && p.s[p.pos] == ')' {
		p.pos++
	} else {
		for {
			a, err := p.parseCompare()
			if err != nil {
				return exprVal{}, err
			}
			args = append(args, a)
			p.skipSpace()
			if p.pos >= len(p.s) {
				return exprVal{}, fmt.Errorf("report: unclosed ( in call %s", name)
			}
			if p.s[p.pos] == ')' {
				p.pos++
				break
			}
			if p.s[p.pos] != ',' {
				return exprVal{}, fmt.Errorf("report: expected , in call %s", name)
			}
			p.pos++
		}
	}
	return exprCall(name, args)
}

func exprCall(name string, args []exprVal) (exprVal, error) {
	switch {
	case strings.EqualFold(name, "UpperCase"):
		if len(args) != 1 {
			return exprVal{}, fmt.Errorf("report: UpperCase wants 1 argument, got %d", len(args))
		}
		return exprStr(strings.ToUpper(args[0].s)), nil
	case strings.EqualFold(name, "LowerCase"):
		if len(args) != 1 {
			return exprVal{}, fmt.Errorf("report: LowerCase wants 1 argument, got %d", len(args))
		}
		return exprStr(strings.ToLower(args[0].s)), nil
	case strings.EqualFold(name, "Trim"):
		if len(args) != 1 {
			return exprVal{}, fmt.Errorf("report: Trim wants 1 argument, got %d", len(args))
		}
		return exprStr(strings.TrimSpace(args[0].s)), nil
	case strings.EqualFold(name, "Substring"):
		if len(args) != 2 && len(args) != 3 {
			return exprVal{}, fmt.Errorf("report: Substring wants 2 or 3 arguments, got %d", len(args))
		}
		r := []rune(args[0].s)
		start, err := exprToNum(args[1], "Substring")
		if err != nil {
			return exprVal{}, err
		}
		si := int(start)
		if si < 0 || si > len(r) {
			return exprVal{}, fmt.Errorf("report: Substring start %d out of range", si)
		}
		if len(args) == 2 {
			return exprStr(string(r[si:])), nil
		}
		ln, err := exprToNum(args[2], "Substring")
		if err != nil {
			return exprVal{}, err
		}
		li := int(ln)
		if li < 0 || si+li > len(r) {
			return exprVal{}, fmt.Errorf("report: Substring length %d out of range", li)
		}
		return exprStr(string(r[si : si+li])), nil
	case strings.EqualFold(name, "ToString"):
		if len(args) != 1 {
			return exprVal{}, fmt.Errorf("report: ToString wants 1 argument, got %d", len(args))
		}
		return exprStr(args[0].s), nil
	case strings.EqualFold(name, "Iif"):
		if len(args) != 3 {
			return exprVal{}, fmt.Errorf("report: Iif wants 3 arguments, got %d", len(args))
		}
		if exprTruth(args[0]) {
			return args[1], nil
		}
		return args[2], nil
	case strings.EqualFold(name, "Abs"):
		if len(args) != 1 {
			return exprVal{}, fmt.Errorf("report: Abs wants 1 argument, got %d", len(args))
		}
		f, err := exprToNum(args[0], "Abs")
		if err != nil {
			return exprVal{}, err
		}
		return exprNum(math.Abs(f)), nil
	case strings.EqualFold(name, "Round"):
		if len(args) != 1 && len(args) != 2 {
			return exprVal{}, fmt.Errorf("report: Round wants 1 or 2 arguments, got %d", len(args))
		}
		f, err := exprToNum(args[0], "Round")
		if err != nil {
			return exprVal{}, err
		}
		digits := 0
		if len(args) == 2 {
			d, err := exprToNum(args[1], "Round")
			if err != nil {
				return exprVal{}, err
			}
			digits = int(d)
			if digits < 0 {
				digits = 0
			}
		}
		mult := math.Pow(10, float64(digits))
		out := math.Round(f*mult) / mult
		if digits == 0 {
			return exprVal{s: strconv.FormatFloat(out, 'f', 0, 64), n: out, isNum: true}, nil
		}
		return exprVal{s: strconv.FormatFloat(out, 'f', digits, 64), n: out, isNum: true}, nil
	case strings.EqualFold(name, "Contains"):
		if len(args) != 2 {
			return exprVal{}, fmt.Errorf("report: Contains wants 2 arguments, got %d", len(args))
		}
		return exprBool(strings.Contains(args[0].s, args[1].s)), nil
	case strings.EqualFold(name, "Replace"):
		if len(args) != 3 {
			return exprVal{}, fmt.Errorf("report: Replace wants 3 arguments, got %d", len(args))
		}
		if args[1].s == "" {
			return exprVal{}, fmt.Errorf("report: Replace: old value must not be empty")
		}
		return exprStr(strings.ReplaceAll(args[0].s, args[1].s, args[2].s)), nil
	case strings.EqualFold(name, "Remove"):
		if len(args) != 2 && len(args) != 3 {
			return exprVal{}, fmt.Errorf("report: Remove wants 2 or 3 arguments, got %d", len(args))
		}
		r := []rune(args[0].s)
		start, err := exprToNum(args[1], "Remove")
		if err != nil {
			return exprVal{}, err
		}
		si := int(start)
		if si < 0 || si > len(r) {
			return exprVal{}, fmt.Errorf("report: Remove start %d out of range", si)
		}
		if len(args) == 2 {
			return exprStr(string(r[:si])), nil
		}
		count, err := exprToNum(args[2], "Remove")
		if err != nil {
			return exprVal{}, err
		}
		ci := int(count)
		if ci < 0 || si+ci > len(r) {
			return exprVal{}, fmt.Errorf("report: Remove count %d out of range", ci)
		}
		return exprStr(string(r[:si]) + string(r[si+ci:])), nil
	case strings.EqualFold(name, "PadLeft"):
		return exprPad(args, false)
	case strings.EqualFold(name, "PadRight"):
		return exprPad(args, true)
	case strings.EqualFold(name, "Insert"):
		if len(args) != 3 {
			return exprVal{}, fmt.Errorf("report: Insert wants 3 arguments, got %d", len(args))
		}
		r := []rune(args[0].s)
		at, err := exprToNum(args[1], "Insert")
		if err != nil {
			return exprVal{}, err
		}
		si := int(at)
		if si < 0 || si > len(r) {
			return exprVal{}, fmt.Errorf("report: Insert start %d out of range", si)
		}
		return exprStr(string(r[:si]) + args[2].s + string(r[si:])), nil
	case strings.EqualFold(name, "Length"):
		if len(args) != 1 {
			return exprVal{}, fmt.Errorf("report: Length wants 1 argument, got %d", len(args))
		}
		return exprNum(float64(len([]rune(args[0].s)))), nil
	case strings.EqualFold(name, "Chr"):
		if len(args) != 1 {
			return exprVal{}, fmt.Errorf("report: Chr wants 1 argument, got %d", len(args))
		}
		code, err := exprToNum(args[0], "Chr")
		if err != nil {
			return exprVal{}, err
		}
		i := int(code)
		if i < 0 || i > unicode.MaxRune {
			return exprVal{}, fmt.Errorf("report: Chr code %d out of range", i)
		}
		return exprStr(string(rune(i))), nil
	case strings.EqualFold(name, "Asc"):
		if len(args) != 1 {
			return exprVal{}, fmt.Errorf("report: Asc wants 1 argument, got %d", len(args))
		}
		r := []rune(args[0].s)
		if len(r) == 0 {
			return exprVal{}, fmt.Errorf("report: Asc: empty string has no character code")
		}
		return exprNum(float64(r[0])), nil
	case strings.EqualFold(name, "TitleCase"):
		if len(args) != 1 {
			return exprVal{}, fmt.Errorf("report: TitleCase wants 1 argument, got %d", len(args))
		}
		return exprStr(exprTitleCase(args[0].s)), nil
	case strings.EqualFold(name, "AddDays"):
		if len(args) != 2 {
			return exprVal{}, fmt.Errorf("report: AddDays wants 2 arguments, got %d", len(args))
		}
		t, hasTime, ok := exprParseStdDate(args[0].s)
		if !ok {
			return exprStr(""), nil
		}
		days, err := exprToNum(args[1], "AddDays")
		if err != nil {
			return exprVal{}, err
		}
		t = t.Add(time.Duration(days * float64(24*time.Hour)))
		return exprStr(exprDateStr(t, hasTime)), nil
	case strings.EqualFold(name, "AddMonths"):
		if len(args) != 2 {
			return exprVal{}, fmt.Errorf("report: AddMonths wants 2 arguments, got %d", len(args))
		}
		t, hasTime, ok := exprParseStdDate(args[0].s)
		if !ok {
			return exprStr(""), nil
		}
		months, err := exprToNum(args[1], "AddMonths")
		if err != nil {
			return exprVal{}, err
		}
		return exprStr(exprDateStr(exprAddMonths(t, int(months)), hasTime)), nil
	case strings.EqualFold(name, "AddYears"):
		if len(args) != 2 {
			return exprVal{}, fmt.Errorf("report: AddYears wants 2 arguments, got %d", len(args))
		}
		t, hasTime, ok := exprParseStdDate(args[0].s)
		if !ok {
			return exprStr(""), nil
		}
		years, err := exprToNum(args[1], "AddYears")
		if err != nil {
			return exprVal{}, err
		}
		return exprStr(exprDateStr(exprAddMonths(t, int(years)*12), hasTime)), nil
	case strings.EqualFold(name, "Day"):
		if len(args) != 1 {
			return exprVal{}, fmt.Errorf("report: Day wants 1 argument, got %d", len(args))
		}
		t, _, ok := exprParseStdDate(args[0].s)
		if !ok {
			return exprStr(""), nil
		}
		return exprNum(float64(t.Day())), nil
	case strings.EqualFold(name, "Month"):
		if len(args) != 1 {
			return exprVal{}, fmt.Errorf("report: Month wants 1 argument, got %d", len(args))
		}
		t, _, ok := exprParseStdDate(args[0].s)
		if !ok {
			return exprStr(""), nil
		}
		return exprNum(float64(int(t.Month()))), nil
	case strings.EqualFold(name, "Year"):
		if len(args) != 1 {
			return exprVal{}, fmt.Errorf("report: Year wants 1 argument, got %d", len(args))
		}
		t, _, ok := exprParseStdDate(args[0].s)
		if !ok {
			return exprStr(""), nil
		}
		return exprNum(float64(t.Year())), nil
	case strings.EqualFold(name, "DayOfWeek"):
		if len(args) != 1 {
			return exprVal{}, fmt.Errorf("report: DayOfWeek wants 1 argument, got %d", len(args))
		}
		t, _, ok := exprParseStdDate(args[0].s)
		if !ok {
			return exprStr(""), nil
		}
		return exprStr(t.Weekday().String()), nil
	case strings.EqualFold(name, "DaysInMonth"):
		if len(args) != 2 {
			return exprVal{}, fmt.Errorf("report: DaysInMonth wants 2 arguments, got %d", len(args))
		}
		year, err := exprToNum(args[0], "DaysInMonth")
		if err != nil {
			return exprVal{}, err
		}
		month, err := exprToNum(args[1], "DaysInMonth")
		if err != nil {
			return exprVal{}, err
		}
		y, m := int(year), int(month)
		if m < 1 || m > 12 {
			return exprVal{}, fmt.Errorf("report: DaysInMonth month %d out of range", m)
		}
		return exprNum(float64(exprDaysInMonth(y, m))), nil
	case strings.EqualFold(name, "DateSerial"):
		if len(args) != 3 {
			return exprVal{}, fmt.Errorf("report: DateSerial wants 3 arguments, got %d", len(args))
		}
		year, err := exprToNum(args[0], "DateSerial")
		if err != nil {
			return exprVal{}, err
		}
		month, err := exprToNum(args[1], "DateSerial")
		if err != nil {
			return exprVal{}, err
		}
		day, err := exprToNum(args[2], "DateSerial")
		if err != nil {
			return exprVal{}, err
		}
		y, m, d := int(year), int(month), int(day)
		if m < 1 || m > 12 {
			return exprVal{}, fmt.Errorf("report: DateSerial month %d out of range", m)
		}
		if d < 1 || d > exprDaysInMonth(y, m) {
			return exprVal{}, fmt.Errorf("report: DateSerial day %d out of range", d)
		}
		return exprStr(time.Date(y, time.Month(m), d, 0, 0, 0, 0, time.UTC).Format(exprDateLayout)), nil
	case strings.EqualFold(name, "Maximum"):
		if len(args) != 2 {
			return exprVal{}, fmt.Errorf("report: Maximum wants 2 arguments, got %d", len(args))
		}
		a, err := exprToNum(args[0], "Maximum")
		if err != nil {
			return exprVal{}, err
		}
		b, err := exprToNum(args[1], "Maximum")
		if err != nil {
			return exprVal{}, err
		}
		return exprNum(math.Max(a, b)), nil
	case strings.EqualFold(name, "Minimum"):
		if len(args) != 2 {
			return exprVal{}, fmt.Errorf("report: Minimum wants 2 arguments, got %d", len(args))
		}
		a, err := exprToNum(args[0], "Minimum")
		if err != nil {
			return exprVal{}, err
		}
		b, err := exprToNum(args[1], "Minimum")
		if err != nil {
			return exprVal{}, err
		}
		return exprNum(math.Min(a, b)), nil
	case strings.EqualFold(name, "Switch"):
		if len(args) < 2 || len(args)%2 != 0 {
			return exprVal{}, fmt.Errorf("report: Switch wants an even number of arguments, got %d", len(args))
		}
		for i := 0; i < len(args); i += 2 {
			if exprTruth(args[i]) {
				return args[i+1], nil
			}
		}
		return exprStr(""), nil
	}
	return exprVal{}, fmt.Errorf("report: unknown function %q", name)
}

// exprPad implements PadLeft and PadRight. Width counts runes, matching
// Substring. The optional third argument supplies the padding character
// (its first rune); a missing or empty padding defaults to a space.
func exprPad(args []exprVal, right bool) (exprVal, error) {
	name := "PadLeft"
	if right {
		name = "PadRight"
	}
	if len(args) != 2 && len(args) != 3 {
		return exprVal{}, fmt.Errorf("report: %s wants 2 or 3 arguments, got %d", name, len(args))
	}
	r := []rune(args[0].s)
	width, err := exprToNum(args[1], name)
	if err != nil {
		return exprVal{}, err
	}
	w := int(width)
	if w < 0 {
		return exprVal{}, fmt.Errorf("report: %s width %d out of range", name, w)
	}
	if len(r) >= w {
		return exprStr(args[0].s), nil
	}
	pad := ' '
	if len(args) == 3 && len([]rune(args[2].s)) > 0 {
		pad = []rune(args[2].s)[0]
	}
	fill := strings.Repeat(string(pad), w-len(r))
	if right {
		return exprStr(args[0].s + fill), nil
	}
	return exprStr(fill + args[0].s), nil
}

// exprTitleCase uppercases the first letter of each word and lowercases
// the rest. A word starts after any non-letter rune.
func exprTitleCase(s string) string {
	r := []rune(strings.ToLower(s))
	prevLetter := false
	for i, c := range r {
		if unicode.IsLetter(c) {
			if !prevLetter {
				r[i] = unicode.ToUpper(c)
			}
			prevLetter = true
			continue
		}
		prevLetter = false
	}
	return string(r)
}

const (
	exprDateLayout     = "2006-01-02"
	exprDateTimeLayout = "2006-01-02 15:04:05"
)

// exprParseStdDate parses a date with the shared exprParseDate helper
// (format.go). It reports hasTime when the input carries a clock part
// and ok=false for anything else; callers turn unparseable input into
// an empty string.
func exprParseStdDate(s string) (t time.Time, hasTime bool, ok bool) {
	s = strings.TrimSpace(s)
	t, ok = exprParseDate(s)
	if !ok {
		return time.Time{}, false, false
	}
	return t, len(s) > len(exprDateLayout), true
}

// exprDateStr formats t as a date, upgrading to a datetime only when the
// input had a time part or the result carries a nonzero clock.
func exprDateStr(t time.Time, hasTime bool) string {
	if hasTime || t.Hour() != 0 || t.Minute() != 0 || t.Second() != 0 {
		return t.Format(exprDateTimeLayout)
	}
	return t.Format(exprDateLayout)
}

// exprAddMonths shifts t by months, clamping the day to the last day of
// the target month like FastReport AddMonths.
func exprAddMonths(t time.Time, months int) time.Time {
	y, m, d := t.Date()
	total := y*12 + int(m) - 1 + months
	ny, nm0 := total/12, total%12
	if nm0 < 0 {
		nm0 += 12
		ny--
	}
	nm := time.Month(nm0 + 1)
	if dim := exprDaysInMonth(ny, int(nm)); d > dim {
		d = dim
	}
	return time.Date(ny, nm, d, t.Hour(), t.Minute(), t.Second(), t.Nanosecond(), t.Location())
}

func exprDaysInMonth(year, month int) int {
	return time.Date(year, time.Month(month)+1, 0, 0, 0, 0, 0, time.UTC).Day()
}
