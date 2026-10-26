// Data dictionary layer (track "data", P2-core): typed view over the
// opaque Dictionary Node plus CSV/JSON/XML loaders and totals.
//
// Element and attribute names mirror real .frx files, e.g.
// FastReport/Demos/Reports/Sales Statistics.frx:
//   - TableDataSource attrs: Name, ReferenceName, DataType, Enabled;
//     children: Column.
//   - Column attrs: Name, DataType, optional Calculated, Expression
//     (plus PropName, Alias, BindableControl which are ignored here).
//   - Relation attrs: Name, ReferenceName, ParentDataSource,
//     ChildDataSource, ParentColumns, ChildColumns, Enabled.
//   - Parameter attrs: Name, DataType, Expression, AsString.
//   - Total attrs: Name, Expression, Evaluator, PrintOn, TotalType,
//     ResetAfterPrint.
package report

import (
	"encoding/csv"
	"encoding/json"
	"encoding/xml"
	"fmt"
	"io"
	"strconv"
	"strings"
)

// DataDictionary is the typed view of a Dictionary node.
type DataDictionary struct {
	Sources    []DataSource
	Relations  []DataRelation
	Parameters []DataParameter
	Totals     []DataTotal
}

// DataSource is one named table of rows with declared columns.
type DataSource struct {
	Name    string
	Columns []DataColumn
	Rows    []DataRow
}

// DataColumn declares one field of a DataSource.
type DataColumn struct {
	Name       string
	DataType   string
	Calculated bool
	Expression string
}

// DataRow is one record: field name to text value.
type DataRow map[string]string

// DataRelation links a parent source to a child source.
type DataRelation struct {
	Name       string
	Parent     string
	Child      string
	ParentCols []string
	ChildCols  []string
}

// DataParameter mirrors a .frx Parameter element.
type DataParameter struct {
	Name       string
	DataType   string
	Expression string
	AsString   string
}

// DataTotal mirrors a .frx Total element.
type DataTotal struct {
	Name            string
	Expression      string
	Evaluator       string
	PrintOn         string
	TotalType       string
	ResetAfterPrint bool
}

// DataParseDictionary converts a Dictionary Node into a DataDictionary.
// Unknown child elements are ignored.
func DataParseDictionary(n Node) (*DataDictionary, error) {
	if n.Name != "Dictionary" {
		return nil, fmt.Errorf("report: want Dictionary node, got %q", n.Name)
	}
	d := &DataDictionary{}
	for _, c := range n.Children {
		switch c.Name {
		case "TableDataSource":
			s := DataSource{Name: attrVal(c.Attrs, "Name")}
			for _, g := range c.Children {
				if g.Name != "Column" {
					continue
				}
				s.Columns = append(s.Columns, DataColumn{
					Name:       attrVal(g.Attrs, "Name"),
					DataType:   attrVal(g.Attrs, "DataType"),
					Calculated: attrVal(g.Attrs, "Calculated") == "true",
					Expression: attrVal(g.Attrs, "Expression"),
				})
			}
			d.Sources = append(d.Sources, s)
		case "Relation":
			d.Relations = append(d.Relations, DataRelation{
				Name:       attrVal(c.Attrs, "Name"),
				Parent:     attrVal(c.Attrs, "ParentDataSource"),
				Child:      attrVal(c.Attrs, "ChildDataSource"),
				ParentCols: dataSplitCols(attrVal(c.Attrs, "ParentColumns")),
				ChildCols:  dataSplitCols(attrVal(c.Attrs, "ChildColumns")),
			})
		case "Parameter":
			d.Parameters = append(d.Parameters, DataParameter{
				Name:       attrVal(c.Attrs, "Name"),
				DataType:   attrVal(c.Attrs, "DataType"),
				Expression: attrVal(c.Attrs, "Expression"),
				AsString:   attrVal(c.Attrs, "AsString"),
			})
		case "Total":
			d.Totals = append(d.Totals, DataTotal{
				Name:            attrVal(c.Attrs, "Name"),
				Expression:      attrVal(c.Attrs, "Expression"),
				Evaluator:       attrVal(c.Attrs, "Evaluator"),
				PrintOn:         attrVal(c.Attrs, "PrintOn"),
				TotalType:       attrVal(c.Attrs, "TotalType"),
				ResetAfterPrint: attrVal(c.Attrs, "ResetAfterPrint") == "true",
			})
		}
	}
	return d, nil
}

// dataSplitCols splits a ParentColumns/ChildColumns attribute value.
// FastReport separates multiple columns with semicolons.
func dataSplitCols(s string) []string {
	var out []string
	for _, p := range strings.Split(s, ";") {
		if p = strings.TrimSpace(p); p != "" {
			out = append(out, p)
		}
	}
	return out
}

// dataLookupRelation returns the child rows of childName related to parent
// through the named relation. It returns nil when the relation or either
// source is missing.
func dataLookupRelation(d *DataDictionary, name string, parent DataRow) []DataRow {
	if d == nil {
		return nil
	}
	var rel *DataRelation
	for i := range d.Relations {
		if d.Relations[i].Name == name {
			rel = &d.Relations[i]
			break
		}
	}
	if rel == nil || len(rel.ParentCols) == 0 || len(rel.ParentCols) != len(rel.ChildCols) {
		return nil
	}
	var child *DataSource
	for i := range d.Sources {
		if d.Sources[i].Name == rel.Child {
			child = &d.Sources[i]
			break
		}
	}
	if child == nil {
		return nil
	}
	var out []DataRow
rows:
	for _, r := range child.Rows {
		for i := range rel.ParentCols {
			if r[rel.ChildCols[i]] != parent[rel.ParentCols[i]] {
				continue rows
			}
		}
		out = append(out, r)
	}
	return out
}

