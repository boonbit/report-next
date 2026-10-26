package report

import (
	"os"
	"strings"
	"testing"
)

func dataDict(t *testing.T, n Node) *DataDictionary {
	t.Helper()
	d, err := DataParseDictionary(n)
	if err != nil {
		t.Fatal(err)
	}
	return d
}

func dataSourceByName(d *DataDictionary, name string) *DataSource {
	for i := range d.Sources {
		if d.Sources[i].Name == name {
			return &d.Sources[i]
		}
	}
	return nil
}

// Dictionary parsed from the real Sales Statistics.frx. Element and
// attribute names asserted here are the file's own: TableDataSource with
// Column children, Relation with ParentDataSource/ChildDataSource and
// ParentColumns/ChildColumns.
func TestDataParseSalesStatistics(t *testing.T) {
	f, err := os.Open("../../../FastReport/Demos/Reports/Sales Statistics.frx")
	if err != nil {
		t.Fatal(err)
	}
	defer f.Close()
	rep, err := ParseFRX(f)
	if err != nil {
		t.Fatal(err)
	}
	d := dataDict(t, rep.Dictionary)
	if len(d.Sources) != 6 {
		t.Fatalf("sources = %d, want 6", len(d.Sources))
	}
	od := dataSourceByName(d, "Order Details")
	if od == nil {
		t.Fatal("missing source Order Details")
	}
	foundPrice := false
	for _, c := range od.Columns {
		if c.Name == "Price" {
			foundPrice = true
			if !c.Calculated {
				t.Error("Price column not Calculated")
			}
			if !strings.Contains(c.Expression, "[Order Details.Quantity]") {
				t.Errorf("Price expression = %q", c.Expression)
			}
		}
	}
	if !foundPrice {
		t.Error("missing Price column in Order Details")
	}
	emp := dataSourceByName(d, "Employees")
	if emp == nil {
		t.Fatal("missing source Employees")
	}
	foundFull := false
	for _, c := range emp.Columns {
		if c.DataType == "" || c.Name == "" {
			t.Errorf("column missing Name/DataType: %+v", c)
		}
		if c.Calculated && !strings.Contains(c.Expression, "[Employees.FirstName]") {
			t.Errorf("FullName expression = %q", c.Expression)
		}
		if c.Calculated {
			foundFull = true
		}
	}
	if !foundFull {
		t.Error("missing calculated column in Employees")
	}
	if len(d.Relations) != 5 {
		t.Fatalf("relations = %d, want 5", len(d.Relations))
	}
	var co *DataRelation
	for i := range d.Relations {
		if d.Relations[i].Name == "CustomersOrders" {
			co = &d.Relations[i]
		}
	}
	if co == nil {
		t.Fatal("missing relation CustomersOrders")
	}
	if co.Parent != "Customers" || co.Child != "Orders" {
		t.Errorf("relation = %+v", co)
	}
	if len(co.ParentCols) != 1 || co.ParentCols[0] != "CustomerID" {
		t.Errorf("parent cols = %v", co.ParentCols)
	}
	if len(co.ChildCols) != 1 || co.ChildCols[0] != "CustomerID" {
		t.Errorf("child cols = %v", co.ChildCols)
	}
	if _, err := DataParseDictionary(Node{Name: "Report"}); err == nil {
		t.Error("non-Dictionary node accepted")
	}
}

