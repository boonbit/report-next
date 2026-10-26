package maxicode

// Small Reed-Solomon encoder over GF(2^6) (primitive polynomial 0x43),
// following the published shift-register algorithm with a precomputed
// generator polynomial.

type mxRS struct {
	size    int
	ecc     int
	log     []int
	alog    []int
	genPoly []int
}

func mxNewRS(polynomial, eccSymbols int) *mxRS {
	size := 0
	for b := 1; b <= polynomial; b <<= 1 {
		size++
	}
	size--
	// Top bit mirrors upstream `b >>= 1`: the first bit above the field.
	top := 1 << uint(size)
	logSize := (1 << uint(size)) - 1
	logTbl := make([]int, logSize+1)
	alogTbl := make([]int, logSize)
	gen := make([]int, eccSymbols+1)
	for p, v := 1, 0; v < logSize; v++ {
		alogTbl[v] = p
		logTbl[p] = v
		p <<= 1
		if p&top != 0 {
			p ^= polynomial
		}
	}
	index := 1
	gen[0] = 1
	for i := 1; i <= eccSymbols; i++ {
		gen[i] = 1
		for k := i - 1; k > 0; k-- {
			if gen[k] != 0 {
				gen[k] = alogTbl[(logTbl[gen[k]]+index)%logSize]
			}
			gen[k] ^= gen[k-1]
		}
		gen[0] = alogTbl[(logTbl[gen[0]]+index)%logSize]
		index++
	}
	return &mxRS{size: logSize, ecc: eccSymbols, log: logTbl, alog: alogTbl, genPoly: gen}
}

func (e *mxRS) encode(data []byte) []byte {
	ecc := make([]byte, e.ecc)
	for i := 0; i < len(data); i++ {
		m := ecc[e.ecc-1] ^ data[i]
		for j := e.ecc - 1; j > 0; j-- {
			if m != 0 && e.genPoly[j] != 0 {
				ecc[j] = ecc[j-1] ^ byte(e.alog[(e.log[m]+e.log[e.genPoly[j]])%e.size])
			} else {
				ecc[j] = ecc[j-1]
			}
		}
		if m != 0 && e.genPoly[0] != 0 {
			ecc[0] = byte(e.alog[(e.log[m]+e.log[e.genPoly[0]])%e.size])
		} else {
			ecc[0] = 0
		}
	}
	return ecc
}

var (
	mxRSECC10 = mxNewRS(0x43, 10)
	mxRSECC20 = mxNewRS(0x43, 20)
	mxRSECC28 = mxNewRS(0x43, 28)
)

func mxRSFor(eccLen int) *mxRS {
	switch eccLen {
	case 10:
		return mxRSECC10
	case 20:
		return mxRSECC20
	default:
		return mxRSECC28
	}
}
