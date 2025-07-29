# Makefile for label-mod

# Build directory
BIN_DIR = bin

# Binary names
BINARY_NAME = label-mod
LINUX_AMD64_BINARY = label-mod-linux-amd64
LINUX_ARM64_BINARY = label-mod-linux-arm64
DARWIN_AMD64_BINARY = label-mod-darwin-amd64
DARWIN_ARM64_BINARY = label-mod-darwin-arm64

# Go build flags
GO_FLAGS = -ldflags="-s -w"

# Version (can be overridden)
VERSION ?= $(shell git describe --tags --always --dirty)

# Default target
.PHONY: all
all: build

# Create bin directory
$(BIN_DIR):
	mkdir -p $(BIN_DIR)

# Build for current platform
.PHONY: build
build: $(BIN_DIR)
	go build $(GO_FLAGS) -o $(BIN_DIR)/$(BINARY_NAME) main.go

# Build for Linux AMD64
.PHONY: build-linux-amd64
build-linux-amd64: $(BIN_DIR)
	GOOS=linux GOARCH=amd64 go build $(GO_FLAGS) -o $(BIN_DIR)/$(LINUX_AMD64_BINARY) main.go

# Build for Linux ARM64
.PHONY: build-linux-arm64
build-linux-arm64: $(BIN_DIR)
	GOOS=linux GOARCH=arm64 go build $(GO_FLAGS) -o $(BIN_DIR)/$(LINUX_ARM64_BINARY) main.go

# Build for Darwin AMD64
.PHONY: build-darwin-amd64
build-darwin-amd64: $(BIN_DIR)
	GOOS=darwin GOARCH=amd64 go build $(GO_FLAGS) -o $(BIN_DIR)/$(DARWIN_AMD64_BINARY) main.go

# Build for Darwin ARM64
.PHONY: build-darwin-arm64
build-darwin-arm64: $(BIN_DIR)
	GOOS=darwin GOARCH=arm64 go build $(GO_FLAGS) -o $(BIN_DIR)/$(DARWIN_ARM64_BINARY) main.go

# Build all platforms
.PHONY: build-all
build-all: build-linux-amd64 build-linux-arm64 build-darwin-amd64 build-darwin-arm64

# Build release binaries
.PHONY: build-release
build-release: clean build-all
	@echo "Building release binaries for version $(VERSION)"
	@echo "Binaries created in $(BIN_DIR)/"

# Create release archives
.PHONY: release
release: build-release
	@echo "Creating release archives for version $(VERSION)"
	cd $(BIN_DIR) && tar -czf label-mod-linux-amd64-$(VERSION).tar.gz $(LINUX_AMD64_BINARY)
	cd $(BIN_DIR) && tar -czf label-mod-linux-arm64-$(VERSION).tar.gz $(LINUX_ARM64_BINARY)
	cd $(BIN_DIR) && tar -czf label-mod-darwin-amd64-$(VERSION).tar.gz $(DARWIN_AMD64_BINARY)
	cd $(BIN_DIR) && tar -czf label-mod-darwin-arm64-$(VERSION).tar.gz $(DARWIN_ARM64_BINARY)
	@echo "Release archives created in $(BIN_DIR)/"

# Run tests
.PHONY: test
test:
	go test -v

# Run tests with coverage
.PHONY: test-coverage
test-coverage:
	go test -v -cover

# Clean build artifacts
.PHONY: clean
clean:
	rm -rf $(BIN_DIR)

# Install (copy to /usr/local/bin)
.PHONY: install
install: build
	sudo cp $(BIN_DIR)/$(BINARY_NAME) /usr/local/bin/

# Show help
.PHONY: help
help:
	@echo "Available targets:"
	@echo "  build              - Build for current platform"
	@echo "  build-linux-amd64  - Build for Linux AMD64"
	@echo "  build-linux-arm64  - Build for Linux ARM64"
	@echo "  build-darwin-amd64 - Build for Darwin AMD64"
	@echo "  build-darwin-arm64 - Build for Darwin ARM64"
	@echo "  build-all          - Build for all platforms"
	@echo "  build-release      - Build all release binaries"
	@echo "  release            - Build and create release archives"
	@echo "  test               - Run tests"
	@echo "  test-coverage      - Run tests with coverage"
	@echo "  clean              - Remove build artifacts"
	@echo "  install            - Install to /usr/local/bin"
	@echo "  help               - Show this help"
	@echo ""
	@echo "Environment variables:"
	@echo "  VERSION            - Version for release (default: git describe)" 