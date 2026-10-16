package counting

import (
	"context"
	"errors"
	"testing"

	"github.com/SanjayM-2002/probabloom"
	"github.com/SanjayM-2002/probabloom/internal/filtertest"
	"github.com/SanjayM-2002/probabloom/internal/hash"
	"github.com/SanjayM-2002/probabloom/internal/probe"
	"github.com/SanjayM-2002/probabloom/internal/sizing"
)

func newFilter(t testing.TB, n uint64, p float64) *Filter {
	params, err := sizing.Optimal(n, p)
	if err != nil {
		t.Fatal(err)
	}
	return New(probe.Double(hash.FNV1a), params)
}

func TestConformance(t *testing.T) {
	filtertest.Run(t, func(t *testing.T, n uint64, p float64) probabloom.Filter {
		return newFilter(t, n, p)
	})
}

func TestRemove(t *testing.T) {
	ctx := context.Background()
	f := newFilter(t, 1000, 0.01)

	f.Add(ctx, []byte("robin"))
	f.Add(ctx, []byte("alice"))

	if err := f.Remove(ctx, []byte("robin")); err != nil {
		t.Fatal(err)
	}
	if ok, _ := f.Contains(ctx, []byte("robin")); ok {
		t.Error("robin still present after Remove")
	}
	if ok, _ := f.Contains(ctx, []byte("alice")); !ok {
		t.Error("removing robin removed alice too")
	}
	if err := f.Remove(ctx, []byte("robin")); !errors.Is(err, ErrNotPresent) {
		t.Errorf("second Remove: got %v, want ErrNotPresent", err)
	}
}

func TestAddTwiceNeedsRemoveTwice(t *testing.T) {
	ctx := context.Background()
	f := newFilter(t, 1000, 0.01)

	f.Add(ctx, []byte("robin"))
	f.Add(ctx, []byte("robin"))
	f.Remove(ctx, []byte("robin"))
	if ok, _ := f.Contains(ctx, []byte("robin")); !ok {
		t.Fatal("added twice, removed once: should still be present")
	}
	f.Remove(ctx, []byte("robin"))
	if ok, _ := f.Contains(ctx, []byte("robin")); ok {
		t.Fatal("added twice, removed twice: should be gone")
	}
}

func TestSaturatedCounterSticks(t *testing.T) {
	ctx := context.Background()
	f := newFilter(t, 1000, 0.01)

	for i := 0; i < 300; i++ {
		f.Add(ctx, []byte("robin"))
	}
	for i := 0; i < 300; i++ {
		f.Remove(ctx, []byte("robin"))
	}
	if ok, _ := f.Contains(ctx, []byte("robin")); !ok {
		t.Fatal("saturated counters should never drop back to 0")
	}
}
