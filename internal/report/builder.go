package report

import "strconv"

// exprNewReport builds a minimal report carrying title that already passes
// Valid: one page, one title band, one text object holding the title.
func exprNewReport(title string) *Report {
	rep := &Report{Version: IRVersion}
	if title != "" {
		rep.Info = map[string]string{"Title": title}
	}
	rep.Pages = []Page{{
		Name: "Page1",
		Bands: []Band{{
			Kind: "ReportTitleBand",
			Name: "ReportTitle1",
			Objects: []Object{{
				Kind: "TextObject",
				Name: "Text1",
				Attrs: []Attr{
					{Name: "Text", Value: title},
					{Name: "Width", Value: "718.2"},
					{Name: "Height", Value: "28.35"},
				},
			}},
		}},
	}}
	return rep
}

// exprAddPage appends a page with a unique non-empty name and returns its
// index. An empty name is generated as Page<N>.
func exprAddPage(rep *Report, name string) int {
	if rep == nil {
		return -1
	}
	if name == "" {
		name = "Page" + strconv.Itoa(len(rep.Pages)+1)
	}
	name = exprUniquePage(rep, name)
	rep.Pages = append(rep.Pages, Page{Name: name})
	return len(rep.Pages) - 1
}

// exprAddBand appends a band to rep.Pages[pageIdx] and returns its index,
// or -1 for a nil report or bad page index. Empty kind defaults to
// DataBand; empty or duplicate names are generated uniquely.
func exprAddBand(rep *Report, pageIdx int, kind, name string) int {
	if rep == nil || pageIdx < 0 || pageIdx >= len(rep.Pages) {
		return -1
	}
	if kind == "" {
		kind = "DataBand"
	}
	p := &rep.Pages[pageIdx]
	if name == "" {
		name = kind + strconv.Itoa(len(p.Bands)+1)
	}
	name = exprUniqueBand(p, name)
	p.Bands = append(p.Bands, Band{Kind: kind, Name: name})
	return len(p.Bands) - 1
}

// exprAddText appends a TextObject to the named band and returns its index,
// or -1 for a nil report or bad page/band index.
func exprAddText(rep *Report, pageIdx, bandIdx int, name, text string) int {
	if rep == nil || pageIdx < 0 || pageIdx >= len(rep.Pages) {
		return -1
	}
	p := &rep.Pages[pageIdx]
	if bandIdx < 0 || bandIdx >= len(p.Bands) {
		return -1
	}
	b := &p.Bands[bandIdx]
	if name == "" {
		name = "Text" + strconv.Itoa(len(b.Objects)+1)
	}
	name = exprUniqueObject(b, name)
	b.Objects = append(b.Objects, Object{
		Kind:  "TextObject",
		Name:  name,
		Attrs: []Attr{{Name: "Text", Value: text}},
	})
	return len(b.Objects) - 1
}

func exprUniquePage(rep *Report, name string) string {
	seen := map[string]bool{}
	for _, p := range rep.Pages {
		seen[p.Name] = true
	}
	return exprDedupe(seen, name)
}

func exprUniqueBand(p *Page, name string) string {
	seen := map[string]bool{}
	for _, b := range p.Bands {
		seen[b.Name] = true
	}
	return exprDedupe(seen, name)
}

func exprUniqueObject(b *Band, name string) string {
	seen := map[string]bool{}
	for _, o := range b.Objects {
		seen[o.Name] = true
	}
	return exprDedupe(seen, name)
}

func exprDedupe(seen map[string]bool, name string) string {
	if !seen[name] {
		return name
	}
	for i := 2; ; i++ {
		cand := name + "_" + strconv.Itoa(i)
		if !seen[cand] {
			return cand
		}
	}
}
