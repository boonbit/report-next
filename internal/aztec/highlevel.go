package aztec

// Aztec high-level encodation: optimal Upper/Lower/Digit/Mixed/Punct +
// binary-shift path search over the state lattice, following the published
// dynamic-programming algorithm.

import "fmt"

// azToken is one emitted unit: a mode value or a binary shift run.
type azToken interface {
	fmt.Stringer
	azPrev() azToken
	azAppend(bits *azBits, text []byte)
}

type azSimpleToken struct {
	prev     azToken
	value    int
	bitCount byte
}

type azShiftToken struct {
	prev       azToken
	shiftStart int
	shiftBytes int
}

func azNewSimpleToken(prev azToken, value int, bitCount byte) azToken {
	return &azSimpleToken{prev, value, bitCount}
}

func azNewShiftToken(prev azToken, start, count int) azToken {
	return &azShiftToken{prev, start, count}
}

func (t *azSimpleToken) azPrev() azToken { return t.prev }

func (t *azSimpleToken) azAppend(bits *azBits, _ []byte) {
	bits.AddBits(t.value, t.bitCount)
}

func (t *azSimpleToken) String() string {
	v := t.value&((1<<t.bitCount)-1) | (1 << t.bitCount)
	return "<" + fmt.Sprintf("%b", v)[1:] + ">"
}

func (t *azShiftToken) azPrev() azToken { return t.prev }

func (t *azShiftToken) azAppend(bits *azBits, text []byte) {
	for i := 0; i < t.shiftBytes; i++ {
		if i == 0 || (i == 31 && t.shiftBytes <= 62) {
			bits.AddBits(31, 5) // BINARY_SHIFT
			switch {
			case t.shiftBytes > 62:
				bits.AddBits(t.shiftBytes-31, 16)
			case i == 0 && t.shiftBytes < 31:
				bits.AddBits(t.shiftBytes, 5)
			case i == 0:
				bits.AddBits(31, 5)
			default: // 32 <= count <= 62 and i == 31
				bits.AddBits(t.shiftBytes-31, 5)
			}
		}
		bits.AddByte(text[t.shiftStart+i])
	}
}

func (t *azShiftToken) String() string {
	return fmt.Sprintf("<%d::%d>", t.shiftStart, t.shiftStart+t.shiftBytes-1)
}

type azMode byte

const (
	azUpper azMode = iota // 5 bits
	azLower               // 5 bits
	azDigit               // 4 bits
	azMixed               // 5 bits
	azPunct               // 5 bits
)

// azLatchTable gives, per mode pair, the latch bits (low half-word) and
// their count (high half-word).
var azLatchTable = map[azMode]map[azMode]int{
	azUpper: {
		azUpper: 0,
		azLower: (5 << 16) + 28,
		azDigit: (5 << 16) + 30,
		azMixed: (5 << 16) + 29,
		azPunct: (10 << 16) + (29 << 5) + 30,
	},
	azLower: {
		azUpper: (9 << 16) + (30 << 4) + 14,
		azLower: 0,
		azDigit: (5 << 16) + 30,
		azMixed: (5 << 16) + 29,
		azPunct: (10 << 16) + (29 << 5) + 30,
	},
	azDigit: {
		azUpper: (4 << 16) + 14,
		azLower: (9 << 16) + (14 << 5) + 28,
		azDigit: 0,
		azMixed: (9 << 16) + (14 << 5) + 29,
		azPunct: (14 << 16) + (14 << 10) + (29 << 5) + 30,
	},
	azMixed: {
		azUpper: (5 << 16) + 29,
		azLower: (5 << 16) + 28,
		azDigit: (10 << 16) + (29 << 5) + 30,
		azMixed: 0,
		azPunct: (5 << 16) + 30,
	},
	azPunct: {
		azUpper: (5 << 16) + 31,
		azLower: (10 << 16) + (31 << 5) + 28,
		azDigit: (10 << 16) + (31 << 5) + 30,
		azMixed: (10 << 16) + (31 << 5) + 29,
		azPunct: 0,
	},
}

// azShiftTable holds the available single-character shifts.
var azShiftTable = map[azMode]map[azMode]int{
	azUpper: {azPunct: 0},
	azLower: {azPunct: 0, azUpper: 28},
	azMixed: {azPunct: 0},
	azDigit: {azPunct: 0, azUpper: 15},
}

var azCharMap map[azMode][]int

