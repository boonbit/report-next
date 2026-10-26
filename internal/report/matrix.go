package report

import (
	"fmt"
	"html"
	"strconv"
	"strings"
)

// TRACK viz (matrix render): MatrixObject HTML renderer.
//
// Real attribute names sampled from FastReport/Demos/Reports/Simple Matrix.frx
// (Matrix2: FixedRows="1" FixedColumns="1" DataSource="MatrixDemo"):
//   MatrixObject  Left Top Width Height FixedRows FixedColumns AutoSize
//                 DataSource Style
//   MatrixColumns/Header  Expression
//   MatrixRows/Header     Expression
//   MatrixCells/Cell      Expression
//   TableColumn   Name Width
//   TableRow      Name Height
//   TableCell     Name Text AllowExpressions Format HorzAlign VertAlign Font
//                 Fill.Color Border.Lines Border.Color TextFill.Color
//                 ColSpan RowSpan, plus nested objects (ShapeObject and
//                 PictureObject in Objects/Pictures Inside The Matrix.frx,
//                 SimpleProgressGauge and SparklineObject in
//                 "Sparkline, Gauge.frx") and non-object nodes (Highlight,
//                 MatrixSortButton, MatrixCollapseButton) which are skipped.
//
// Dynamic pivot aggregation is out of scope: there is no data binding for
// matrix descriptors yet, so vizMatrixHTML renders the static template grid
// (TableColumn/TableRow/TableCell children) as an HTML table. The typed
// descriptor parse (MatrixColumns/MatrixRows/MatrixCells expressions) rides
// along as data-* attributes for inspection.

// vizMatrixDescriptor is one typed Header/Cell descriptor expression.
type vizMatrixDescriptor struct {
	expression string
	display    string
}

// vizMatrix is the parsed form of a MatrixObject.
type vizMatrix struct {
	obj        Object
	colHeaders []vizMatrixDescriptor
	rowHeaders []vizMatrixDescriptor
	cells      []vizMatrixDescriptor
	columns    []tabColumn
	rows       []tabRow
}

// vizMatrixDescOf types one MatrixColumns/MatrixRows/MatrixCells group child:
// Header and Cell items carry Expression (DisplayText accepted too).
func vizMatrixDescOf(n Node) vizMatrixDescriptor {
	return vizMatrixDescriptor{
		expression: strings.TrimSpace(attrVal(n.Attrs, "Expression")),
		display:    strings.TrimSpace(attrVal(n.Attrs, "DisplayText")),
	}
}

// vizMatrixCellKeeps reports whether a TableCell child node is a nested
// report object worth rendering (ShapeObject, PictureObject,
// SimpleProgressGauge, ...). Styling/behavior nodes (Highlight, Condition,
// MatrixSortButton, MatrixCollapseButton, MSChartSeries) are skipped.
func vizMatrixCellKeeps(name string) bool {
	return strings.HasSuffix(name, "Object") || strings.HasSuffix(name, "Gauge")
}

// vizParseMatrix types a MatrixObject's verbatim children into descriptors
// plus the template grid. TableColumn/TableRow/TableCell parsing mirrors
// tabParseTable; anything else (MatrixColumns/MatrixRows/MatrixCells,
// unknown elements) is either typed or ignored, never an error.
// vizMatrixDescOnly types the descriptor groups (MatrixColumns/MatrixRows/
// MatrixCells) without requiring the template grid, for pivot aggregation.
func vizMatrixDescOnly(o Object) (*vizMatrix, error) {
	if o.Kind != "MatrixObject" {
		return nil, fmt.Errorf("report: vizParseMatrix wants MatrixObject, got %q", o.Kind)
	}
	m := &vizMatrix{obj: o}
	for _, c := range o.Children {
		switch c.Name {
		case "MatrixColumns":
			for _, n := range c.Children {
				if n.Name == "Header" {
					m.colHeaders = append(m.colHeaders, vizMatrixDescOf(n))
				}
			}
		case "MatrixRows":
			for _, n := range c.Children {
				if n.Name == "Header" {
					m.rowHeaders = append(m.rowHeaders, vizMatrixDescOf(n))
				}
			}
		case "MatrixCells":
			for _, n := range c.Children {
				if n.Name == "Cell" {
					m.cells = append(m.cells, vizMatrixDescOf(n))
				}
			}
		}
	}
	return m, nil
}

func vizParseMatrix(o Object) (*vizMatrix, error) {
	m, err := vizMatrixDescOnly(o)
	if err != nil {
		return nil, err
	}
	for _, c := range o.Children {
		switch c.Name {
		case "TableColumn":
			m.columns = append(m.columns, tabColumn{
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
					if vizMatrixCellKeeps(nn.Name) {
						cell.objects = append(cell.objects, tabObjectOf(nn))
					}
				}
				r.cells = append(r.cells, cell)
			}
			m.rows = append(m.rows, r)
		}
	}
	if len(m.columns) == 0 && len(m.rows) == 0 {
		return nil, fmt.Errorf("report: matrix %q has no TableColumn/TableRow children", o.Name)
	}
	return m, nil
}

