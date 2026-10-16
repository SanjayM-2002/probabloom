package hash

import (
	"encoding/binary"
	"math/bits"
)

const (
	xxP1 uint64 = 0x9E3779B185EBCA87
	xxP2 uint64 = 0xC2B2AE3D27D4EB4F
	xxP3 uint64 = 0x165667B19E3779F9
	xxP4 uint64 = 0x85EBCA77C2B2AE63
	xxP5 uint64 = 0x27D4EB2F165667C5
)

func xxRound(acc, input uint64) uint64 {
	acc += input * xxP2
	acc = bits.RotateLeft64(acc, 31)
	return acc * xxP1
}

func xxMerge(acc, val uint64) uint64 {
	acc ^= xxRound(0, val)
	return acc*xxP1 + xxP4
}

func XXH64(seed uint64, data []byte) uint64 {
	n := uint64(len(data))
	var h uint64

	if len(data) >= 32 {
		v1, v2, v3, v4 := seed+xxP1+xxP2, seed+xxP2, seed, seed-xxP1
		for len(data) >= 32 {
			v1 = xxRound(v1, binary.LittleEndian.Uint64(data[0:]))
			v2 = xxRound(v2, binary.LittleEndian.Uint64(data[8:]))
			v3 = xxRound(v3, binary.LittleEndian.Uint64(data[16:]))
			v4 = xxRound(v4, binary.LittleEndian.Uint64(data[24:]))
			data = data[32:]
		}
		h = bits.RotateLeft64(v1, 1) + bits.RotateLeft64(v2, 7) +
			bits.RotateLeft64(v3, 12) + bits.RotateLeft64(v4, 18)
		h = xxMerge(h, v1)
		h = xxMerge(h, v2)
		h = xxMerge(h, v3)
		h = xxMerge(h, v4)
	} else {
		h = seed + xxP5
	}
	h += n

	for len(data) >= 8 {
		h ^= xxRound(0, binary.LittleEndian.Uint64(data))
		h = bits.RotateLeft64(h, 27)*xxP1 + xxP4
		data = data[8:]
	}
	if len(data) >= 4 {
		h ^= uint64(binary.LittleEndian.Uint32(data)) * xxP1
		h = bits.RotateLeft64(h, 23)*xxP2 + xxP3
		data = data[4:]
	}
	for _, b := range data {
		h ^= uint64(b) * xxP5
		h = bits.RotateLeft64(h, 11) * xxP1
	}

	h ^= h >> 33
	h *= xxP2
	h ^= h >> 29
	h *= xxP3
	h ^= h >> 32
	return h
}
