// Like standard, but each slot is a counter instead of a bit, so items can be removed.
package counting

import (
	"context"
	"errors"
	"math"

	"github.com/SanjayM-2002/probabloom"
	"github.com/SanjayM-2002/probabloom/internal/probe"
	"github.com/SanjayM-2002/probabloom/internal/sizing"
)

var ErrNotPresent = errors.New("item is not in the filter")

type Filter struct {
	counters []uint8
	probe    probe.Func
	params   sizing.Params
}

var _ probabloom.Remover = (*Filter)(nil)

func New(p probe.Func, params sizing.Params) *Filter {
	return &Filter{counters: make([]uint8, params.M), probe: p, params: params}
}

func (f *Filter) Add(_ context.Context, data []byte) error {
	for _, p := range f.positions(data) {
		if f.counters[p] < math.MaxUint8 {
			f.counters[p]++
		}
	}
	return nil
}

func (f *Filter) Contains(_ context.Context, data []byte) (bool, error) {
	for _, p := range f.positions(data) {
		if f.counters[p] == 0 {
			return false, nil
		}
	}
	return true, nil
}

// Removing an item that was never added (but is a false positive) will
// decrement other items' counters and can cause false negatives.
func (f *Filter) Remove(_ context.Context, data []byte) error {
	pos := f.positions(data)
	for _, p := range pos {
		if f.counters[p] == 0 {
			return ErrNotPresent
		}
	}
	for _, p := range pos {
		// a saturated counter no longer knows its real count, so it stays put
		if c := f.counters[p]; c > 0 && c < math.MaxUint8 {
			f.counters[p]--
		}
	}
	return nil
}

func (f *Filter) Params() sizing.Params { return f.params }

func (f *Filter) SizeBytes() uint64 { return uint64(len(f.counters)) }

func (f *Filter) positions(data []byte) []uint64 {
	pos := make([]uint64, f.params.K)
	f.probe(data, pos)
	for i := range pos {
		pos[i] %= f.params.M
	}
	return pos
}
