// Package pdffont loads a bring-your-own TrueType font (BYOF) and builds
// small 8-bit PDF subsets covering ASCII plus Thai (U+0E00-U+0E7F).
//
// The parser reads the raw sfnt tables directly: cmap format 4 (BMP, with
// a format 12 fallback), hmtx advances, hhea/head metrics and the glyf
// outlines needed for subsetting. TrueType outlines ('glyf') are required;
// CFF-flavoured fonts ('OTTO') are rejected. TTC collections are accepted
// by selecting the first font with Thai coverage, else the first font.
//
// The subset keeps an 8-bit encoding: ASCII maps to itself, Thai runes map
// to 0x80+ slots in sorted order. The subset sfnt rewrites cmap so each
// used byte code addresses its glyph directly; /Widths and /ToUnicode are
// derived from the same assignment.
package pdffont

import (
	"encoding/binary"
	"fmt"
	"hash/fnv"
	"os"
	"sort"
	"strings"
	"unicode/utf16"
)

// FntEnvVar is the environment variable pointing at a user-supplied .ttf.
const FntEnvVar = "REPORT_FONT_TTF"

// Code ranges served by a subset.
const (
	fntASCIILo = 0x20
	fntASCIIHi = 0x7E
	fntThaiLo  = 0x0E00
	fntThaiHi  = 0x0E7F
	// fntExtraBase is the first code assigned to non-ASCII runes.
	fntExtraBase = 0x80
	// fntExtraSlots is the count of codes from fntExtraBase to 0xFF.
	fntExtraSlots = 128
)

// FntFont is a parsed TrueType font ready for subsetting.
type FntFont struct {
	raw  []byte
	base int // sfnt header offset inside raw (nonzero for TTC members)

	upm         int
	ascent      int // hhea, font units
	descent     int // hhea, font units (usually negative)
	xMin, yMin  int
	xMax, yMax  int
	italic      bool
	italicAngle float64
	capHeight   int // font units, 0 when unknown
	psName      string
	numGlyphs   int
	cmap        map[rune]uint16 // unicode code point -> glyph id
	adv         []int           // per-glyph advance width, font units
	lsb         []int           // per-glyph left side bearing, font units
	locaShort   bool
	head, hhea  []byte
	maxp, hmtx  []byte
	loca, glyf  []byte
	nameTab     []byte // optional, may be nil
	os2Tab      []byte // optional, may be nil
	postTab     []byte // optional, may be nil
}

// FntLoadTTF reads path and parses it as TTF (or TTC, see package doc).
func FntLoadTTF(path string) (*FntFont, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("pdffont: read %q: %w", path, err)
	}
	return FntParseTTF(data)
}

// fntLoadTTF is the track-named entry point for loading a font file.
func fntLoadTTF(path string) (*FntFont, error) {
	return FntLoadTTF(path)
}

// FntParseTTF parses an in-memory sfnt (TTF or TTC).
func FntParseTTF(data []byte) (*FntFont, error) {
	if len(data) < 12 {
		return nil, fmt.Errorf("pdffont: file too short")
	}
	if binary.BigEndian.Uint32(data) == 0x74746366 { // 'ttcf'
		return fntParseCollection(data)
	}
	return fntParseAt(data, 0)
}

// fntParseAt parses one sfnt header at absolute offset base.
func fntParseAt(data []byte, base int) (*FntFont, error) {
	if len(data) < base+12 {
		return nil, fmt.Errorf("pdffont: file too short")
	}
	ver := binary.BigEndian.Uint32(data[base:])
	switch ver {
	case 0x00010000, 0x74727565: // 1.0, 'true'
	case 0x4F54544F: // 'OTTO': CFF outlines, no glyf table to subset
		return nil, fmt.Errorf("pdffont: CFF outlines unsupported")
	default:
		return nil, fmt.Errorf("pdffont: bad sfnt version %#x", ver)
	}
	numTables := int(binary.BigEndian.Uint16(data[base+4:]))
	if len(data) < base+12+16*numTables {
		return nil, fmt.Errorf("pdffont: truncated table directory")
	}
	tabs := map[string][]byte{}
	for i := 0; i < numTables; i++ {
		rec := data[base+12+16*i:]
		tag := string(rec[:4])
		off := int(binary.BigEndian.Uint32(rec[8:]))
		n := int(binary.BigEndian.Uint32(rec[12:]))
		if off < 0 || n < 0 || len(data) < off+n {
			return nil, fmt.Errorf("pdffont: table %q out of range", tag)
		}
		if _, dup := tabs[tag]; !dup {
			tabs[tag] = data[off : off+n]
		}
	}
	for _, tag := range []string{"head", "hhea", "maxp", "hmtx", "cmap", "loca", "glyf"} {
		if len(tabs[tag]) == 0 {
			return nil, fmt.Errorf("pdffont: missing table %q", tag)
		}
	}
	f := &FntFont{raw: data, base: base}
	f.head, f.hhea = tabs["head"], tabs["hhea"]
	f.maxp, f.hmtx = tabs["maxp"], tabs["hmtx"]
	f.loca, f.glyf = tabs["loca"], tabs["glyf"]
	f.nameTab = tabs["name"]
	f.os2Tab = tabs["OS/2"]
	f.postTab = tabs["post"]
	if len(f.head) < 54 {
		return nil, fmt.Errorf("pdffont: short head table")
	}
	if binary.BigEndian.Uint32(f.head[12:]) != 0x5F0F3CF5 {
		return nil, fmt.Errorf("pdffont: bad head magic")
	}
	f.upm = int(binary.BigEndian.Uint16(f.head[18:]))
	if f.upm <= 0 {
		return nil, fmt.Errorf("pdffont: bad units-per-em %d", f.upm)
	}
	f.xMin = int(int16(binary.BigEndian.Uint16(f.head[36:])))
	f.yMin = int(int16(binary.BigEndian.Uint16(f.head[38:])))
	f.xMax = int(int16(binary.BigEndian.Uint16(f.head[40:])))
	f.yMax = int(int16(binary.BigEndian.Uint16(f.head[42:])))
	f.italic = binary.BigEndian.Uint16(f.head[44:])&0x02 != 0
	f.locaShort = int16(binary.BigEndian.Uint16(f.head[50:])) == 0
	if len(f.hhea) < 36 {
		return nil, fmt.Errorf("pdffont: short hhea table")
	}
	f.ascent = int(int16(binary.BigEndian.Uint16(f.hhea[4:])))
	f.descent = int(int16(binary.BigEndian.Uint16(f.hhea[6:])))
	numHMetrics := int(binary.BigEndian.Uint16(f.hhea[34:]))
	if len(f.maxp) < 6 {
		return nil, fmt.Errorf("pdffont: short maxp table")
	}
	f.numGlyphs = int(binary.BigEndian.Uint16(f.maxp[4:]))
	if f.numGlyphs < 1 {
		return nil, fmt.Errorf("pdffont: no glyphs")
	}
	if err := f.parseHmtx(numHMetrics); err != nil {
		return nil, err
	}
	cmap, err := fntParseCmap(tabs["cmap"])
	if err != nil {
		return nil, err
	}
	f.cmap = cmap
	f.psName = fntPostScriptName(tabs["name"])
	f.italicAngle = fntItalicAngle(tabs["post"])
	f.capHeight = fntCapHeight(tabs["OS/2"], f.ascent)
	return f, nil
}

