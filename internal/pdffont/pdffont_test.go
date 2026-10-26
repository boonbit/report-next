package pdffont

import (
	"bytes"
	"encoding/binary"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"testing"
)

// fntTestFont builds a minimal valid TTF in memory: upm 1000, four glyphs
// (gid0 empty; 'A' gid1 adv 600; U+0E01 gid2 adv 500; U+0E02 gid3 adv 550
// as a composite of gid1+gid2), name ID 6 "FntTest".
func fntTestFont(t *testing.T) []byte {
	t.Helper()
	putU16 := func(b []byte, off, v int) {
		binary.BigEndian.PutUint16(b[off:], uint16(int16(v)))
	}
	head := make([]byte, 54)
	binary.BigEndian.PutUint32(head[0:], 0x00010000)
	binary.BigEndian.PutUint32(head[12:], 0x5F0F3CF5)
	binary.BigEndian.PutUint16(head[18:], 1000)
	putU16(head, 36, 0)
	putU16(head, 38, -200)
	putU16(head, 40, 600)
	putU16(head, 42, 800)
	binary.BigEndian.PutUint16(head[48:], 2) // fontDirectionHint
	binary.BigEndian.PutUint16(head[50:], 0) // short loca

	hhea := make([]byte, 36)
	binary.BigEndian.PutUint32(hhea[0:], 0x00010000)
	putU16(hhea, 4, 800)
	putU16(hhea, 6, -200)
	binary.BigEndian.PutUint16(hhea[34:], 4)

	maxp := make([]byte, 32)
	binary.BigEndian.PutUint32(maxp[0:], 0x00010000)
	binary.BigEndian.PutUint16(maxp[4:], 4)

	hmtx := make([]byte, 16)
	for i, adv := range []int{500, 600, 500, 550} {
		binary.BigEndian.PutUint16(hmtx[4*i:], uint16(adv))
	}

	empty := make([]byte, 10)
	comp := []byte{}
	comp = append(comp, 0xFF, 0xFF)                         // numContours = -1
	comp = append(comp, make([]byte, 8)...)                 // bbox
	comp = append(comp, 0x00, 0x23, 0x00, 0x01, 0, 0, 0, 0) // more, gid1
	comp = append(comp, 0x01, 0x03, 0x00, 0x02, 0, 0, 0, 0) // instr, gid2
	comp = append(comp, 0x00, 0x00)                         // zero instructions
	glyf := append(append(append([]byte{}, empty...), empty...), empty...)
	glyf = append(glyf, comp...)

	loca := make([]byte, 10)
	for i, o := range []int{0, 5, 10, 15, 29} {
		binary.BigEndian.PutUint16(loca[2*i:], uint16(o))
	}

	sub := make([]byte, 40)
	binary.BigEndian.PutUint16(sub[0:], 4)
	binary.BigEndian.PutUint16(sub[2:], 40)
	binary.BigEndian.PutUint16(sub[6:], 6) // segCountX2 = 3 segs
	binary.BigEndian.PutUint16(sub[8:], 4)
	binary.BigEndian.PutUint16(sub[10:], 1)
	binary.BigEndian.PutUint16(sub[12:], 2)
	for i, v := range []int{0x41, 0xE02, 0xFFFF} {
		binary.BigEndian.PutUint16(sub[14+2*i:], uint16(v))
	}
	// reservedPad at sub[20] stays zero
	for i, v := range []int{0x41, 0xE01, 0xFFFF} {
		binary.BigEndian.PutUint16(sub[22+2*i:], uint16(v))
	}
	for i, v := range []int{(1 - 0x41) & 0xFFFF, (2 - 0xE01) & 0xFFFF, 1} {
		binary.BigEndian.PutUint16(sub[28+2*i:], uint16(v))
	}
	cmap := []byte{0, 0, 0, 1, 0, 3, 0, 1, 0, 0, 0, 12}
	cmap = append(cmap, sub...)

	nameStr := "FntTest"
	nameRec := make([]byte, 12)
	binary.BigEndian.PutUint16(nameRec[0:], 3)
	binary.BigEndian.PutUint16(nameRec[2:], 1)
	binary.BigEndian.PutUint16(nameRec[4:], 0x409)
	binary.BigEndian.PutUint16(nameRec[6:], 6)
	binary.BigEndian.PutUint16(nameRec[8:], uint16(len(nameStr)*2))
	binary.BigEndian.PutUint16(nameRec[10:], 0)
	var nameStrBytes []byte
	for _, r := range nameStr {
		var b [2]byte
		binary.BigEndian.PutUint16(b[:], uint16(r))
		nameStrBytes = append(nameStrBytes, b[:]...)
	}
	nameTab := []byte{0, 0, 0, 1, 0, 18}
	nameTab = append(nameTab, nameRec...)
	nameTab = append(nameTab, nameStrBytes...)

	os2 := make([]byte, 96)
	binary.BigEndian.PutUint16(os2[0:], 2)
	binary.BigEndian.PutUint16(os2[88:], 700)

	post := make([]byte, 32)
	binary.BigEndian.PutUint32(post[0:], 0x00030000)

	tabs := map[string][]byte{
		"OS/2": os2, "cmap": cmap, "glyf": glyf, "head": head,
		"hhea": hhea, "hmtx": hmtx, "loca": loca, "maxp": maxp,
		"name": nameTab, "post": post,
	}
	var tags []string
	for tag := range tabs {
		tags = append(tags, tag)
	}
	sort.Strings(tags)
	out := make([]byte, 12+16*len(tags))
	binary.BigEndian.PutUint32(out[0:], 0x00010000)
	binary.BigEndian.PutUint16(out[4:], uint16(len(tags)))
	off := 12 + 16*len(tags)
	for i, tag := range tags {
		copy(out[12+16*i:], tag)
		binary.BigEndian.PutUint32(out[12+16*i+8:], uint32(off))
		binary.BigEndian.PutUint32(out[12+16*i+12:], uint32(len(tabs[tag])))
		off += (len(tabs[tag]) + 3) &^ 3
	}
	for _, tag := range tags {
		out = append(out, tabs[tag]...)
		for len(out)%4 != 0 {
			out = append(out, 0)
		}
	}
	var sum uint32
	for i := 0; i+4 <= len(out); i += 4 {
		sum += binary.BigEndian.Uint32(out[i:])
	}
	_ = sum // test font checksum adjustment not required by the parser
	return out
}

