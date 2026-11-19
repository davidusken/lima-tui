.PHONY: build run install fmt vet test lint check snapshot demo clean

build:
	go build -o lima-tui .

run:
	go run .

install:
	go install .

fmt:
	gofmt -l -w .

vet:
	go vet ./...

test:
	go test -race ./...

lint:
	golangci-lint run

# Everything CI checks, except lint (needs golangci-lint installed)
check: fmt vet test build

# Build a local release (all platforms) into dist/ without publishing
snapshot:
	goreleaser release --snapshot --clean

# Re-record the README screenshots and demo GIFs (needs vhs; uses a fake
# limactl, so no real VM is touched)
demo:
	./assets/demo/record.sh

clean:
	rm -rf lima-tui dist/
