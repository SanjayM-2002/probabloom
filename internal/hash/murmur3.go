package hash

import (
	"encoding/binary"
	"math/bits"
)

const (
	mmC1 uint64 = 0x87c37b91114253d5
	mmC2 uint64 = 0x4cf5ad432745937f
)

// Mix64 is Murmur3's finalizer: every input bit affects every output bit.
func Mix64(k uint64) uint64 {
	k ^= k >> 33
	k *= 0xff51afd7ed558ccd
	k ^= k >> 33
	k *= 0xc4ceb9fe1a85ec53
	k ^= k >> 33
	return k
}

// MurmurHash3 x64_128, returning the first 64 bits.
func Murmur3(seed uint64, data []byte) uint64 {
	n := uint64(len(data))
	h1, h2 := seed, seed

	for len(data) >= 16 {
		k1 := binary.LittleEndian.Uint64(data[0:])
		k2 := binary.LittleEndian.Uint64(data[8:])

		k1 *= mmC1
		k1 = bits.RotateLeft64(k1, 31)
		k1 *= mmC2
		h1 ^= k1
		h1 = bits.RotateLeft64(h1, 27)
		h1 += h2
		h1 = h1*5 + 0x52dce729

		k2 *= mmC2
		k2 = bits.RotateLeft64(k2, 33)
		k2 *= mmC1
		h2 ^= k2
		h2 = bits.RotateLeft64(h2, 31)
		h2 += h1
		h2 = h2*5 + 0x38495ab5

		data = data[16:]
	}

	var k1, k2 uint64
	for i := len(data) - 1; i >= 8; i-- {
		k2 ^= uint64(data[i]) << ((i - 8) * 8)
	}
	if len(data) > 8 {
		k2 *= mmC2
		k2 = bits.RotateLeft64(k2, 33)
		k2 *= mmC1
		h2 ^= k2
	}
	for i := min(len(data), 8) - 1; i >= 0; i-- {
		k1 ^= uint64(data[i]) << (i * 8)
	}
	if len(data) > 0 {
		k1 *= mmC1
		k1 = bits.RotateLeft64(k1, 31)
		k1 *= mmC2
		h1 ^= k1
	}

	h1 ^= n
	h2 ^= n
	h1 += h2
	h2 += h1
	h1 = Mix64(h1)
	h2 = Mix64(h2)
	h1 += h2
	return h1
}
