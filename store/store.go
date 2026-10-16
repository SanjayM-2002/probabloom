package store

import "context"

type BitStore interface {
	Set(ctx context.Context, positions []uint64) error
	Test(ctx context.Context, positions []uint64) (bool, error)
	SizeBytes() uint64
}

// Factory creates an m-bit store. Strategies that pick their own size
// (partitioned, blocked) or grow (scalable) need this instead of a store.
type Factory func(m uint64) BitStore
