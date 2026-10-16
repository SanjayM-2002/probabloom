package registry

import (
	"context"
	"testing"

	"github.com/SanjayM-2002/probabloom"
	"github.com/SanjayM-2002/probabloom/internal/filtertest"
)

func TestMatrix(t *testing.T) {
	for _, backend := range Backends() {
		for _, strategy := range Strategies() {
			for _, h := range Hashes() {
				for _, pos := range Positions() {
					t.Run(backend+"/"+strategy+"/"+h+"/"+pos, func(t *testing.T) {
						t.Parallel()
						filtertest.Run(t, func(t *testing.T, n uint64, p float64) probabloom.Filter {
							f, _, err := New(context.Background(), Config{
								Backend: backend, Strategy: strategy, Hash: h, Positions: pos, N: n, P: p,
							})
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
}

func TestNewRejectsUnknown(t *testing.T) {
	valid := Config{Backend: "memory", Strategy: "standard", Hash: "fnv1a", Positions: "seeded", N: 100, P: 0.01}
	if _, _, err := New(context.Background(), valid); err != nil {
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
			if _, _, err := New(context.Background(), c); err == nil {
				t.Fatal("expected error")
			}
		})
	}
}
