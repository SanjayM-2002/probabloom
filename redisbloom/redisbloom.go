// Adapter over Redis's built-in bloom filter (BF.* commands, Redis 8+ or
// Redis Stack). Redis picks the strategy (scalable) and hash (MurmurHash2),
// so this backend takes no strategy, hash or positions setting.
//
// Importing this package registers the "redisbloom" backend.
package redisbloom

import (
	"context"
	"fmt"
	"strings"

	"github.com/redis/go-redis/v9"

	"github.com/SanjayM-2002/probabloom"
	"github.com/SanjayM-2002/probabloom/registry"
)

func init() {
	registry.RegisterBackend("redisbloom", open)
	// errors are returned to the caller; don't also print go-redis's internal retry logs
	redis.SetLogger(quiet{})
}

type quiet struct{}

func (quiet) Printf(context.Context, string, ...any) {}

type Filter struct {
	rdb *redis.Client
	key string
}

var _ probabloom.Filter = (*Filter)(nil)

func open(ctx context.Context, cfg registry.Config) (probabloom.Filter, error) {
	if cfg.Strategy != "" || cfg.Hash != "" || cfg.Positions != "" {
		return nil, fmt.Errorf("redisbloom uses Redis's own strategy and hash; strategy, hash and positions can't be set")
	}
	if cfg.RedisKey == "" {
		return nil, fmt.Errorf("redisbloom needs a key")
	}
	rdb := redis.NewClient(&redis.Options{Addr: cfg.RedisAddr, DB: cfg.RedisDB, DialerRetries: 1})
	if err := rdb.Ping(ctx).Err(); err != nil {
		rdb.Close()
		return nil, fmt.Errorf("redisbloom: can't reach Redis at %s: %w", cfg.RedisAddr, err)
	}
	f, err := New(ctx, rdb, cfg.RedisKey, cfg.N, cfg.P)
	if err != nil {
		rdb.Close()
		return nil, err
	}
	return f, nil
}

// New reserves the filter, or reuses it if the key already exists with the
// same capacity. Redis doesn't report a filter's error rate, so only the
// capacity can be checked.
func New(ctx context.Context, rdb *redis.Client, key string, n uint64, p float64) (*Filter, error) {
	err := rdb.BFReserve(ctx, key, p, int64(n)).Err()
	if err != nil && !strings.Contains(err.Error(), "item exists") {
		return nil, fmt.Errorf("redisbloom: BF.RESERVE %s: %w", key, err)
	}
	if err != nil {
		info, err := rdb.BFInfo(ctx, key).Result()
		if err != nil {
			return nil, fmt.Errorf("redisbloom: BF.INFO %s: %w", key, err)
		}
		if uint64(info.Capacity) != n {
			return nil, fmt.Errorf("redisbloom: key %s already exists with capacity %d, not %d", key, info.Capacity, n)
		}
	}
	return &Filter{rdb: rdb, key: key}, nil
}

func (f *Filter) Add(ctx context.Context, data []byte) error {
	return f.rdb.BFAdd(ctx, f.key, data).Err()
}

func (f *Filter) Contains(ctx context.Context, data []byte) (bool, error) {
	return f.rdb.BFExists(ctx, f.key, data).Result()
}

func (f *Filter) Info(ctx context.Context) (registry.Info, error) {
	info, err := f.rdb.BFInfo(ctx, f.key).Result()
	if err != nil {
		return registry.Info{}, err
	}
	return registry.Info{SizeBytes: uint64(info.Size), Layers: int(info.Filters)}, nil
}

func (f *Filter) Close() error { return f.rdb.Close() }
