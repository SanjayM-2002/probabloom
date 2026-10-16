// Values are not reduced mod m; each strategy maps them onto its own layout.
package probe

import (
	"fmt"
	"sort"

	"github.com/SanjayM-2002/probabloom/internal/hash"
)

type Func func(data []byte, out []uint64)

type Mode func(h hash.Func) Func

var modes = map[string]Mode{
	"seeded":   Seeded,
	"double":   Double,
	"enhanced": Enhanced,
}

// v_i = h(seed=i, data)
func Seeded(h hash.Func) Func {
	return func(data []byte, out []uint64) {
		for i := range out {
			out[i] = h(uint64(i), data)
		}
	}
}

// Kirsch-Mitzenmacher: v_i = h1 + i*h2. h2 is forced odd so it's never 0,
// which would put every probe on h1.
func Double(h hash.Func) Func {
	return func(data []byte, out []uint64) {
		h1 := h(0, data)
		h2 := h(1, data) | 1
		for i := range out {
			out[i] = h1 + uint64(i)*h2
		}
	}
}

// Dillinger-Manolios: v_i = h1 + i*h2 + (i^3-i)/6. The cubic term stops the
// probes from forming a straight line, which double hashing can do when h2
// shares a factor with m.
func Enhanced(h hash.Func) Func {
	return func(data []byte, out []uint64) {
		x := h(0, data)
		y := h(1, data)
		for i := range out {
			out[i] = x
			x += y
			y += uint64(i + 1)
		}
	}
}

func Get(name string) (Mode, error) {
	m, ok := modes[name]
	if !ok {
		return nil, fmt.Errorf("unknown positions mode %q (available: %v)", name, Names())
	}
	return m, nil
}

func Names() []string {
	names := make([]string, 0, len(modes))
	for n := range modes {
		names = append(names, n)
	}
	sort.Strings(names)
	return names
}
