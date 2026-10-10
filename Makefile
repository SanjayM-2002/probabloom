.PHONY: test test-v bench fp vet depcheck

test:
	go test ./...

test-v:
	go test ./... -v -run 'Conformance|Matrix' | grep -E 'target|^(ok|FAIL|---)'

bench:
	go test ./... -run=^$$ -bench=. -benchmem

fp:
	go run ./cmd/probabloom fp

vet:
	go vet ./...

depcheck:
	@! go list -deps ./strategy/... ./store/memory | grep -i redis && echo "ok: core has no redis dependency"
