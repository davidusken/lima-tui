.PHONY: build run install fmt vet snapshot clean

build:
	go build -o lima-tui .

run:
	go run .

install:
	go install .

fmt:
	go fmt ./...

vet:
	go vet ./...

# Build a local release (all platforms) into dist/ without publishing
snapshot:
	goreleaser release --snapshot --clean

clean:
	rm -rf lima-tui dist/
