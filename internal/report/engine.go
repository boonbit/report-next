package report

import (
	"fmt"
	"regexp"
	"strconv"
	"strings"

	"report-next/internal/fontmetrics"
)

// TRACK engine (P3): prepared print document.
//
// Real .frx attribute names mirrored from inspected fixtures:
//   FastReport/Demos/Reports/Groups.frx: GroupHeaderBand Condition
//   ("[Products.ProductName].Substring(0,1)"), nested DataBand DataSource
//   ("Products"), Total attrs Name, TotalType ("Count"), Evaluator
//   ("Data1"), PrintOn ("GroupFooter1").
//   FastReport/Demos/Reports/Products by Suppliers.frx: DataBand
//   DataSource ("Suppliers", "Products"); Total TotalType ("Count").

// Assumption: FastReport geometry attributes (Left/Top/Width/Height) are
// pixels at 96dpi, while ReportPage PaperHeight/PaperWidth and *Margin
// attributes are millimetres; 1mm = 3.7795px (96/25.4).
const engMM = 3.7795

// engDocument is a prepared report: one body of band HTML per output page.
type engDocument struct {
	Pages []engPage
}

// engPage is one output page: Body holds concatenated RenderBand fragments.
type engPage struct {
	Body string
}

// engPrepare binds DataBands to DataDictionary rows, breaks on GroupHeader
// Condition change, evaluates Totals over the in-scope rows, repeats
// PageHeader/PageFooter bands, and paginates by accumulated band heights
// against the usable page height.
func engPrepare(rep Report, dict *DataDictionary) (*engDocument, error) {
	params := engParams(dict)
	doc := &engDocument{}
	for _, p := range rep.Pages {
		var titles, headers, footers, summaries, overlays []Band
		var flow []engBand
		for _, b := range p.Bands {
			// Titles/headers/footers render verbatim (never decomposed),
			// so bind their matrices up front; flow bands pivot inside
			// engDecompose via the same helper.
			b = engBindMatrixBand(b, dict)
			switch b.Kind {
			case "ReportTitleBand":
				titles = append(titles, b)
			case "PageHeaderBand", "ColumnHeaderBand":
				headers = append(headers, b)
			case "PageFooterBand", "ColumnFooterBand":
				footers = append(footers, b)
			case "ReportSummaryBand":
				summaries = append(summaries, b)
			case "OverlayBand":
				overlays = append(overlays, b)
			default:
				flow = append(flow, engDecompose(b, dict))
			}
		}
		var body []engInst
		engExpand(flow, dict, params, &body)
		// ReportSummaryBands print once after all flow instances, in
		// band order; each expands like a standalone static band so
		// nested data bands still resolve. OverlayBands never join the
		// flow; they render onto every prepared page in engPaginate.
		for _, s := range summaries {
			engExpandOne(engDecompose(s, dict), flow, dict, params, &body)
		}
		doc.Pages = append(doc.Pages, engPaginate(p, titles, headers, body, footers, overlays, params)...)
	}
	if doc.Pages == nil {
		doc.Pages = []engPage{{}}
	}
	return doc, nil
}

// engRenderDocumentHTML renders a prepared document as full HTML with one
// section per page.
func engRenderDocumentHTML(doc *engDocument) string {
	var sb strings.Builder
	sb.WriteString("<!DOCTYPE html>\n<html><head><meta charset=\"utf-8\">" +
		"<style>.page{position:relative;overflow:hidden;page-break-after:always;}</style>" +
		"</head><body>\n")
	if doc != nil {
		for i, pg := range doc.Pages {
			fmt.Fprintf(&sb, "<section class=\"page\" data-page=\"%d\">\n%s\n</section>\n", i+1, pg.Body)
		}
	}
	sb.WriteString("</body></html>\n")
	return sb.String()
}

// engRenderPageHTML renders a single 1-based prepared page, keeping its
// original data-page number. Out-of-range pages render the full document.
func engRenderPageHTML(doc *engDocument, page int) string {
	if doc == nil || page < 1 || page > len(doc.Pages) {
		return engRenderDocumentHTML(doc)
	}
	var sb strings.Builder
	sb.WriteString("<!DOCTYPE html>\n<html><head><meta charset=\"utf-8\">" +
		"<style>.page{position:relative;overflow:hidden;page-break-after:always;}</style>" +
		"</head><body>\n")
	fmt.Fprintf(&sb, "<section class=\"page\" data-page=\"%d\">\n%s\n</section>\n",
		page, doc.Pages[page-1].Body)
	sb.WriteString("</body></html>\n")
	return sb.String()
}

