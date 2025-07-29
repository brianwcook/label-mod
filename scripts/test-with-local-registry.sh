#!/bin/bash

set -e

echo "Setting up local registry for testing..."

# Build the tool first
echo "Building label-mod..."
make build

# Setup the test registry
echo "Setting up test registry..."
./scripts/setup-test-registry.sh

# Wait a moment for registry to be fully ready
sleep 2

# Run the tests
echo "Running tests with local registry..."
go test -v

# Cleanup
echo "Cleaning up test registry..."
./scripts/cleanup-test-registry.sh

echo "Tests completed!" 