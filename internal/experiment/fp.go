package experiment

import (
	"context"
	"fmt"

	"github.com/SanjayM-2002/probabloom"
)

type FPResult struct {
	Added          int
	Queried        int
	FalsePositives int
}

func (r FPResult) Rate() float64 {
	return float64(r.FalsePositives) / float64(r.Queried)
}

func MeasureFP(ctx context.Context, f probabloom.Filter, n int) (FPResult, error) {
	for i := 0; i < n; i++ {
		if err := f.Add(ctx, []byte(fmt.Sprintf("added-%d", i))); err != nil {
			return FPResult{}, err
		}
	}
	res := FPResult{Added: n, Queried: n}
	for i := 0; i < n; i++ {
		ok, err := f.Contains(ctx, []byte(fmt.Sprintf("absent-%d", i)))
		if err != nil {
			return FPResult{}, err
		}
		if ok {
			res.FalsePositives++
		}
	}
	return res, nil
}
