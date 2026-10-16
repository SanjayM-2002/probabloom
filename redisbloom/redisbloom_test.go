package redisbloom

import (
	"context"
	"fmt"
	"math/rand/v2"
	"os"
	"strconv"
	"testing"

	"github.com/redis/go-redis/v9"

	"github.com/SanjayM-2002/probabloom"
	"github.com/SanjayM-2002/probabloom/internal/filtertest"
	"github.com/SanjayM-2002/probabloom/registry"
)

// Needs REDIS_ADDR. Uses database REDIS_DB (default 15) and deletes its keys.
func testConfig(t *testing.T, n uint64, p float64) registry.Config {
	addr := os.Getenv("REDIS_ADDR")
	if addr == "" {
		t.Skip("REDIS_ADDR not set")
	}
	db := 15
	if s := os.Getenv("REDIS_DB"); s != "" {
		db, _ = strconv.Atoi(s)
	}
	key := fmt.Sprintf("probabloom:test:%x", rand.Uint64())
	t.Cleanup(func() {
		rdb := redis.NewClient(&redis.Options{Addr: addr, DB: db})
		rdb.Del(context.Background(), key)
		rdb.Close()
	})
	return registry.Config{Backend: "redisbloom", N: n, P: p, RedisAddr: addr, RedisDB: db, RedisKey: key}
}

func openTest(t *testing.T, cfg registry.Config) probabloom.Filter {
	f, err := registry.New(context.Background(), cfg)
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { f.(*Filter).Close() })
	return f
}

func TestConformance(t *testing.T) {
	// every operation is a network round trip, so use fewer items
	filtertest.RunWith(t, func(t *testing.T, n uint64, p float64) probabloom.Filter {
		return openTest(t, testConfig(t, n, p))
	}, filtertest.Options{N: 5_000, Rates: []float64{0.1, 0.01}})
}

func TestReopenKeepsItems(t *testing.T) {
	ctx := context.Background()
	cfg := testConfig(t, 1000, 0.01)

	openTest(t, cfg).Add(ctx, []byte("robin"))
	if ok, _ := openTest(t, cfg).Contains(ctx, []byte("robin")); !ok {
		t.Fatal("robin missing after reopening the same key")
	}
}

func TestReopenWithDifferentCapacityFails(t *testing.T) {
	cfg := testConfig(t, 1000, 0.01)
	openTest(t, cfg)
	cfg.N = 2000
	if _, err := registry.New(context.Background(), cfg); err == nil {
		t.Fatal("expected error for a capacity mismatch")
	}
}

func TestRejectsStrategy(t *testing.T) {
	cfg := testConfig(t, 1000, 0.01)
	cfg.Strategy = "counting"
	if _, err := registry.New(context.Background(), cfg); err == nil {
		t.Fatal("expected error when a strategy is set")
	}
}

func TestInfo(t *testing.T) {
	ctx := context.Background()
	f := openTest(t, testConfig(t, 100, 0.01))
	for i := 0; i < 500; i++ {
		f.Add(ctx, []byte(fmt.Sprintf("user-%d", i)))
	}
	info, err := registry.Describe(ctx, f)
	if err != nil {
		t.Fatal(err)
	}
	if info.Layers < 2 || info.SizeBytes == 0 {
		t.Errorf("after 5x capacity got %+v, expected Redis to have grown", info)
	}
}
