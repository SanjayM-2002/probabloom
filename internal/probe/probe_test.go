package probe

import (
	"slices"
	"testing"

	"github.com/SanjayM-2002/probabloom/internal/hash"
)

func TestModes(t *testing.T) {
	for _, name := range Names() {
		mode, err := Get(name)
		if err != nil {
			t.Fatal(err)
		}
		f := mode(hash.FNV1a)

		a := make([]uint64, 8)
		b := make([]uint64, 8)
		f([]byte("robin"), a)
		f([]byte("robin"), b)
		if !slices.Equal(a, b) {
			t.Errorf("%s: not deterministic", name)
		}

		seen := map[uint64]bool{}
		for _, v := range a {
			if seen[v] {
				t.Errorf("%s: repeated value %#x in %v", name, v, a)
			}
			seen[v] = true
		}
	}
	if _, err := Get("nope"); err == nil {
		t.Error("Get(nope): expected error")
	}
}

func TestDoubleUsesTwoHashCalls(t *testing.T) {
	calls := 0
	counting := func(seed uint64, data []byte) uint64 {
		calls++
		return hash.FNV1a(seed, data)
	}
	Double(counting)([]byte("robin"), make([]uint64, 7))
	if calls != 2 {
		t.Errorf("double made %d hash calls, want 2", calls)
	}
}
