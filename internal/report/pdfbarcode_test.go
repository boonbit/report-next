package report

import (
	"bytes"
	"testing"

	"report-next/internal/aztec"
	"report-next/internal/datamatrix"
	"report-next/internal/maxicode"
	"report-next/internal/pdf417"
	"report-next/internal/qr"
)

func pdfRectCount(b []byte) int {
	return bytes.Count(b, []byte(" re f\n"))
}

func pdfDarkCount(grid [][]bool) int {
	n := 0
	for _, row := range grid {
		for _, v := range row {
			if v {
				n++
			}
		}
	}
	return n
}

func TestPDFExportBarcodeDataMatrix(t *testing.T) {
	rep := loadHello(t)
	rep.Pages[0].Bands[0].Objects = append(rep.Pages[0].Bands[0].Objects,
		Object{Kind: "BarcodeObject", Name: "DM",
			Attrs: objAttrs("Left", "10", "Top", "10", "Width", "100", "Height", "100",
				"Text", "AB", "Barcode", "Datamatrix")})
	out, skipped, err := pdfExportReport(rep)
	if err != nil {
		t.Fatal(err)
	}
	if skipped != 0 {
		t.Fatalf("skipped = %d, want 0", skipped)
	}
	code, err := datamatrix.Encode([]byte("AB"), 0)
	if err != nil {
		t.Fatal(err)
	}
	if want := pdfDarkCount(code.Matrix()); pdfRectCount(out) != want {
		t.Fatalf("rects = %d, want %d dark modules", pdfRectCount(out), want)
	}
	if !bytes.Contains(out, []byte("(AB)")) {
		t.Fatal("datamatrix caption missing from pdf")
	}
}

func TestPDFExportBarcodeQR(t *testing.T) {
	rep := loadHello(t)
	rep.Pages[0].Bands[0].Objects = append(rep.Pages[0].Bands[0].Objects,
		Object{Kind: "BarcodeObject", Name: "QR",
			Attrs: objAttrs("Left", "10", "Top", "10", "Width", "100", "Height", "100",
				"Text", "http://fast-report.com", "Barcode", "QR Code",
				"Barcode.ErrorCorrection", "L", "ShowText", "false")})
	out, skipped, err := pdfExportReport(rep)
	if err != nil {
		t.Fatal(err)
	}
	if skipped != 0 {
		t.Fatalf("skipped = %d, want 0", skipped)
	}
	code, err := qr.QrEncodeWithLevel([]byte("http://fast-report.com"), qr.QrL)
	if err != nil {
		t.Fatal(err)
	}
	if want := pdfDarkCount(code.QrMatrix()); pdfRectCount(out) != want {
		t.Fatalf("rects = %d, want %d dark modules", pdfRectCount(out), want)
	}
	if bytes.Contains(out, []byte("fast-report")) {
		t.Fatal("ShowText=false must hide the caption")
	}
}

func TestPDFExportBarcodeCode39(t *testing.T) {
	rep := loadHello(t)
	rep.Pages[0].Bands[0].Objects = append(rep.Pages[0].Bands[0].Objects,
		Object{Kind: "BarcodeObject", Name: "C39",
			Attrs: objAttrs("Left", "10", "Top", "10", "Width", "200", "Height", "50",
				"Text", "123456")})
	out, skipped, err := pdfExportReport(rep)
	if err != nil {
		t.Fatal(err)
	}
	if skipped != 0 {
		t.Fatalf("skipped = %d, want 0", skipped)
	}
	want := 0
	for i := 0; i < len(objCode39Pattern("123456")); i++ {
		if c := objCode39Pattern("123456")[i]; c == '5' || c == '6' {
			want++
		}
	}
	if pdfRectCount(out) != want {
		t.Fatalf("rects = %d, want %d bars", pdfRectCount(out), want)
	}
	if !bytes.Contains(out, []byte("(123456)")) {
		t.Fatal("code39 caption missing from pdf")
	}
}

func TestPDFExportBarcodePDF417(t *testing.T) {
	rep := loadHello(t)
	rep.Pages[0].Bands[0].Objects = append(rep.Pages[0].Bands[0].Objects,
		Object{Kind: "BarcodeObject", Name: "P417",
			Attrs: objAttrs("Left", "10", "Top", "10", "Width", "200", "Height", "100",
				"Text", "AB", "Barcode", "PDF417", "Barcode.ErrorCorrection", "Auto")})
	out, skipped, err := pdfExportReport(rep)
	if err != nil {
		t.Fatal(err)
	}
	if skipped != 0 {
		t.Fatalf("skipped = %d, want 0", skipped)
	}
	code, err := pdf417.Encode([]byte("AB"), 2)
	if err != nil {
		t.Fatal(err)
	}
	if want := pdfDarkCount(code.Matrix()); pdfRectCount(out) != want {
		t.Fatalf("rects = %d, want %d dark modules", pdfRectCount(out), want)
	}
	if !bytes.Contains(out, []byte("(AB)")) {
		t.Fatal("pdf417 caption missing from pdf")
	}
}

