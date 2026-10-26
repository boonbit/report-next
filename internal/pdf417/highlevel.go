package pdf417

// PDF417 high-level encodation (text/byte/numeric submodes), following the
// published algorithm: latch 900/901/902/924, shift 913, numeric chunks of
// 44 digits via base-900 of 1+digits, byte sixpacks of 6 bytes into 5
// base-900 codewords, text pairs packed as 30*h+l.

import (
	"errors"
	"math/big"
)

type pdfEncodingMode byte
type pdfSubMode byte

const (
	pdfEncText pdfEncodingMode = iota
	pdfEncNumeric
	pdfEncBinary

	pdfSubUpper pdfSubMode = iota
	pdfSubLower
	pdfSubMixed
	pdfSubPunct

	pdfLatchText       = 900
	pdfLatchBytePadded = 901
	pdfLatchNumeric    = 902
	pdfLatchByte       = 924
	pdfShiftByte       = 913

	pdfMinNumericCount = 13
)

var (
	pdfMixedMap map[rune]int
	pdfPunctMap map[rune]int
)

func init() {
	pdfMixedMap = map[rune]int{}
	for idx, ch := range []rune{
		48, 49, 50, 51, 52, 53, 54, 55, 56, 57, 38, 13, 9, 44, 58,
		35, 45, 46, 36, 47, 43, 37, 42, 61, 94, 0, 32, 0, 0, 0,
	} {
		if ch > 0 {
			pdfMixedMap[ch] = idx
		}
	}
	pdfPunctMap = map[rune]int{}
	for idx, ch := range []rune{
		59, 60, 62, 64, 91, 92, 93, 95, 96, 126, 33, 13, 9, 44, 58,
		10, 45, 46, 36, 47, 34, 124, 42, 40, 41, 63, 123, 125, 39, 0,
	} {
		if ch > 0 {
			pdfPunctMap[ch] = idx
		}
	}
}

func pdfRuneToInt(r rune) int {
	if r >= '0' && r <= '9' {
		return int(r - '0')
	}
	return -1
}

func pdfDigitCount(data []rune) int {
	cnt := 0
	for _, r := range data {
		if pdfRuneToInt(r) == -1 {
			break
		}
		cnt++
	}
	return cnt
}

func pdfEncodeNumeric(digits []rune) ([]int, error) {
	digitCount := len(digits)
	chunkCount := digitCount / 44
	if digitCount%44 != 0 {
		chunkCount++
	}
	codeWords := []int{}
	for i := 0; i < chunkCount; i++ {
		start := i * 44
		end := start + 44
		if end > digitCount {
			end = digitCount
		}
		chunk := digits[start:end]
		chunkNum := big.NewInt(0)
		if _, ok := chunkNum.SetString("1"+string(chunk), 10); !ok {
			return nil, errors.New("pdf417: bad numeric chunk " + string(chunk))
		}
		cws := []int{}
		for chunkNum.Cmp(big.NewInt(0)) > 0 {
			newChunk, cw := chunkNum.DivMod(chunkNum, big.NewInt(900), big.NewInt(0))
			chunkNum = newChunk
			cws = append([]int{int(cw.Int64())}, cws...)
		}
		codeWords = append(codeWords, cws...)
	}
	return codeWords, nil
}

func pdfTextCount(msg []rune) int {
	result := 0
	isText := func(ch rune) bool {
		return ch == '\t' || ch == '\n' || ch == '\r' || (ch >= 32 && ch <= 126)
	}
	for i, ch := range msg {
		numericCount := pdfDigitCount(msg[i:])
		if numericCount >= pdfMinNumericCount || (numericCount == 0 && !isText(ch)) {
			break
		}
		result++
	}
	return result
}

