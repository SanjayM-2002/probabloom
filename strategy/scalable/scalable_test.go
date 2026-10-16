package scalable

import (
	"context"
	"fmt"
	"testing"

	"github.com/SanjayM-2002/probabloom"
	"github.com/SanjayM-2002/probabloom/internal/filtertest"
	"github.com/SanjayM-2002/probabloom/internal/hash"
	"github.com/SanjayM-2002/probabloom/internal/probe"
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

// add 20x the initial capacity: the filter must grow and still hold its fp target
func TestGrowsAndKeepsFPRate(t *testing.T) {
	ctx := context.Background()
	const n0, p = 10_000, 0.01
	f := newFilter(t, n0, p)

	const added = 20 * n0
	for i := 0; i < added; i++ {
		f.Add(ctx, []byte(fmt.Sprintf("user-%d", i)))
	}
	if f.Layers() < 4 {
		t.Errorf("layers = %d after %dx capacity, expected the filter to grow", f.Layers(), added/n0)
	}
	for i := 0; i < added; i++ {
		if ok, _ := f.Contains(ctx, []byte(fmt.Sprintf("user-%d", i))); !ok {
			t.Fatalf("false negative for user-%d", i)
		}
	}

	fp := 0
	for i := 0; i < added; i++ {
		if ok, _ := f.Contains(ctx, []byte(fmt.Sprintf("absent-%d", i))); ok {
			fp++
		}
	}
	got := float64(fp) / added
	t.Logf("layers %d, size %d KB, target %.4f, measured %.4f", f.Layers(), f.SizeBytes()/1024, p, got)
	if got > p {
		t.Errorf("false-positive rate %.4f exceeds overall target %.4f", got, p)
	}
}

func TestDuplicatesDontUseCapacity(t *testing.T) {
	ctx := context.Background()
	f := newFilter(t, 100, 0.01)
	for i := 0; i < 10_000; i++ {
		f.Add(ctx, []byte("robin"))
	}
	if f.Layers() != 1 {
		t.Errorf("layers = %d after adding the same item repeatedly, want 1", f.Layers())
	}
}