// parseHmtx expands the horizontal metrics into per-glyph advances.
func (f *FntFont) parseHmtx(numHMetrics int) error {
	if numHMetrics < 1 || numHMetrics > f.numGlyphs {
		return fmt.Errorf("pdffont: bad numberOfHMetrics %d", numHMetrics)
	}
	want := 4*numHMetrics + 2*(f.numGlyphs-numHMetrics)
	if len(f.hmtx) < want {
		return fmt.Errorf("pdffont: short hmtx table")
	}
	f.adv = make([]int, f.numGlyphs)
	f.lsb = make([]int, f.numGlyphs)
	lastAdv := 0
	for i := 0; i < f.numGlyphs; i++ {
		if i < numHMetrics {
			lastAdv = int(binary.BigEndian.Uint16(f.hmtx[4*i:]))
			f.lsb[i] = int(int16(binary.BigEndian.Uint16(f.hmtx[4*i+2:])))
		} else {
			f.lsb[i] = int(int16(binary.BigEndian.Uint16(f.hmtx[4*numHMetrics+2*(i-numHMetrics):])))
		}
		f.adv[i] = lastAdv
	}
	return nil
}

// fntCmapSub records one candidate cmap subtable.
type fntCmapSub struct {
	platform, encoding uint16
	format             uint16
	offset             int
}

// fntParseCmap parses the cmap table, preferring a Unicode BMP format 4
// subtable and falling back to format 12.
func fntParseCmap(data []byte) (map[rune]uint16, error) {
	if len(data) < 4 {
		return nil, fmt.Errorf("pdffont: short cmap table")
	}
	num := int(binary.BigEndian.Uint16(data[2:]))
	var subs []fntCmapSub
	for i := 0; i < num; i++ {
		if len(data) < 4+8*i+8 {
			return nil, fmt.Errorf("pdffont: short cmap header")
		}
		rec := data[4+8*i:]
		off := int(binary.BigEndian.Uint32(rec[4:]))
		var format uint16
		if len(data) >= off+2 {
			format = binary.BigEndian.Uint16(data[off:])
		}
		subs = append(subs, fntCmapSub{
			platform: binary.BigEndian.Uint16(rec[0:]),
			encoding: binary.BigEndian.Uint16(rec[2:]),
			format:   format,
			offset:   off,
		})
	}
	// Preference: (3,1)/fmt4, (0,3)/fmt4, (3,10)/fmt12, (0,4)/fmt12,
	// then any fmt4, then any fmt12.
	pick := func(pid, eid uint16, format uint16) *fntCmapSub {
		for i := range subs {
			if subs[i].platform == pid && subs[i].encoding == eid && subs[i].format == format {
				return &subs[i]
			}
		}
		return nil
	}
	cand := pick(3, 1, 4)
	if cand == nil {
		cand = pick(0, 3, 4)
	}
	if cand == nil {
		cand = pick(3, 10, 12)
	}
	if cand == nil {
		cand = pick(0, 4, 12)
	}
	if cand == nil {
		for i := range subs {
			if subs[i].format == 4 {
				cand = &subs[i]
				break
			}
		}
	}
	if cand == nil {
		for i := range subs {
			if subs[i].format == 12 {
				cand = &subs[i]
				break
			}
		}
	}
	if cand == nil {
		return nil, fmt.Errorf("pdffont: no format 4/12 cmap subtable")
	}
	if cand.format == 12 {
		return fntParseCmap12(data[cand.offset:])
	}
	return fntParseCmap4(data[cand.offset:])
}

