package report

import "testing"

func TestSrvExportCSVFromDict(t *testing.T) {
	dict := &DataDictionary{Sources: []DataSource{{
		Name:    "Customers",
		Columns: []DataColumn{{Name: "Name"}, {Name: "City"}},
		Rows: []DataRow{
			{"Name": "Amy", "City": "Bangkok"},
			{"Name": "Bo, Jr.", "City": "Chiang \"Mai\""},
		},
	}}}
	got := srvExportCSV(dict, "Customers", nil)
	want := "Name,City\nAmy,Bangkok\n\"Bo, Jr.\",\"Chiang \"\"Mai\"\"\"\n"
	if got != want {
		t.Fatalf("csv = %q, want %q", got, want)
	}
}

func TestSrvExportCSVRowsOverride(t *testing.T) {
	rows := []DataRow{{"b": "2", "a": "1"}}
	got := srvExportCSV(nil, "Missing", rows)
	if got != "a,b\n1,2\n" {
		t.Fatalf("csv = %q", got)
	}
}

func TestSrvExportCSVEmpty(t *testing.T) {
	if got := srvExportCSV(nil, "Missing", nil); got != "" {
		t.Fatalf("csv = %q, want empty", got)
	}
	if got := srvExportCSV(&DataDictionary{}, "Missing", nil); got != "" {
		t.Fatalf("csv = %q, want empty", got)
	}
}
