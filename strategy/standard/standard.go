// Classic bloom filter: position i = hash(seed=i, data) % m.
package standard

import (
	"context"

	"github.com/SanjayM-2002/probabloom"
	"github.com/SanjayM-2002/probabloom/internal/hash"
	"github.com/SanjayM-2002/probabloom/internal/sizing"
	"github.com/SanjayM-2002/probabloom/store"
)

type Filter struct {
	bits   store.BitStore
	hash   hash.Func
	params sizing.Params
}

var _ probabloom.Filter = (*Filter)(nil)

func New(bits store.BitStore, hashFn hash.Func, params sizing.Params) *Filter {
	return &Filter{bits: bits, hash: hashFn, params: params}
}

func (f *Filter) Add(ctx context.Context, data []byte) error {
	return f.bits.Set(ctx, f.positions(data))
}

func (f *Filter) Contains(ctx context.Context, data []byte) (bool, error) {
	return f.bits.Test(ctx, f.positions(data))
}

func (f *Filter) Params() sizing.Params { return f.params }

func (f *Filter) positions(data []byte) []uint64 {
	pos := make([]uint64, f.params.K)
	for i := range pos {
		pos[i] = f.hash(uint64(i), data) % f.params.M
	}
	return pos
}
