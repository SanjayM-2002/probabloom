// Scalable bloom filter (Almeida et al., 2007): a chain of partitioned
// filters. When the newest layer reaches its capacity a new one is added
// with growth× the capacity and tightening× the false-positive rate, so the
// overall rate stays below p however many items are added:
//
//	p0 = p*(1-tightening), layer i gets p0*tightening^i, and the sum is p.
package scalable

import (
	"context"
	"encoding/binary"
	"fmt"
	"math"
	"sync"

	"github.com/SanjayM-2002/probabloom"
	"github.com/SanjayM-2002/probabloom/internal/probe"
	"github.com/SanjayM-2002/probabloom/internal/sizing"
	"github.com/SanjayM-2002/probabloom/store"
	"github.com/SanjayM-2002/probabloom/strategy/partitioned"
)

const (
	growth     = 2
	tightening = 0.5
)

type layer struct {
	filter   *partitioned.Filter
	capacity uint64
	count    uint64
}

type Filter struct {
	mu      sync.RWMutex
	newBits store.Factory
	probe   probe.Func
	n0      uint64
	p0      float64
	layers  []*layer
}

var _ probabloom.Filter = (*Filter)(nil)

func New(newBits store.Factory, p probe.Func, n uint64, fp float64) (*Filter, error) {
	f := &Filter{newBits: newBits, probe: p, n0: n, p0: fp * (1 - tightening)}
	if err := f.grow(); err != nil {
		return nil, err
	}
	return f, nil
}

func (f *Filter) grow() error {
	i := len(f.layers)
	capacity := f.n0 * uint64(math.Pow(growth, float64(i)))
	params, err := sizing.Optimal(capacity, f.p0*math.Pow(tightening, float64(i)))
	if err != nil {
		return err
	}
	f.layers = append(f.layers, &layer{
		filter:   partitioned.New(f.newBits, f.probe, params),
		capacity: capacity,
	})
	return nil
}

func (f *Filter) Add(ctx context.Context, data []byte) error {
	f.mu.Lock()
	defer f.mu.Unlock()

	// skip items that are (probably) already in, so duplicates don't use up capacity
	if ok, err := f.contains(ctx, data); err != nil || ok {
		return err
	}
	last := f.layers[len(f.layers)-1]
	if last.count >= last.capacity {
		if err := f.grow(); err != nil {
			return err
		}
		last = f.layers[len(f.layers)-1]
	}
	if err := last.filter.Add(ctx, data); err != nil {
		return err
	}
	last.count++
	return nil
}

func (f *Filter) Contains(ctx context.Context, data []byte) (bool, error) {
	f.mu.RLock()
	defer f.mu.RUnlock()
	return f.contains(ctx, data)
}

func (f *Filter) contains(ctx context.Context, data []byte) (bool, error) {
	for _, l := range f.layers {
		if ok, err := l.filter.Contains(ctx, data); err != nil || ok {
			return ok, err
		}
	}
	return false, nil
}

func (f *Filter) Layers() int {
	f.mu.RLock()
	defer f.mu.RUnlock()
	return len(f.layers)
}

func (f *Filter) Params() sizing.Params {
	f.mu.RLock()
	defer f.mu.RUnlock()
	return f.layers[0].filter.Params()
}

func (f *Filter) SizeBytes() uint64 {
	f.mu.RLock()
	defer f.mu.RUnlock()
	var total uint64
	for _, l := range f.layers {
		total += l.filter.SizeBytes()
	}
	return total
}

// layout: layer count, then per layer: capacity, count, payload length, payload
func (f *Filter) MarshalBinary() ([]byte, error) {
	f.mu.RLock()
	defer f.mu.RUnlock()

	buf := binary.LittleEndian.AppendUint64(nil, uint64(len(f.layers)))
	for _, l := range f.layers {
		payload, err := l.filter.MarshalBinary()
		if err != nil {
			return nil, err
		}
		buf = binary.LittleEndian.AppendUint64(buf, l.capacity)
		buf = binary.LittleEndian.AppendUint64(buf, l.count)
		buf = binary.LittleEndian.AppendUint64(buf, uint64(len(payload)))
		buf = append(buf, payload...)
	}
	return buf, nil
}

func (f *Filter) UnmarshalBinary(data []byte) error {
	f.mu.Lock()
	defer f.mu.Unlock()

	next := func() (uint64, error) {
		if len(data) < 8 {
			return 0, fmt.Errorf("scalable: truncated data")
		}
		v := binary.LittleEndian.Uint64(data)
		data = data[8:]
		return v, nil
	}

	n, err := next()
	if err != nil {
		return err
	}
	for uint64(len(f.layers)) < n {
		if err := f.grow(); err != nil {
			return err
		}
	}
	for i := uint64(0); i < n; i++ {
		l := f.layers[i]
		capacity, err := next()
		if err != nil {
			return err
		}
		if capacity != l.capacity {
			return fmt.Errorf("scalable: layer %d saved with capacity %d, expected %d", i, capacity, l.capacity)
		}
		if l.count, err = next(); err != nil {
			return err
		}
		size, err := next()
		if err != nil {
			return err
		}
		if uint64(len(data)) < size {
			return fmt.Errorf("scalable: truncated data")
		}
		if err := l.filter.UnmarshalBinary(data[:size]); err != nil {
			return err
		}
		data = data[size:]
	}
	if len(data) != 0 {
		return fmt.Errorf("scalable: %d trailing bytes", len(data))
	}
	return nil
}
