package aztec

// Generic Reed-Solomon encoder over a binary Galois field, following the
// published algorithm: log/alog tables from the primitive polynomial,
// generator polynomial product (x + a^(i-1+base)), info*monomial/generation
// division remainder as the EC words.

// azField is GF(2^m) with its primitive polynomial pp.
type azField struct {
	size   int
	base   int
	aLog   []int
	logTbl []int
}

func azNewField(pp, fieldSize, b int) *azField {
	f := &azField{size: fieldSize, base: b, aLog: make([]int, fieldSize), logTbl: make([]int, fieldSize)}
	x := 1
	for i := 0; i < fieldSize; i++ {
		f.aLog[i] = x
		x *= 2
		if x >= fieldSize {
			x = (x ^ pp) & (fieldSize - 1)
		}
	}
	for i := 0; i < fieldSize; i++ {
		f.logTbl[f.aLog[i]] = i
	}
	return f
}

func (f *azField) addOrSub(a, b int) int { return a ^ b }

func (f *azField) multiply(a, b int) int {
	if a == 0 || b == 0 {
		return 0
	}
	return f.aLog[(f.logTbl[a]+f.logTbl[b])%(f.size-1)]
}

func (f *azField) divide(a, b int) int {
	if b == 0 {
		panic("aztec: divide by zero")
	}
	if a == 0 {
		return 0
	}
	return f.aLog[(f.logTbl[a]-f.logTbl[b])%(f.size-1)]
}

func (f *azField) inverse(a int) int { return f.aLog[(f.size-1)-f.logTbl[a]] }

func (f *azField) zero() *azPoly { return azNewPoly(f, []int{0}) }

// azPoly is a polynomial with coefficients in f, highest degree first.
type azPoly struct {
	f     *azField
	coeff []int
}

func azNewPoly(f *azField, coeff []int) *azPoly {
	for len(coeff) > 1 && coeff[0] == 0 {
		coeff = coeff[1:]
	}
	return &azPoly{f, coeff}
}

func (p *azPoly) degree() int { return len(p.coeff) - 1 }

func (p *azPoly) zero() bool { return p.coeff[0] == 0 }

func (p *azPoly) at(degree int) int { return p.coeff[p.degree()-degree] }

func (p *azPoly) addOrSub(q *azPoly) *azPoly {
	if p.zero() {
		return q
	}
	if q.zero() {
		return p
	}
	small, large := p.coeff, q.coeff
	if len(small) > len(large) {
		small, large = large, small
	}
	out := make([]int, len(large))
	diff := len(large) - len(small)
	copy(out, large[:diff])
	for i := diff; i < len(large); i++ {
		out[i] = p.f.addOrSub(small[i-diff], large[i])
	}
	return azNewPoly(p.f, out)
}

func (p *azPoly) mulMonomial(degree, c int) *azPoly {
	if c == 0 {
		return p.f.zero()
	}
	out := make([]int, len(p.coeff)+degree)
	for i, v := range p.coeff {
		out[i] = p.f.multiply(v, c)
	}
	return azNewPoly(p.f, out)
}

func (p *azPoly) multiply(q *azPoly) *azPoly {
	if p.zero() || q.zero() {
		return p.f.zero()
	}
	out := make([]int, len(p.coeff)+len(q.coeff)-1)
	for i, a := range p.coeff {
		for j, b := range q.coeff {
			out[i+j] = p.f.addOrSub(out[i+j], p.f.multiply(a, b))
		}
	}
	return azNewPoly(p.f, out)
}

func (p *azPoly) divide(q *azPoly) (quot, rem *azPoly) {
	quot, rem = p.f.zero(), p
	f := p.f
	denomLead := q.at(q.degree())
	invDenom := f.inverse(denomLead)
	for rem.degree() >= q.degree() && !rem.zero() {
		diff := rem.degree() - q.degree()
		scale := f.multiply(rem.at(rem.degree()), invDenom)
		term := q.mulMonomial(diff, scale)
		quot = quot.addOrSub(azMonomial(f, diff, scale))
		rem = rem.addOrSub(term)
	}
	return quot, rem
}

func azMonomial(f *azField, degree, c int) *azPoly {
	if c == 0 {
		return f.zero()
	}
	out := make([]int, degree+1)
	out[0] = c
	return azNewPoly(f, out)
}

// azRSEncoder caches generator polynomials per degree.
type azRSEncoder struct {
	f   *azField
	gen []*azPoly
}

func azNewRS(f *azField) *azRSEncoder {
	return &azRSEncoder{f, []*azPoly{azNewPoly(f, []int{1})}}
}

func (rs *azRSEncoder) generator(degree int) *azPoly {
	for len(rs.gen) <= degree {
		d := len(rs.gen)
		last := rs.gen[d-1]
		next := last.multiply(azNewPoly(rs.f, []int{1, rs.f.aLog[d-1+rs.f.base]}))
		rs.gen = append(rs.gen, next)
	}
	return rs.gen[degree]
}

func (rs *azRSEncoder) encode(data []int, eccCount int) []int {
	gen := rs.generator(eccCount)
	info := azNewPoly(rs.f, data).mulMonomial(eccCount, 1)
	_, rem := info.divide(gen)
	out := make([]int, eccCount)
	copy(out[eccCount-len(rem.coeff):], rem.coeff)
	return out
}
