package report

import (
	"fmt"
	"html"
	"strconv"
	"strings"
)

// TRACK tab (P5 render): TableObject HTML renderer + SubreportObject
// placeholder.
//
// Real attribute names sampled from internal/report/testdata/table.frx
// (byte-identical Table1/Table2/Table3 to FastReport/Demos/Reports/Table.frx)
// and from Pictures Inside The Matrix.frx:
//   TableObject     Left Top Width Height
//   TableColumn     Width AutoSize MinWidth MaxWidth
//   TableRow        Height AutoSize
//   TableCell       Text Font HorzAlign VertAlign Fill.Color Border.Lines
//                   Border.Color Padding ColSpan RowSpan, plus nested objects
//                   (CheckBoxObject in Cell13, PictureObject in Cell22)
//   SubreportObject Left Top Width Height ReportPage PrintOnParent
//                   (sampled from FastReport/Demos/Reports/Subreport.frx)

// tabColumn is one TableColumn: only Width styles the output.
type tabColumn struct {
	name  string
	width string
}

// tabCell is one TableCell with parsed spans and nested objects.
type tabCell struct {
	name    string
	attrs   []Attr
	colSpan int
	rowSpan int
	objects []Object
}

// tabRow is one TableRow with its cells in document order.
type tabRow struct {
	name  string
	attrs []Attr
	cells []tabCell
}

// tabTable is the parsed form of a TableObject's
// TableColumn/TableRow/TableCell children.
type tabTable struct {
	obj     Object
	columns []tabColumn
	rows    []tabRow
}

// tabObjectOf converts a verbatim child Node (TableCell content such as a
// nested CheckBoxObject, or a TableObject under a ChildBand) to an Object,
// following the pageOf convention (Name split out of Attrs).
func tabObjectOf(n Node) Object {
	return Object{Kind: n.Name, Name: attrVal(n.Attrs, "Name"), Attrs: withoutName(n.Attrs), Children: n.Children}
}

// tabSpan parses a ColSpan/RowSpan value: empty means 1.
func tabSpan(table, cell, attr, v string) (int, error) {
	if strings.TrimSpace(v) == "" {
		return 0, nil
	}
	n, err := strconv.Atoi(strings.TrimSpace(v))
	if err != nil || n < 1 {
		return 0, fmt.Errorf("report: table %q cell %q bad %s %q", table, cell, attr, v)
	}
	return n, nil
}

// tabParseTable types a TableObject's verbatim children into columns, rows
// and cells. Unknown row children (anything but TableCell) are skipped;
// nested cell objects are kept as Objects for the renderer.
func tabParseTable(o Object) (*tabTable, error) {
	if o.Kind != "TableObject" {
		return nil, fmt.Errorf("report: tabParseTable wants TableObject, got %q", o.Kind)
	}
	t := &tabTable{obj: o}
	for _, c := range o.Children {
		switch c.Name {
		case "TableColumn":
			t.columns = append(t.columns, tabColumn{
				name:  attrVal(c.Attrs, "Name"),
				width: strings.TrimSpace(attrVal(c.Attrs, "Width")),
			})
		case "TableRow":
			r := tabRow{name: attrVal(c.Attrs, "Name"), attrs: c.Attrs}
			for _, n := range c.Children {
				if n.Name != "TableCell" {
					continue
				}
				cell := tabCell{name: attrVal(n.Attrs, "Name"), attrs: withoutName(n.Attrs)}
				var err error
				if cell.colSpan, err = tabSpan(o.Name, cell.name, "ColSpan", attrVal(n.Attrs, "ColSpan")); err != nil {
					return nil, err
				}
				if cell.rowSpan, err = tabSpan(o.Name, cell.name, "RowSpan", attrVal(n.Attrs, "RowSpan")); err != nil {
					return nil, err
				}
				for _, nn := range n.Children {
					cell.objects = append(cell.objects, tabObjectOf(nn))
				}
				r.cells = append(r.cells, cell)
			}
			t.rows = append(t.rows, r)
		}
	}
	if len(t.columns) == 0 && len(t.rows) == 0 {
		return nil, fmt.Errorf("report: table %q has no TableColumn/TableRow children", o.Name)
	}
	return t, nil
}

// tabColor normalizes a Fill.Color value for CSS: "r, g, b" triples become
// rgb(...); Transparent means no background; names pass through as-is.
func tabColor(v string) string {
	v = strings.TrimSpace(v)
	if v == "" || strings.EqualFold(v, "Transparent") {
		return ""
	}
	parts := strings.Split(v, ",")
	if len(parts) == 3 {
		nums := make([]string, 0, 3)
		for _, p := range parts {
			p = strings.TrimSpace(p)
			if _, err := strconv.Atoi(p); err != nil {
				return v
			}
			nums = append(nums, p)
		}
		return "rgb(" + strings.Join(nums, ",") + ")"
	}
	return v
}

