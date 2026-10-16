package hash

import (
	stdfnv "hash/fnv"
	"testing"
)

func TestFNV1aMatchesStdlib(t *testing.T) {
	inputs := []string{"", "a", "foobar", "robin", "the quick brown fox"}
	for _, in := range inputs {
		ref := stdfnv.New64a()
		ref.Write([]byte(in))
		want := ref.Sum64()

		if got := FNV1a(0, []byte(in)); got != want {
			t.Errorf("FNV1a(0, %q) = %#x, want %#x", in, got, want)
		}
	}
}

func TestFNV1aSeedsDiffer(t *testing.T) {
	data := []byte("robin")
	seen := map[uint64]uint64{}
	for seed := uint64(0); seed < 16; seed++ {
		h := FNV1a(seed, data)
		if prev, ok := seen[h]; ok {
			t.Fatalf("seeds %d and %d produced the same hash %#x", prev, seed, h)
		}
		seen[h] = seed
	}
}