func fntTestLoad(t *testing.T) *FntFont {
	t.Helper()
	f, err := FntParseTTF(fntTestFont(t))
	if err != nil {
		t.Fatal(err)
	}
	return f
}

func TestFntLoadAndMetrics(t *testing.T) {
	path := filepath.Join(t.TempDir(), "test.ttf")
	if err := os.WriteFile(path, fntTestFont(t), 0o644); err != nil {
		t.Fatal(err)
	}
	for _, load := range []func(string) (*FntFont, error){FntLoadTTF, fntLoadTTF} {
		f, err := load(path)
		if err != nil {
			t.Fatal(err)
		}
		if f.FntUnitsPerEm() != 1000 {
			t.Fatalf("upm = %d", f.FntUnitsPerEm())
		}
		if f.FntAscent() != 800 || f.FntDescent() != -200 {
			t.Fatalf("metrics = %v %v", f.FntAscent(), f.FntDescent())
		}
		if f.FntCapHeight() != 700 {
			t.Fatalf("capheight = %v", f.FntCapHeight())
		}
		if got := f.FntBBox(); got != [4]float64{0, -200, 600, 800} {
			t.Fatalf("bbox = %v", got)
		}
		if f.FntPostScriptName() != "FntTest" {
			t.Fatalf("psname = %q", f.FntPostScriptName())
		}
		if !f.FntHas('A') || !f.FntHas(0xE01) || f.FntHas('B') {
			t.Fatal("coverage mismatch")
		}
		if f.FntAdvance('A') != 600 || f.FntAdvance(0xE01) != 500 {
			t.Fatal("advance mismatch")
		}
		if f.FntAdvance('B') != 500 { // .notdef fallback
			t.Fatalf("missing advance = %d", f.FntAdvance('B'))
		}
		if w := f.FntWidth("Aก", 10); w != 11 { // (600+500)*10/1000
			t.Fatalf("width = %v", w)
		}
		if len(f.FntMissing("AกB")) != 1 {
			t.Fatalf("missing = %q", f.FntMissing("AกB"))
		}
	}
}

