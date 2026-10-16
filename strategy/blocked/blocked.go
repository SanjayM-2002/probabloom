// The bit array is split into 512-bit blocks (one 64-byte cache line). The
// first hash value picks a block and all k bits go inside it, so a lookup
// touches one cache line instead of k.
package blocked

import (
	"context"
	"math"
	"math/bits"

	"github.com/SanjayM-2002/probabloom"
	"github.com/SanjayM-2002/probabloom/internal/hash"
	"github.com/SanjayM-2002/probabloom/internal/probe"
	"github.com/SanjayM-2002/probabloom/internal/sizing"
	"github.com/SanjayM-2002/probabloom/store"
)

const blockBits = 512

type Filter struct {
	bits   store.BitStore
	probe  probe.Func
	k      uint64
	blocks uint64
}

var _ probabloom.Filter = (*Filter)(nil)

func New(newBits store.Factory, p probe.Func, n uint64, fp float64) (*Filter, error) {
	params, err := sizing.Optimal(n, fp)
	if err != nil {
		return nil, err
	}
	// items land in blocks unevenly, so the standard m is not enough;
	// add blocks until the blocked formula meets the target
	blocks := (params.M + blockBits - 1) / blockBits
	for fpRate(n, blocks, params.K) > fp {
		blocks += max(1, blocks/50)
	}
	return &Filter{bits: newBits(blocks * blockBits), probe: p, k: params.K, blocks: blocks}, nil
}

// Block loads are Poisson(n/blocks); a block holding i items is a tiny
// standard filter with 512 bits, so FP = sum_i P(i) * (1 - (1-1/512)^(k*i))^k.
func fpRate(n, blocks, k uint64) float64 {
	lambda := float64(n) / float64(blocks)
	pmf := math.Exp(-lambda)
	limit := int(lambda + 10*math.Sqrt(lambda) + 20)

	var fp float64
	for i := 0; i <= limit; i++ {
		if i > 0 {
			pmf *= lambda / float64(i)
		}
		set := 1 - math.Pow(1-1.0/blockBits, float64(k)*float64(i))
		fp += pmf * math.Pow(set, float64(k))
	}
	return fp
}

func (f *Filter) Add(ctx context.Context, data []byte) error {
	return f.bits.Set(ctx, f.positions(data))
}

func (f *Filter) Contains(ctx context.Context, data []byte) (bool, error) {
	return f.bits.Test(ctx, f.positions(data))
}

func (f *Filter) Params() sizing.Params { return sizing.Params{M: f.blocks * blockBits, K: f.k} }

func (f *Filter) SizeBytes() uint64 { return f.bits.SizeBytes() }

func (f *Filter) positions(data []byte) []uint64 {
	vals := make([]uint64, f.k+1)
	f.probe(data, vals)

	// Each value is remixed before we slice bits out of it: FNV's low bits are
	// weak, and with double hashing every value shares h1, so the in-block
	// bits would otherwise correlate with the block choice.
	block, _ := bits.Mul64(hash.Mix64(vals[0]), f.blocks)
	base := block * blockBits

	pos := vals[1:]
	for i := range pos {
		bit, _ := bits.Mul64(hash.Mix64(pos[i]), blockBits)
		pos[i] = base + bit
	}
	return pos
}

func (f *Filter) MarshalBinary() ([]byte, error) { return store.Marshal(f.bits) }

func (f *Filter) UnmarshalBinary(data []byte) error { return store.Unmarshal(f.bits, data) }
