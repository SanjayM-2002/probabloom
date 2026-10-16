package memory

import (
	"context"
	"sync"
	"testing"
)

func TestBitsSetAndTest(t *testing.T) {
	ctx := context.Background()
	b := NewBits(130)

	positions := []uint64{0, 63, 64, 129}
	if ok, _ := b.Test(ctx, positions); ok {
		t.Fatal("fresh bit array reported bits set")
	}

	if err := b.Set(ctx, positions); err != nil {
		t.Fatal(err)
	}
	for _, p := range positions {
		if ok, _ := b.Test(ctx, []uint64{p}); !ok {
			t.Errorf("bit %d not set", p)
		}
	}

	for _, p := range []uint64{1, 62, 65, 128} {
		if ok, _ := b.Test(ctx, []uint64{p}); ok {
			t.Errorf("bit %d unexpectedly set", p)
		}
	}

	if ok, _ := b.Test(ctx, []uint64{0, 1}); ok {
		t.Error("Test returned true with one bit unset")
	}
}

func TestBitsSize(t *testing.T) {
	b := NewBits(130)
	if b.Len() != 130 {
		t.Errorf("Len = %d, want 130", b.Len())
	}
	if b.SizeBytes() != 24 {
		t.Errorf("SizeBytes = %d, want 24", b.SizeBytes())
	}
}

func TestBitsMarshalRoundTrip(t *testing.T) {
	ctx := context.Background()
	a := NewBits(200)
	a.Set(ctx, []uint64{0, 64, 199})

	data, err := a.MarshalBinary()
	if err != nil {
		t.Fatal(err)
	}
	b := NewBits(200)
	if err := b.UnmarshalBinary(data); err != nil {
		t.Fatal(err)
	}
	if ok, _ := b.Test(ctx, []uint64{0, 64, 199}); !ok {
		t.Error("bits lost in round trip")
	}
	if ok, _ := b.Test(ctx, []uint64{1}); ok {
		t.Error("unexpected bit after round trip")
	}
	if err := NewBits(300).UnmarshalBinary(data); err == nil {
		t.Error("expected error loading into a different size")
	}
}

func TestBitsConcurrent(t *testing.T) {
	ctx := context.Background()
	b := NewBits(4096)
	var wg sync.WaitGroup
	for g := 0; g < 8; g++ {
		wg.Add(1)
		go func(g int) {
			defer wg.Done()
			for i := 0; i < 4096; i += 8 {
				p := uint64(i + g)
				b.Set(ctx, []uint64{p})
				if ok, _ := b.Test(ctx, []uint64{p}); !ok {
					t.Errorf("bit %d not visible after Set", p)
				}
			}
		}(g)
	}
	wg.Wait()
	for p := uint64(0); p < 4096; p++ {
		if ok, _ := b.Test(ctx, []uint64{p}); !ok {
			t.Fatalf("bit %d lost under concurrent writes", p)
		}
	}
}
