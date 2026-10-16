package blocked

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
	f, err := New(memory.Factory, probe.Double(hash.FNV1a), n, p)
	if err != nil {
		t.Fatal(err)
	}
	return f
}

func TestConformance(t *testing.T) {
	filtertest.Run(t, func(t *testing.T, n uint64, p float64) probabloom.Filter {
		return newFilter(t, n, p)
	})
}

func TestAllPositionsInOneBlock(t *testing.T) {
	f := newFilter(t, 100_000, 0.01)
	for _, item := range []string{"robin", "alice", "bob", "x"} {
		pos := f.positions([]byte(item))
		if uint64(len(pos)) != f.k {
			t.Fatalf("got %d positions, want %d", len(pos), f.k)
		}
		block := pos[0] / blockBits
		for _, p := range pos {
			if p/blockBits != block {
				t.Errorf("%s: positions %v span more than one block", item, pos)
				break
			}
		}
	}
}

// the formula must agree with a standard filter when every block holds
// its fair share, and the size we pick must be bigger than the standard m
func TestSizingCompensates(t *testing.T) {
	for _, p := range []float64{0.1, 0.01, 0.001} {
		f := newFilter(t, 1_000_000, p)
		std, _ := sizing.Optimal(1_000_000, p)
		got := fpRate(1_000_000, f.blocks, f.k)
		t.Logf("p=%v: standard m=%d, blocked m=%d (+%.0f%%), formula fp=%.5f",
			p, std.M, f.blocks*blockBits, 100*(float64(f.blocks*blockBits)/float64(std.M)-1), got)
		if got > p {
			t.Errorf("p=%v: chosen size gives fp %.5f > target", p, got)
		}
		if f.blocks*blockBits < std.M {
			t.Errorf("p=%v: blocked m %d smaller than standard m %d", p, f.blocks*blockBits, std.M)
		}
	}
}
