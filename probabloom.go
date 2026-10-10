package probabloom

import "context"

// ctx/error are there so redis-backed filters fit the same interface;
// in-memory filters ignore ctx and never error.
type Filter interface {
	Add(ctx context.Context, data []byte) error
	Contains(ctx context.Context, data []byte) (bool, error)
}

type Remover interface {
	Filter
	Remove(ctx context.Context, data []byte) error
}