func TestDataLoaders(t *testing.T) {
	csvF, err := os.Open("testdata/customers.csv")
	if err != nil {
		t.Fatal(err)
	}
	defer csvF.Close()
	cs, err := DataLoadCSV(csvF, "Customers")
	if err != nil {
		t.Fatal(err)
	}
	jsonF, err := os.Open("testdata/customers.json")
	if err != nil {
		t.Fatal(err)
	}
	defer jsonF.Close()
	js, err := DataLoadJSON(jsonF, "Customers")
	if err != nil {
		t.Fatal(err)
	}
	xmlF, err := os.Open("testdata/customers.xml")
	if err != nil {
		t.Fatal(err)
	}
	defer xmlF.Close()
	xs, err := DataLoadXML(xmlF, "Customers")
	if err != nil {
		t.Fatal(err)
	}
	for _, s := range []*DataSource{cs, js, xs} {
		if s.Name != "Customers" {
			t.Errorf("name = %q", s.Name)
		}
		if len(s.Rows) != 3 {
			t.Errorf("rows = %d, want 3", len(s.Rows))
		}
		if len(s.Columns) != 6 {
			t.Errorf("columns = %d, want 6", len(s.Columns))
		}
		if s.Rows[0]["CustomerID"] != "ALFKI" {
			t.Errorf("rows[0].CustomerID = %q", s.Rows[0]["CustomerID"])
		}
		if s.Rows[2]["Orders"] != "7" {
			t.Errorf("rows[2].Orders = %q", s.Rows[2]["Orders"])
		}
	}
	if cs.Rows[1]["City"] != "México D.F." {
		t.Errorf("csv city = %q", cs.Rows[1]["City"])
	}
	if _, err := DataLoadCSV(strings.NewReader(""), "X"); err == nil {
		t.Error("empty csv accepted")
	}
	if _, err := DataLoadJSON(strings.NewReader(`{"a":1}`), "X"); err == nil {
		t.Error("non-array json accepted")
	}
	if _, err := DataLoadXML(strings.NewReader(`<a/>`), "X"); err == nil {
		t.Error("recordless xml accepted")
	}
}

func TestDataLookupRelation(t *testing.T) {
	d := &DataDictionary{
		Sources: []DataSource{
			{Name: "Customers", Rows: []DataRow{{"CustomerID": "ALFKI"}, {"CustomerID": "ANATR"}}},
			{Name: "Orders", Rows: []DataRow{
				{"OrderID": "1", "CustomerID": "ALFKI"},
				{"OrderID": "2", "CustomerID": "ALFKI"},
				{"OrderID": "3", "CustomerID": "ANATR"},
			}},
		},
		Relations: []DataRelation{{
			Name: "CustomersOrders", Parent: "Customers", Child: "Orders",
			ParentCols: []string{"CustomerID"}, ChildCols: []string{"CustomerID"},
		}},
	}
	got := dataLookupRelation(d, "CustomersOrders", DataRow{"CustomerID": "ALFKI"})
	if len(got) != 2 || got[0]["OrderID"] != "1" || got[1]["OrderID"] != "2" {
		t.Fatalf("lookup = %v", got)
	}
	if n := len(dataLookupRelation(d, "CustomersOrders", DataRow{"CustomerID": "ZZZ"})); n != 0 {
		t.Fatalf("unknown parent matched %d rows", n)
	}
	if dataLookupRelation(d, "Nope", DataRow{"CustomerID": "ALFKI"}) != nil {
		t.Error("unknown relation matched rows")
	}
	if dataLookupRelation(nil, "CustomersOrders", DataRow{}) != nil {
		t.Error("nil dictionary matched rows")
	}
}

func TestDataAggregateTotal(t *testing.T) {
	rows := []DataRow{{"Orders": "6"}, {"Orders": "4"}, {"Orders": "7"}}
	for fn, want := range map[string]float64{"sum": 17, "count": 3, "avg": 17.0 / 3, "min": 4, "max": 7} {
		got, err := DataAggregateTotal(fn, "Orders", rows)
		if err != nil {
			t.Fatalf("%s: %v", fn, err)
		}
		if got != want {
			t.Errorf("%s = %v, want %v", fn, got, want)
		}
	}
	bad := []DataRow{{"Orders": "6"}, {"Orders": "abc"}}
	if _, err := DataAggregateTotal("sum", "Orders", bad); err == nil {
		t.Error("bad number accepted")
	}
	if _, err := DataAggregateTotal("median", "Orders", rows); err == nil {
		t.Error("unknown func accepted")
	}
	if _, err := DataAggregateTotal("avg", "Orders", nil); err == nil {
		t.Error("avg of no rows accepted")
	}
	if v, err := DataAggregateTotal("count", "Orders", nil); err != nil || v != 0 {
		t.Errorf("count of no rows = %v, %v", v, err)
	}
}
