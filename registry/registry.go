// Separate package from probabloom because it imports every strategy/store,
// which would otherwise be an import cycle.
package registry

import (
	"context"
	"fmt"
	"slices"

	"github.com/SanjayM-2002/probabloom"
	"github.com/SanjayM-2002/probabloom/internal/hash"
	"github.com/SanjayM-2002/probabloom/internal/sizing"
	"github.com/SanjayM-2002/probabloom/store/memory"
	"github.com/SanjayM-2002/probabloom/strategy/standard"
)

var (
	backends   = []string{"memory"}
	strategies = []string{"standard"}
)

func Backends() []string { return slices.Clone(backends) }

func Strategies() []string { return slices.Clone(strategies) }

func Hashes() []string { return hash.Names() }

type Config struct {
	Backend  string
	Strategy string
	Hash     string
	N        uint64
	P        float64
}

type Info struct {
	M         uint64
	K         uint64
	SizeBytes uint64
}

func New(_ context.Context, cfg Config) (probabloom.Filter, Info, error) {
	params, err := sizing.Optimal(cfg.N, cfg.P)
	if err != nil {
		return nil, Info{}, err
	}
	hashFn, err := hash.Get(cfg.Hash)
	if err != nil {
		return nil, Info{}, err
	}

	switch cfg.Backend {
	case "memory":
		bits := memory.NewBits(params.M)
		info := Info{M: params.M, K: params.K, SizeBytes: bits.SizeBytes()}

		switch cfg.Strategy {
		case "standard":
			return standard.New(bits, hashFn, params), info, nil
		}
		return nil, Info{}, fmt.Errorf("unknown strategy %q (available: %v)", cfg.Strategy, strategies)
	}
	return nil, Info{}, fmt.Errorf("unknown backend %q (available: %v)", cfg.Backend, backends)
}