func init() {
	azCharMap = map[azMode][]int{}
	azCharMap[azUpper] = make([]int, 256)
	azCharMap[azLower] = make([]int, 256)
	azCharMap[azDigit] = make([]int, 256)
	azCharMap[azMixed] = make([]int, 256)
	azCharMap[azPunct] = make([]int, 256)
	azCharMap[azUpper][' '] = 1
	for c := 'A'; c <= 'Z'; c++ {
		azCharMap[azUpper][c] = int(c-'A') + 2
	}
	azCharMap[azLower][' '] = 1
	for c := 'a'; c <= 'z'; c++ {
		azCharMap[azLower][c] = int(c-'a') + 2
	}
	azCharMap[azDigit][' '] = 1
	for c := '0'; c <= '9'; c++ {
		azCharMap[azDigit][c] = int(c-'0') + 2
	}
	azCharMap[azDigit][','] = 12
	azCharMap[azDigit]['.'] = 13
	mixed := []int{
		0, ' ', 1, 2, 3, 4, 5, 6, 7, 8, 9, 10,
		11, 12, 13, 27, 28, 29, 30, 31, '@', '\\', '^',
		'_', '`', '|', '~', 127,
	}
	for i, v := range mixed {
		azCharMap[azMixed][v] = i
	}
	punct := []int{
		0, '\r', 0, 0, 0, 0, '!', '\'', '#', '$', '%', '&', '\'',
		'(', ')', '*', '+', ',', '-', '.', '/', ':', ';', '<', '=', '>', '?',
		'[', ']', '{', '}',
	}
	for i, v := range punct {
		if v > 0 {
			azCharMap[azPunct][v] = i
		}
	}
}

type azState struct {
	mode       azMode
	tokens     azToken
	shiftBytes int
	bitCount   int
}

type azStateSlice []*azState

var azInitialState = &azState{mode: azUpper}

func (m azMode) azBitCount() byte {
	if m == azDigit {
		return 4
	}
	return 5
}

func (s *azState) latchAndAppend(mode azMode, value int) *azState {
	bitCount, tokens := s.bitCount, s.tokens
	if mode != s.mode {
		latch := azLatchTable[s.mode][mode]
		tokens = azNewSimpleToken(tokens, latch&0xFFFF, byte(latch>>16))
		bitCount += latch >> 16
	}
	tokens = azNewSimpleToken(tokens, value, mode.azBitCount())
	return &azState{mode: mode, tokens: tokens, bitCount: bitCount + int(mode.azBitCount())}
}

func (s *azState) shiftAndAppend(mode azMode, value int) *azState {
	// Shifts exist only to UPPER and PUNCT, both with 5-bit values.
	tokens := azNewSimpleToken(s.tokens, azShiftTable[s.mode][mode], s.mode.azBitCount())
	tokens = azNewSimpleToken(tokens, value, 5)
	return &azState{
		mode: s.mode, tokens: tokens,
		bitCount: s.bitCount + int(s.mode.azBitCount()) + 5,
	}
}

func (s *azState) addBinaryShiftChar(index int) *azState {
	tokens, mode, bitCnt := s.tokens, s.mode, s.bitCount
	if s.mode == azPunct || s.mode == azDigit {
		latch := azLatchTable[s.mode][azUpper]
		tokens = azNewSimpleToken(tokens, latch&0xFFFF, byte(latch>>16))
		bitCnt += latch >> 16
		mode = azUpper
	}
	delta := 8
	if s.shiftBytes == 0 || s.shiftBytes == 31 {
		delta = 18
	} else if s.shiftBytes == 62 {
		delta = 9
	}
	result := &azState{mode: mode, tokens: tokens, shiftBytes: s.shiftBytes + 1, bitCount: bitCnt + delta}
	if result.shiftBytes == 2047+31 {
		// The string is as long as it's allowed to be; end it.
		result = result.endBinaryShift(index + 1)
	}
	return result
}

func (s *azState) endBinaryShift(index int) *azState {
	if s.shiftBytes == 0 {
		return s
	}
	return &azState{
		mode:     s.mode,
		tokens:   azNewShiftToken(s.tokens, index-s.shiftBytes, s.shiftBytes),
		bitCount: s.bitCount,
	}
}

func (s *azState) noWorseThan(o *azState) bool {
	mySize := s.bitCount + (azLatchTable[s.mode][o.mode] >> 16)
	if o.shiftBytes > 0 && (s.shiftBytes == 0 || s.shiftBytes > o.shiftBytes) {
		mySize += 10 // cost of entering binary shift mode
	}
	return mySize <= o.bitCount
}