// fntParseCmap4 parses a format 4 (BMP segment) subtable.
func fntParseCmap4(data []byte) (map[rune]uint16, error) {
	if len(data) < 16 {
		return nil, fmt.Errorf("pdffont: short cmap format 4")
	}
	segCount := int(binary.BigEndian.Uint16(data[6:])) / 2
	if segCount < 1 || len(data) < 16+8*segCount {
		return nil, fmt.Errorf("pdffont: bad cmap format 4 segCount")
	}
	end := data[14:]
	start := data[16+2*segCount:]
	delta := data[16+4*segCount:]
	rangeOff := data[16+6*segCount:]
	m := map[rune]uint16{}
	for i := 0; i < segCount; i++ {
		e := int(binary.BigEndian.Uint16(end[2*i:]))
		s := int(binary.BigEndian.Uint16(start[2*i:]))
		d := int(int16(binary.BigEndian.Uint16(delta[2*i:])))
		r := int(binary.BigEndian.Uint16(rangeOff[2*i:]))
		if s == 0xFFFF && e == 0xFFFF {
			break
		}
		if e > 0xFFFF {
			e = 0xFFFF
		}
		for c := s; c <= e; c++ {
			var gid int
			if r == 0 {
				gid = (c + d) & 0xFFFF
			} else {
				p := 16 + 6*segCount + 2*i + r + 2*(c-s)
				if p+2 > len(data) {
					return nil, fmt.Errorf("pdffont: cmap glyph index out of range")
				}
				gid = int(binary.BigEndian.Uint16(data[p:]))
				if gid != 0 {
					gid = (gid + d) & 0xFFFF
				}
			}
			if gid != 0 {
				m[rune(c)] = uint16(gid)
			}
		}
	}
	return m, nil
}

// fntParseCmap12 parses a format 12 (full Unicode segment) subtable.
func fntParseCmap12(data []byte) (map[rune]uint16, error) {
	if len(data) < 16 {
		return nil, fmt.Errorf("pdffont: short cmap format 12")
	}
	n := int(binary.BigEndian.Uint32(data[12:]))
	if len(data) < 16+12*n {
		return nil, fmt.Errorf("pdffont: bad cmap format 12 groups")
	}
	m := map[rune]uint16{}
	for i := 0; i < n; i++ {
		g := data[16+12*i:]
		s := binary.BigEndian.Uint32(g[0:])
		e := binary.BigEndian.Uint32(g[4:])
		gid := binary.BigEndian.Uint32(g[8:])
		for c := s; c <= e; c++ {
			if gid != 0 {
				m[rune(c)] = uint16(gid & 0xFFFF)
			}
			gid++
			if c == e {
				break // avoid overflow at 0xFFFFFFFF
			}
		}
	}
	return m, nil
}

// fntPostScriptName reads name ID 6 (PostScript name), preferring Unicode
// English, and sanitizes it for use in a PDF BaseFont name.
func fntPostScriptName(data []byte) string {
	name := fntNameString(data, 6)
	if name == "" {
		return "FntFont"
	}
	var sb strings.Builder
	for _, r := range name {
		if r >= 'A' && r <= 'Z' || r >= 'a' && r <= 'z' || r >= '0' && r <= '9' {
			sb.WriteRune(r)
		} else {
			sb.WriteByte('-')
		}
	}
	out := sb.String()
	if out == "" {
		return "FntFont"
	}
	if out[0] >= '0' && out[0] <= '9' {
		out = "Fnt" + out
	}
	return out
}

// fntNameString decodes one name record by id.
func fntNameString(data []byte, id uint16) string {
	if len(data) < 6 {
		return ""
	}
	count := int(binary.BigEndian.Uint16(data[2:]))
	strOff := int(binary.BigEndian.Uint16(data[4:]))
	type rec struct {
		plat, enc, lang uint16
		s               []byte
	}
	var recs []rec
	for i := 0; i < count; i++ {
		if len(data) < 6+12*i+12 {
			return ""
		}
		r := data[6+12*i:]
		n := int(binary.BigEndian.Uint16(r[8:]))
		off := int(binary.BigEndian.Uint16(r[10:]))
		if strOff+off+n > len(data) {
			continue
		}
		if binary.BigEndian.Uint16(r[6:]) != id {
			continue
		}
		recs = append(recs, rec{binary.BigEndian.Uint16(r[0:]), binary.BigEndian.Uint16(r[2:]), binary.BigEndian.Uint16(r[4:]), data[strOff+off : strOff+off+n]})
	}
	decode := func(r rec) string {
		if r.plat == 1 { // Macintosh: MacRoman, ASCII-safe subset
			var sb strings.Builder
			for _, b := range r.s {
				if b < 0x80 {
					sb.WriteByte(b)
				} else {
					sb.WriteByte('?')
				}
			}
			return sb.String()
		}
		if len(r.s)%2 == 1 {
			return ""
		}
		u := make([]uint16, len(r.s)/2)
		for i := range u {
			u[i] = binary.BigEndian.Uint16(r.s[2*i:])
		}
		return string(utf16.Decode(u))
	}
	best := ""
	bestScore := -1
	for _, r := range recs {
		score := 0
		switch {
		case r.plat == 3 && r.enc == 1 && r.lang == 0x409:
			score = 4
		case r.plat == 3 && r.enc == 1:
			score = 3
		case r.plat == 0:
			score = 2
		case r.plat == 1 && r.enc == 0:
			score = 1
		}
		if score > bestScore {
			if s := decode(r); s != "" {
				best, bestScore = s, score
			}
		}
	}
	return best
}