func pdfEncodeText(text []rune, submode pdfSubMode) (pdfSubMode, []int) {
	isAlphaUpper := func(ch rune) bool { return ch == ' ' || (ch >= 'A' && ch <= 'Z') }
	isAlphaLower := func(ch rune) bool { return ch == ' ' || (ch >= 'a' && ch <= 'z') }
	isMixed := func(ch rune) bool { _, ok := pdfMixedMap[ch]; return ok }
	isPunct := func(ch rune) bool { _, ok := pdfPunctMap[ch]; return ok }

	idx := 0
	var tmp []int
	for idx < len(text) {
		ch := text[idx]
		switch submode {
		case pdfSubUpper:
			if isAlphaUpper(ch) {
				if ch == ' ' {
					tmp = append(tmp, 26)
				} else {
					tmp = append(tmp, int(ch-'A'))
				}
			} else if isAlphaLower(ch) {
				submode = pdfSubLower
				tmp = append(tmp, 27) // lower latch
				continue
			} else if isMixed(ch) {
				submode = pdfSubMixed
				tmp = append(tmp, 28) // mixed latch
				continue
			} else {
				tmp = append(tmp, 29) // punctuation switch
				tmp = append(tmp, pdfPunctMap[ch])
				break
			}
			break
		case pdfSubLower:
			if isAlphaLower(ch) {
				if ch == ' ' {
					tmp = append(tmp, 26)
				} else {
					tmp = append(tmp, int(ch-'a'))
				}
			} else if isAlphaUpper(ch) {
				tmp = append(tmp, 27) // upper switch
				tmp = append(tmp, int(ch-'A'))
				break
			} else if isMixed(ch) {
				submode = pdfSubMixed
				tmp = append(tmp, 28) // mixed latch
				continue
			} else {
				tmp = append(tmp, 29) // punctuation switch
				tmp = append(tmp, pdfPunctMap[ch])
				break
			}
			break
		case pdfSubMixed:
			if isMixed(ch) {
				tmp = append(tmp, pdfMixedMap[ch])
			} else if isAlphaUpper(ch) {
				submode = pdfSubUpper
				tmp = append(tmp, 28) // upper latch
				continue
			} else if isAlphaLower(ch) {
				submode = pdfSubLower
				tmp = append(tmp, 27) // lower latch
				continue
			} else {
				if idx+1 < len(text) {
					if isPunct(text[idx+1]) {
						submode = pdfSubPunct
						tmp = append(tmp, 25) // punctuation latch
						continue
					}
				}
				tmp = append(tmp, 29) // punctuation switch
				tmp = append(tmp, pdfPunctMap[ch])
			}
			break
		default: // pdfSubPunct
			if isPunct(ch) {
				tmp = append(tmp, pdfPunctMap[ch])
			} else {
				submode = pdfSubUpper
				tmp = append(tmp, 29) // upper latch
				continue
			}
		}
		idx++
	}
	h := 0
	result := []int{}
	for i, val := range tmp {
		if i%2 != 0 {
			h = h*30 + val
			result = append(result, h)
		} else {
			h = val
		}
	}
	if len(tmp)%2 != 0 {
		result = append(result, h*30+29)
	}
	return submode, result
}

func pdfBinaryCount(msg []byte) int {
	result := 0
	for i := range msg {
		if pdfDigitCount([]rune(string(msg[i:]))) >= pdfMinNumericCount {
			break
		}
		if pdfTextCount([]rune(string(msg[i:]))) > 5 {
			break
		}
		result++
	}
	return result
}

func pdfEncodeBinary(data []byte, startmode pdfEncodingMode) []int {
	result := []int{}
	count := len(data)
	switch {
	case count == 1 && startmode == pdfEncText:
		result = append(result, pdfShiftByte)
	case count%6 == 0:
		result = append(result, pdfLatchByte)
	default:
		result = append(result, pdfLatchBytePadded)
	}
	idx := 0
	if count >= 6 {
		words := make([]int, 5)
		for count-idx >= 6 {
			var t int64
			for i := 0; i < 6; i++ {
				t = t<<8 + int64(data[idx+i])
			}
			for i := 0; i < 5; i++ {
				words[4-i] = int(t % 900)
				t /= 900
			}
			result = append(result, words...)
			idx += 6
		}
	}
	for i := idx; i < count; i++ {
		result = append(result, int(data[i]&0xff))
	}
	return result
}

func pdfHighLevelEncode(data []byte) ([]int, error) {
	encodingMode := pdfEncText
	textSubMode := pdfSubUpper
	result := []int{}
	for len(data) > 0 {
		numericCount := pdfDigitCount([]rune(string(data)))
		if numericCount >= pdfMinNumericCount || numericCount == len(data) {
			result = append(result, pdfLatchNumeric)
			encodingMode = pdfEncNumeric
			textSubMode = pdfSubUpper
			numData, err := pdfEncodeNumeric([]rune(string(data[:numericCount])))
			if err != nil {
				return nil, err
			}
			result = append(result, numData...)
			data = data[numericCount:]
			continue
		}
		textCount := pdfTextCount([]rune(string(data)))
		if textCount >= 5 || textCount == len(data) {
			if encodingMode != pdfEncText {
				result = append(result, pdfLatchText)
				encodingMode = pdfEncText
				textSubMode = pdfSubUpper
			}
			var txtData []int
			textSubMode, txtData = pdfEncodeText([]rune(string(data[:textCount])), textSubMode)
			result = append(result, txtData...)
			data = data[textCount:]
			continue
		}
		binaryCount := pdfBinaryCount(data)
		if binaryCount == 0 {
			binaryCount = 1
		}
		chunk := data[:binaryCount]
		if len(chunk) != 1 || encodingMode != pdfEncText {
			encodingMode = pdfEncBinary
			textSubMode = pdfSubUpper
		}
		result = append(result, pdfEncodeBinary(chunk, encodingMode)...)
		data = data[binaryCount:]
	}
	return result, nil
}
