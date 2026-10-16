// The m bits are split into k equal slices and hash i only sets a bit in slice i,
// so an item's k bits never overlap each other.
package partitioned

import (
	"context"

	"github.com/SanjayM-2002/probabloom"
	"github.com/SanjayM-2002/probabloom/internal/probe"
	"github.com/SanjayM-2002/probabloom/internal/sizing"
	"github.com/SanjayM-2002/probabloom/store"
)

type Filter struct {
	bits  store.BitStore
	probe probe.Func
	k     uint64
	slice uint64
}

var _ probabloom.Filter = (*Filter)(nil)

func New(newBits store.Factory, p probe.Func, params sizing.Params) *Filter {
	slice := (params.M + params.K - 1) / params.K
	return &Filter{bits: newBits(slice * params.K), probe: p, k: params.K, slice: slice}
}

func (f *Filter) Add(ctx context.Context, data []byte) error {
	return f.bits.Set(ctx, f.positions(data))
}

func (f *Filter) Contains(ctx context.Context, data []byte) (bool, error) {
	return f.bits.Test(ctx, f.positions(data))
}

func (f *Filter) Params() sizing.Params { return sizing.Params{M: f.slice * f.k, K: f.k} }

func (f *Filter) SizeBytes() uint64 { return f.bits.SizeBytes() }

func (f *Filter) positions(data []byte) []uint64 {
	pos := make([]uint64, f.k)
	f.probe(data, pos)
	for i := range pos {
		pos[i] = uint64(i)*f.slice + pos[i]%f.slice
	}
	return pos
}