// engBand is a band plus its nested sub-bands (frx nests DataBand and
// GroupFooterBand objects inside their GroupHeaderBand/DataBand element;
// the flat builder form keeps them as sibling bands instead).
type engBand struct {
	band Band
	subs []engBand
}

// engDecompose splits a band's Band-suffixed objects into sub-bands and
// drops structural Sort objects so they never render as fallback boxes.
func engDecompose(b Band, dict *DataDictionary) engBand {
	e := engBand{band: Band{Kind: b.Kind, Name: b.Name, Attrs: b.Attrs}}
	for _, o := range b.Objects {
		if o.Kind == "Sort" {
			continue
		}
		if strings.HasSuffix(o.Kind, "Band") {
			e.subs = append(e.subs, engDecompose(engObjectBand(o), dict))
			continue
		}
		// A MatrixObject bound to a known source pivots into a static
		// TableObject so the table renderer displays aggregated data.
		// Anything unresolvable keeps the static template fallback.
		if tab, ok := engPivotObject(o, dict); ok {
			e.band.Objects = append(e.band.Objects, tab)
			continue
		}
		e.band.Objects = append(e.band.Objects, o)
	}
	return e
}

// engPivotObject converts a data-bound MatrixObject into a static
// TableObject. It reports false when there is nothing to bind, so the
// caller keeps the static template fallback.
func engPivotObject(o Object, dict *DataDictionary) (Object, bool) {
	if o.Kind != "MatrixObject" || dict == nil {
		return Object{}, false
	}
	tab, err := vizPivotTableObject(o, dict)
	if err != nil {
		return Object{}, false
	}
	return tab, true
}

// engBindMatrixBand pre-binds the top-level MatrixObjects of bands that
// render verbatim (titles, headers, footers); flow bands pivot inside
// engDecompose.
func engBindMatrixBand(b Band, dict *DataDictionary) Band {
	for i, o := range b.Objects {
		if tab, ok := engPivotObject(o, dict); ok {
			b.Objects[i] = tab
		}
	}
	return b
}

// engObjectBand converts a nested band object (whose TextObjects ride along
// as Node children) back into a Band.
func engObjectBand(o Object) Band {
	b := Band{Kind: o.Kind, Name: o.Name, Attrs: o.Attrs}
	for _, c := range o.Children {
		if c.Name == "Sort" {
			continue
		}
		b.Objects = append(b.Objects, Object{
			Kind: c.Name, Name: attrVal(c.Attrs, "Name"),
			Attrs: withoutName(c.Attrs), Children: c.Children,
		})
	}
	return b
}

// engInst is one rendered band occurrence with its bound fields.
type engInst struct {
	band   Band
	fields map[string]string
	height float64
}

func engHeight(b Band) float64 {
	if h := engAttrF(b.Attrs, "Height"); h > 0 {
		return h
	}
	h := 0.0
	for _, o := range b.Objects {
		h += engAttrF(o.Attrs, "Height")
	}
	return h
}

// engLineHeightFactor is the line step for wrapped CanGrow text, as a
// multiple of the font size. It lives in the engine so HTML and PDF share
// one measuring path through fontmetrics.
const engLineHeightFactor = 1.15

// engAttrBool reads a booleanish attr: true/1/yes/always (and the
// FastReport ShiftMode name smAlways), case-insensitive. Absent or any
// other value is false.
func engAttrBool(v string) bool {
	switch strings.ToLower(strings.TrimSpace(v)) {
	case "true", "1", "yes", "always", "smalways":
		return true
	}
	return false
}

// engSetAttr upserts one attr in place, preserving document order.
func engSetAttr(a []Attr, name, val string) []Attr {
	for i := range a {
		if a[i].Name == name {
			a[i].Value = val
			return a
		}
	}
	return append(a, Attr{Name: name, Value: val})
}