// fntItalicAngle reads the post table italic angle in degrees.
func fntItalicAngle(data []byte) float64 {
	if len(data) < 8 {
		return 0
	}
	return float64(int32(binary.BigEndian.Uint32(data[4:]))) / 65536
}

// fntCapHeight reads OS/2 sCapHeight (offset 88, v2+), else falls back to
// the hhea ascent passed in.
func fntCapHeight(data []byte, ascent int) int {
	if len(data) >= 90 && binary.BigEndian.Uint16(data[0:]) >= 2 {
		if v := int(int16(binary.BigEndian.Uint16(data[88:]))); v > 0 && v <= 2*ascent+2000 {
			return v
		}
	}
	return ascent
}

// fntScale converts font units to thousandths of an em.
func (f *FntFont) fntScale(v int) float64 {
	return float64(v) * 1000 / float64(f.upm)
}

// FntUnitsPerEm returns the head units-per-em value.
func (f *FntFont) FntUnitsPerEm() int { return f.upm }

// FntAscent returns the hhea ascent in thousandths of an em.
func (f *FntFont) FntAscent() float64 { return f.fntScale(f.ascent) }

// FntDescent returns the hhea descent in thousandths of an em (negative).
func (f *FntFont) FntDescent() float64 { return f.fntScale(f.descent) }

// FntCapHeight returns the cap height in thousandths of an em.
func (f *FntFont) FntCapHeight() float64 { return f.fntScale(f.capHeight) }

// FntBBox returns the head bounding box in thousandths of an em.
func (f *FntFont) FntBBox() [4]float64 {
	return [4]float64{f.fntScale(f.xMin), f.fntScale(f.yMin), f.fntScale(f.xMax), f.fntScale(f.yMax)}
}

// FntPostScriptName returns the sanitized PostScript name (name ID 6).
func (f *FntFont) FntPostScriptName() string { return f.psName }

// FntItalicAngle returns the post table italic angle in degrees.
func (f *FntFont) FntItalicAngle() float64 { return f.italicAngle }

// FntHas reports whether r maps to a real (nonzero) glyph.
func (f *FntFont) FntHas(r rune) bool {
	gid, ok := f.cmap[r]
	return ok && gid != 0
}

// FntGlyphIndex returns the glyph id for r, or -1 when missing.
func (f *FntFont) FntGlyphIndex(r rune) int {
	gid, ok := f.cmap[r]
	if !ok || gid == 0 {
		return -1
	}
	return int(gid)
}

// FntAdvance returns the advance width of r in thousandths of an em.
// Missing runes fall back to the .notdef (gid 0) advance.
func (f *FntFont) FntAdvance(r rune) int {
	gid := 0
	if g, ok := f.cmap[r]; ok {
		gid = int(g)
	}
	if gid < 0 || gid >= len(f.adv) {
		return 0
	}
	return int(float64(f.adv[gid])*1000/float64(f.upm) + 0.5)
}

// FntWidth returns the width of s at size points.
func (f *FntFont) FntWidth(s string, size float64) float64 {
	var w int
	for _, r := range s {
		w += f.FntAdvance(r)
	}
	return float64(w) * size / 1000
}

// FntMissing lists distinct runes of s with no glyph, in sorted order.
func (f *FntFont) FntMissing(s string) []rune {
	var missing []rune
	seen := map[rune]bool{}
	for _, r := range s {
		if !seen[r] {
			seen[r] = true
			if !f.FntHas(r) {
				missing = append(missing, r)
			}
		}
	}
	sort.Slice(missing, func(i, j int) bool { return missing[i] < missing[j] })
	return missing
}

// FntSubset is one 8-bit subset: the sfnt bytes plus the code assignment
// (rune -> byte code) and the PDF structures derived from it.
type FntSubset struct {
	Data      []byte         // standalone subset sfnt
	Tag       string         // 6 uppercase letters, subset prefix
	Name      string         // sanitized PostScript name
	FirstChar int            // lowest used byte code
	LastChar  int            // highest used byte code
	Widths    []int          // per code FirstChar..LastChar, 1000em units
	Names     map[int]string // code -> PDF glyph name (/uniXXXX)
	Unicodes  map[int]rune   // code -> unicode code point
	Encode    map[rune]byte  // rune -> byte code

	Ascent      float64
	Descent     float64
	CapHeight   float64
	BBox        [4]float64
	ItalicAngle float64
}

// FntEncodeString encodes s into the subset byte codes as a Go string
// (bytes 0x00-0xFF preserved). Runes outside the subset are an error.
func (s *FntSubset) FntEncodeString(str string) (string, error) {
	var sb strings.Builder
	sb.Grow(len(str))
	for _, r := range str {
		c, ok := s.Encode[r]
		if !ok {
			return "", fmt.Errorf("pdffont: rune U+%04X not in subset", r)
		}
		sb.WriteByte(c)
	}
	return sb.String(), nil
}

