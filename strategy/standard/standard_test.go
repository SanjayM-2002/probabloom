package standard

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

func TestPositionsInRange(t *testing.T) {
	f := newFilter(t, 1000, 0.01)
	pos := f.positions([]byte("robin"))
	if uint64(len(pos)) != f.params.K {
		t.Fatalf("got %d positions, want k=%d", len(pos), f.params.K)
	}
	for _, p := range pos {
		if p >= f.params.M {
			t.Errorf("position %d out of range m=%d", p, f.params.M)
		}
	}
}
