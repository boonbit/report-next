package report

import "fmt"

// Severity classifies a validation issue.
type Severity string

const (
	SeverityError   Severity = "error"
	SeverityWarning Severity = "warning"
)

// Issue is one validation finding.
type Issue struct {
	Path     string   `json:"path"`
	Message  string   `json:"message"`
	Severity Severity `json:"severity"`
}

func (i Issue) String() string {
	return fmt.Sprintf("%s: %s (%s)", i.Path, i.Message, i.Severity)
}

// Validate checks a report and returns every issue found. A report is
// accepted when no issue has SeverityError.
func Validate(rep Report) []Issue {
	var issues []Issue
	add := func(path, format string, args ...any) {
		issues = append(issues, Issue{Path: path, Message: fmt.Sprintf(format, args...), Severity: SeverityError})
	}
	if rep.Version != IRVersion {
		add("version", "unsupported version %q, want %q", rep.Version, IRVersion)
	}
	styles := map[string]bool{}
	for i, s := range rep.Styles {
		p := fmt.Sprintf("styles[%d]", i)
		if s.Name == "" {
			add(p, "style name is empty")
		} else if styles[s.Name] {
			add(p, "duplicate style %q", s.Name)
		}
		styles[s.Name] = true
	}
	seenPages := map[string]bool{}
	for pi, p := range rep.Pages {
		pp := fmt.Sprintf("pages[%d]", pi)
		if p.Name == "" {
			add(pp, "page name is empty")
		} else if seenPages[p.Name] {
			add(pp, "duplicate page %q", p.Name)
		}
		seenPages[p.Name] = true
		seenBands := map[string]bool{}
		for bi, b := range p.Bands {
			bp := fmt.Sprintf("%s.bands[%d]", pp, bi)
			if b.Kind == "" {
				add(bp, "band kind is empty")
			}
			if b.Name == "" {
				add(bp, "band name is empty")
			} else if seenBands[b.Name] {
				add(bp, "duplicate band %q", b.Name)
			}
			seenBands[b.Name] = true
			seenObjs := map[string]bool{}
			for oi, o := range b.Objects {
				op := fmt.Sprintf("%s.objects[%d]", bp, oi)
				if o.Kind == "" {
					add(op, "object kind is empty")
				}
				if o.Name == "" {
					add(op, "object name is empty")
				} else if seenObjs[o.Name] {
					add(op, "duplicate object %q", o.Name)
				}
				seenObjs[o.Name] = true
				if st := attrVal(o.Attrs, "Style"); st != "" && !styles[st] {
					add(op, "unknown style %q", st)
				}
			}
		}
	}
	return issues
}

// Valid reports whether Validate found no errors.
func Valid(rep Report) bool {
	for _, is := range Validate(rep) {
		if is.Severity == SeverityError {
			return false
		}
	}
	return true
}