// FntToUnicodeCMap renders the /ToUnicode CMap mapping each used code
// back to its unicode code point (surrogate pairs beyond the BMP).
func (s *FntSubset) FntToUnicodeCMap() string {
	codes := make([]int, 0, len(s.Unicodes))
	for c := range s.Unicodes {
		codes = append(codes, c)
	}
	sort.Ints(codes)
	var sb strings.Builder
	sb.WriteString("/CIDInit /ProcSet findresource begin\n12 dict begin\nbegincmap\n")
	sb.WriteString("/CIDSystemInfo <</Registry (Adobe) /Ordering (UCS) /Supplement 0>> def\n")
	sb.WriteString("/CMapName /Adobe-Identity-UCS def\n/CMapType 2 def\n")
	sb.WriteString("1 begincodespacerange\n<00> <FF>\nendcodespacerange\n")
	fmt.Fprintf(&sb, "%d beginbfchar\n", len(codes))
	for _, c := range codes {
		sb.WriteString("<")
		sb.WriteString(fntHexByte(byte(c)))
		sb.WriteString("> <")
		sb.WriteString(fntHexUTF16(s.Unicodes[c]))
		sb.WriteString(">\n")
	}
	sb.WriteString("endbfchar\nendcmap\nCMapName currentdict /CMap defineresource pop\nend\nend\n")
	return sb.String()
}

func fntHexByte(b byte) string {
	const digits = "0123456789ABCDEF"
	return string([]byte{digits[b>>4], digits[b&0x0F]})
}

// fntHexUTF16 renders r as UTF-16BE hex (surrogate pair beyond the BMP).
func fntHexUTF16(r rune) string {
	var sb strings.Builder
	for _, u := range utf16.Encode([]rune{r}) {
		fmt.Fprintf(&sb, "%04X", u)
	}
	return sb.String()
}

// FntSubsetFor builds a subset covering exactly the distinct runes given.
// ASCII runes keep their codes; other runes take 0x80+ slots in sorted
// order. Runes without glyphs, or more non-ASCII runes than slots, are an
// error naming the first offender.
func (f *FntFont) FntSubsetFor(runes []rune) (*FntSubset, error) {
	uniq := map[rune]bool{}
	for _, r := range runes {
		uniq[r] = true
	}
	var ascii, extra []rune
	for r := range uniq {
		if r >= fntASCIILo && r <= fntASCIIHi {
			ascii = append(ascii, r)
		} else {
			extra = append(extra, r)
		}
	}
	sort.Slice(ascii, func(i, j int) bool { return ascii[i] < ascii[j] })
	sort.Slice(extra, func(i, j int) bool { return extra[i] < extra[j] })
	if len(extra) > fntExtraSlots {
		return nil, fmt.Errorf("pdffont: %d non-ascii runes exceed %d slots", len(extra), fntExtraSlots)
	}
	for _, r := range extra {
		if !f.FntHas(r) {
			return nil, fmt.Errorf("pdffont: no glyph for U+%04X", r)
		}
	}
	for _, r := range ascii {
		if !f.FntHas(r) {
			return nil, fmt.Errorf("pdffont: no glyph for U+%04X", r)
		}
	}
	encode := map[rune]byte{}
	var codes []int
	for _, r := range ascii {
		encode[r] = byte(r)
		codes = append(codes, int(r))
	}
	for i, r := range extra {
		encode[r] = byte(fntExtraBase + i)
		codes = append(codes, fntExtraBase+i)
	}
	sort.Ints(codes)
	newGids, codeToNew, remap, err := f.fntCloseGlyphs(encode)
	if err != nil {
		return nil, err
	}
	data, err := f.fntBuildSubset(newGids, codeToNew, remap)
	if err != nil {
		return nil, err
	}
	sub := &FntSubset{
		Data:        data,
		Tag:         fntSubsetTag(f.psName, codes),
		Name:        f.psName,
		Encode:      encode,
		Names:       map[int]string{},
		Unicodes:    map[int]rune{},
		Ascent:      f.FntAscent(),
		Descent:     f.FntDescent(),
		CapHeight:   f.FntCapHeight(),
		BBox:        f.FntBBox(),
		ItalicAngle: f.italicAngle,
	}
	sub.FirstChar, sub.LastChar = codes[0], codes[len(codes)-1]
	advOf := map[int]int{} // code -> advance
	for code, newGid := range codeToNew {
		advOf[code] = int(float64(f.adv[int(newGids[newGid])])*1000/float64(f.upm) + 0.5)
	}
	for c := sub.FirstChar; c <= sub.LastChar; c++ {
		sub.Widths = append(sub.Widths, advOf[c])
		if newGid, ok := codeToNew[c]; ok {
			_ = newGid
			for r, b := range encode {
				if int(b) == c {
					sub.Names[c] = fmt.Sprintf("uni%04X", r)
					sub.Unicodes[c] = r
					break
				}
			}
		} else {
			sub.Names[c] = ".notdef"
		}
	}
	return sub, nil
}

