# probabloom

Bloom filters in Go, built from first principles, for checking whether a
username is taken without hitting the database every time.

## Combinations

| Backend | Strategy | Hash | Positions |
|---|---|---|---|
| `memory` | `standard`, `counting`, `partitioned`, `blocked`, `scalable` | `fnv1a`, `xxh64`, `murmur3` | `seeded`, `double`, `enhanced` |
| `redisbloom` | Redis's own (scalable) | MurmurHash2 | — |

That's 45 memory combinations plus Redis.

| Strategy | |
|---|---|
| standard | k bits anywhere in one array |
| counting | 4-bit counters, supports `remove` |
| partitioned | k slices, one bit per slice |
| blocked | all k bits in one 64-byte block (one cache line) |
| scalable | grows when full, for an unknown n |

## Usage

```
go run ./cmd/probabloom [flags] <command> [names]
```

| Flag | Default |
|---|---|
| `-backend` | `memory` |
| `-strategy` / `-hash` / `-positions` | `standard` / `xxh64` / `double` (memory only) |
| `-n` / `-p` | `1000000` / `0.01` |
| `-load FILE` | names to add first |
| `-file FILE` | memory only: keep the filter between runs |
| `-redis-addr` / `-redis-db` / `-key` | `localhost:6379` / `0` / `probabloom:usernames` |

| Command | |
|---|---|
| `info` | size and shape |
| `add` / `check` / `remove NAME...` | `remove` is counting only |
| `available NAME...` | filter first, then the database (the `-load` file) |
| `repl` | interactive add / check / remove |
| `fp` | measure the real false-positive rate and speed |
| `bench` | `fp` for all 45 memory combinations |

Names are normalized (trimmed, NFKC, lowercased): `Robin` == ` robin `.

```bash
go run ./cmd/probabloom -load testdata/usernames.txt available robin zara
go run ./cmd/probabloom -strategy=counting -file users.bloom add robin
go run ./cmd/probabloom -backend=redisbloom check robin
go run ./cmd/probabloom bench
```

## Notes

- Use `xxh64` or `murmur3`. `fnv1a` with `double`/`enhanced` isn't uniform
  on username-like keys.
- `blocked` sizes itself with its own formula and uses 2–8% more memory.
- Redis needs Redis 8 (or Redis Stack); `make redis-up` starts one.
  `probabloom-lite` is the CLI without Redis compiled in.

## Make

`test`, `test-race`, `test-redis`, `bench`, `bench-cli`, `fp`, `build`, `depcheck`
