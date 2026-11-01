.PHONY: run test vet smoke check icon

run:
	go run ./cmd/client

test:
	go test -race -tags ci ./...

vet:
	go vet -tags ci ./...

smoke:
	mkdir -p tmp
	go run -tags ci ./cmd/client --smoke --data-dir ./tmp/smoke --screenshot ./tmp/desktop.png

icon:
	go run ./tools/icon

check: test vet smoke
	go run ./tools/check
