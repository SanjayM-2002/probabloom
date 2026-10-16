// Like standard, but each slot is a 4-bit counter instead of a bit, so items
// can be removed. Two counters are packed into each byte.
package counting

import (
	"context"
	"errors"
	"fmt"
	"sync"

	"github.com/SanjayM-2002/probabloom"
	"github.com/SanjayM-2002/probabloom/internal/probe"
	"github.com/SanjayM-2002/probabloom/internal/sizing"
)

const maxCount = 15

var ErrNotPresent = errors.New("item is not in the filter")

type Filter struct {
	mu       sync.RWMutex
	counters []uint8
	probe    probe.Func
	params   sizing.Params
}

var _ probabloom.Remover = (*Filter)(nil)

func New(p probe.Func, params sizing.Params) *Filter {
	return &Filter{counters: make([]uint8, (params.M+1)/2), probe: p, params: params}
}

func (f *Filter) get(i uint64) uint8 {
	if i%2 == 0 {
		return f.counters[i/2] & 0x0f
	}
	return f.counters[i/2] >> 4
}

func (f *Filter) set(i uint64, v uint8) {
	if i%2 == 0 {
		f.counters[i/2] = f.counters[i/2]&0xf0 | v
	} else {
		f.counters[i/2] = f.counters[i/2]&0x0f | v<<4
	}
}

func (f *Filter) Add(_ context.Context, data []byte) error {
	pos := f.positions(data)
	f.mu.Lock()
	defer f.mu.Unlock()
	for _, p := range pos {
		if c := f.get(p); c < maxCount {
			f.set(p, c+1)
		}
	}
	return nil
}

func (f *Filter) Contains(_ context.Context, data []byte) (bool, error) {
	pos := f.positions(data)
	f.mu.RLock()
	defer f.mu.RUnlock()
	for _, p := range pos {
		if f.get(p) == 0 {
			return false, nil
		}
	}
	return true, nil
}

// Removing an item that was never added (but is a false positive) will
// decrement other items' counters and can cause false negatives.
func (f *Filter) Remove(_ context.Context, data []byte) error {
	pos := f.positions(data)
	f.mu.Lock()
	defer f.mu.Unlock()
	for _, p := range pos {
		if f.get(p) == 0 {
			return ErrNotPresent
		}
	}
	for _, p := range pos {
		// a saturated counter no longer knows its real count, so it stays put
		if c := f.get(p); c > 0 && c < maxCount {
			f.set(p, c-1)
		}
	}
	return nil
}

func (f *Filter) Params() sizing.Params { return f.params }

func (f *Filter) SizeBytes() uint64 { return uint64(len(f.counters)) }

func (f *Filter) MarshalBinary() ([]byte, error) {
	f.mu.RLock()
	defer f.mu.RUnlock()
	return append([]byte(nil), f.counters...), nil
}

func (f *Filter) UnmarshalBinary(data []byte) error {
	f.mu.Lock()
	defer f.mu.Unlock()
	if len(data) != len(f.counters) {
		return fmt.Errorf("counting: got %d bytes, want %d", len(data), len(f.counters))
	}
	copy(f.counters, data)
	return nil
}

func (f *Filter) positions(data []byte) []uint64 {
	pos := make([]uint64, f.params.K)
	f.probe(data, pos)
	for i := range pos {
		pos[i] %= f.params.M
	}
	return pos
}