// engGrowInstance stretches TextObjects carrying CanGrow="true" to their
// wrapped text height (measured with fontmetrics at the object's font
// size and width) and grows the band instance to enclose them. Objects
// below a grown object move down by the same delta only when the band
// opts into ShiftMode; otherwise positions stay and content may overlap
// as before. Bands and objects without the new attrs are untouched.
func engGrowInstance(in *engInst, params map[string]string) {
	b := &in.band
	// Instances share their template band's backing arrays across rows,
	// so detach before rewriting geometry per evaluated row text.
	objs := make([]Object, len(b.Objects))
	for i, o := range b.Objects {
		objs[i] = o
		objs[i].Attrs = append([]Attr(nil), o.Attrs...)
	}
	b.Objects = objs
	b.Attrs = append([]Attr(nil), b.Attrs...)
	shift := engAttrBool(attrVal(b.Attrs, "ShiftMode"))
	grew := false
	for i := range b.Objects {
		o := &b.Objects[i]
		if o.Kind != "TextObject" || !engAttrBool(attrVal(o.Attrs, "CanGrow")) {
			continue
		}
		text := engEvalText(attrVal(o.Attrs, "Text"), in.fields, params)
		size, face := pdfFont(o.Attrs)
		// pdfFont speaks points but the sheet geometry (and the HTML
		// preview) is 96dpi pixels: measure in px like the browser does,
		// or grown boxes come out ~3/4 short and spill.
		size *= 96.0 / 72.0
		top, _, w, h := objRectOf(o.Attrs)
		need := float64(engWrapLines(text, face, size, w)) * size * engLineHeightFactor
		if need <= h {
			continue
		}
		grew = true
		delta := need - h
		o.Attrs = engSetAttr(o.Attrs, "Height", strconv.FormatFloat(need, 'f', 2, 64))
		if !shift {
			continue
		}
		for j := range b.Objects {
			if j == i {
				continue
			}
			if t := objFloatAttr(b.Objects[j].Attrs, "Top"); t >= top+h {
				b.Objects[j].Attrs = engSetAttr(b.Objects[j].Attrs, "Top",
					strconv.FormatFloat(t+delta, 'f', 2, 64))
			}
		}
	}
	if !grew {
		return
	}
	bottom := 0.0
	for _, o := range b.Objects {
		if e := objFloatAttr(o.Attrs, "Top") + objFloatAttr(o.Attrs, "Height"); e > bottom {
			bottom = e
		}
	}
	if bottom > in.height {
		in.height = bottom
		b.Attrs = engSetAttr(b.Attrs, "Height", strconv.FormatFloat(bottom, 'f', 2, 64))
	}
}

// engWrapLines counts the lines text needs inside width at size, using
// greedy word wrap with rune-level hard breaks for oversize words.
// Newlines in the text force line breaks.
func engWrapLines(text string, face fontmetrics.Face, size, width float64) int {
	if size <= 0 {
		size = 10
	}
	if width <= 0 {
		return 1
	}
	space := face.Width(" ", size)
	total := 0
	for _, para := range strings.Split(text, "\n") {
		words := strings.Fields(para)
		if len(words) == 0 {
			total++
			continue
		}
		lines := 1
		cur := 0.0
		for _, w := range words {
			ww := face.Width(w, size)
			if ww > width {
				if cur > 0 {
					lines++
					cur = 0
				}
				for _, r := range w {
					rw := face.Width(string(r), size)
					if cur > 0 && cur+rw > width {
						lines++
						cur = 0
					}
					cur += rw
				}
				continue
			}
			switch {
			case cur == 0:
				cur = ww
			case cur+space+ww <= width:
				cur += space + ww
			default:
				lines++
				cur = ww
			}
		}
		total += lines
	}
	return total
}

