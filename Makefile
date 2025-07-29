# Makefile for label-mod

# Build directory
BIN_DIR = bin

# Binary names
BINARY_NAME = label-mod
LINUX_BINARY = label-mod-linux-amd64

# Go build flags
GO_FLAGS = -ldflags="-s -w"

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
.PHONY: build-linux
build-linux: $(BIN_DIR)
	GOOS=linux GOARCH=amd64 go build $(GO_FLAGS) -o $(BIN_DIR)/$(LINUX_BINARY) main.go

# Build all platforms
.PHONY: build-all
build-all: build build-linux

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
	@echo "  build        - Build for current platform"
	@echo "  build-linux  - Build for Linux AMD64"
	@echo "  build-all    - Build for all platforms"
	@echo "  test         - Run tests"
	@echo "  test-coverage- Run tests with coverage"
	@echo "  clean        - Remove build artifacts"
	@echo "  install      - Install to /usr/local/bin"
	@echo "  help         - Show this help" 