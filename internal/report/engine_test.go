package report

import (
	"strings"
	"testing"
)

// Builder-built grouped report over CSV rows: texts render, group headers
// break on Condition change, group footers total via DataAggregateTotal,
// and overflow paginates with repeated page bands.
func TestEngPrepareGroupsTotalsPagination(t *testing.T) {
	csvData := "Name,Category,Price\n" +
		"Apple,Fruit,1.5\nBanana,Fruit,0.75\nCherry,Fruit,2\nDate,Fruit,3\nElder,Fruit,4\n" +
		"Carrot,Veg,2.25\nDaikon,Veg,1\nEndive,Veg,1.5\nFennel,Veg,2\nGarlic,Veg,0.5\n"
	src, err := DataLoadCSV(strings.NewReader(csvData), "Products")
	if err != nil {
		t.Fatalf("DataLoadCSV: %v", err)
	}
	dict := &DataDictionary{
		Sources: []DataSource{*src},
		Totals: []DataTotal{
			{Name: "TotalProducts", TotalType: "Count", Evaluator: "Data1", PrintOn: "GroupFooter1"},
		},
	}

	rep := &Report{Version: IRVersion, Pages: []Page{{
		Name:  "Page1",
		Attrs: []Attr{{Name: "PaperHeight", Value: "60"}},
	}}}
	setBand := func(kind, name string, attrs ...Attr) int {
		i := exprAddBand(rep, 0, kind, name)
		rep.Pages[0].Bands[i].Attrs = append(rep.Pages[0].Bands[i].Attrs, attrs...)
		return i
	}
	setText := func(bi int, text string, attrs ...Attr) {
		oi := exprAddText(rep, 0, bi, "", text)
		rep.Pages[0].Bands[bi].Objects[oi].Attrs =
			append(rep.Pages[0].Bands[bi].Objects[oi].Attrs, attrs...)
	}
	ph := setBand("PageHeaderBand", "PageHeader1", Attr{"Height", "20"})
	setText(ph, "HEADER")
	gh := setBand("GroupHeaderBand", "GroupHeader1",
		Attr{"Height", "18"}, Attr{"Condition", "[Category]"})
	setText(gh, "[Category]")
	db := setBand("DataBand", "Data1",
		Attr{"Height", "18"}, Attr{"DataSource", "Products"})
	setText(db, "[Name] [Price]")
	gf := setBand("GroupFooterBand", "GroupFooter1", Attr{"Height", "18"})
	setText(gf, "Total products: [TotalProducts]")
	pf := setBand("PageFooterBand", "PageFooter1", Attr{"Height", "20"})
	setText(pf, "[PageN]")

	doc, err := engPrepare(*rep, dict)
	if err != nil {
		t.Fatalf("engPrepare: %v", err)
	}
	if len(doc.Pages) < 2 {
		t.Fatalf("want multi-page overflow, got %d page(s)", len(doc.Pages))
	}
	html := engRenderDocumentHTML(doc)
	for _, want := range []string{"Apple", "Carrot", "Fruit", "Veg", "HEADER"} {
		if !strings.Contains(html, want) {
			t.Errorf("HTML missing %q", want)
		}
	}
	if got := strings.Count(html, "Total products: 5"); got != 2 {
		t.Errorf("want one group-scoped total per group (2), got %d", got)
	}
	for _, stale := range []string{"[PageN]", "[TotalProducts]", "[Category]", "[Name]"} {
		if strings.Contains(html, stale) {
			t.Errorf("HTML still contains unevaluated %q", stale)
		}
	}
	if got := strings.Count(html, "<section"); got != len(doc.Pages) {
		t.Errorf("want one section per page: %d sections vs %d pages", got, len(doc.Pages))
	}
	if !strings.Contains(html, ">1<") || !strings.Contains(html, ">2<") {
		t.Errorf("want repeated page footers numbered 1 and 2")
	}
}

func TestEngPrepareNilDict(t *testing.T) {
	rep := exprNewReport("hello")
	doc, err := engPrepare(*rep, nil)
	if err != nil {
		t.Fatalf("engPrepare: %v", err)
	}
	if len(doc.Pages) != 1 {
		t.Fatalf("want 1 page, got %d", len(doc.Pages))
	}
	if html := engRenderDocumentHTML(nil); !strings.Contains(html, "<html>") {
		t.Errorf("nil document should still render shell, got %q", html)
	}
}
