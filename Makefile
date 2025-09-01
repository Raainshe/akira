# Akira Makefile
VERSION ?= $(shell git describe --tags --always --dirty)
BUILD_TIME = $(shell date -u '+%Y-%m-%d_%H:%M:%S')
LDFLAGS = -ldflags "-X main.Version=$(VERSION) -X main.BuildTime=$(BUILD_TIME)"

.PHONY: build build-linux build-darwin build-windows install clean

# Build for current platform
build:
	go build $(LDFLAGS) -o bin/akira .

# Build for Linux
build-linux:
	GOOS=linux GOARCH=amd64 go build $(LDFLAGS) -o bin/akira-linux-amd64 .

# Build for macOS
build-darwin:
	GOOS=darwin GOARCH=amd64 go build $(LDFLAGS) -o bin/akira-darwin-amd64 .

# Build for Windows
build-windows:
	GOOS=windows GOARCH=amd64 go build $(LDFLAGS) -o bin/akira-windows-amd64.exe .

# Build all platforms
build-all: build-linux build-darwin build-windows

# Install to system (requires sudo)
install: build
	sudo cp bin/akira /usr/local/bin/
	sudo chmod +x /usr/local/bin/akira

# Install to user's home directory (no sudo required)
install-user: build
	mkdir -p $(HOME)/.local/bin
	cp bin/akira $(HOME)/.local/bin/
	chmod +x $(HOME)/.local/bin/akira
	@echo "Add $(HOME)/.local/bin to your PATH if not already there"

# Create release archive
release: build-all
	mkdir -p releases
	tar -czf releases/akira-$(VERSION)-linux-amd64.tar.gz -C bin akira-linux-amd64
	tar -czf releases/akira-$(VERSION)-darwin-amd64.tar.gz -C bin akira-darwin-amd64
	zip -j releases/akira-$(VERSION)-windows-amd64.zip bin/akira-windows-amd64.exe

# Clean build artifacts
clean:
	rm -rf bin/ releases/

# Run tests
test:
	go test ./...

# Run with race detection
test-race:
	go test -race ./...

# Format code
fmt:
	go fmt ./...

# Lint code
lint:
	golangci-lint run

# Show help
help:
	@echo "Available targets:"
	@echo "  build        - Build for current platform"
	@echo "  build-linux  - Build for Linux"
	@echo "  build-darwin - Build for macOS"
	@echo "  build-windows- Build for Windows"
	@echo "  build-all    - Build for all platforms"
	@echo "  install      - Install to system (requires sudo)"
	@echo "  install-user - Install to user directory"
	@echo "  release      - Create release archives"
	@echo "  clean        - Clean build artifacts"
	@echo "  test         - Run tests"
	@echo "  fmt          - Format code"
	@echo "  lint         - Lint code"
