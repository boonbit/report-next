package report

import (
	"encoding/base64"
	"strings"
	"testing"
)

// bindPNG1x1 is a 1x1 transparent PNG used as the golden image payload.
const bindPNG1x1 = "iVBORw0KGgoAAAANSUhEUgAAAAEAAAABCAYAAAAfFcSJAAAADUlEQVR42mP8z8BQDwAEhQGAhKmMIQAAAABJRU5ErkJggg=="

func bindPicAttrs(col string) []Attr {
	return objAttrs("Left", "9.45", "Top", "28.35", "Width", "103.95", "Height", "132.3",
		"DataColumn", col)
}

func TestPictureBoundDataURIGolden(t *testing.T) {
	o := Object{Kind: "PictureObject", Name: "Picture1", Attrs: bindPicAttrs("Products.Photo")}
	fields := map[string]string{"Photo": bindPNG1x1}
	got := objPictureBoundHTML(o, fields, nil)
	if !strings.Contains(got, "<img") || !strings.Contains(got, "data:image/png;base64,"+bindPNG1x1) {
		t.Fatalf("png data URI missing:\n%s", got)
	}
	if !strings.Contains(got, `data-kind="PictureObject"`) {
		t.Errorf("picture kind missing:\n%s", got)
	}
	// Bracketed refs resolve through the expression path too.
	oBrack := Object{Kind: "PictureObject", Name: "Picture1", Attrs: bindPicAttrs("[Products.Photo]")}
	if got := objPictureBoundHTML(oBrack, fields, nil); !strings.Contains(got, "data:image/png;base64,") {
		t.Errorf("bracketed ref missing data URI:\n%s", got)
	}
	// Params are a fallback source when fields lack the column.
	if got := objPictureBoundHTML(o, nil, map[string]string{"Photo": bindPNG1x1}); !strings.Contains(got, "data:image/png;base64,") {
		t.Errorf("param ref missing data URI:\n%s", got)
	}
	// Raw (non-base64) binary bytes stored as a string render as well.
	raw, err := base64.StdEncoding.DecodeString(bindPNG1x1)
	if err != nil {
		t.Fatal(err)
	}
	if got := objPictureBoundHTML(o, map[string]string{"Photo": string(raw)}, nil); !strings.Contains(got, "data:image/png;base64,"+bindPNG1x1) {
		t.Errorf("raw bytes missing canonical data URI:\n%s", got)
	}
	// JPEG magic sniffs to image/jpeg.
	jpeg := base64.StdEncoding.EncodeToString([]byte{0xFF, 0xD8, 0xFF, 0xE0, 0x00, 0x01, 0x02, 0x03})
	if got := objPictureBoundHTML(o, map[string]string{"Photo": jpeg}, nil); !strings.Contains(got, "data:image/jpeg;base64,") {
		t.Errorf("jpeg magic missing data URI:\n%s", got)
	}
	// A data: URI value is accepted and re-emitted canonically.
	if got := objPictureBoundHTML(o, map[string]string{"Photo": "data:image/png;base64," + bindPNG1x1}, nil); !strings.Contains(got, "data:image/png;base64,"+bindPNG1x1) {
		t.Errorf("data URI input missing canonical output:\n%s", got)
	}
}

func TestPictureBoundFallback(t *testing.T) {
	o := Object{Kind: "PictureObject", Name: "Picture1", Attrs: bindPicAttrs("Products.Photo")}
	// Unresolvable without row context: the existing labeled placeholder.
	if got := objPictureBoundHTML(o, nil, nil); !strings.Contains(got, "Picture: Products.Photo") {
		t.Errorf("unresolvable ref missing placeholder:\n%s", got)
	}
	// Non-image bytes fall back to the placeholder.
	if got := objPictureBoundHTML(o, map[string]string{"Photo": "not-an-image!!!"}, nil); !strings.Contains(got, "Picture: Products.Photo") {
		t.Errorf("non-image missing placeholder:\n%s", got)
	}
	// Empty values fall back to the placeholder.
	if got := objPictureBoundHTML(o, map[string]string{"Photo": ""}, nil); !strings.Contains(got, "Picture: Products.Photo") {
		t.Errorf("empty value missing placeholder:\n%s", got)
	}
	// No DataColumn at all falls back to the name label.
	bare := Object{Kind: "PictureObject", Name: "Picture1",
		Attrs: objAttrs("Left", "9.45", "Width", "103.95", "Height", "132.3")}
	if got := objPictureBoundHTML(bare, map[string]string{"Photo": bindPNG1x1}, nil); !strings.Contains(got, "Picture: Picture1") {
		t.Errorf("bare picture missing name placeholder:\n%s", got)
	}
}

