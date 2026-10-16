package registry

import (
	"context"
	"os"
	"path/filepath"
	"testing"

	"github.com/SanjayM-2002/probabloom"
	"github.com/SanjayM-2002/probabloom/internal/filtertest"
)

func memoryConfig(strategy, h, pos string, n uint64, p float64) Config {
	return Config{Backend: "memory", Strategy: strategy, Hash: h, Positions: pos, N: n, P: p}
}

func TestMatrix(t *testing.T) {
	for _, strategy := range Strategies() {
		for _, h := range Hashes() {
			for _, pos := range Positions() {
				t.Run(strategy+"/"+h+"/"+pos, func(t *testing.T) {
					t.Parallel()
					filtertest.Run(t, func(t *testing.T, n uint64, p float64) probabloom.Filter {
						f, err := New(context.Background(), memoryConfig(strategy, h, pos, n, p))
						if err != nil {
							t.Fatal(err)
						}
						return f
					})
				})
			}
		}
	}
}

func TestNewRejectsUnknown(t *testing.T) {
	valid := memoryConfig("standard", "fnv1a", "seeded", 100, 0.01)
	if _, err := New(context.Background(), valid); err != nil {
		t.Fatalf("valid config: %v", err)
	}

	cases := map[string]func(c *Config){
		"backend":   func(c *Config) { c.Backend = "nope" },
		"strategy":  func(c *Config) { c.Strategy = "nope" },
		"hash":      func(c *Config) { c.Hash = "nope" },
		"positions": func(c *Config) { c.Positions = "nope" },
		"n":         func(c *Config) { c.N = 0 },
		"p":         func(c *Config) { c.P = 2 },
	}
	for name, mutate := range cases {
		t.Run(name, func(t *testing.T) {
			c := valid
			mutate(&c)
			if _, err := New(context.Background(), c); err == nil {
				t.Fatal("expected error")
			}
		})
	}
}

func TestSaveAndLoadFile(t *testing.T) {
	ctx := context.Background()
	path := filepath.Join(t.TempDir(), "users.bloom")

	for _, strategy := range Strategies() {
		t.Run(strategy, func(t *testing.T) {
			cfg := memoryConfig(strategy, "xxh64", "double", 1000, 0.01)
			f, err := New(ctx, cfg)
			if err != nil {
				t.Fatal(err)
			}
			// past capacity, so scalable has more than one layer to save
			for _, name := range []string{"robin", "alice", "bob"} {
				f.Add(ctx, []byte(name))
			}
			for i := 0; i < 3000; i++ {
				f.Add(ctx, []byte{byte(i), byte(i >> 8), 'x'})
			}
			if err := SaveFile(path, cfg, f); err != nil {
				t.Fatal(err)
			}

			g, got, err := LoadFile(ctx, path)
			if err != nil {
				t.Fatal(err)
			}
			if got != cfg {
				t.Errorf("loaded config %+v, want %+v", got, cfg)
			}
			for _, name := range []string{"robin", "alice", "bob"} {
				if ok, _ := g.Contains(ctx, []byte(name)); !ok {
					t.Errorf("%s missing after reload", name)
				}
			}
			before, _ := Describe(ctx, f)
			after, _ := Describe(ctx, g)
			if before != after {
				t.Errorf("shape changed: before %+v, after %+v", before, after)
			}
		})
	}
}

func TestLoadRejectsGarbage(t *testing.T) {
	path := filepath.Join(t.TempDir(), "bad.bloom")
	if err := os.WriteFile(path, []byte("hello world"), 0o644); err != nil {
		t.Fatal(err)
	}
	if _, _, err := LoadFile(context.Background(), path); err == nil {
		t.Fatal("expected error")
	}
}
