package username

import (
	"context"
	"errors"
	"fmt"
	"sync"
	"sync/atomic"
	"testing"

	"github.com/SanjayM-2002/probabloom/registry"
)

func TestNormalize(t *testing.T) {
	same := []string{"robin", "Robin", "  ROBIN  ", "ｒｏｂｉｎ", "\trobin\n"}
	for _, in := range same {
		got, err := Normalize(in)
		if err != nil || got != "robin" {
			t.Errorf("Normalize(%q) = %q, %v; want robin", in, got, err)
		}
	}

	// NFKC also folds compatibility forms like ligatures and superscripts
	if got, _ := Normalize("ﬁnn²"); got != "finn2" {
		t.Errorf("Normalize(ﬁnn²) = %q, want finn2", got)
	}

	for _, bad := range []string{"", "   ", "robin hood", "rob\x00in"} {
		if _, err := Normalize(bad); !errors.Is(err, ErrInvalid) {
			t.Errorf("Normalize(%q): got %v, want ErrInvalid", bad, err)
		}
	}
}

func newChecker(t *testing.T, taken ...string) (*Checker, *MemoryDB) {
	f, err := registry.New(context.Background(), registry.Config{
		Backend: "memory", Strategy: "standard", Hash: "xxh64", Positions: "double", N: 10_000, P: 0.01,
	})
	if err != nil {
		t.Fatal(err)
	}
	db := NewMemoryDB()
	c := NewChecker(f, db)
	for _, name := range taken {
		if _, err := c.Claim(context.Background(), name); err != nil {
			t.Fatal(err)
		}
	}
	return c, db
}

func TestCheckSkipsDBWhenFilterSaysNo(t *testing.T) {
	ctx := context.Background()
	c, db := newChecker(t)
	// fill the filter to capacity so false positives actually show up
	for i := 0; i < 10_000; i++ {
		if _, err := c.Claim(ctx, fmt.Sprintf("user_%d", i)); err != nil {
			t.Fatal(err)
		}
	}

	for i := 0; i < 1000; i++ {
		res, err := c.Check(ctx, fmt.Sprintf("free_%d", i))
		if err != nil {
			t.Fatal(err)
		}
		if !res.Available {
			t.Fatalf("%s reported taken", res.Name)
		}
	}
	// p=0.01, so ~10 lookups expected
	if db.Lookups > 50 {
		t.Errorf("%d database lookups for 1000 free names; the filter isn't helping", db.Lookups)
	}
	t.Logf("1000 free names, %d database lookups", db.Lookups)
}

func TestCheckTakenNames(t *testing.T) {
	ctx := context.Background()
	c, _ := newChecker(t, "robin")

	res, err := c.Check(ctx, "  Robin ")
	if err != nil {
		t.Fatal(err)
	}
	if res.Available || !res.CheckedDB || res.Name != "robin" {
		t.Errorf("got %+v, want robin taken after a database check", res)
	}
}

func TestClaimTwiceFails(t *testing.T) {
	ctx := context.Background()
	c, _ := newChecker(t, "robin")
	if _, err := c.Claim(ctx, "ROBIN"); !errors.Is(err, ErrTaken) {
		t.Fatalf("got %v, want ErrTaken", err)
	}
}

func TestConcurrentClaimsOneWinner(t *testing.T) {
	ctx := context.Background()
	c, _ := newChecker(t)

	var wins atomic.Int32
	var wg sync.WaitGroup
	for i := 0; i < 50; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			if _, err := c.Claim(ctx, "robin"); err == nil {
				wins.Add(1)
			}
		}()
	}
	wg.Wait()
	if wins.Load() != 1 {
		t.Fatalf("%d goroutines claimed robin, want exactly 1", wins.Load())
	}
}
