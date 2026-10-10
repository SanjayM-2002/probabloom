package hash

import (
	"fmt"
	"sort"
)

type Func func(seed uint64, data []byte) uint64

var funcs = map[string]Func{
	"fnv1a": FNV1a,
}

func Get(name string) (Func, error) {
	f, ok := funcs[name]
	if !ok {
		return nil, fmt.Errorf("unknown hash %q (available: %v)", name, Names())
	}
	return f, nil
}

func Names() []string {
	names := make([]string, 0, len(funcs))
	for n := range funcs {
		names = append(names, n)
	}
	sort.Strings(names)
	return names
}
