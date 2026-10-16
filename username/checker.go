package username

import (
	"context"
	"errors"
	"sync"

	"github.com/SanjayM-2002/probabloom"
)

var ErrTaken = errors.New("username is taken")

// DB is the source of truth. The filter only decides whether a lookup is needed.
type DB interface {
	Exists(ctx context.Context, name string) (bool, error)
	// Insert must fail with ErrTaken if the name exists, atomically (a unique
	// constraint in a real database), so two people can't claim the same name.
	Insert(ctx context.Context, name string) error
}

type Result struct {
	Name      string
	Available bool
	// false means the filter answered on its own ("definitely not taken")
	CheckedDB bool
}

type Checker struct {
	filter probabloom.Filter
	db     DB
}

func NewChecker(f probabloom.Filter, db DB) *Checker {
	return &Checker{filter: f, db: db}
}

func (c *Checker) Check(ctx context.Context, raw string) (Result, error) {
	name, err := Normalize(raw)
	if err != nil {
		return Result{}, err
	}
	maybe, err := c.filter.Contains(ctx, []byte(name))
	if err != nil {
		return Result{}, err
	}
	if !maybe {
		return Result{Name: name, Available: true}, nil
	}
	exists, err := c.db.Exists(ctx, name)
	if err != nil {
		return Result{}, err
	}
	return Result{Name: name, Available: !exists, CheckedDB: true}, nil
}

// Claim writes to the database first: if the process dies before the filter
// is updated, the filter just misses a name the database still rejects.
func (c *Checker) Claim(ctx context.Context, raw string) (string, error) {
	name, err := Normalize(raw)
	if err != nil {
		return "", err
	}
	if err := c.db.Insert(ctx, name); err != nil {
		return "", err
	}
	return name, c.filter.Add(ctx, []byte(name))
}

// MemoryDB is a DB for tests and demos.
type MemoryDB struct {
	mu      sync.Mutex
	names   map[string]struct{}
	Lookups int
}

func NewMemoryDB(names ...string) *MemoryDB {
	db := &MemoryDB{names: map[string]struct{}{}}
	for _, n := range names {
		db.names[n] = struct{}{}
	}
	return db
}

func (db *MemoryDB) Exists(_ context.Context, name string) (bool, error) {
	db.mu.Lock()
	defer db.mu.Unlock()
	db.Lookups++
	_, ok := db.names[name]
	return ok, nil
}

func (db *MemoryDB) Insert(_ context.Context, name string) error {
	db.mu.Lock()
	defer db.mu.Unlock()
	if _, ok := db.names[name]; ok {
		return ErrTaken
	}
	db.names[name] = struct{}{}
	return nil
}
