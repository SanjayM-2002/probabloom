package registry

import (
	"context"
	"fmt"
	"testing"
)

// go test ./registry -run=^$ -bench=. -benchmem
func BenchmarkMatrix(b *testing.B) {
	ctx := context.Background()
	keys := make([][]byte, 4096)
	for i := range keys {
		keys[i] = []byte(fmt.Sprintf("user_%06d", i))
	}

	for _, strategy := range Strategies() {
		for _, h := range Hashes() {
			for _, pos := range Positions() {
				f, err := New(ctx, memoryConfig(strategy, h, pos, 1_000_000, 0.01))
				if err != nil {
					b.Fatal(err)
				}
				for _, k := range keys[:len(keys)/2] {
					f.Add(ctx, k)
				}
				name := strategy + "/" + h + "/" + pos
				b.Run(name+"/add", func(b *testing.B) {
					for i := 0; i < b.N; i++ {
						f.Add(ctx, keys[i%len(keys)])
					}
				})
				b.Run(name+"/contains", func(b *testing.B) {
					for i := 0; i < b.N; i++ {
						f.Contains(ctx, keys[i%len(keys)])
					}
				})
			}
		}
	}
}
