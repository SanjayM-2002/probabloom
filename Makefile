REDIS_ADDR ?= localhost:6379

.PHONY: build test test-race test-redis bench bench-cli fp vet depcheck redis-up redis-down

build:
	go build -o bin/probabloom ./cmd/probabloom
	go build -o bin/probabloom-lite ./cmd/probabloom-lite

test:
	go test ./...

test-race:
	go test -race ./...

# redis tests use database 15 and delete their keys
test-redis:
	REDIS_ADDR=$(REDIS_ADDR) go test ./redisbloom/... -v -run . | grep -E 'target|^(--- |ok|FAIL)'

bench:
	go test ./registry -run=^$$ -bench=. -benchmem

bench-cli:
	go run ./cmd/probabloom bench

fp:
	go run ./cmd/probabloom fp

vet:
	go vet ./...

depcheck:
	@! go list -deps ./strategy/... ./store/... ./cmd/probabloom-lite | grep -i redis && echo "ok: core and lite binary have no redis dependency"

redis-up:
	docker compose up -d

redis-down:
	docker compose down
