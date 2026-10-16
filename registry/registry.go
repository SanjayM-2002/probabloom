// Separate package from probabloom because it imports every strategy/store,
// which would otherwise be an import cycle.
package registry

import (
	"context"
	"fmt"
	"slices"

	"github.com/SanjayM-2002/probabloom"
	"github.com/SanjayM-2002/probabloom/internal/hash"
	"github.com/SanjayM-2002/probabloom/internal/probe"
	"github.com/SanjayM-2002/probabloom/internal/sizing"
	"github.com/SanjayM-2002/probabloom/store/memory"
	"github.com/SanjayM-2002/probabloom/strategy/blocked"
	"github.com/SanjayM-2002/probabloom/strategy/counting"
	"github.com/SanjayM-2002/probabloom/strategy/partitioned"
	"github.com/SanjayM-2002/probabloom/strategy/scalable"
	"github.com/SanjayM-2002/probabloom/strategy/standard"
)

var (
	backends   = []string{"memory"}
	strategies = []string{"standard", "counting", "partitioned", "blocked", "scalable"}
)

func Backends() []string { return slices.Clone(backends) }

func Strategies() []string { return slices.Clone(strategies) }

func Hashes() []string { return hash.Names() }

func Positions() []string { return probe.Names() }

type Config struct {
	Backend   string
	Strategy  string
	Hash      string
	Positions string
	N         uint64
	P         float64
}

type Info struct {
	M         uint64
	K         uint64
	SizeBytes uint64
}

type filter interface {
	probabloom.Filter
	Params() sizing.Params
	SizeBytes() uint64
}

func New(_ context.Context, cfg Config) (probabloom.Filter, Info, error) {
	f, err := build(cfg)
	if err != nil {
		return nil, Info{}, err
	}
	p := f.Params()
	return f, Info{M: p.M, K: p.K, SizeBytes: f.SizeBytes()}, nil
}

func build(cfg Config) (filter, error) {
	params, err := sizing.Optimal(cfg.N, cfg.P)
	if err != nil {
		return nil, err
	}
	hashFn, err := hash.Get(cfg.Hash)
	if err != nil {
		return nil, err
	}
	mode, err := probe.Get(cfg.Positions)
	if err != nil {
		return nil, err
	}
	pr := mode(hashFn)

	if cfg.Backend != "memory" {
		return nil, fmt.Errorf("unknown backend %q (available: %v)", cfg.Backend, backends)
	}
	newBits := memory.Factory

	switch cfg.Strategy {
	case "standard":
		return standard.New(newBits, pr, params), nil
	case "counting":
		return counting.New(pr, params), nil
	case "partitioned":
		return partitioned.New(newBits, pr, params), nil
	case "blocked":
		f, err := blocked.New(newBits, pr, cfg.N, cfg.P)
		if err != nil {
			return nil, err
		}
		return f, nil
	case "scalable":
		f, err := scalable.New(newBits, pr, cfg.N, cfg.P)
		if err != nil {
			return nil, err
		}
		return f, nil
	}
	return nil, fmt.Errorf("unknown strategy %q (available: %v)", cfg.Strategy, strategies)
}
