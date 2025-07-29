#!/bin/bash

set -e

echo "Cleaning up test registry..."

# Stop and remove the registry container
podman stop test-registry 2>/dev/null || true
podman rm test-registry 2>/dev/null || true

# Remove the volume
podman volume rm test-registry-data 2>/dev/null || true

# Remove test image files
rm -f Dockerfile.test

echo "Test registry cleanup complete!" 