// dataSourceOf builds a DataSource from ordered headers and string rows.
func dataSourceOf(name string, headers []string, rows []DataRow) *DataSource {
	s := &DataSource{Name: name}
	for _, h := range headers {
		s.Columns = append(s.Columns, DataColumn{Name: h, DataType: "System.String"})
	}
	s.Rows = rows
	return s
}

// DataLoadCSV loads CSV data: the first record is the header row.
func DataLoadCSV(r io.Reader, name string) (*DataSource, error) {
	cr := csv.NewReader(r)
	cr.FieldsPerRecord = -1
	recs, err := cr.ReadAll()
	if err != nil {
		return nil, fmt.Errorf("report: csv: %w", err)
	}
	if len(recs) == 0 {
		return nil, fmt.Errorf("report: csv: empty input")
	}
	headers := recs[0]
	var rows []DataRow
	for _, rec := range recs[1:] {
		row := DataRow{}
		for i, h := range headers {
			if i < len(rec) {
				row[h] = rec[i]
			} else {
				row[h] = ""
			}
		}
		rows = append(rows, row)
	}
	return dataSourceOf(name, headers, rows), nil
}

// DataLoadJSON loads an array of flat objects.
func DataLoadJSON(r io.Reader, name string) (*DataSource, error) {
	dec := json.NewDecoder(r)
	dec.UseNumber()
	var arr []map[string]any
	if err := dec.Decode(&arr); err != nil {
		return nil, fmt.Errorf("report: json: %w", err)
	}
	var headers []string
	seen := map[string]bool{}
	var rows []DataRow
	for _, obj := range arr {
		row := DataRow{}
		for k, v := range obj {
			if !seen[k] {
				seen[k] = true
				headers = append(headers, k)
			}
			row[k] = dataScalar(v)
		}
		rows = append(rows, row)
	}
	return dataSourceOf(name, headers, rows), nil
}

// dataScalar renders one JSON scalar as text.
func dataScalar(v any) string {
	switch t := v.(type) {
	case nil:
		return ""
	case string:
		return t
	case json.Number:
		return t.String()
	case bool:
		return strconv.FormatBool(t)
	default:
		return fmt.Sprintf("%v", t)
	}
}

// DataLoadXML loads repeating record elements: every direct child element
// of the root is one row; its attributes and child elements are fields.
func DataLoadXML(r io.Reader, name string) (*DataSource, error) {
	dec := xml.NewDecoder(r)
	var headers []string
	seen := map[string]bool{}
	add := func(h string) {
		if !seen[h] {
			seen[h] = true
			headers = append(headers, h)
		}
	}
	var rows []DataRow
	depth := 0
	var cur DataRow
	var field string
	var text strings.Builder
	for {
		tok, err := dec.Token()
		if err == io.EOF {
			break
		}
		if err != nil {
			return nil, fmt.Errorf("report: xml: %w", err)
		}
		switch t := tok.(type) {
		case xml.StartElement:
			depth++
			switch depth {
			case 2:
				cur = DataRow{}
				for _, a := range t.Attr {
					add(a.Name.Local)
					cur[a.Name.Local] = a.Value
				}
			case 3:
				field = t.Name.Local
				text.Reset()
			}
		case xml.CharData:
			if depth == 3 {
				text.Write(t)
			}
		case xml.EndElement:
			switch depth {
			case 3:
				add(field)
				cur[field] = strings.TrimSpace(text.String())
				field = ""
			case 2:
				rows = append(rows, cur)
				cur = nil
			}
			depth--
		}
	}
	if depth != 0 {
		return nil, fmt.Errorf("report: xml: truncated document")
	}
	if rows == nil {
		return nil, fmt.Errorf("report: xml: no record elements")
	}
	return dataSourceOf(name, headers, rows), nil
}

// DataAggregateTotal aggregates col over rows. fn is one of
// sum, count, avg, min, max. count ignores col and counts rows.
func DataAggregateTotal(fn, col string, rows []DataRow) (float64, error) {
	switch fn {
	case "count":
		return float64(len(rows)), nil
	case "sum", "avg", "min", "max":
		if len(rows) == 0 {
			if fn == "sum" {
				return 0, nil
			}
			return 0, fmt.Errorf("report: %s of no rows", fn)
		}
		vals := make([]float64, 0, len(rows))
		for i, r := range rows {
			v, err := strconv.ParseFloat(strings.TrimSpace(r[col]), 64)
			if err != nil {
				return 0, fmt.Errorf("report: %s: row %d col %q: %w", fn, i, col, err)
			}
			vals = append(vals, v)
		}
		switch fn {
		case "sum":
			s := 0.0
			for _, v := range vals {
				s += v
			}
			return s, nil
		case "avg":
			s := 0.0
			for _, v := range vals {
				s += v
			}
			return s / float64(len(vals)), nil
		case "min":
			m := vals[0]
			for _, v := range vals[1:] {
				if v < m {
					m = v
				}
			}
			return m, nil
		default: // max
			m := vals[0]
			for _, v := range vals[1:] {
				if v > m {
					m = v
				}
			}
			return m, nil
		}
	default:
		return 0, fmt.Errorf("report: unknown aggregate %q", fn)
	}
}