// engExpand flattens flow bands into band instances in print order.
func engExpand(flow []engBand, dict *DataDictionary, params map[string]string, out *[]engInst) {
	for i := 0; i < len(flow); i++ {
		e := flow[i]
		if e.band.Kind != "GroupHeaderBand" {
			engExpandOne(e, flow, dict, params, out)
			continue
		}
		datas, footer, consumed := engGroupMembers(e, flow[i+1:])
		i += consumed
		rows := engGroupRows(datas, dict)
		emit := func(b Band, fields map[string]string) {
			*out = append(*out, engInst{band: b, fields: fields, height: engHeight(b)})
		}
		if len(rows) == 0 {
			emit(e.band, map[string]string{})
			for _, d := range datas {
				engExpandOne(d, flow, dict, params, out)
			}
			if footer != nil {
				emit(footer.band, engScopeTotals(dict, nil))
			}
			continue
		}
		for _, g := range engPartition(rows, attrVal(e.band.Attrs, "Condition"), params) {
			emit(e.band, engCopyRow(g[0]))
			for _, r := range g {
				fm := engCopyRow(r)
				for _, d := range datas {
					emit(d.band, fm)
				}
			}
			emit(footerBand(footer), engMergeRow(g[len(g)-1], engScopeTotals(dict, g)))
		}
	}
}

// engExpandOne expands a single non-group band: optional nested headers,
// one instance per data row (or one static instance), nested footers with
// totals, then nested detail DataBands sequentially.
func engExpandOne(e engBand, flow []engBand, dict *DataDictionary, params map[string]string, out *[]engInst) {
	if e.band.Kind != "DataBand" {
		var fields map[string]string
		if strings.HasSuffix(e.band.Kind, "FooterBand") {
			fields = engScopeTotals(dict, engEvaluatorRows(e.band.Name, flow, dict))
		} else {
			fields = map[string]string{}
		}
		*out = append(*out, engInst{band: e.band, fields: fields, height: engHeight(e.band)})
		for _, s := range e.subs {
			engExpandOne(s, flow, dict, params, out)
		}
		return
	}
	rows := engRows(dict, attrVal(e.band.Attrs, "DataSource"))
	var heads, details, foots []engBand
	for _, s := range e.subs {
		switch s.band.Kind {
		case "DataBand":
			details = append(details, s)
		case "DataHeaderBand":
			heads = append(heads, s)
		case "DataFooterBand", "GroupFooterBand":
			foots = append(foots, s)
		default:
			details = append(details, s)
		}
	}
	for _, h := range heads {
		*out = append(*out, engInst{band: h.band, fields: map[string]string{}, height: engHeight(h.band)})
	}
	for _, r := range rows {
		*out = append(*out, engInst{band: e.band, fields: engCopyRow(r), height: engHeight(e.band)})
	}
	totals := engScopeTotals(dict, rows)
	var last DataRow
	if len(rows) > 0 {
		last = rows[len(rows)-1]
	}
	for _, f := range foots {
		*out = append(*out, engInst{band: f.band, fields: engMergeRow(last, totals), height: engHeight(f.band)})
	}
	for _, d := range details {
		engExpandOne(d, flow, dict, params, out)
	}
}

// engEvaluatorRows resolves a footer's Totals Evaluator band names to their
// source rows, so standalone footers still aggregate the right data.
func engEvaluatorRows(footer string, flow []engBand, dict *DataDictionary) []DataRow {
	if dict == nil || footer == "" {
		return nil
	}
	want := map[string]bool{}
	for _, t := range dict.Totals {
		if t.PrintOn == footer && t.Evaluator != "" {
			want[t.Evaluator] = true
		}
	}
	if len(want) == 0 {
		return nil
	}
	var rows []DataRow
	var walk func(e engBand)
	walk = func(e engBand) {
		if want[e.band.Name] && e.band.Kind == "DataBand" {
			rows = append(rows, engRows(dict, attrVal(e.band.Attrs, "DataSource"))...)
		}
		for _, s := range e.subs {
			walk(s)
		}
	}
	for _, e := range flow {
		walk(e)
	}
	return rows
}

// engGroupMembers collects a group header's data bands and footer from
// either nested sub-bands (parsed .frx) or following flat siblings
// (builder form), returning how many siblings were consumed.
func engGroupMembers(header engBand, rest []engBand) (datas []engBand, footer *engBand, consumed int) {
	if len(header.subs) > 0 {
		for i := range header.subs {
			switch header.subs[i].band.Kind {
			case "DataBand":
				datas = append(datas, header.subs[i])
			case "GroupFooterBand":
				f := header.subs[i]
				if footer == nil {
					footer = &f
				}
			}
		}
		return datas, footer, 0
	}
	for _, s := range rest {
		switch s.band.Kind {
		case "DataBand", "DataHeaderBand", "DataFooterBand":
			datas = append(datas, s)
			consumed++
		case "GroupFooterBand":
			f := s
			footer = &f
			consumed++
			return datas, footer, consumed
		default:
			return datas, footer, consumed
		}
	}
	return datas, footer, consumed
}

