#!/bin/bash

set -e

echo "Setting up test registry..."

# Configure podman to use HTTP for localhost
mkdir -p ~/.config/containers
cat > ~/.config/containers/registries.conf << 'EOF'
unqualified-search-registries = ["docker.io"]

[[registry]]
location = "localhost:5000"
insecure = true
EOF

# Start the registry using podman directly
podman run -d --name test-registry -p 5000:5000 \
  -e REGISTRY_STORAGE_DELETE_ENABLED=true \
  -e REGISTRY_HTTP_ADDR=0.0.0.0:5000 \
  -e REGISTRY_STORAGE_FILESYSTEM_MAXTHREADS=100 \
  -v test-registry-data:/var/lib/registry \
  registry:2

# Wait for registry to be ready
echo "Waiting for registry to be ready..."
for i in {1..30}; do
    if curl -s http://localhost:5000/v2/ > /dev/null; then
        echo "Registry is ready!"
        break
    fi
    if [ $i -eq 30 ]; then
        echo "Registry failed to start"
        echo "=== Registry container logs ==="
        podman logs test-registry || true
        echo "=== Registry container status ==="
        podman ps -a || true
        exit 1
    fi
    sleep 1
done

# Build and push test images
echo "Building test images..."

# Create a simple test image with labels
cat > Dockerfile.test << 'EOF'
FROM alpine:latest
LABEL test.key=value
LABEL test.label=original-value
LABEL test.update.label=old-value
LABEL test.remove.label=to-be-removed
LABEL test.modify.label=modify-me
LABEL maintainer=test@example.com
LABEL quay.expires-after=2024-12-31
CMD ["echo", "Hello from test image"]
EOF

# Build the test image
echo "Building test image..."
if ! podman build -f Dockerfile.test -t localhost:5000/test/labeltest:latest .; then
    echo "Failed to build test image"
    exit 1
fi

# Push to local registry with TLS verification disabled
echo "Pushing test image..."
if ! podman push --tls-verify=false localhost:5000/test/labeltest:latest; then
    echo "Failed to push test image"
    echo "=== Podman images ==="
    podman images || true
    exit 1
fi

# Create a digest reference by pulling and getting the digest
echo "Creating digest reference..."
if ! podman pull --tls-verify=false localhost:5000/test/labeltest:latest; then
    echo "Failed to pull test image"
    exit 1
fi

DIGEST=$(podman images --digests | grep "localhost:5000/test/labeltest" | awk '{print $3}' | head -1)
if [ -z "$DIGEST" ]; then
    echo "Failed to get image digest"
    echo "=== Podman images with digests ==="
    podman images --digests || true
    exit 1
fi
echo "Image digest: $DIGEST"

echo "Test registry setup complete!"
echo "Registry URL: localhost:5000"
echo "Test image: localhost:5000/test/labeltest:latest"
echo "Test digest reference: localhost:5000/test/labeltest@${DIGEST}" 