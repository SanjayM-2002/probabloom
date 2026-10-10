package filtertest

import (
	"context"
	"fmt"
	"testing"

	"github.com/SanjayM-2002/probabloom"
)

type NewFunc func(t *testing.T, n uint64, p float64) probabloom.Filter

func Run(t *testing.T, newFilter NewFunc) {
	t.Run("EmptyContainsNothing", func(t *testing.T) {
		f := newFilter(t, 1000, 0.01)
		for i := 0; i < 1000; i++ {
			if mustContains(t, f, key("absent", i)) {
				t.Fatalf("empty filter reported %q as present", key("absent", i))
			}
		}
	})

	t.Run("NoFalseNegatives", func(t *testing.T) {
		const n = 50_000
		f := newFilter(t, n, 0.01)
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
		for _, p := range []float64{0.1, 0.01, 0.001} {
			t.Run(fmt.Sprintf("p=%v", p), func(t *testing.T) {
				const n = 100_000
				f := newFilter(t, n, p)
				for i := 0; i < n; i++ {
					mustAdd(t, f, key("user", i))
				}
				fp := 0
				for i := 0; i < n; i++ {
					if mustContains(t, f, key("absent", i)) {
						fp++
					}
				}
				got := float64(fp) / n
				t.Logf("target %.4f, measured %.4f", p, got)
				// 50% slack for sampling noise
				if got > p*1.5 {
					t.Errorf("false-positive rate %.4f exceeds target %.4f by more than 50%%", got, p)
				}
			})
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
