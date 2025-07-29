# label-mod

A Go-based tool to remove and update labels on container images using the registry API directly, without pulling large image blobs. This is particularly useful when dealing with very large images (>50GB) or when container runtime tools like `buildah` and `skopeo` are not available due to uid/gid issues.

## Features

- **No image blob downloads**: Works directly with the registry API
- **Label removal**: Remove specific labels from container images
- **Label updates**: Update existing labels or add new ones
- **Digest reference support**: Works with both tag and digest references
- **Multiple tagging**: Support for tagging with multiple tags
- **JSON output**: Structured output for programmatic use
- **Authentication support**: Works with Quay and other registries

## Prerequisites

- Go 1.21 or later
- podman (for local testing)

## Installation

1. Clone or download the files
2. Install dependencies:
   ```bash
   go mod tidy
   ```
3. Build the program:
   ```bash
   make build
   ```
   Or manually:
   ```bash
   go build -o bin/label-mod main.go
   ```

## Configuration

Set your registry credentials as environment variables:

```bash
# For Quay.io
export QUAY_USERNAME="your-username"
export QUAY_PASSWORD="your-password"

# Or for other registries
export REGISTRY_USERNAME="your-username"
export REGISTRY_PASSWORD="your-password"
```

## Usage

```bash
# Remove labels
./bin/label-mod remove-labels <image> <label1> [label2] ... [--tag <new-tag>]

# Update labels
./bin/label-mod update-labels <image> <key=value> [key=value] ... [--tag <new-tag>]

# Modify labels (remove and update in one command)
./bin/label-mod modify-labels <image> [--remove <label1>] [--update <key=value>] [--tag <new-tag>]

# Test image (view current labels)
./bin/label-mod test <image>
```

## Examples

### Remove expiration label from your target image:

```bash
./bin/label-mod remove-labels quay.io/redhat-user-workloads/bcook-tenant/simple-container-a9695:tree-b83d54e2488749d83c2bd81dfcb9ed4bef28656d quay.expires-after
```

### Test with your test images:

```bash
# Test current labels
./bin/label-mod test quay.io/bcook/labeltest/test:latest

# Remove a label
./bin/label-mod remove-labels quay.io/bcook/labeltest/test:latest quay.expires-after

# Update a label
./bin/label-mod update-labels quay.io/bcook/labeltest/test:latest quay.expires-after=2024-12-31
```

### Update multiple labels:

```bash
./bin/label-mod update-labels quay.io/bcook/labeltest/test:latest \
  quay.expires-after=2024-12-31 \
  maintainer=bcook@redhat.com \
  version=1.0.0
```

### Work with digest references:

```bash
# Test digest reference
./bin/label-mod test quay.io/repo/image@sha256:abc123...

# Remove label with new tag
./bin/label-mod remove-labels quay.io/repo/image@sha256:abc123... label-name --tag new-tag

# Update label with digest reference
./bin/label-mod update-labels quay.io/repo/image@sha256:abc123... new.label=value --tag updated
```

### Multiple tagging:

```bash
# Tag with multiple tags
./bin/label-mod update-labels quay.io/repo/image:latest new.label=value --tag v1.0 --tag latest --tag stable
```

### Combined operations:

```bash
# Remove and update labels in one command
./bin/label-mod modify-labels quay.io/repo/image:latest \
  --remove old.label \
  --update new.label=value \
  --tag modified
```

## Testing

The project includes comprehensive tests that verify all functionality. Tests can be run with either a local registry (recommended) or external registries.

### Local Registry Testing (Recommended)

The project includes a local container registry setup for self-contained testing. This approach is faster, more reliable, and doesn't require external registry access.

#### Prerequisites

- **podman** - Container runtime for running the local registry
- **Go 1.21+** - For building and running tests

#### Quick Test

```bash
# Run all tests with local registry (setup, test, cleanup)
./scripts/test-local.sh
```

#### Manual Testing

```bash
# Setup local registry
./scripts/setup-test-registry.sh

# Run tests
go test -v

# Cleanup (optional - script handles this automatically)
./scripts/cleanup-test-registry.sh
```

#### Test Configuration

Tests use the local registry by default:
- **Registry**: `localhost:5000`
- **Test Image**: `localhost:5000/test/labeltest:latest`
- **Test Labels**: Pre-configured test labels for all operations

#### Environment Variables

You can override the test configuration:

```bash
# Use external registry for testing
export LABEL_MOD_TEST_REPO="quay.io/bcook/labeltest/test"
export LABEL_MOD_TEST_TAG="has-label"

# Run tests
go test -v
```

### External Registry Testing

Tests can also run against external registries by setting environment variables:

```bash
export LABEL_MOD_TEST_REPO="quay.io/bcook/labeltest/test"
export LABEL_MOD_TEST_TAG="has-label"
go test -v
```

## How It Works

1. **Manifest Retrieval**: Fetches the image manifest from the registry
2. **Config Extraction**: Downloads only the config blob (typically < 1KB)
3. **Label Modification**: Modifies the labels in the config JSON
4. **Config Upload**: Uploads the modified config blob back to the registry
5. **Manifest Update**: Updates the manifest to reference the new config digest
6. **Manifest Upload**: Uploads the updated manifest

This approach avoids downloading the large image layers while still allowing label manipulation.

## Error Handling

The tools provide detailed error messages for common issues:

- Authentication failures
- Network connectivity problems
- Invalid image references
- Missing labels
- Registry API errors

## Security Notes

- Credentials are transmitted using Basic Authentication
- Consider using token-based authentication for production use
- The tools do not store credentials locally
- All communication uses HTTPS

## Troubleshooting

### Authentication Issues

If you get authentication errors:

1. Verify your credentials are set correctly:
   ```bash
   echo "Username: $QUAY_USERNAME"
   echo "Password: $QUAY_PASSWORD"
   ```

2. Test with a simple curl command:
   ```bash
   curl -u "$QUAY_USERNAME:$QUAY_PASSWORD" \
     "https://quay.io/v2/bcook/labeltest/manifests/latest"
   ```

### Network Issues

If you get network errors:

1. Check your internet connection
2. Verify the registry URL is accessible
3. Check if you're behind a corporate firewall

### Permission Issues

If you get permission errors:

1. Ensure the binary is executable: `chmod +x bin/label-mod`
2. Check that you have write permissions to the repository
3. Verify your account has push access to the registry

## License

This project is provided as-is for educational and operational purposes. 