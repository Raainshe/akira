# Akira Makefile
VERSION ?= $(shell git describe --tags --always --dirty)
BUILD_TIME = $(shell date -u '+%Y-%m-%d_%H:%M:%S')
LDFLAGS = -ldflags "-X main.Version=$(VERSION) -X main.BuildTime=$(BUILD_TIME)"

.PHONY: build build-linux build-darwin build-windows install clean docker-build docker-up docker-down docker-dev docker-dev-down gen-fake-torrents

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

# Docker
docker-build:
	VERSION=$(VERSION) BUILD_TIME=$(BUILD_TIME) docker compose build

docker-up:
	docker compose up -d --build

docker-down:
	docker compose down

# Dev stack with Air live reload (source mounted; uses docker-compose.dev.yml overlay)
docker-dev:
	docker compose -f docker-compose.yml -f docker-compose.dev.yml up --build -d

docker-dev-down:
	docker compose -f docker-compose.yml -f docker-compose.dev.yml down

# Generate fake torrents for local testing (see scripts/genfake/main.go)
gen-fake-torrents:
	go run ./scripts/genfake -count 3 -out testdata/fake-torrents

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
	@echo "  docker-build    - Build production Docker image"
	@echo "  docker-up       - Start Akira (production image, detached)"
	@echo "  docker-down     - Stop production docker compose stack"
	@echo "  docker-dev      - Start dev stack with Air live reload (detached)"
	@echo "  docker-dev-down - Stop dev docker compose stack"
	@echo "  gen-fake-torrents - Generate fake .torrent files and magnets"
	@echo "  test         - Run tests"
	@echo "  fmt          - Format code"
	@echo "  lint         - Lint code"
