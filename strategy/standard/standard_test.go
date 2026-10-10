package standard

import (
	"context"
	"fmt"
	"testing"

	"github.com/SanjayM-2002/probabloom"
	"github.com/SanjayM-2002/probabloom/internal/filtertest"
	"github.com/SanjayM-2002/probabloom/internal/hash"
	"github.com/SanjayM-2002/probabloom/internal/sizing"
	"github.com/SanjayM-2002/probabloom/store/memory"
)

func newMemory(t testing.TB, n uint64, p float64) *Filter {
	params, err := sizing.Optimal(n, p)
	if err != nil {
		t.Fatal(err)
	}
	return New(memory.NewBits(params.M), hash.FNV1a, params)
}

func TestConformance(t *testing.T) {
	filtertest.Run(t, func(t *testing.T, n uint64, p float64) probabloom.Filter {
		return newMemory(t, n, p)
	})
}

func TestPositions(t *testing.T) {
	f := newMemory(t, 1000, 0.01)
	pos := f.positions([]byte("robin"))

	if uint64(len(pos)) != f.params.K {
		t.Fatalf("got %d positions, want k=%d", len(pos), f.params.K)
	}
	for _, p := range pos {
		if p >= f.params.M {
			t.Errorf("position %d out of range m=%d", p, f.params.M)
		}
	}
	again := f.positions([]byte("robin"))
	for i := range pos {
		if pos[i] != again[i] {
			t.Fatal("positions are not deterministic")
		}
	}
}

func BenchmarkAdd(b *testing.B) {
	f := newMemory(b, 1_000_000, 0.01)
	ctx := context.Background()
	keys := benchKeys(1024)
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		f.Add(ctx, keys[i%len(keys)])
	}
}

func BenchmarkContains(b *testing.B) {
	f := newMemory(b, 1_000_000, 0.01)
	ctx := context.Background()
	keys := benchKeys(1024)
	for _, k := range keys[:512] {
		f.Add(ctx, k)
	}
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		f.Contains(ctx, keys[i%len(keys)])
	}
}

func benchKeys(n int) [][]byte {
	keys := make([][]byte, n)
	for i := range keys {
		keys[i] = []byte(fmt.Sprintf("user_%06d", i))
	}
	return keys
}
