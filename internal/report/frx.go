package report

import (
	"encoding/xml"
	"fmt"
	"io"
	"sort"
	"strings"
)

// ParseFRX reads a .frx document into the P0 IR. Whitespace-only
// character data is dropped; everything else is preserved in order.
func ParseFRX(r io.Reader) (Report, error) {
	dec := xml.NewDecoder(r)
	for {
		tok, err := dec.Token()
		if err != nil {
			return Report{}, err
		}
		if start, ok := tok.(xml.StartElement); ok {
			root, err := decodeElement(dec, start)
			if err != nil {
				return Report{}, err
			}
			if root.Name != "Report" {
				return Report{}, fmt.Errorf("report: root is %q, want Report", root.Name)
			}
			return reportOf(root), nil
		}
	}
}

func decodeElement(dec *xml.Decoder, start xml.StartElement) (Node, error) {
	n := Node{Name: start.Name.Local}
	for _, a := range start.Attr {
		name := a.Name.Local
		if a.Name.Space != "" {
			name = a.Name.Space + ":" + name
		}
		n.Attrs = append(n.Attrs, Attr{name, a.Value})
	}
	for {
		tok, err := dec.Token()
		if err != nil {
			return n, err
		}
		switch t := tok.(type) {
		case xml.StartElement:
			c, err := decodeElement(dec, t)
			if err != nil {
				return n, err
			}
			n.Children = append(n.Children, c)
		case xml.CharData:
			if strings.TrimSpace(string(t)) != "" {
				n.Text += string(t)
			}
		case xml.EndElement:
			return n, nil
		}
	}
}

func reportOf(root Node) Report {
	rep := Report{Version: IRVersion}
	for _, a := range root.Attrs {
		switch {
		case a.Name == "ScriptLanguage":
			rep.ScriptLanguage = a.Value
		case strings.HasPrefix(a.Name, "ReportInfo."):
			if rep.Info == nil {
				rep.Info = map[string]string{}
			}
			rep.Info[strings.TrimPrefix(a.Name, "ReportInfo.")] = a.Value
		default:
			rep.Attrs = append(rep.Attrs, a)
		}
	}
	for _, c := range root.Children {
		switch c.Name {
		case "Styles":
			for _, s := range c.Children {
				rep.Styles = append(rep.Styles, Style{Name: attrVal(s.Attrs, "Name"), Attrs: withoutName(s.Attrs)})
			}
		case "Dictionary":
			rep.Dictionary = c
		case "ReportPage":
			rep.Pages = append(rep.Pages, pageOf(c))
		default:
			rep.Extra = append(rep.Extra, c)
		}
	}
	return rep
}

func withoutName(a []Attr) []Attr {
	out := a[:0:0]
	for _, at := range a {
		if at.Name != "Name" {
			out = append(out, at)
		}
	}
	return out
}

func pageOf(n Node) Page {
	p := Page{Name: attrVal(n.Attrs, "Name"), Attrs: withoutName(n.Attrs)}
	for _, c := range n.Children {
		if strings.HasSuffix(c.Name, "Band") {
			b := Band{Kind: c.Name, Name: attrVal(c.Attrs, "Name"), Attrs: withoutName(c.Attrs)}
			for _, o := range c.Children {
				b.Objects = append(b.Objects, Object{Kind: o.Name, Name: attrVal(o.Attrs, "Name"), Attrs: withoutName(o.Attrs), Children: o.Children})
			}
			p.Bands = append(p.Bands, b)
		} else {
			p.Extra = append(p.Extra, c)
		}
	}
	return p
}

// WriteFRX serializes the IR back to .frx XML.
func WriteFRX(w io.Writer, rep Report) error {
	root := Node{Name: "Report"}
	if rep.ScriptLanguage != "" {
		root.Attrs = append(root.Attrs, Attr{"ScriptLanguage", rep.ScriptLanguage})
	}
	for _, k := range sortedInfoKeys(rep.Info) {
		root.Attrs = append(root.Attrs, Attr{"ReportInfo." + k, rep.Info[k]})
	}
	root.Attrs = append(root.Attrs, rep.Attrs...)
	if len(rep.Styles) > 0 {
		sn := Node{Name: "Styles"}
		for _, s := range rep.Styles {
			sn.Children = append(sn.Children, Node{Name: "Style", Attrs: append([]Attr{{"Name", s.Name}}, s.Attrs...)})
		}
		root.Children = append(root.Children, sn)
	}
	dict := rep.Dictionary
	dict.Name = "Dictionary"
	root.Children = append(root.Children, dict)
	for _, p := range rep.Pages {
		pn := Node{Name: "ReportPage", Attrs: append([]Attr{{"Name", p.Name}}, p.Attrs...)}
		for _, b := range p.Bands {
			bn := Node{Name: b.Kind, Attrs: append([]Attr{{"Name", b.Name}}, b.Attrs...)}
			for _, o := range b.Objects {
				bn.Children = append(bn.Children, Node{Name: o.Kind, Attrs: append([]Attr{{"Name", o.Name}}, o.Attrs...), Children: o.Children})
			}
			pn.Children = append(pn.Children, bn)
		}
		pn.Children = append(pn.Children, p.Extra...)
		root.Children = append(root.Children, pn)
	}
	root.Children = append(root.Children, rep.Extra...)

	if _, err := io.WriteString(w, xml.Header); err != nil {
		return err
	}
	enc := xml.NewEncoder(w)
	enc.Indent("", "  ")
	if err := writeNode(enc, root); err != nil {
		return err
	}
	return enc.Flush()
}

func writeNode(enc *xml.Encoder, n Node) error {
	el := xml.StartElement{Name: xml.Name{Local: n.Name}}
	for _, a := range n.Attrs {
		el.Attr = append(el.Attr, xml.Attr{Name: xml.Name{Local: a.Name}, Value: a.Value})
	}
	if err := enc.EncodeToken(el); err != nil {
		return err
	}
	if n.Text != "" {
		if err := enc.EncodeToken(xml.CharData(n.Text)); err != nil {
			return err
		}
	}
	for _, c := range n.Children {
		if err := writeNode(enc, c); err != nil {
			return err
		}
	}
	return enc.EncodeToken(el.End())
}

func sortedInfoKeys(m map[string]string) []string {
	out := make([]string, 0, len(m))
	for k := range m {
		out = append(out, k)
	}
	sort.Strings(out)
	return out
}