// vizMatrixExprs joins descriptor expressions for a data-* attribute.
func vizMatrixExprs(dd []vizMatrixDescriptor) string {
	var out []string
	for _, d := range dd {
		if d.expression != "" {
			out = append(out, d.expression)
		} else if d.display != "" {
			out = append(out, d.display)
		}
	}
	return strings.Join(out, "; ")
}

// vizMatrixHTML renders a MatrixObject as an absolutely-positioned HTML grid
// table: column widths from TableColumn Width, row heights from TableRow
// Height, ColSpan/RowSpan on cells, and nested cell objects via RenderBand on
// a synthetic Band. Descriptor expressions are exposed as data-* attributes.
// Matrices without a template grid keep the labeled-box fallback.
func vizMatrixHTML(o Object) string {
	m, err := vizParseMatrix(o)
	if err != nil {
		return objFallbackHTML(o)
	}
	var sb strings.Builder
	fmt.Fprintf(&sb, `<div class="matrix" data-kind="MatrixObject" data-name="%s" data-col-expr="%s" data-row-expr="%s" data-cell-expr="%s" style="%s">`,
		html.EscapeString(o.Name),
		html.EscapeString(vizMatrixExprs(m.colHeaders)),
		html.EscapeString(vizMatrixExprs(m.rowHeaders)),
		html.EscapeString(vizMatrixExprs(m.cells)),
		objAbsStyle(o.Attrs, "overflow:hidden;"))
	sb.WriteString(`<table style="border-collapse:collapse;width:100%;height:100%;table-layout:fixed;">`)
	if len(m.columns) > 0 {
		sb.WriteString("<colgroup>")
		for _, c := range m.columns {
			if f, ferr := strconv.ParseFloat(c.width, 64); ferr == nil && c.width != "" {
				fmt.Fprintf(&sb, `<col style="width:%gpx;">`, f)
			} else {
				sb.WriteString("<col>")
			}
		}
		sb.WriteString("</colgroup>")
	}
	for _, r := range m.rows {
		sb.WriteString("<tr")
		if h := strings.TrimSpace(attrVal(r.attrs, "Height")); h != "" {
			if f, ferr := strconv.ParseFloat(h, 64); ferr == nil {
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
			if st := tabCellStyle(cell.attrs, m.columns, i); st != "" {
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

// vizPivotField extracts the field name from a descriptor expression:
// "[MatrixDemo.Year]" -> "Year", "[Year]" -> "Year". Anything else
// (function calls, bare text) is an error.
func vizPivotField(expr string) (string, error) {
	expr = strings.TrimSpace(expr)
	if !strings.HasPrefix(expr, "[") || !strings.HasSuffix(expr, "]") {
		return "", fmt.Errorf("report: pivot expression %q is not [field]", expr)
	}
	inner := strings.TrimSpace(expr[1 : len(expr)-1])
	if inner == "" || strings.ContainsAny(inner, "()") {
		return "", fmt.Errorf("report: pivot expression %q is not [field]", expr)
	}
	if i := strings.LastIndex(inner, "."); i >= 0 {
		inner = inner[i+1:]
	}
	return inner, nil
}

// vizPivot is one aggregated matrix: distinct column keys across the top,
// composite row keys down the side, sums (or counts for non-numeric cells),
// plus row/column/grand totals.
type vizPivot struct {
	cols     []string
	rows     [][]string
	rowKeys  []string
	cells    map[string]map[string]float64
	rowTotal map[string]float64
	colTotal map[string]float64
	grand    float64
	counted  bool // true when values were non-numeric (counts, not sums)
}

// vizPivotGrid aggregates a MatrixObject's DataSource rows over its
// MatrixColumns/MatrixRows header expressions and its MatrixCells value
// expression. Only the first Cell expression is used.
func vizPivotGrid(o Object, dict *DataDictionary) (*vizPivot, error) {
	m, err := vizMatrixDescOnly(o)
	if err != nil {
		return nil, err
	}
	source := strings.TrimSpace(attrVal(o.Attrs, "DataSource"))
	if source == "" {
		return nil, fmt.Errorf("report: matrix %q has no DataSource", o.Name)
	}
	var rows []DataRow
	for _, ds := range dict.Sources {
		if ds.Name == source {
			rows = ds.Rows
		}
	}
	if rows == nil {
		return nil, fmt.Errorf("report: matrix source %q not found", source)
	}
	var colFields, rowFields []string
	for _, h := range m.colHeaders {
		f, err := vizPivotField(h.expression)
		if err != nil {
			return nil, err
		}
		colFields = append(colFields, f)
	}
	for _, h := range m.rowHeaders {
		f, err := vizPivotField(h.expression)
		if err != nil {
			return nil, err
		}
		rowFields = append(rowFields, f)
	}
	if len(m.cells) == 0 {
		return nil, fmt.Errorf("report: matrix %q has no Cell", o.Name)
	}
	valField, err := vizPivotField(m.cells[0].expression)
	if err != nil {
		return nil, err
	}
	keyOf := func(r DataRow, fields []string) (string, error) {
		parts := make([]string, len(fields))
		for i, f := range fields {
			v, ok := r[f]
			if !ok {
				return "", fmt.Errorf("report: matrix field %q missing", f)
			}
			parts[i] = v
		}
		return strings.Join(parts, "\x00"), nil
	}
	g := &vizPivot{cells: map[string]map[string]float64{}, rowTotal: map[string]float64{}, colTotal: map[string]float64{}}
	seenCol := map[string]bool{}
	seenRow := map[string]bool{}
	numeric := true
	for _, r := range rows {
		if _, err := strconv.ParseFloat(r[valField], 64); err != nil {
			numeric = false
		}
	}
	for _, r := range rows {
		ck, err := keyOf(r, colFields)
		if err != nil {
			return nil, err
		}
		rk, err := keyOf(r, rowFields)
		if err != nil {
			return nil, err
		}
		if !seenCol[ck] {
			seenCol[ck] = true
			g.cols = append(g.cols, ck)
		}
		if !seenRow[rk] {
			seenRow[rk] = true
			g.rowKeys = append(g.rowKeys, rk)
			g.rows = append(g.rows, strings.Split(rk, "\x00"))
		}
		v := 1.0
		if numeric {
			v, _ = strconv.ParseFloat(r[valField], 64)
		}
		if g.cells[rk] == nil {
			g.cells[rk] = map[string]float64{}
		}
		g.cells[rk][ck] += v
		g.rowTotal[rk] += v
		g.colTotal[ck] += v
		g.grand += v
	}
	g.counted = !numeric
	return g, nil
}

// vizPivotTableObject converts an aggregated pivot into a static TableObject
// (corner + column headers + row headers + cells + totals) that the table
// renderer displays verbatim.
func vizPivotTableObject(o Object, dict *DataDictionary) (Object, error) {
	g, err := vizPivotGrid(o, dict)
	if err != nil {
		return Object{}, err
	}
	cell := func(name, text string) Node {
		return Node{Name: "TableCell", Attrs: []Attr{{"Name", name}, {"Text", text}}}
	}
	tab := Object{Kind: "TableObject", Name: o.Name + "_pivot"}
	width := strconv.Itoa(120)
	for i := 0; i <= len(g.cols); i++ {
		tab.Children = append(tab.Children, Node{Name: "TableColumn",
			Attrs: []Attr{{"Name", "PivotCol" + strconv.Itoa(i)}, {"Width", width}}})
	}
	head := Node{Name: "TableRow", Attrs: []Attr{{"Name", "PivotHead"}}}
	head.Children = append(head.Children, cell("PivotCorner", ""))
	for i, ck := range g.cols {
		head.Children = append(head.Children, cell("PivotCH"+strconv.Itoa(i), strings.ReplaceAll(ck, "\x00", " / ")))
	}
	head.Children = append(head.Children, cell("PivotCHT", "Total"))
	tab.Children = append(tab.Children, head)
	for ri := range g.rows {
		rk := g.rowKeys[ri]
		rn := Node{Name: "TableRow", Attrs: []Attr{{"Name", "PivotR" + strconv.Itoa(ri)}}}
		rn.Children = append(rn.Children, cell("PivotRH"+strconv.Itoa(ri), strings.Join(g.rows[ri], " / ")))
		for ci, ck := range g.cols {
			rn.Children = append(rn.Children, cell(
				fmt.Sprintf("PivotC%d_%d", ri, ci), vizPivotNum(g.cells[rk][ck])))
		}
		rn.Children = append(rn.Children, cell(fmt.Sprintf("PivotRT%d", ri), vizPivotNum(g.rowTotal[rk])))
		tab.Children = append(tab.Children, rn)
	}
	tot := Node{Name: "TableRow", Attrs: []Attr{{"Name", "PivotTotal"}}}
	tot.Children = append(tot.Children, cell("PivotTT", "Total"))
	for i, ck := range g.cols {
		tot.Children = append(tot.Children, cell("PivotTC"+strconv.Itoa(i), vizPivotNum(g.colTotal[ck])))
	}
	tot.Children = append(tot.Children, cell("PivotGT", vizPivotNum(g.grand)))
	tab.Children = append(tab.Children, tot)
	return tab, nil
}

func vizPivotNum(f float64) string {
	return strconv.FormatFloat(f, 'f', -1, 64)
}