func TestPDFExportBarcodeAztec(t *testing.T) {
	rep := loadHello(t)
	rep.Pages[0].Bands[0].Objects = append(rep.Pages[0].Bands[0].Objects,
		Object{Kind: "BarcodeObject", Name: "Azt",
			Attrs: objAttrs("Left", "10", "Top", "10", "Width", "100", "Height", "100",
				"Text", "tel:(71) 555-4444", "Barcode", "Aztec", "Barcode.ErrorCorrection", "33",
				"ShowText", "false")})
	out, skipped, err := pdfExportReport(rep)
	if err != nil {
		t.Fatal(err)
	}
	if skipped != 0 {
		t.Fatalf("skipped = %d, want 0", skipped)
	}
	code, err := aztec.Encode([]byte("tel:(71) 555-4444"), 33, 0)
	if err != nil {
		t.Fatal(err)
	}
	if want := pdfDarkCount(code.Matrix()); pdfRectCount(out) != want {
		t.Fatalf("rects = %d, want %d dark modules", pdfRectCount(out), want)
	}
	if bytes.Contains(out, []byte("555-4444")) {
		t.Fatal("ShowText=false must hide caption")
	}
}

func TestPDFExportBarcodeFallback(t *testing.T) {
	rep := loadHello(t)
	rep.Pages[0].Bands[0].Objects = append(rep.Pages[0].Bands[0].Objects,
		Object{Kind: "BarcodeObject", Name: "Mxi",
			Attrs: objAttrs("Left", "10", "Top", "10", "Width", "100", "Height", "100",
				"Text", "123456", "Barcode", "MaxiCode", "Barcode.Mode", "9")},
		Object{Kind: "BarcodeObject", Name: "P417bad",
			Attrs: objAttrs("Left", "10", "Top", "120", "Width", "100", "Height", "100",
				"Text", "123456", "Barcode", "PDF417", "Barcode.ErrorCorrection", "99")},
		Object{Kind: "BarcodeObject", Name: "DMbig",
			Attrs: objAttrs("Left", "10", "Top", "230", "Width", "100", "Height", "100",
				"Text", "1234567890", "Barcode", "Datamatrix", "Barcode.SymbolSize", "Size10x10")})
	out, skipped, err := pdfExportReport(rep)
	if err != nil {
		t.Fatal(err)
	}
	if skipped != 0 {
		t.Fatalf("skipped = %d, want 0", skipped)
	}
	if got := pdfRectCount(out); got != 0 {
		t.Fatalf("rects = %d, want 0 for unsupported/oversize 2D", got)
	}
	if bytes.Contains(out, []byte("MaxiCode")) || bytes.Contains(out, []byte("(1234567890)")) {
		t.Fatal("fallback must draw neither symbol nor caption")
	}
}

func TestPDFExportBarcodeMaxi(t *testing.T) {
	rep := loadHello(t)
	rep.Pages[0].Bands[0].Objects = append(rep.Pages[0].Bands[0].Objects,
		Object{Kind: "BarcodeObject", Name: "Mxi",
			Attrs: objAttrs("Left", "10", "Top", "10", "Width", "100", "Height", "110",
				"Text", "HELLO WORLD 123", "Barcode", "MaxiCode", "Barcode.Mode", "4")})
	out, skipped, err := pdfExportReport(rep)
	if err != nil {
		t.Fatal(err)
	}
	if skipped != 0 {
		t.Fatalf("skipped = %d, want 0", skipped)
	}
	code, err := maxicode.Encode(4, []byte("HELLO WORLD 123"))
	if err != nil {
		t.Fatal(err)
	}
	// Dark modules plus the finder core overlay rect.
	if want := pdfDarkCount(code.Matrix()) + 1; pdfRectCount(out) != want {
		t.Fatalf("rects = %d, want %d", pdfRectCount(out), want)
	}
	if !bytes.Contains(out, []byte("(HELLO WORLD 123)")) {
		t.Fatal("maxicode caption missing from pdf")
	}
}

func TestPDFExportBarcodeNonASCIICaption(t *testing.T) {
	rep := loadHello(t)
	rep.Pages[0].Bands[0].Objects = append(rep.Pages[0].Bands[0].Objects,
		Object{Kind: "BarcodeObject", Name: "QRth",
			Attrs: objAttrs("Left", "10", "Top", "10", "Width", "100", "Height", "100",
				"Text", "สวัสดี", "Barcode", "QR Code")})
	out, skipped, err := pdfExportReport(rep)
	if err != nil {
		t.Fatal(err)
	}
	if skipped != 1 {
		t.Fatalf("skipped = %d, want 1 for the non-ASCII caption", skipped)
	}
	if pdfRectCount(out) == 0 {
		t.Fatal("qr symbol should still be drawn")
	}
}
