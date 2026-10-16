package hash

import (
	"testing"

	"github.com/cespare/xxhash/v2"
)

func TestXXH64MatchesReference(t *testing.T) {
	for _, seed := range []uint64{0, 1, 42, 1 << 40} {
		for _, in := range testInputs() {
			d := xxhash.NewWithSeed(seed)
			d.Write(in)
			if got, want := XXH64(seed, in), d.Sum64(); got != want {
				t.Errorf("XXH64(seed=%d, len=%d) = %#x, want %#x", seed, len(in), got, want)
			}
		}
	}
}
