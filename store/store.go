package store

import (
	"context"
	"encoding"
	"errors"
)

type BitStore interface {
	Set(ctx context.Context, positions []uint64) error
	Test(ctx context.Context, positions []uint64) (bool, error)
	SizeBytes() uint64
}

// Factory creates an m-bit store. Strategies that pick their own size
// (partitioned, blocked) or grow (scalable) need this instead of a store.
type Factory func(m uint64) BitStore

var ErrNotPersistable = errors.New("store does not support saving")

func Marshal(b BitStore) ([]byte, error) {
	m, ok := b.(encoding.BinaryMarshaler)
	if !ok {
		return nil, ErrNotPersistable
	}
	return m.MarshalBinary()
}

func Unmarshal(b BitStore, data []byte) error {
	u, ok := b.(encoding.BinaryUnmarshaler)
	if !ok {
		return ErrNotPersistable
	}
	return u.UnmarshalBinary(data)
}
