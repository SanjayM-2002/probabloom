package hash

import (
	"fmt"
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

func TestGet(t *testing.T) {
	if _, err := Get("fnv1a"); err != nil {
		t.Fatalf("Get(fnv1a): %v", err)
	}
	if _, err := Get("nope"); err == nil {
		t.Fatal("Get(nope): expected error")
	}
}

func BenchmarkFNV1a(b *testing.B) {
	for _, size := range []int{8, 16, 32, 64} {
		data := make([]byte, size)
		b.Run(fmt.Sprintf("%dB", size), func(b *testing.B) {
			b.SetBytes(int64(size))
			for i := 0; i < b.N; i++ {
				FNV1a(uint64(i), data)
			}
		})
	}
}
