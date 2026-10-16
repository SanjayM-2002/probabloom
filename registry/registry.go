// Separate package from probabloom because it imports every strategy/store,
// which would otherwise be an import cycle.
package registry

import (
	"context"
	"fmt"
	"slices"
	"sort"
	"sync"

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

var strategies = []string{"standard", "counting", "partitioned", "blocked", "scalable"}

type Config struct {
	Backend   string  `json:"backend"`
	Strategy  string  `json:"strategy,omitempty"`
	Hash      string  `json:"hash,omitempty"`
	Positions string  `json:"positions,omitempty"`
	N         uint64  `json:"n"`
	P         float64 `json:"p"`

	RedisAddr string `json:"-"`
	RedisDB   int    `json:"-"`
	RedisKey  string `json:"-"`
}

type Info struct {
	M         uint64 // 0 when the backend doesn't expose it
	K         uint64
	SizeBytes uint64
	Layers    int
}

// BackendFunc opens a filter on a backend. Backends other than memory
// register themselves from an init function, so a binary only contains the
// backends it imports.
type BackendFunc func(ctx context.Context, cfg Config) (probabloom.Filter, error)

var (
	mu       sync.RWMutex
	backends = map[string]BackendFunc{"memory": newMemory}
)

func RegisterBackend(name string, fn BackendFunc) {
	mu.Lock()
	defer mu.Unlock()
	if _, dup := backends[name]; dup {
		panic("registry: backend registered twice: " + name)
	}
	backends[name] = fn
}

func Backends() []string {
	mu.RLock()
	defer mu.RUnlock()
	names := make([]string, 0, len(backends))
	for n := range backends {
		names = append(names, n)
	}
	sort.Strings(names)
	return names
}

func Strategies() []string { return slices.Clone(strategies) }

func Hashes() []string { return hash.Names() }

func Positions() []string { return probe.Names() }

func New(ctx context.Context, cfg Config) (probabloom.Filter, error) {
	mu.RLock()
	open, ok := backends[cfg.Backend]
	mu.RUnlock()
	if !ok {
		return nil, fmt.Errorf("unknown backend %q (available: %v)", cfg.Backend, Backends())
	}
	return open(ctx, cfg)
}

func Describe(ctx context.Context, f probabloom.Filter) (Info, error) {
	if d, ok := f.(interface {
		Info(context.Context) (Info, error)
	}); ok {
		return d.Info(ctx)
	}
	var info Info
	if p, ok := f.(interface{ Params() sizing.Params }); ok {
		info.M, info.K = p.Params().M, p.Params().K
	}
	if s, ok := f.(interface{ SizeBytes() uint64 }); ok {
		info.SizeBytes = s.SizeBytes()
	}
	info.Layers = 1
	if l, ok := f.(interface{ Layers() int }); ok {
		info.Layers = l.Layers()
	}
	return info, nil
}

func newMemory(_ context.Context, cfg Config) (probabloom.Filter, error) {
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