func footerBand(f *engBand) Band {
	if f == nil {
		return Band{}
	}
	return f.band
}

// engGroupRows returns the rows of the group's first DataBand source.
func engGroupRows(datas []engBand, dict *DataDictionary) []DataRow {
	for _, d := range datas {
		if d.band.Kind == "DataBand" {
			if rows := engRows(dict, attrVal(d.band.Attrs, "DataSource")); len(rows) > 0 {
				return rows
			}
		}
	}
	return nil
}

// engPartition splits rows into consecutive runs with equal group keys
// (FastReport groups expect sorted input).
func engPartition(rows []DataRow, cond string, params map[string]string) [][]DataRow {
	var groups [][]DataRow
	for _, r := range rows {
		fm := engCopyRow(r)
		key := engGroupKey(cond, fm, params)
		if len(groups) == 0 || engGroupKey(cond, engCopyRow(groups[len(groups)-1][0]), params) != key {
			groups = append(groups, []DataRow{r})
			continue
		}
		groups[len(groups)-1] = append(groups[len(groups)-1], r)
	}
	return groups
}

// engSheet is one output page worth of band instances; title marks the
// first sheet, which also carries the report titles.
type engSheet struct {
	title bool
	insts []engInst
}

// engSplitSheets flows body instances into sheets by accumulated height
// against the usable page height. Titles ride only on the first sheet.
// Instances grow first (CanGrow), then paginate whole: an instance that
// does not fit the remaining room moves to a new sheet (KeepTogether).
func engSplitSheets(p Page, titles, headers []Band, body []engInst, footers []Band, params map[string]string) []engSheet {
	budget := engBudget(p)
	fixed := 0.0
	for _, b := range headers {
		fixed += engHeight(b)
	}
	for _, b := range footers {
		fixed += engHeight(b)
	}
	titleH := 0.0
	for _, b := range titles {
		titleH += engHeight(b)
	}
	room := budget - fixed
	if room < 1 {
		room = 1
	}
	for i := range body {
		engGrowInstance(&body[i], params)
	}
	sheets := []engSheet{{title: true}}
	used := titleH
	put := func(in engInst) {
		cur := &sheets[len(sheets)-1]
		limit := room
		if cur.title {
			limit -= titleH
		}
		// KeepTogether: a band instance is atomic. When it does not
		// fit the remaining room it moves whole to a new sheet instead
		// of splitting. Bands without the attr keep the historical
		// behavior (same atomic move); an explicit KeepTogether="false"
		// opts out and lets the instance overflow the current sheet.
		keep := true
		if v := strings.TrimSpace(attrVal(in.band.Attrs, "KeepTogether")); v != "" {
			keep = engAttrBool(v)
		}
		if len(cur.insts) > 0 && keep && used+in.height > limit {
			sheets = append(sheets, engSheet{})
			cur = &sheets[len(sheets)-1]
			used = 0
			limit = room
		}
		cur.insts = append(cur.insts, in)
		used += in.height
	}
	for _, in := range body {
		put(in)
	}
	if len(sheets) > 1 && len(sheets[len(sheets)-1].insts) == 0 {
		sheets = sheets[:len(sheets)-1]
	}
	return sheets
}

