package aztec

// azBits is an MSB-first bit list: AddBits appends the low `count` bits of
// v highest bit first.
type azBits struct {
	bits []bool
}

func (b *azBits) Len() int { return len(b.bits) }

func (b *azBits) Get(i int) bool { return b.bits[i] }

func (b *azBits) AddBit(v bool) { b.bits = append(b.bits, v) }

func (b *azBits) AddBits(v int, count byte) {
	for i := int(count) - 1; i >= 0; i-- {
		b.bits = append(b.bits, v>>uint(i)&1 == 1)
	}
}

func (b *azBits) AddByte(v byte) {
	for i := 7; i >= 0; i-- {
		b.bits = append(b.bits, v>>uint(i)&1 == 1)
	}
}
