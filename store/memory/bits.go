package memory

import (
	"context"

	"github.com/SanjayM-2002/probabloom/store"
)

// Not safe for concurrent use.
type Bits struct {
	words []uint64
	m     uint64
}

var _ store.BitStore = (*Bits)(nil)

func NewBits(m uint64) *Bits {
	return &Bits{words: make([]uint64, (m+63)/64), m: m}
}

func (b *Bits) Set(_ context.Context, positions []uint64) error {
	for _, p := range positions {
		b.words[p/64] |= 1 << (p % 64)
	}
	return nil
}

func (b *Bits) Test(_ context.Context, positions []uint64) (bool, error) {
	for _, p := range positions {
		if b.words[p/64]&(1<<(p%64)) == 0 {
			return false, nil
		}
	}
	return true, nil
}

func (b *Bits) Len() uint64 { return b.m }

func (b *Bits) SizeBytes() uint64 { return uint64(len(b.words)) * 8 }

func Factory(m uint64) store.BitStore { return NewBits(m) }
