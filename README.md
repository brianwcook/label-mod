# Remove OCI Labels

This project provides tools to remove and update labels on container images using the registry API directly, without pulling large image blobs. This is particularly useful when dealing with very large images (>50GB) or when container runtime tools like `buildah` and `skopeo` are not available due to uid/gid issues.

## Features

- **No image blob downloads**: Works directly with the registry API
- **Label removal**: Remove specific labels from container images
- **Label updates**: Update existing labels or add new ones
- **Multiple formats**: Available as both Go program and shell script
- **Authentication support**: Works with Quay and other registries

## Prerequisites

### For Go version:
- Go 1.21 or later
- `jq` (for JSON processing)

### For Shell script version:
- `curl`
- `jq`
- `bash`

## Installation

### Go Version

1. Clone or download the files
2. Install dependencies:
   ```bash
   go mod tidy
   ```
3. Build the program:
   ```bash
   go build -o remove-labels main.go
   ```

### Shell Script Version

1. Make the script executable (already done):
   ```bash
   chmod +x remove-labels.sh
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

### Go Version

```bash
# Remove labels
go run main.go remove-labels <image> <label1> [label2] ...

# Update labels
go run main.go update-labels <image> <key=value> [key=value] ...

# Test image (view current labels)
go run main.go test <image>
```

### Shell Script Version

```bash
# Remove labels
./remove-labels.sh remove-labels <image> <label1> [label2] ...

# Update labels
./remove-labels.sh update-labels <image> <key=value> [key=value] ...

# Test image (view current labels)
./remove-labels.sh test <image>
```

## Examples

### Remove expiration label from your target image:

```bash
# Using Go version
go run main.go remove-labels quay.io/redhat-user-workloads/bcook-tenant/simple-container-a9695:tree-b83d54e2488749d83c2bd81dfcb9ed4bef28656d quay.expires-after

# Using shell script
./remove-labels.sh remove-labels quay.io/redhat-user-workloads/bcook-tenant/simple-container-a9695:tree-b83d54e2488749d83c2bd81dfcb9ed4bef28656d quay.expires-after
```

### Test with your test images:

```bash
# Test current labels
./remove-labels.sh test quay.io/bcook/labeltest/test:latest

# Remove a label
./remove-labels.sh remove-labels quay.io/bcook/labeltest/test:latest quay.expires-after

# Update a label
./remove-labels.sh update-labels quay.io/bcook/labeltest/test:latest quay.expires-after=2024-12-31
```

### Update multiple labels:

```bash
./remove-labels.sh update-labels quay.io/bcook/labeltest/test:latest \
  quay.expires-after=2024-12-31 \
  maintainer=bcook@redhat.com \
  version=1.0.0
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

1. Ensure the script is executable: `chmod +x remove-labels.sh`
2. Check that you have write permissions to the repository
3. Verify your account has push access to the registry

## License

This project is provided as-is for educational and operational purposes. 