// tabCellStyle builds a <td> style: column width, cell border, fill and the
// shared text style (font, color, alignment).
func tabCellStyle(a []Attr, cols []tabColumn, idx int) string {
	var sb strings.Builder
	if idx < len(cols) {
		if w := cols[idx].width; w != "" {
			if f, err := strconv.ParseFloat(w, 64); err == nil {
				fmt.Fprintf(&sb, "width:%gpx;", f)
			}
		}
	}
	if lines := strings.TrimSpace(attrVal(a, "Border.Lines")); lines != "" && !strings.EqualFold(lines, "None") {
		fmt.Fprintf(&sb, "border:%gpx solid %s;", objBorderWidth(a), objStrokeColor(a))
	}
	if c := tabColor(attrVal(a, "Fill.Color")); c != "" {
		sb.WriteString("background:" + c + ";")
	}
	sb.WriteString(textStyle(a))
	return sb.String()
}

// tabRenderTableHTML renders a TableObject as an absolutely-positioned
// <table>: column widths from TableColumn Width, row heights from TableRow
// Height, ColSpan/RowSpan on cells (both attested in table.frx: Cell1
// ColSpan="3", Cell4 RowSpan="3", Cell23 RowSpan="4"), and nested cell
// objects via RenderBand on a synthetic Band. Unparseable tables keep the
// labeled-box fallback so layout stays visible.
func tabRenderTableHTML(o Object) string {
	t, err := tabParseTable(o)
	if err != nil {
		return objFallbackHTML(o)
	}
	var sb strings.Builder
	fmt.Fprintf(&sb, `<div class="table" data-kind="TableObject" data-name="%s" style="%s">`,
		html.EscapeString(o.Name), objAbsStyle(o.Attrs, "overflow:hidden;"))
	sb.WriteString(`<table style="border-collapse:collapse;width:100%;height:100%;table-layout:fixed;">`)
	if len(t.columns) > 0 {
		sb.WriteString("<colgroup>")
		for _, c := range t.columns {
			if f, err := strconv.ParseFloat(c.width, 64); err == nil && c.width != "" {
				fmt.Fprintf(&sb, `<col style="width:%gpx;">`, f)
			} else {
				sb.WriteString("<col>")
			}
		}
		sb.WriteString("</colgroup>")
	}
	for _, r := range t.rows {
		sb.WriteString("<tr")
		if h := strings.TrimSpace(attrVal(r.attrs, "Height")); h != "" {
			if f, err := strconv.ParseFloat(h, 64); err == nil {
				fmt.Fprintf(&sb, ` style="height:%gpx;"`, f)
			}
		}
		sb.WriteString(">")
		for i, cell := range r.cells {
			sb.WriteString("<td")
			if cell.name != "" {
				fmt.Fprintf(&sb, ` data-cell="%s"`, html.EscapeString(cell.name))
			}
			if cell.colSpan > 1 {
				fmt.Fprintf(&sb, ` colspan="%d"`, cell.colSpan)
			}
			if cell.rowSpan > 1 {
				fmt.Fprintf(&sb, ` rowspan="%d"`, cell.rowSpan)
			}
			if st := tabCellStyle(cell.attrs, t.columns, i); st != "" {
				fmt.Fprintf(&sb, ` style="%s"`, st)
			}
			sb.WriteString(">")
			sb.WriteString(html.EscapeString(attrVal(cell.attrs, "Text")))
			if len(cell.objects) > 0 {
				sb.WriteString(RenderBand(Band{Objects: cell.objects}))
			}
			sb.WriteString("</td>")
		}
		sb.WriteString("</tr>")
	}
	sb.WriteString("</table></div>")
	return sb.String()
}

// tabSubreportHTML renders a SubreportObject as a labeled placeholder box.
// Real .frx files (Subreport.frx, Side-by-Side Subreports.frx) carry only
// geometry plus ReportPage, so there is no page content to inline yet.
func tabSubreportHTML(o Object) string {
	label := "Subreport: " + o.Name
	if p := strings.TrimSpace(attrVal(o.Attrs, "ReportPage")); p != "" {
		label += " (Page: " + p + ")"
	}
	return fmt.Sprintf(`<div class="subreport" data-kind="SubreportObject" style="%s">%s</div>`,
		objAbsStyle(o.Attrs, "border:1px dashed gray;overflow:hidden;"),
		html.EscapeString(label))
}