// engPaginate flows titles (first page only), headers, body and footers
// into pages by accumulated height against the usable page height.
// Overlays render onto every page without consuming page room.
func engPaginate(p Page, titles, headers []Band, body []engInst, footers, overlays []Band, params map[string]string) []engPage {
	sheets := engSplitSheets(p, titles, headers, body, footers, params)
	total := len(sheets)
	var pages []engPage
	for i, sh := range sheets {
		nums := map[string]string{
			"PageN":      strconv.Itoa(i + 1),
			"TotalPages": strconv.Itoa(total),
		}
		var sb strings.Builder
		if sh.title {
			for _, b := range titles {
				sb.WriteString(RenderBand(engEvalBand(b, nums, params)))
				sb.WriteString("\n")
			}
		}
		for _, b := range headers {
			sb.WriteString(RenderBand(engEvalBand(b, nums, params)))
			sb.WriteString("\n")
		}
		for _, in := range sh.insts {
			sb.WriteString(RenderBand(engEvalBand(in.band, engMergeNums(in.fields, nums), params)))
			sb.WriteString("\n")
		}
		for _, b := range footers {
			sb.WriteString(RenderBand(engEvalBand(b, nums, params)))
			sb.WriteString("\n")
		}
		for _, b := range overlays {
			sb.WriteString(RenderBand(engEvalBand(b, nums, params)))
			sb.WriteString("\n")
		}
		pages = append(pages, engPage{Body: sb.String()})
	}
	return pages
}

// engOutlineEntry is one table-of-contents item: the evaluated
// OutlineExpression title and its 1-based prepared page number.
type engOutlineEntry struct {
	Title string
	Page  int
}

// engOutline evaluates every band OutlineExpression attr per prepared page.
// Static text renders verbatim; [field] references resolve through the same
// engEvalText path used for Text objects, against the band instance fields
// (data rows, totals) plus PageN/TotalPages numbering.
func engOutline(rep Report, dict *DataDictionary) []engOutlineEntry {
	out := make([]engOutlineEntry, 0)
	params := engParams(dict)
	type laidOut struct {
		titles, headers, footers []Band
		sheets                   []engSheet
	}
	var all []laidOut
	total := 0
	for _, p := range rep.Pages {
		var titles, headers, footers, summaries []Band
		var flow []engBand
		for _, b := range p.Bands {
			b = engBindMatrixBand(b, dict)
			switch b.Kind {
			case "ReportTitleBand":
				titles = append(titles, b)
			case "PageHeaderBand", "ColumnHeaderBand":
				headers = append(headers, b)
			case "PageFooterBand", "ColumnFooterBand":
				footers = append(footers, b)
			case "ReportSummaryBand":
				summaries = append(summaries, b)
			case "OverlayBand":
				// Overlays repeat on every page and carry no entries.
				continue
			default:
				flow = append(flow, engDecompose(b, dict))
			}
		}
		var body []engInst
		engExpand(flow, dict, params, &body)
		// Same trailing-summary order as engPrepare so outline page
		// numbers match the rendered document.
		for _, s := range summaries {
			engExpandOne(engDecompose(s, dict), flow, dict, params, &body)
		}
		sheets := engSplitSheets(p, titles, headers, body, footers, params)
		all = append(all, laidOut{titles: titles, headers: headers, footers: footers, sheets: sheets})
		total += len(sheets)
	}
	if total == 0 {
		total = 1
	}
	pageNo := 0
	for _, lo := range all {
		for _, sh := range lo.sheets {
			pageNo++
			nums := map[string]string{
				"PageN":      strconv.Itoa(pageNo),
				"TotalPages": strconv.Itoa(total),
			}
			if sh.title {
				for _, b := range lo.titles {
					if expr := strings.TrimSpace(attrVal(b.Attrs, "OutlineExpression")); expr != "" {
						out = append(out, engOutlineEntry{Title: engEvalText(expr, nums, params), Page: pageNo})
					}
				}
			}
			for _, b := range lo.headers {
				if expr := strings.TrimSpace(attrVal(b.Attrs, "OutlineExpression")); expr != "" {
					out = append(out, engOutlineEntry{Title: engEvalText(expr, nums, params), Page: pageNo})
				}
			}
			for _, in := range sh.insts {
				if expr := strings.TrimSpace(attrVal(in.band.Attrs, "OutlineExpression")); expr != "" {
					out = append(out, engOutlineEntry{
						Title: engEvalText(expr, engMergeNums(in.fields, nums), params),
						Page:  pageNo,
					})
				}
			}
			for _, b := range lo.footers {
				if expr := strings.TrimSpace(attrVal(b.Attrs, "OutlineExpression")); expr != "" {
					out = append(out, engOutlineEntry{Title: engEvalText(expr, nums, params), Page: pageNo})
				}
			}
		}
	}
	return out
}