func TestCheckBoxBoundTruthTable(t *testing.T) {
	truthy := []string{"true", "True", "TRUE", "checked", "Checked", "CHECKED",
		"yes", "Yes", "YES", "y", "on", "1", "2", "-1", "0.5", "some text"}
	falsy := []string{"", "false", "False", "FALSE", "unchecked", "Unchecked",
		"no", "No", "n", "off", "Off", "0", "0.0", "   "}
	for _, v := range truthy {
		// Static Checked="false" must lose to a truthy bound value.
		o := Object{Kind: "CheckBoxObject", Name: "C",
			Attrs: objAttrs("Width", "18.9", "Height", "18.9",
				"Checked", "false", "DataColumn", "Products.Discontinued")}
		if got := objCheckBoxBoundHTML(o, map[string]string{"Discontinued": v}, nil); !strings.Contains(got, "&#10003;") {
			t.Errorf("bound %q: want tick, got:\n%s", v, got)
		}
	}
	for _, v := range falsy {
		// Static Checked="true" must lose to a falsy bound value.
		o := Object{Kind: "CheckBoxObject", Name: "C",
			Attrs: objAttrs("Width", "18.9", "Height", "18.9",
				"Checked", "true", "DataColumn", "Products.Discontinued")}
		if got := objCheckBoxBoundHTML(o, map[string]string{"Discontinued": v}, nil); strings.Contains(got, "&#10003;") || strings.Contains(got, "&#10005;") {
			t.Errorf("bound %q: want unchecked, got:\n%s", v, got)
		}
	}
	// Bracketed CheckState refs resolve against fields.
	o := Object{Kind: "CheckBoxObject", Name: "C",
		Attrs: objAttrs("Width", "18.9", "Height", "18.9",
			"Checked", "false", "CheckState", "[Products.Flag]")}
	if got := objCheckBoxBoundHTML(o, map[string]string{"Flag": "yes"}, nil); !strings.Contains(got, "&#10003;") {
		t.Errorf("bracketed CheckState missing tick:\n%s", got)
	}
	// Unresolvable refs keep the static rendering.
	static := Object{Kind: "CheckBoxObject", Name: "C",
		Attrs: objAttrs("Width", "18.9", "Height", "18.9",
			"Checked", "true", "DataColumn", "Products.Missing")}
	if got := objCheckBoxBoundHTML(static, map[string]string{"Other": "false"}, nil); !strings.Contains(got, "&#10003;") {
		t.Errorf("unresolvable ref should keep static tick:\n%s", got)
	}
}

func TestCheckBoxStaticBackwardCompat(t *testing.T) {
	// Checked="true" still ticks with no row context.
	got := objCheckBoxHTML(Object{Kind: "CheckBoxObject", Name: "C",
		Attrs: objAttrs("Width", "18.9", "Height", "18.9", "Checked", "true")})
	if !strings.Contains(got, "&#10003;") {
		t.Errorf("static Checked=true missing tick:\n%s", got)
	}
	// Checked="false" plus an unresolvable DataColumn still renders unchecked.
	got = objCheckBoxHTML(Object{Kind: "CheckBoxObject", Name: "C",
		Attrs: objAttrs("Width", "18.9", "Height", "18.9", "Checked", "false", "DataColumn", "Products.Discontinued")})
	if strings.Contains(got, "&#10003;") || strings.Contains(got, "&#10005;") {
		t.Errorf("static Checked=false shows a mark:\n%s", got)
	}
	// A static CheckState wins over a conflicting static Checked.
	got = objCheckBoxHTML(Object{Kind: "CheckBoxObject", Name: "C",
		Attrs: objAttrs("Width", "18.9", "Height", "18.9", "Checked", "false", "CheckState", "Checked")})
	if !strings.Contains(got, "&#10003;") {
		t.Errorf("static CheckState should win over Checked=false:\n%s", got)
	}
	got = objCheckBoxHTML(Object{Kind: "CheckBoxObject", Name: "C",
		Attrs: objAttrs("Width", "18.9", "Height", "18.9", "Checked", "true", "CheckState", "false")})
	if strings.Contains(got, "&#10003;") || strings.Contains(got, "&#10005;") {
		t.Errorf("static CheckState=false should win over Checked=true:\n%s", got)
	}
}
