package sizing

import (
	"math"
	"testing"
)

func TestOptimal1Percent(t *testing.T) {
	got, err := Optimal(1_000_000, 0.01)
	if err != nil {
		t.Fatal(err)
	}
	if got.M != 9_585_059 {
		t.Errorf("M = %d, want 9585059", got.M)
	}
	if got.K != 7 {
		t.Errorf("K = %d, want 7", got.K)
	}
	if fp := EstimatedFPRate(got, 1_000_000); math.Abs(fp-0.01) > 0.001 {
		t.Errorf("EstimatedFPRate = %v, want ~0.01", fp)
	}
}

func TestOptimalRejectsBadInput(t *testing.T) {
	cases := []struct {
		n uint64
		p float64
	}{{0, 0.01}, {100, 0}, {100, 1}, {100, -0.5}}
	for _, c := range cases {
		if _, err := Optimal(c.n, c.p); err == nil {
			t.Errorf("Optimal(%d, %v): expected error", c.n, c.p)
		}
	}
}