func TestFntBadInput(t *testing.T) {
	for _, data := range [][]byte{
		{},
		{0, 1, 0, 0},
		bytes.Repeat([]byte{0}, 64),
		append([]byte("OTTO"), bytes.Repeat([]byte{0}, 8)...),
	} {
		if _, err := FntParseTTF(data); err == nil {
			t.Fatalf("accepted %d bytes", len(data))
		}
	}
	if _, err := FntLoadTTF(filepath.Join(t.TempDir(), "nope.ttf")); err == nil {
		t.Fatal("accepted missing file")
	}
}

func TestFntSubsetBasics(t *testing.T) {
	f := fntTestLoad(t)
	sub, err := f.FntSubsetFor([]rune{'A', 0xE01, 0xE02, 'A'})
	if err != nil {
		t.Fatal(err)
	}
	if sub.Encode['A'] != 0x41 || sub.Encode[0xE01] != 0x80 || sub.Encode[0xE02] != 0x81 {
		t.Fatalf("encode = %v", sub.Encode)
	}
	if sub.FirstChar != 0x41 || sub.LastChar != 0x81 {
		t.Fatalf("range = %x..%x", sub.FirstChar, sub.LastChar)
	}
	if len(sub.Widths) != sub.LastChar-sub.FirstChar+1 {
		t.Fatalf("widths len = %d", len(sub.Widths))
	}
	if sub.Widths[0] != 600 || sub.Widths[0x80-0x41] != 500 || sub.Widths[0x81-0x41] != 550 {
		t.Fatalf("widths = %v", sub.Widths)
	}
	// Widths must agree with the font metrics for every mapped code.
	for code, uni := range sub.Unicodes {
		if sub.Widths[code-sub.FirstChar] != f.FntAdvance(uni) {
			t.Fatalf("code %x width %d != advance %d", code, sub.Widths[code-sub.FirstChar], f.FntAdvance(uni))
		}
	}
	if len(sub.Tag) != 6 {
		t.Fatalf("tag = %q", sub.Tag)
	}
	for _, r := range sub.Tag {
		if r < 'A' || r > 'Z' {
			t.Fatalf("tag = %q", sub.Tag)
		}
	}
	// The subset sfnt must re-parse and still contain the used glyphs.
	rf, err := FntParseTTF(sub.Data)
	if err != nil {
		t.Fatalf("subset re-parse: %v", err)
	}
	if rf.numGlyphs != 4 {
		t.Fatalf("subset glyphs = %d", rf.numGlyphs)
	}
	if rf.FntAdvance('A') != 600 || rf.FntAdvance(0x80) != 500 || rf.FntAdvance(0x81) != 550 {
		t.Fatal("subset advances mismatch")
	}
	// ToUnicode must round-trip every mapped codepoint.
	cmap := sub.FntToUnicodeCMap()
	if !strings.Contains(cmap, "<80> <0E01>") || !strings.Contains(cmap, "<41> <0041>") {
		t.Fatalf("cmap missing entries:\n%s", cmap)
	}
	round := map[int]rune{}
	inBF := false
	for _, line := range strings.Split(cmap, "\n") {
		if strings.HasSuffix(line, "beginbfchar") {
			inBF = true
			continue
		}
		if strings.HasSuffix(line, "endbfchar") {
			break
		}
		if !inBF {
			continue
		}
		var c, u int
		if _, err := fmt.Sscanf(line, "<%X> <%X>", &c, &u); err != nil {
			continue
		}
		round[c] = rune(u)
	}
	if len(round) != len(sub.Unicodes) {
		t.Fatalf("round-trip %d != %d", len(round), len(sub.Unicodes))
	}
	for c, r := range sub.Unicodes {
		if round[c] != r {
			t.Fatalf("code %x: %U != %U", c, round[c], r)
		}
	}
	enc, err := sub.FntEncodeString("Aกข")
	if err != nil {
		t.Fatal(err)
	}
	if enc != string([]byte{0x41, 0x80, 0x81}) {
		t.Fatalf("encoded = %q", enc)
	}
	if _, err := sub.FntEncodeString("AB"); err == nil {
		t.Fatal("encoded missing rune")
	}
}

