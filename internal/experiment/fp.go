package experiment

import (
	"context"
	"fmt"
	"time"

	"github.com/SanjayM-2002/probabloom"
)

type Result struct {
	Added          int
	Queried        int
	FalsePositives int
	AddTime        time.Duration
	ContainsTime   time.Duration
}

func (r Result) Rate() float64 { return float64(r.FalsePositives) / float64(r.Queried) }

func (r Result) AddNs() float64 { return float64(r.AddTime.Nanoseconds()) / float64(r.Added) }

func (r Result) ContainsNs() float64 {
	return float64(r.ContainsTime.Nanoseconds()) / float64(r.Queried)
}

func Keys(prefix string, n int) [][]byte {
	keys := make([][]byte, n)
	for i := range keys {
		keys[i] = []byte(fmt.Sprintf("%s-%d", prefix, i))
	}
	return keys
}

// Run adds every key in added, then queries every key in absent (which must
// not overlap with added). Every "maybe" for absent is a false positive.
func Run(ctx context.Context, f probabloom.Filter, added, absent [][]byte) (Result, error) {
	res := Result{Added: len(added), Queried: len(absent)}

	start := time.Now()
	for _, k := range added {
		if err := f.Add(ctx, k); err != nil {
			return Result{}, err
		}
	}
	res.AddTime = time.Since(start)

	start = time.Now()
	for _, k := range absent {
		ok, err := f.Contains(ctx, k)
		if err != nil {
			return Result{}, err
		}
		if ok {
			res.FalsePositives++
		}
	}
	res.ContainsTime = time.Since(start)
	return res, nil
}