func (s *azState) toBitList(text []byte) *azBits {
	var tokens []azToken
	for t := s.endBinaryShift(len(text)).tokens; t != nil; t = t.azPrev() {
		tokens = append(tokens, t)
	}
	out := new(azBits)
	for i := len(tokens) - 1; i >= 0; i-- {
		tokens[i].azAppend(out, text)
	}
	return out
}

func azSimplify(states azStateSlice) azStateSlice {
	var result azStateSlice
	for _, next := range states {
		add := true
		var kept azStateSlice
		for _, prev := range result {
			if add && prev.noWorseThan(next) {
				add = false
			}
			if !(add && next.noWorseThan(prev)) {
				kept = append(kept, prev)
			}
		}
		if add {
			result = append(kept, next)
		} else {
			result = kept
		}
	}
	return result
}

func azUpdateForChar(list azStateSlice, data []byte, index int) azStateSlice {
	var result azStateSlice
	for _, s := range list {
		if r := azStateForChar(s, data, index); len(r) > 0 {
			result = append(result, r...)
		}
	}
	return azSimplify(result)
}

func azStateForChar(s *azState, data []byte, index int) azStateSlice {
	var result azStateSlice
	ch := data[index]
	inCurrent := azCharMap[s.mode][ch] > 0
	var noBinary *azState
	for mode := azUpper; mode <= azPunct; mode++ {
		inMode := azCharMap[mode][ch]
		if inMode == 0 {
			continue
		}
		if noBinary == nil {
			noBinary = s.endBinaryShift(index)
		}
		// Try generating the character by latching to its mode.
		if !inCurrent || mode == s.mode || mode == azDigit {
			// If the character is in the current table, don't latch elsewhere
			// except possibly digit (4 bits); any other latch would tie after
			// this character and never save bits.
			result = append(result, noBinary.latchAndAppend(mode, inMode))
		}
		// Try generating the character by shifting to its mode.
		if shift, ok := azShiftTable[s.mode][mode]; !inCurrent && ok {
			_ = shift
			result = append(result, noBinary.shiftAndAppend(mode, inMode))
		}
	}
	if s.shiftBytes > 0 || azCharMap[s.mode][ch] == 0 {
		// Going binary is never worthwhile unless already binary or the
		// character is missing from the current mode.
		result = append(result, s.addBinaryShiftChar(index))
	}
	return result
}

func azUpdateForPair(list azStateSlice, data []byte, index, pairCode int) azStateSlice {
	var result azStateSlice
	for _, s := range list {
		result = append(result, azStateForPair(s, data, index, pairCode)...)
	}
	return azSimplify(result)
}

func azStateForPair(s *azState, data []byte, index, pairCode int) azStateSlice {
	noBinary := s.endBinaryShift(index)
	// Possibility 1: latch to PUNCT, then append the pair code.
	result := azStateSlice{noBinary.latchAndAppend(azPunct, pairCode)}
	if s.mode != azPunct {
		// Possibility 2: shift to PUNCT (every mode but PUNCT can shift).
		result = append(result, noBinary.shiftAndAppend(azPunct, pairCode))
	}
	if pairCode == 3 || pairCode == 4 {
		// Both characters are in DIGITS: two plain digits sometimes win.
		digitState := noBinary.
			latchAndAppend(azDigit, 16-pairCode). // period or comma in DIGIT
			latchAndAppend(azDigit, 1)            // space in DIGIT
		result = append(result, digitState)
	}
	if s.shiftBytes > 0 {
		// Binary only makes sense when already binary.
		result = append(result, s.addBinaryShiftChar(index).addBinaryShiftChar(index+1))
	}
	return result
}

func azHighLevelEncode(data []byte) *azBits {
	states := azStateSlice{azInitialState}
	for index := 0; index < len(data); index++ {
		var next byte
		if index+1 < len(data) {
			next = data[index+1]
		}
		pairCode := 0
		switch cur := data[index]; {
		case cur == '\r' && next == '\n':
			pairCode = 2
		case cur == '.' && next == ' ':
			pairCode = 3
		case cur == ',' && next == ' ':
			pairCode = 4
		case cur == ':' && next == ' ':
			pairCode = 5
		}
		if pairCode > 0 {
			states = azUpdateForPair(states, data, index, pairCode)
			index++
		} else {
			states = azUpdateForChar(states, data, index)
		}
	}
	best := -1
	for i, s := range states {
		if best < 0 || s.bitCount < states[best].bitCount {
			best = i
		}
	}
	if best < 0 {
		return new(azBits)
	}
	return states[best].toBitList(data)
}