// fntGlyphData returns the raw glyph bytes for an old glyph id.
func (f *FntFont) fntGlyphData(gid int) ([]byte, error) {
	n := f.numGlyphs
	var offs []int
	if f.locaShort {
		if len(f.loca) < 2*(n+1) {
			return nil, fmt.Errorf("pdffont: short loca table")
		}
		for i := 0; i <= n; i++ {
			offs = append(offs, 2*int(binary.BigEndian.Uint16(f.loca[2*i:])))
		}
	} else {
		if len(f.loca) < 4*(n+1) {
			return nil, fmt.Errorf("pdffont: short loca table")
		}
		for i := 0; i <= n; i++ {
			offs = append(offs, int(binary.BigEndian.Uint32(f.loca[4*i:])))
		}
	}
	if gid < 0 || gid >= n || offs[gid] > offs[gid+1] || offs[gid+1] > len(f.glyf) {
		return nil, fmt.Errorf("pdffont: bad loca entry %d", gid)
	}
	return f.glyf[offs[gid]:offs[gid+1]], nil
}

// Component glyph flags (TrueType glyf composites).
const (
	fntFlagArgsWords     = 0x0001
	fntFlagWeHaveScale   = 0x0008
	fntFlagMoreComps     = 0x0020
	fntFlagWeHaveXYScale = 0x0040
	fntFlagWeHave2x2     = 0x0080
	fntFlagWeHaveInstr   = 0x0100
)

// fntCompositeComponents lists component glyph ids of a composite glyph,
// or nil when the glyph is simple (or empty).
func fntCompositeComponents(glyph []byte) ([]uint16, error) {
	if len(glyph) < 10 || int16(binary.BigEndian.Uint16(glyph[0:])) != -1 {
		return nil, nil
	}
	var comps []uint16
	p := 10
	for {
		if len(glyph) < p+4 {
			return nil, fmt.Errorf("pdffont: truncated composite")
		}
		flags := binary.BigEndian.Uint16(glyph[p:])
		gid := binary.BigEndian.Uint16(glyph[p+2:])
		comps = append(comps, gid)
		p += 4
		if flags&fntFlagArgsWords != 0 {
			p += 4
		} else {
			p += 2
		}
		switch {
		case flags&fntFlagWeHaveScale != 0:
			p += 2
		case flags&fntFlagWeHaveXYScale != 0:
			p += 4
		case flags&fntFlagWeHave2x2 != 0:
			p += 8
		}
		if len(glyph) < p {
			return nil, fmt.Errorf("pdffont: truncated composite args")
		}
		if flags&fntFlagMoreComps == 0 {
			if flags&fntFlagWeHaveInstr != 0 {
				if len(glyph) < p+2 {
					return nil, fmt.Errorf("pdffont: truncated composite instructions")
				}
				p += 2 + int(binary.BigEndian.Uint16(glyph[p:]))
				if len(glyph) < p {
					return nil, fmt.Errorf("pdffont: truncated composite instructions")
				}
			}
			break
		}
	}
	return comps, nil
}

// fntRemapComposite rewrites component glyph ids via remap (old -> new).
func fntRemapComposite(glyph []byte, remap map[uint16]uint16) ([]byte, error) {
	out := make([]byte, len(glyph))
	copy(out, glyph)
	if len(glyph) < 10 || int16(binary.BigEndian.Uint16(glyph[0:])) != -1 {
		return out, nil
	}
	p := 10
	for {
		if len(glyph) < p+4 {
			return nil, fmt.Errorf("pdffont: truncated composite")
		}
		flags := binary.BigEndian.Uint16(glyph[p:])
		old := binary.BigEndian.Uint16(glyph[p+2:])
		nw, ok := remap[old]
		if !ok {
			return nil, fmt.Errorf("pdffont: composite references missing glyph %d", old)
		}
		binary.BigEndian.PutUint16(out[p+2:], nw)
		p += 4
		if flags&fntFlagArgsWords != 0 {
			p += 4
		} else {
			p += 2
		}
		switch {
		case flags&fntFlagWeHaveScale != 0:
			p += 2
		case flags&fntFlagWeHaveXYScale != 0:
			p += 4
		case flags&fntFlagWeHave2x2 != 0:
			p += 8
		}
		if flags&fntFlagMoreComps == 0 {
			break
		}
	}
	return out, nil
}

// fntCloseGlyphs computes the subset glyph set: gid 0 plus every encoded
// glyph plus transitively referenced composite components. It returns the
// new glyph order (new -> old), the code -> new-gid map, and the
// old -> new gid remap for composite rewriting.
func (f *FntFont) fntCloseGlyphs(encode map[rune]byte) ([]uint16, map[int]int, map[uint16]uint16, error) {
	need := map[uint16]bool{0: true}
	for r := range encode {
		need[f.cmap[r]] = true
	}
	for {
		grown := false
		for gid := range need {
			glyph, err := f.fntGlyphData(int(gid))
			if err != nil {
				return nil, nil, nil, err
			}
			comps, err := fntCompositeComponents(glyph)
			if err != nil {
				return nil, nil, nil, err
			}
			for _, c := range comps {
				if int(c) >= f.numGlyphs {
					return nil, nil, nil, fmt.Errorf("pdffont: composite references glyph %d", c)
				}
				if !need[c] {
					need[c] = true
					grown = true
				}
			}
		}
		if !grown {
			break
		}
	}
	var olds []uint16
	for gid := range need {
		olds = append(olds, gid)
	}
	sort.Slice(olds, func(i, j int) bool {
		if olds[i] == 0 {
			return true
		}
		if olds[j] == 0 {
			return false
		}
		return olds[i] < olds[j]
	})
	remap := map[uint16]uint16{}
	newGids := make([]uint16, len(olds))
	for new, old := range olds {
		remap[old] = uint16(new)
		newGids[new] = old
	}
	codeToNew := map[int]int{}
	for r, b := range encode {
		codeToNew[int(b)] = int(remap[f.cmap[r]])
	}
	return newGids, codeToNew, remap, nil
}