func TestFntCompositeClosure(t *testing.T) {
	f := fntTestLoad(t)
	sub, err := f.FntSubsetFor([]rune{0xE02})
	if err != nil {
		t.Fatal(err)
	}
	rf, err := FntParseTTF(sub.Data)
	if err != nil {
		t.Fatal(err)
	}
	if rf.numGlyphs != 4 { // gid0 + composite + its two components
		t.Fatalf("subset glyphs = %d", rf.numGlyphs)
	}
	// Every composite reference must resolve inside the subset.
	for gid := 0; gid < rf.numGlyphs; gid++ {
		g, err := rf.fntGlyphData(gid)
		if err != nil {
			t.Fatal(err)
		}
		comps, err := fntCompositeComponents(g)
		if err != nil {
			t.Fatal(err)
		}
		for _, c := range comps {
			if int(c) >= rf.numGlyphs {
				t.Fatalf("dangling component %d in %d", c, gid)
			}
		}
	}
	// A subset without composites drops the unneeded glyphs.
	sub2, err := f.FntSubsetFor([]rune{'A'})
	if err != nil {
		t.Fatal(err)
	}
	rf2, err := FntParseTTF(sub2.Data)
	if err != nil {
		t.Fatal(err)
	}
	if rf2.numGlyphs != 2 {
		t.Fatalf("subset glyphs = %d", rf2.numGlyphs)
	}
}

func TestFntSubsetErrors(t *testing.T) {
	f := fntTestLoad(t)
	if _, err := f.FntSubsetFor([]rune{'B'}); err == nil {
		t.Fatal("accepted missing glyph")
	}
	if _, err := f.FntSubsetFor(nil); err == nil {
		t.Fatal("accepted empty subset")
	}
	var many []rune
	for r := rune(0xE00); r < rune(0xE00)+fntExtraSlots+1; r++ {
		many = append(many, r)
	}
	if _, err := f.FntSubsetFor(many); err == nil {
		t.Fatal("accepted slot overflow")
	}
}

func TestFntParseCmap12(t *testing.T) {
	sub := make([]byte, 16+24)
	binary.BigEndian.PutUint16(sub[0:], 12)
	binary.BigEndian.PutUint16(sub[2:], uint16(len(sub)))
	binary.BigEndian.PutUint32(sub[4:], 0)
	binary.BigEndian.PutUint32(sub[12:], 2)
	// Group 1: U+0041-U+0042 -> gid 1-2.
	binary.BigEndian.PutUint32(sub[16:], 0x41)
	binary.BigEndian.PutUint32(sub[20:], 0x42)
	binary.BigEndian.PutUint32(sub[24:], 1)
	// Group 2: U+1F600 -> gid 7 (astral).
	binary.BigEndian.PutUint32(sub[28:], 0x1F600)
	binary.BigEndian.PutUint32(sub[32:], 0x1F600)
	binary.BigEndian.PutUint32(sub[36:], 7)
	m, err := fntParseCmap12(sub)
	if err != nil {
		t.Fatal(err)
	}
	if m['A'] != 1 || m['B'] != 2 || m[0x1F600] != 7 {
		t.Fatalf("cmap12 = %v", m)
	}
}

func TestFntHexUTF16(t *testing.T) {
	if fntHexUTF16('A') != "0041" || fntHexUTF16(0xE01) != "0E01" {
		t.Fatal("bmp hex mismatch")
	}
	if fntHexUTF16(0x1F600) != "D83DDE00" {
		t.Fatalf("astral hex = %s", fntHexUTF16(0x1F600))
	}
}

func TestFntSystemFont(t *testing.T) {
	path := os.Getenv(FntEnvVar)
	if path == "" {
		t.Skip("REPORT_FONT_TTF not set")
	}
	data, err := os.ReadFile(path)
	if err != nil {
		t.Skipf("unreadable font: %v", err)
	}
	f, err := FntParseTTF(data)
	if err != nil {
		t.Fatalf("parse: %v", err)
	}
	const sample = "Hello, World! 0123456789"
	var want int
	for _, r := range sample {
		if !f.FntHas(r) {
			t.Skipf("font lacks %q", r)
		}
		want += f.FntAdvance(r)
	}
	if got := f.FntWidth(sample, 12); got != float64(want)*12/1000 {
		t.Fatalf("width %v != sum %d", got, want)
	}
	var runes []rune
	for _, r := range sample {
		runes = append(runes, r)
	}
	sub, err := f.FntSubsetFor(runes)
	if err != nil {
		t.Fatal(err)
	}
	if len(sub.Data) >= len(data) {
		t.Fatalf("subset %d not smaller than %d", len(sub.Data), len(data))
	}
	for code, uni := range sub.Unicodes {
		if sub.Widths[code-sub.FirstChar] != f.FntAdvance(uni) {
			t.Fatalf("code %x width mismatch", code)
		}
	}
}
