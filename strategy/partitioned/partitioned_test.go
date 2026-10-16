package partitioned

import (
	"testing"

	"github.com/SanjayM-2002/probabloom"
	"github.com/SanjayM-2002/probabloom/internal/filtertest"
	"github.com/SanjayM-2002/probabloom/internal/hash"
	"github.com/SanjayM-2002/probabloom/internal/probe"
	"github.com/SanjayM-2002/probabloom/internal/sizing"
	"github.com/SanjayM-2002/probabloom/store/memory"
)

func newFilter(t testing.TB, n uint64, p float64) *Filter {
	params, err := sizing.Optimal(n, p)
	if err != nil {
		t.Fatal(err)
	}
	return New(memory.Factory, probe.Double(hash.FNV1a), params)
}

func TestConformance(t *testing.T) {
	filtertest.Run(t, func(t *testing.T, n uint64, p float64) probabloom.Filter {
		return newFilter(t, n, p)
	})
}

func TestOnePositionPerSlice(t *testing.T) {
	f := newFilter(t, 1000, 0.01)
	for _, item := range []string{"robin", "alice", "bob", "x"} {
		for i, p := range f.positions([]byte(item)) {
			lo, hi := uint64(i)*f.slice, uint64(i+1)*f.slice
			if p < lo || p >= hi {
				t.Errorf("%s: position %d = %d, want in slice [%d, %d)", item, i, p, lo, hi)
			}
		}
	}
}
