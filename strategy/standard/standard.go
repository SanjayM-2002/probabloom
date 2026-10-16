// Classic bloom filter: k positions anywhere in one m-bit array.
package standard

import (
	"context"

	"github.com/SanjayM-2002/probabloom"
	"github.com/SanjayM-2002/probabloom/internal/probe"
	"github.com/SanjayM-2002/probabloom/internal/sizing"
	"github.com/SanjayM-2002/probabloom/store"
)

type Filter struct {
	bits   store.BitStore
	probe  probe.Func
	params sizing.Params
}

var _ probabloom.Filter = (*Filter)(nil)

func New(newBits store.Factory, p probe.Func, params sizing.Params) *Filter {
	return &Filter{bits: newBits(params.M), probe: p, params: params}
}

func (f *Filter) Add(ctx context.Context, data []byte) error {
	return f.bits.Set(ctx, f.positions(data))
}

func (f *Filter) Contains(ctx context.Context, data []byte) (bool, error) {
	return f.bits.Test(ctx, f.positions(data))
}

func (f *Filter) Params() sizing.Params { return f.params }

func (f *Filter) SizeBytes() uint64 { return f.bits.SizeBytes() }

func (f *Filter) positions(data []byte) []uint64 {
	pos := make([]uint64, f.params.K)
	f.probe(data, pos)
	for i := range pos {
		pos[i] %= f.params.M
	}
	return pos
}

func (f *Filter) MarshalBinary() ([]byte, error) { return store.Marshal(f.bits) }

func (f *Filter) UnmarshalBinary(data []byte) error { return store.Unmarshal(f.bits, data) }
