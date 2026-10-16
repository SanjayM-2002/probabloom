package hash

import (
	"fmt"
	"testing"
)

func TestGet(t *testing.T) {
	for _, name := range Names() {
		if _, err := Get(name); err != nil {
			t.Errorf("Get(%q): %v", name, err)
		}
	}
	if _, err := Get("nope"); err == nil {
		t.Fatal("Get(nope): expected error")
	}
}

func TestSeedsDiffer(t *testing.T) {
	data := []byte("robin")
	for _, name := range Names() {
		h, _ := Get(name)
		seen := map[uint64]uint64{}
		for seed := uint64(0); seed < 64; seed++ {
			v := h(seed, data)
			if prev, ok := seen[v]; ok {
				t.Fatalf("%s: seeds %d and %d gave the same hash %#x", name, prev, seed, v)
			}
			seen[v] = seed
		}
	}
}

// inputs that hit every code path: empty, tails of each size, one and many full blocks
func testInputs() [][]byte {
	var out [][]byte
	for _, n := range []int{0, 1, 3, 4, 7, 8, 9, 15, 16, 17, 31, 32, 33, 63, 64, 100, 1000} {
		b := make([]byte, n)
		for i := range b {
			b[i] = byte(i*31 + 7)
		}
		out = append(out, b)
	}
	return out
}

func BenchmarkHashes(b *testing.B) {
	for _, name := range Names() {
		h, _ := Get(name)
		for _, size := range []int{8, 16, 32, 64, 256} {
			data := make([]byte, size)
			b.Run(fmt.Sprintf("%s/%dB", name, size), func(b *testing.B) {
				b.SetBytes(int64(size))
				for i := 0; i < b.N; i++ {
					h(uint64(i), data)
				}
			})
		}
	}
}
