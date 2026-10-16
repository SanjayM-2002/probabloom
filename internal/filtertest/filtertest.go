package filtertest

import (
	"context"
	"encoding"
	"fmt"
	"sync"
	"testing"

	"github.com/SanjayM-2002/probabloom"
)

type NewFunc func(t *testing.T, n uint64, p float64) probabloom.Filter

type Options struct {
	N     int       // items per test; network backends use fewer
	Rates []float64 // false-positive targets to check
}

var defaults = Options{N: 100_000, Rates: []float64{0.1, 0.01, 0.001}}

func Run(t *testing.T, newFilter NewFunc) { RunWith(t, newFilter, defaults) }

func RunWith(t *testing.T, newFilter NewFunc, opts Options) {
	n := opts.N

	t.Run("EmptyContainsNothing", func(t *testing.T) {
		f := newFilter(t, 1000, 0.01)
		for i := 0; i < 1000; i++ {
			if mustContains(t, f, key("absent", i)) {
				t.Fatalf("empty filter reported %q as present", key("absent", i))
			}
		}
	})

	t.Run("NoFalseNegatives", func(t *testing.T) {
		f := newFilter(t, uint64(n), 0.01)
		for i := 0; i < n; i++ {
			mustAdd(t, f, key("user", i))
		}
		for i := 0; i < n; i++ {
			if !mustContains(t, f, key("user", i)) {
				t.Fatalf("false negative: %q was added but Contains returned false", key("user", i))
			}
		}
	})

	t.Run("FalsePositiveRate", func(t *testing.T) {
		for _, p := range opts.Rates {
			t.Run(fmt.Sprintf("p=%v", p), func(t *testing.T) {
				f := newFilter(t, uint64(n), p)
				for i := 0; i < n; i++ {
					mustAdd(t, f, key("user", i))
				}
				fp := 0
				for i := 0; i < n; i++ {
					if mustContains(t, f, key("absent", i)) {
						fp++
					}
				}
				got := float64(fp) / float64(n)
				t.Logf("target %.4f, measured %.4f", p, got)
				// 50% slack for sampling noise
				if got > p*1.5 {
					t.Errorf("false-positive rate %.4f exceeds target %.4f by more than 50%%", got, p)
				}
			})
		}
	})

	t.Run("Concurrent", func(t *testing.T) {
		const workers = 8
		per := max(n/50, 100)
		f := newFilter(t, uint64(workers*per), 0.01)

		var wg sync.WaitGroup
		for w := 0; w < workers; w++ {
			wg.Add(1)
			go func(w int) {
				defer wg.Done()
				for i := 0; i < per; i++ {
					k := key(fmt.Sprintf("w%d", w), i)
					if err := f.Add(context.Background(), k); err != nil {
						t.Errorf("Add(%q): %v", k, err)
						return
					}
					if ok, err := f.Contains(context.Background(), k); err != nil || !ok {
						t.Errorf("Contains(%q) = %v, %v right after Add", k, ok, err)
						return
					}
				}
			}(w)
		}
		wg.Wait()

		for w := 0; w < workers; w++ {
			for i := 0; i < per; i++ {
				if k := key(fmt.Sprintf("w%d", w), i); !mustContains(t, f, k) {
					t.Fatalf("false negative after concurrent adds: %q", k)
				}
			}
		}
	})

	t.Run("SaveAndLoad", func(t *testing.T) {
		f := newFilter(t, 10_000, 0.01)
		m, ok := f.(encoding.BinaryMarshaler)
		if !ok {
			t.Skip("filter does not support saving")
		}
		for i := 0; i < 10_000; i++ {
			mustAdd(t, f, key("user", i))
		}
		data, err := m.MarshalBinary()
		if err != nil {
			t.Fatal(err)
		}

		g := newFilter(t, 10_000, 0.01)
		if err := g.(encoding.BinaryUnmarshaler).UnmarshalBinary(data); err != nil {
			t.Fatal(err)
		}
		for i := 0; i < 10_000; i++ {
			if !mustContains(t, g, key("user", i)) {
				t.Fatalf("%q missing after save and load", key("user", i))
			}
		}
		for i := 0; i < 1000; i++ {
			if got, want := mustContains(t, g, key("absent", i)), mustContains(t, f, key("absent", i)); got != want {
				t.Fatalf("loaded filter disagrees with the original on %q", key("absent", i))
			}
		}
	})
}

func key(prefix string, i int) []byte {
	return []byte(fmt.Sprintf("%s-%d", prefix, i))
}

func mustAdd(t *testing.T, f probabloom.Filter, data []byte) {
	t.Helper()
	if err := f.Add(context.Background(), data); err != nil {
		t.Fatalf("Add(%q): %v", data, err)
	}
}

func mustContains(t *testing.T, f probabloom.Filter, data []byte) bool {
	t.Helper()
	ok, err := f.Contains(context.Background(), data)
	if err != nil {
		t.Fatalf("Contains(%q): %v", data, err)
	}
	return ok
}
