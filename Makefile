.PHONY: build run clean install

# Build the binary
build:
	go build -o lima-tui .

# Run the application
run:
	go run .

# Clean build artifacts
clean:
	rm -f lima-tui

# Install dependencies
deps:
	go mod tidy
	go mod download

# Install the binary to PATH
install: build
	cp lima-tui /usr/local/bin/

# Development - run with hot reload (if you have air installed)
dev:
	air

# Format code
fmt:
	go fmt ./...

# Cross-compile for different platforms
build-all:
	GOOS=darwin GOARCH=amd64 go build -o lima-tui-darwin-amd64 .
	GOOS=darwin GOARCH=arm64 go build -o lima-tui-darwin-arm64 .
	GOOS=linux GOARCH=amd64 go build -o lima-tui-linux-amd64 .
