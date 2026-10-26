// Package fontmetrics measures text widths for PDF base-14 fonts.
//
// The tables below are transcribed from the published Adobe Font Metrics for
// the standard PDF fonts, ASCII 32-126 only. They are layout-grade: famous
// invariants hold exactly (space 278, digits 556, Helvetica A 667 M 833 W
// 944; Courier uniformly 600) but individual glyphs were not certified
// against the AFM files. Non-ASCII input falls back to the face average and
// is reported by ASCIIOnly. Thai and other non-WinAnsi scripts cannot use
// base-14 fonts at all; embedding a real font is future work.
package fontmetrics

// Face is one measurable PDF base-14 font.
type Face int

const (
	Helvetica Face = iota
	HelveticaBold
	Courier
	CourierBold
)

// BaseName returns the PDF base-14 resource name.
func (f Face) BaseName() string {
	switch f {
	case HelveticaBold:
		return "Helvetica-Bold"
	case Courier:
		return "Courier"
	case CourierBold:
		return "Courier-Bold"
	default:
		return "Helvetica"
	}
}

// avgWidth is the fallback for code points outside the tables.
func (f Face) avgWidth() float64 {
	switch f {
	case Courier, CourierBold:
		return 600
	default:
		return 500
	}
}

// widths holds units-per-1000em for ASCII 32..126.
func (f Face) widths() [95]float64 {
	switch f {
	case HelveticaBold:
		return helvBold
	case Courier, CourierBold:
		return courier
	default:
		return helv
	}
}

// Width returns the width of s at size points.
func (f Face) Width(s string, size float64) float64 {
	var w float64
	for _, r := range s {
		if r >= 32 && r <= 126 {
			w += float64(f.widths()[r-32])
		} else {
			w += f.avgWidth()
		}
	}
	return w * size / 1000
}

// ASCIIOnly reports whether s renders exactly (printable ASCII only).
func ASCIIOnly(s string) bool {
	for _, r := range s {
		if r < 32 || r > 126 {
			return false
		}
	}
	return true
}

var courier = [95]float64{
	600, 600, 600, 600, 600, 600, 600, 600, 600, 600, 600, 600, 600, 600, 600, 600,
	600, 600, 600, 600, 600, 600, 600, 600, 600, 600, 600, 600, 600, 600, 600, 600,
	600, 600, 600, 600, 600, 600, 600, 600, 600, 600, 600, 600, 600, 600, 600, 600,
	600, 600, 600, 600, 600, 600, 600, 600, 600, 600, 600, 600, 600, 600, 600, 600,
	600, 600, 600, 600, 600, 600, 600, 600, 600, 600, 600, 600, 600, 600, 600, 600,
	600, 600, 600, 600, 600, 600, 600, 600, 600, 600, 600, 600, 600, 600, 600,
}

var helv = [95]float64{
	// 32..47: space ! " # $ % & ' ( ) * + , - . /
	278, 333, 355, 556, 556, 889, 667, 191, 333, 333, 389, 584, 278, 333, 278, 278,
	// 48..63: 0-9 : ; < = > ?
	556, 556, 556, 556, 556, 556, 556, 556, 556, 556, 278, 278, 584, 584, 584, 556,
	// 64..79: @ A-O
	1015, 667, 667, 722, 722, 667, 611, 778, 722, 278, 500, 667, 556, 833, 722, 778,
	// 80..95: P-Z [ \ ] ^ _
	667, 778, 667, 667, 611, 722, 667, 944, 667, 667, 611, 278, 278, 278, 469, 556,
	// 96..111: ` a-o
	333, 556, 556, 500, 556, 556, 278, 556, 556, 222, 222, 500, 222, 833, 556, 556,
	// 112..126: p-z { | } ~
	556, 556, 333, 500, 278, 556, 500, 722, 500, 500, 500, 334, 260, 334, 584,
}

var helvBold = [95]float64{
	// 32..47
	278, 333, 474, 556, 556, 889, 722, 238, 333, 333, 389, 584, 278, 333, 278, 278,
	// 48..63
	556, 556, 556, 556, 556, 556, 556, 556, 556, 556, 333, 333, 584, 584, 584, 611,
	// 64..79
	975, 722, 722, 722, 722, 667, 611, 778, 722, 278, 556, 722, 611, 833, 722, 778,
	// 80..95
	667, 778, 722, 722, 667, 611, 722, 667, 944, 667, 667, 611, 333, 278, 333, 569,
	// 96..111
	333, 556, 611, 556, 611, 556, 333, 611, 611, 278, 278, 556, 278, 889, 611, 611,
	// 112..126
	611, 611, 389, 556, 333, 611, 556, 778, 556, 556, 500, 392, 280, 392, 584,
}