// engBudget is the usable page height in pixels.
func engBudget(p Page) float64 {
	h := engAttrF(p.Attrs, "PaperHeight")
	if h <= 0 {
		h = 297 // A4 default, mm
	}
	u := h*engMM - (engAttrF(p.Attrs, "TopMargin")+engAttrF(p.Attrs, "BottomMargin"))*engMM
	if u < 1 {
		u = 1
	}
	return u
}

func engAttrF(a []Attr, name string) float64 {
	v := strings.TrimSpace(attrVal(a, name))
	if v == "" {
		return 0
	}
	f, err := strconv.ParseFloat(v, 64)
	if err != nil {
		return 0
	}
	return f
}

// engRows finds a source by exact name, else by last dotted segment.
func engRows(dict *DataDictionary, source string) []DataRow {
	if dict == nil {
		return nil
	}
	for i := range dict.Sources {
		if dict.Sources[i].Name == source {
			return dict.Sources[i].Rows
		}
	}
	short := source
	if k := strings.LastIndexByte(source, '.'); k >= 0 {
		short = strings.TrimSpace(source[k+1:])
	}
	for i := range dict.Sources {
		if dict.Sources[i].Name == short {
			return dict.Sources[i].Rows
		}
	}
	return nil
}

func engParams(dict *DataDictionary) map[string]string {
	m := map[string]string{}
	if dict == nil {
		return m
	}
	for _, p := range dict.Parameters {
		if p.AsString != "" {
			m[p.Name] = p.AsString
		} else if p.Expression != "" {
			m[p.Name] = p.Expression
		}
	}
	return m
}

func engTotalCol(expr string) string {
	e := strings.TrimSpace(expr)
	if strings.HasPrefix(e, "[") && strings.HasSuffix(e, "]") && len(e) > 1 {
		e = strings.TrimSpace(e[1 : len(e)-1])
	}
	if i := strings.LastIndexByte(e, '.'); i >= 0 {
		e = strings.TrimSpace(e[i+1:])
	}
	return e
}

func engTotalFunc(t string) string {
	switch strings.ToLower(strings.TrimSpace(t)) {
	case "count":
		return "count"
	case "sum":
		return "sum"
	case "avg", "average":
		return "avg"
	case "min":
		return "min"
	case "max":
		return "max"
	}
	return ""
}

// engScopeTotals evaluates every known Total over rows via
// DataAggregateTotal, keyed by total name for [Name] lookup.
func engScopeTotals(dict *DataDictionary, rows []DataRow) map[string]string {
	m := map[string]string{}
	if dict == nil {
		return m
	}
	for _, t := range dict.Totals {
		fn := engTotalFunc(t.TotalType)
		if fn == "" {
			continue
		}
		col := engTotalCol(t.Expression)
		if col == "" && fn != "count" {
			continue
		}
		if v, err := DataAggregateTotal(fn, col, rows); err == nil {
			m[t.Name] = exprNumStr(v)
		}
	}
	return m
}

// Dotted Substring calls from real .frx files
// (Groups.frx Condition "[Products.ProductName].Substring(0,1)") have no
// function-call equivalent in exprEval, so translate them to literals
// before evaluation.
var (
	engDblSubRe = regexp.MustCompile(`\[\[([^\[\]]+?)\]\.Substring\(\s*(\d+)\s*(?:,\s*(\d+)\s*)?\)\]`)
	engSglSubRe = regexp.MustCompile(`\[([^\[\]]+?)\]\.Substring\(\s*(\d+)\s*(?:,\s*(\d+)\s*)?\)\]`)
)

func engSubstCalls(s string, fields map[string]string) string {
	apply := func(re *regexp.Regexp, s string) string {
		return re.ReplaceAllStringFunc(s, func(m string) string {
			g := re.FindStringSubmatch(m)
			si, _ := strconv.Atoi(g[2])
			li := -1
			if g[3] != "" {
				li, _ = strconv.Atoi(g[3])
			}
			return engSubstr(engFieldOf(strings.TrimSpace(g[1]), fields), si, li)
		})
	}
	return apply(engSglSubRe, apply(engDblSubRe, s))
}

