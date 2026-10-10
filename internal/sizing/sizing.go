package sizing

import (
	"fmt"
	"math"
)

type Params struct {
	M uint64
	K uint64
}

// m = ceil(-n*ln(p) / ln(2)^2), k = round(m/n * ln(2))
func Optimal(n uint64, p float64) (Params, error) {
	if n == 0 {
		return Params{}, fmt.Errorf("n must be > 0")
	}
	if p <= 0 || p >= 1 {
		return Params{}, fmt.Errorf("p must be in (0, 1), got %v", p)
	}

	m := math.Ceil(-float64(n) * math.Log(p) / (math.Ln2 * math.Ln2))
	k := math.Round(m / float64(n) * math.Ln2)
	if k < 1 {
		k = 1
	}
	return Params{M: uint64(m), K: uint64(k)}, nil
}

func EstimatedFPRate(p Params, n uint64) float64 {
	return math.Pow(1-math.Exp(-float64(p.K)*float64(n)/float64(p.M)), float64(p.K))
}
