package report

// IRVersion is the current report IR version.
const IRVersion = "report/v1"

// Attr is one XML attribute, kept in document order.
type Attr struct {
	Name  string `json:"name"`
	Value string `json:"value"`
}

// Report is the P0 IR: enough to carry one .frx page through a
// parse -> write -> parse round-trip without losing data.
// Unknown elements (e.g. DialogPage) ride along as opaque Nodes.
type Report struct {
	Version        string            `json:"version"`
	ScriptLanguage string            `json:"scriptLanguage,omitempty"`
	Info           map[string]string `json:"info,omitempty"`  // ReportInfo.* dotted attributes
	Attrs          []Attr            `json:"attrs,omitempty"` // other root attributes, in order
	Styles         []Style           `json:"styles,omitempty"`
	Dictionary     Node              `json:"dictionary"`
	Pages          []Page            `json:"pages"`
	Extra          []Node            `json:"extra,omitempty"` // DialogPage etc.
}

// Style is one named entry of the Styles sheet.
type Style struct {
	Name  string `json:"name"`
	Attrs []Attr `json:"attrs,omitempty"`
}

// Node is an opaque element: name, attributes, text, children.
type Node struct {
	Name     string `json:"name"`
	Attrs    []Attr `json:"attrs,omitempty"`
	Text     string `json:"text,omitempty"`
	Children []Node `json:"children,omitempty"`
}

// Page is one ReportPage.
type Page struct {
	Name  string `json:"name"`
	Attrs []Attr `json:"attrs,omitempty"`
	Bands []Band `json:"bands"`
	Extra []Node `json:"extra,omitempty"` // non-band children, preserved verbatim
}

// Band is one *Band element.
type Band struct {
	Kind    string   `json:"kind"` // ReportTitleBand, PageFooterBand, ...
	Name    string   `json:"name"`
	Attrs   []Attr   `json:"attrs,omitempty"`
	Objects []Object `json:"objects"`
}

// Object is one report object (TextObject, ...). Children holds nested
// content such as TableObject columns/rows and TableCell objects; P5 types
// them, P0/P1 preserve them verbatim.
type Object struct {
	Kind     string `json:"kind"`
	Name     string `json:"name"`
	Attrs    []Attr `json:"attrs,omitempty"`
	Children []Node `json:"children,omitempty"`
}

func attrVal(a []Attr, name string) string {
	for _, at := range a {
		if at.Name == name {
			return at.Value
		}
	}
	return ""
}