func engSubstr(s string, start, length int) string {
	r := []rune(s)
	if start < 0 {
		start = 0
	}
	if start > len(r) {
		start = len(r)
	}
	end := len(r)
	if length >= 0 && start+length < end {
		end = start + length
	}
	return string(r[start:end])
}

func engFieldOf(name string, fields map[string]string) string {
	if v, ok := fields[name]; ok {
		return v
	}
	short := name
	if i := strings.LastIndexByte(name, '.'); i >= 0 {
		short = strings.TrimSpace(name[i+1:])
	}
	if v, ok := fields[short]; ok {
		return v
	}
	for k, v := range fields {
		if strings.EqualFold(k, name) || strings.EqualFold(k, short) {
			return v
		}
	}
	return ""
}

// engGroupKey evaluates a GroupHeader Condition to a break key.
func engGroupKey(cond string, fields, params map[string]string) string {
	c := strings.TrimSpace(cond)
	if c == "" {
		return ""
	}
	t := engSubstCalls(c, fields)
	if v, err := exprEval(t, fields, params); err == nil {
		return v
	}
	return t
}

// engEvalText substitutes [references] (and dotted Substring calls) and
// leaves unresolvable text (e.g. [PageN] before numbering) untouched.
func engEvalText(text string, fields, params map[string]string) string {
	if !strings.Contains(text, "[") {
		return text
	}
	t := engSubstCalls(text, fields)
	v, err := exprEval(t, fields, params)
	if err != nil {
		return text
	}
	return v
}

// engEvalBand returns a copy of b with TextObject texts evaluated and
// Format applied, ready for RenderBand.
func engEvalBand(b Band, fields, params map[string]string) Band {
	out := Band{Kind: b.Kind, Name: b.Name, Attrs: b.Attrs}
	for _, o := range b.Objects {
		if o.Kind == "CheckBoxObject" {
			attrs := make([]Attr, len(o.Attrs))
			copy(attrs, o.Attrs)
			o.Attrs = attrs
			if objCheckedStateBound(o.Attrs, fields, params) {
				o.Attrs = engSetAttr(o.Attrs, "Checked", "true")
			} else {
				o.Attrs = engSetAttr(o.Attrs, "Checked", "false")
			}
			out.Objects = append(out.Objects, o)
			continue
		}
		if o.Kind == "PictureObject" {
			if uri, ok := objPictureDataURI(o, fields, params); ok {
				attrs := make([]Attr, len(o.Attrs))
				copy(attrs, o.Attrs)
				o.Attrs = attrs
				o.Attrs = engSetAttr(o.Attrs, "DataURI", uri)
			}
			out.Objects = append(out.Objects, o)
			continue
		}
		if o.Kind != "TextObject" {
			out.Objects = append(out.Objects, o)
			continue
		}
		val := engEvalText(attrVal(o.Attrs, "Text"), fields, params)
		if fk := strings.TrimSpace(attrVal(o.Attrs, "Format")); fk != "" {
			digits := -1
			if d := strings.TrimSpace(attrVal(o.Attrs, "Format.DecimalDigits")); d != "" {
				if n, err := strconv.Atoi(d); err == nil {
					digits = n
				}
			}
			val = exprFormatValue(fk, val, digits)
		}
		attrs := make([]Attr, len(o.Attrs))
		copy(attrs, o.Attrs)
		set := false
		for i := range attrs {
			if attrs[i].Name == "Text" {
				attrs[i].Value = val
				set = true
				break
			}
		}
		if !set {
			attrs = append(attrs, Attr{Name: "Text", Value: val})
		}
		o.Attrs = attrs
		out.Objects = append(out.Objects, o)
	}
	return out
}

func engCopyRow(r DataRow) map[string]string {
	m := make(map[string]string, len(r))
	for k, v := range r {
		m[k] = v
	}
	return m
}

func engMergeRow(r DataRow, extra map[string]string) map[string]string {
	m := engCopyRow(r)
	for k, v := range extra {
		m[k] = v
	}
	return m
}

func engMergeNums(fields, nums map[string]string) map[string]string {
	m := make(map[string]string, len(fields)+len(nums))
	for k, v := range fields {
		m[k] = v
	}
	for k, v := range nums {
		m[k] = v
	}
	return m
}
