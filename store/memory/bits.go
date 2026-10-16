package memory

import (
	"context"
	"encoding/binary"
	"fmt"
	"sync/atomic"

	"github.com/SanjayM-2002/probabloom/store"
)

// Safe for concurrent use: bits are only ever set, never cleared, so atomic
// OR and atomic loads are enough without a lock.
type Bits struct {
	words []uint64
	m     uint64
}

var _ store.BitStore = (*Bits)(nil)

func NewBits(m uint64) *Bits {
	return &Bits{words: make([]uint64, (m+63)/64), m: m}
}

func Factory(m uint64) store.BitStore { return NewBits(m) }

func (b *Bits) Set(_ context.Context, positions []uint64) error {
	for _, p := range positions {
		atomic.OrUint64(&b.words[p/64], 1<<(p%64))
	}
	return nil
}

func (b *Bits) Test(_ context.Context, positions []uint64) (bool, error) {
	for _, p := range positions {
		if atomic.LoadUint64(&b.words[p/64])&(1<<(p%64)) == 0 {
			return false, nil
		}
	}
	return true, nil
}

func (b *Bits) Len() uint64 { return b.m }

func (b *Bits) SizeBytes() uint64 { return uint64(len(b.words)) * 8 }

// layout: m (8 bytes) followed by the words, all little-endian
func (b *Bits) MarshalBinary() ([]byte, error) {
	buf := make([]byte, 8+8*len(b.words))
	binary.LittleEndian.PutUint64(buf, b.m)
	for i := range b.words {
		binary.LittleEndian.PutUint64(buf[8+8*i:], atomic.LoadUint64(&b.words[i]))
	}
	return buf, nil
}

func (b *Bits) UnmarshalBinary(data []byte) error {
	if len(data) != 8+8*len(b.words) {
		return fmt.Errorf("bits: got %d bytes, want %d", len(data), 8+8*len(b.words))
	}
	if m := binary.LittleEndian.Uint64(data); m != b.m {
		return fmt.Errorf("bits: saved with m=%d, this store has m=%d", m, b.m)
	}
	for i := range b.words {
		atomic.StoreUint64(&b.words[i], binary.LittleEndian.Uint64(data[8+8*i:]))
	}
	return nil
}
