#!/bin/bash

set -e

echo "🚀 Starting local registry tests..."

# Colors for output
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
BLUE='\033[0;34m'
NC='\033[0m' # No Color

# Function to print colored output
print_status() {
    echo -e "${BLUE}[INFO]${NC} $1"
}

print_success() {
    echo -e "${GREEN}[SUCCESS]${NC} $1"
}

print_warning() {
    echo -e "${YELLOW}[WARNING]${NC} $1"
}

print_error() {
    echo -e "${RED}[ERROR]${NC} $1"
}

# Check if podman is available
if ! command -v podman &> /dev/null; then
    print_error "podman is not installed. Please install podman first."
    exit 1
fi

print_success "podman is available"

# Build the tool
print_status "Building label-mod..."
make build
print_success "Build completed"

# Setup the test registry
print_status "Setting up test registry..."
./scripts/setup-test-registry.sh
print_success "Test registry setup completed"

# Wait a moment for registry to be fully ready
sleep 2

# Run the tests
print_status "Running tests with local registry..."
if go test -v; then
    print_success "All tests passed!"
else
    print_error "Some tests failed!"
    exit 1
fi

# Cleanup
print_status "Cleaning up test registry..."
./scripts/cleanup-test-registry.sh
print_success "Cleanup completed"

print_success "🎉 All tests completed successfully!" 