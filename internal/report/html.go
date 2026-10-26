package report

import (
	"fmt"
	"html"
	"strconv"
	"strings"
)

// RenderBand renders one band to a self-contained HTML fragment.
// Units are FastReport pixels (1/96 inch). P0 supports TextObject only;
// unknown objects render as empty boxes so layout stays visible.
func RenderBand(b Band) string {
	var sb strings.Builder
	fmt.Fprintf(&sb, `<div class="band" style="position:relative;width:%spx;height:%spx;">`,
		num(b.Attrs, "Width"), num(b.Attrs, "Height"))
	for _, o := range b.Objects {
		sb.WriteString(renderObject(o))
	}
	sb.WriteString(`</div>`)
	return sb.String()
}

func renderObject(o Object) string {
	switch o.Kind {
	case "LineObject":
		return objLineHTML(o)
	case "ShapeObject":
		return objShapeHTML(o)
	case "CheckBoxObject":
		return objCheckBoxHTML(o)
	case "PictureObject":
		return objPictureHTML(o)
	case "BarcodeObject":
		return objBarcodeHTML(o)
	case "TableObject":
		return tabRenderTableHTML(o)
	case "MatrixObject":
		return vizMatrixHTML(o)
	case "LinearGauge", "SimpleGauge", "RadialGauge", "SimpleProgressGauge":
		return vizGaugeHTML(o)
	case "SubreportObject":
		return tabSubreportHTML(o)
	case "TextObject":
		style := fmt.Sprintf("position:absolute;left:%spx;top:%spx;width:%spx;height:%spx;%s",
			num(o.Attrs, "Left"), num(o.Attrs, "Top"),
			num(o.Attrs, "Width"), num(o.Attrs, "Height"), textStyle(o.Attrs))
		inner := html.EscapeString(attrVal(o.Attrs, "Text"))
		if v := objHyperlinkOf(o); v != "" {
			inner = objLinkHTML(v, inner)
		}
		return fmt.Sprintf(`<div class="text" style="%s">%s</div>`,
			style, inner)
	default:
		return objFallbackHTML(o)
	}
}

// objHyperlinkOf reads a TextObject hyperlink target. Real .frx files
// carry it as a dotted attribute (Hyperlink.Value="..."); the nested
// <Hyperlink Value="..."/> element form is accepted too.
func objHyperlinkOf(o Object) string {
	if v := strings.TrimSpace(attrVal(o.Attrs, "Hyperlink.Value")); v != "" {
		return v
	}
	for _, c := range o.Children {
		if c.Name == "Hyperlink" {
			if v := strings.TrimSpace(attrVal(c.Attrs, "Value")); v != "" {
				return v
			}
		}
	}
	return ""
}

// objLinkHTML wraps already-escaped inner text in an anchor for URL targets
// (http/https/mailto open in a new tab) or a page-jump anchor carrying
// data-goto-page for plain integer targets. Anything else renders as-is.
func objLinkHTML(link, inner string) string {
	v := strings.TrimSpace(link)
	low := strings.ToLower(v)
	if strings.HasPrefix(low, "http://") || strings.HasPrefix(low, "https://") ||
		strings.HasPrefix(low, "mailto:") {
		return fmt.Sprintf(`<a href="%s" target="_blank" rel="noopener">%s</a>`,
			html.EscapeString(v), inner)
	}
	if _, err := strconv.Atoi(v); err == nil {
		return fmt.Sprintf(`<a data-goto-page="%s">%s</a>`,
			html.EscapeString(v), inner)
	}
	return inner
}

func num(a []Attr, name string) string {
	v := attrVal(a, name)
	if v == "" {
		return "0"
	}
	if _, err := strconv.ParseFloat(v, 64); err != nil {
		return "0"
	}
	return v
}

// textStyle converts Font="Name, 10pt, style=Bold" and TextFill.Color.
func textStyle(a []Attr) string {
	var sb strings.Builder
	if strings.Contains(attrVal(a, "Text"), "\n") {
		sb.WriteString("white-space:pre-wrap;")
	}
	// CanGrow boxes are measured server-side with engLineHeightFactor, so
	// pin the browser to the same rhythm or grown text re-wraps and spills.
	if strings.EqualFold(strings.TrimSpace(attrVal(a, "CanGrow")), "true") {
		sb.WriteString("line-height:115%;")
	}
	if c := attrVal(a, "TextFill.Color"); c != "" {
		fmt.Fprintf(&sb, "color:%s;", c)
	}
	font := attrVal(a, "Font")
	if font == "" {
		return sb.String()
	}
	parts := strings.Split(font, ",")
	name := strings.TrimSpace(parts[0])
	fmt.Fprintf(&sb, "font-family:'%s';", name)
	for _, p := range parts[1:] {
		p = strings.TrimSpace(p)
		if strings.HasSuffix(p, "pt") {
			if f, err := strconv.ParseFloat(strings.TrimSuffix(p, "pt"), 64); err == nil {
				fmt.Fprintf(&sb, "font-size:%.2fpx;", f*96.0/72.0)
			}
		} else if strings.HasPrefix(p, "style=") {
			for _, s := range strings.Split(strings.TrimPrefix(p, "style="), " ") {
				switch strings.TrimSpace(s) {
				case "Bold":
					sb.WriteString("font-weight:bold;")
				case "Italic":
					sb.WriteString("font-style:italic;")
				case "Underline":
					sb.WriteString("text-decoration:underline;")
				}
			}
		}
	}
	switch attrVal(a, "HorzAlign") {
	case "Center":
		sb.WriteString("text-align:center;")
	case "Right":
		sb.WriteString("text-align:right;")
	case "Justify":
		sb.WriteString("text-align:justify;")
	}
	if attrVal(a, "VertAlign") == "Center" {
		sb.WriteString("display:flex;align-items:center;")
	}
	return sb.String()
}
