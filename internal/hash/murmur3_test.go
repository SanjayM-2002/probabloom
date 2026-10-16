package hash

import (
	"testing"

	"github.com/spaolacci/murmur3"
)

func TestMurmur3MatchesReference(t *testing.T) {
	// the reference takes a 32-bit seed
	for _, seed := range []uint32{0, 1, 42, 1 << 31} {
		for _, in := range testInputs() {
			if got, want := Murmur3(uint64(seed), in), murmur3.Sum64WithSeed(in, seed); got != want {
				t.Errorf("Murmur3(seed=%d, len=%d) = %#x, want %#x", seed, len(in), got, want)
			}
		}
	}
}
