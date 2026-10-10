package store

import "context"

type BitStore interface {
	Set(ctx context.Context, positions []uint64) error
	Test(ctx context.Context, positions []uint64) (bool, error)
}