// fntBuildSubset assembles a standalone subset sfnt from the glyphs in
// newGids (new -> old) with a fresh cmap mapping byte codes to new gids.
func (f *FntFont) fntBuildSubset(newGids []uint16, codeToNew map[int]int, remap map[uint16]uint16) ([]byte, error) {
	n := len(newGids)
	// Gather (and remap) glyph bytes.
	glyphs := make([][]byte, n)
	for new, old := range newGids {
		g, err := f.fntGlyphData(int(old))
		if err != nil {
			return nil, err
		}
		glyphs[new], err = fntRemapComposite(g, remap)
		if err != nil {
			return nil, err
		}
	}
	// loca: keep the short format unless offsets overflow it. Short
	// offsets are halved, so pad odd-length glyphs first.
	for i, g := range glyphs {
		if len(g)%2 == 1 {
			glyphs[i] = append(g, 0)
		}
	}
	offs := make([]int, n+1)
	for i, g := range glyphs {
		offs[i+1] = offs[i] + len(g)
	}
	useShort := f.locaShort && offs[n] <= 2*0xFFFF
	var loca []byte
	if useShort {
		loca = make([]byte, 2*(n+1))
		for i, o := range offs {
			binary.BigEndian.PutUint16(loca[2*i:], uint16(o/2))
		}
	} else {
		loca = make([]byte, 4*(n+1))
		for i, o := range offs {
			binary.BigEndian.PutUint32(loca[4*i:], uint32(o))
		}
	}
	var glyf []byte
	for _, g := range glyphs {
		glyf = append(glyf, g...)
	}
	// hmtx with one (advance, lsb) pair per glyph.
	hmtx := make([]byte, 4*n)
	for new, old := range newGids {
		binary.BigEndian.PutUint16(hmtx[4*new:], uint16(f.adv[int(old)]))
		binary.BigEndian.PutUint16(hmtx[4*new+2:], uint16(int16(f.lsb[int(old)])))
	}
	// hhea: patch numberOfHMetrics.
	hhea := make([]byte, len(f.hhea))
	copy(hhea, f.hhea)
	binary.BigEndian.PutUint16(hhea[34:], uint16(n))
	// maxp: patch numGlyphs (layout v0.5 and v1.0 share the prefix).
	maxp := make([]byte, len(f.maxp))
	copy(maxp, f.maxp)
	binary.BigEndian.PutUint16(maxp[4:], uint16(n))
	// head: zero the checksum adjustment, fix loca format.
	head := make([]byte, len(f.head))
	copy(head, f.head)
	binary.BigEndian.PutUint32(head[8:], 0)
	if useShort {
		binary.BigEndian.PutUint16(head[50:], 0)
	} else {
		binary.BigEndian.PutUint16(head[50:], 1)
	}
	// name: copy, or a minimal empty table when absent.
	nameTab := f.nameTab
	if len(nameTab) == 0 {
		nameTab = []byte{0, 0, 0, 0, 0, 6}
	}
	// post: fresh format 3.0 (no glyph names; /Differences uses uni names).
	post := make([]byte, 32)
	binary.BigEndian.PutUint32(post[0:], 0x00030000)
	cmap, err := fntBuildCmap4(codeToNew)
	if err != nil {
		return nil, err
	}
	tables := []struct {
		tag  string
		data []byte
	}{
		{"OS/2", f.os2Tab},
		{"cmap", cmap},
		{"glyf", glyf},
		{"head", head},
		{"hhea", hhea},
		{"hmtx", hmtx},
		{"loca", loca},
		{"maxp", maxp},
		{"name", nameTab},
		{"post", post},
	}
	var have []struct {
		tag  string
		data []byte
	}
	for _, t := range tables {
		if len(t.data) > 0 {
			have = append(have, t)
		}
	}
	sort.Slice(have, func(i, j int) bool { return have[i].tag < have[j].tag })
	nt := len(have)
	out := make([]byte, 12+16*nt)
	binary.BigEndian.PutUint32(out[0:], 0x00010000)
	binary.BigEndian.PutUint16(out[4:], uint16(nt))
	// Offset-table search fields over the table directory.
	p := 1
	for p*2 <= nt {
		p *= 2
	}
	binary.BigEndian.PutUint16(out[6:], uint16(16*p))
	binary.BigEndian.PutUint16(out[8:], uint16(fntLog2(p)))
	binary.BigEndian.PutUint16(out[10:], uint16(16*nt-16*p))
	off := 12 + 16*nt
	headDir := 0
	for i, t := range have {
		if t.tag == "head" {
			headDir = i
		}
		copy(out[12+16*i:], t.tag)
		binary.BigEndian.PutUint32(out[12+16*i+4:], fntChecksum(t.data))
		binary.BigEndian.PutUint32(out[12+16*i+8:], uint32(off))
		binary.BigEndian.PutUint32(out[12+16*i+12:], uint32(len(t.data)))
		off += (len(t.data) + 3) &^ 3
	}
	for _, t := range have {
		pad := ((len(t.data) + 3) &^ 3) - len(t.data)
		out = append(out, t.data...)
		for i := 0; i < pad; i++ {
			out = append(out, 0)
		}
	}
	// head.checkSumAdjustment = 0xB1B0AFBA - file sum.
	var sum uint32
	for i := 0; i+4 <= len(out); i += 4 {
		sum += binary.BigEndian.Uint32(out[i:])
	}
	adj := 0xB1B0AFBA - sum
	headOff := int(binary.BigEndian.Uint32(out[12+16*headDir+8:]))
	binary.BigEndian.PutUint32(out[headOff+8:], adj)
	return out, nil
}

