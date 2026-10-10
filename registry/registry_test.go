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
				t.Run(backend+"/"+strategy+"/"+h, func(t *testing.T) {
					filtertest.Run(t, func(t *testing.T, n uint64, p float64) probabloom.Filter {
						f, _, err := New(context.Background(), Config{
							Backend: backend, Strategy: strategy, Hash: h, N: n, P: p,
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

func TestNewRejectsUnknown(t *testing.T) {
	valid := Config{Backend: "memory", Strategy: "standard", Hash: "fnv1a", N: 100, P: 0.01}

	cases := map[string]func(c *Config){
		"backend":  func(c *Config) { c.Backend = "nope" },
		"strategy": func(c *Config) { c.Strategy = "nope" },
		"hash":     func(c *Config) { c.Hash = "nope" },
		"n":        func(c *Config) { c.N = 0 },
		"p":        func(c *Config) { c.P = 2 },
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
