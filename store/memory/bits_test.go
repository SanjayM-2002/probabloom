package memory

import (
	"context"
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