func fntLog2(p int) int {
	n := 0
	for p > 1 {
		p /= 2
		n++
	}
	return n
}

// fntChecksum sums data as big-endian uint32 words (zero-padded).
func fntChecksum(data []byte) uint32 {
	var sum uint32
	for i := 0; i < len(data); i += 4 {
		var w [4]byte
		copy(w[:], data[i:])
		sum += binary.BigEndian.Uint32(w[:])
	}
	return sum
}

// fntBuildCmap4 builds a format 4 subtable mapping codes to new gids,
// wrapped in a one-record cmap table.
func fntBuildCmap4(codeToNew map[int]int) ([]byte, error) {
	codes := make([]int, 0, len(codeToNew))
	for c := range codeToNew {
		if c < 0 || c > 0xFF {
			return nil, fmt.Errorf("pdffont: code %d outside 8-bit range", c)
		}
		codes = append(codes, c)
	}
	if len(codes) == 0 {
		return nil, fmt.Errorf("pdffont: empty subset")
	}
	sort.Ints(codes)
	type seg struct {
		start, end int
		gid0       int // glyph id of start
	}
	var segs []seg
	for _, c := range codes {
		g := codeToNew[c]
		if n := len(segs); n > 0 {
			last := &segs[n-1]
			if c == last.end+1 && g == last.gid0+(c-last.start) {
				last.end = c
				continue
			}
		}
		segs = append(segs, seg{c, c, g})
	}
	segCount := len(segs) + 1
	sub := make([]byte, 16+8*segCount)
	binary.BigEndian.PutUint16(sub[0:], 4)
	binary.BigEndian.PutUint16(sub[2:], uint16(16+8*segCount))
	binary.BigEndian.PutUint16(sub[4:], 0) // language
	binary.BigEndian.PutUint16(sub[6:], uint16(2*segCount))
	p := 1
	for p*2 <= segCount {
		p *= 2
	}
	binary.BigEndian.PutUint16(sub[8:], uint16(2*p))
	binary.BigEndian.PutUint16(sub[10:], uint16(fntLog2(p)))
	binary.BigEndian.PutUint16(sub[12:], uint16(2*segCount-2*p))
	for i, s := range segs {
		binary.BigEndian.PutUint16(sub[14+2*i:], uint16(s.end))
	}
	binary.BigEndian.PutUint16(sub[14+2*(segCount-1):], 0xFFFF)
	base := 16 + 2*segCount
	binary.BigEndian.PutUint16(sub[base-2:], 0) // reservedPad
	for i, s := range segs {
		binary.BigEndian.PutUint16(sub[base+2*i:], uint16(s.start))
	}
	binary.BigEndian.PutUint16(sub[base+2*(segCount-1):], 0xFFFF)
	for i, s := range segs {
		binary.BigEndian.PutUint16(sub[base+2*segCount+2*i:], uint16((s.gid0-s.start)&0xFFFF))
	}
	binary.BigEndian.PutUint16(sub[base+2*segCount+2*(segCount-1):], 1)
	// idRangeOffset words stay zero; no glyphIdArray.
	out := make([]byte, 12)
	binary.BigEndian.PutUint16(out[0:], 0)
	binary.BigEndian.PutUint16(out[2:], 1)
	binary.BigEndian.PutUint16(out[4:], 3)
	binary.BigEndian.PutUint16(out[6:], 1)
	binary.BigEndian.PutUint32(out[8:], 12)
	return append(out, sub...), nil
}

// fntSubsetTag derives a deterministic 6-letter subset prefix.
func fntSubsetTag(psName string, codes []int) string {
	h := fnv.New32a()
	h.Write([]byte(psName))
	var buf [4]byte
	for _, c := range codes {
		binary.BigEndian.PutUint32(buf[:], uint32(c))
		h.Write(buf[:])
	}
	sum := h.Sum32()
	tag := make([]byte, 6)
	for i := range tag {
		tag[i] = byte('A' + sum%26)
		sum /= 26
		if sum == 0 {
			sum += 0x9E3779B9
		}
	}
	return string(tag)
}

// fntParseCollection selects one font from a TTC: the first with Thai
// coverage, else the first that parses.
func fntParseCollection(data []byte) (*FntFont, error) {
	if len(data) < 12 {
		return nil, fmt.Errorf("pdffont: collection too short")
	}
	num := int(binary.BigEndian.Uint32(data[8:12]))
	if num < 1 || len(data) < 12+4*num {
		return nil, fmt.Errorf("pdffont: bad collection header")
	}
	var first *FntFont
	for i := 0; i < num; i++ {
		off := int(binary.BigEndian.Uint32(data[12+4*i:]))
		f, err := fntParseAt(data, off)
		if err != nil {
			continue
		}
		if first == nil {
			first = f
		}
		if _, ok := f.cmap[fntThaiLo+1]; ok { // U+0E01 ก
			return f, nil
		}
	}
	if first == nil {
		return nil, fmt.Errorf("pdffont: no usable font in collection")
	}
	return first, nil
}
