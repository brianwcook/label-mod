# Releases

This document describes how to create releases for the `label-mod` tool.

## Automated Releases

### GitHub Actions

The repository includes GitHub Actions workflows for automated releases:

- **`.github/workflows/test.yml`** - Runs tests on pull requests and pushes
- **`.github/workflows/release.yml`** - Creates releases when tags are pushed

### Creating a Release

1. **Create and push a tag:**
   ```bash
   git tag -a v1.0.0 -m "Release v1.0.0"
   git push origin v1.0.0
   ```

2. **GitHub Actions will automatically:**
   - Build for all platforms (Linux AMD64/ARM64, Darwin AMD64/ARM64)
   - Run tests
   - Create a GitHub release with all binaries
   - Generate release notes

## Manual Releases

### Using the Release Script

```bash
# Create release with current git version
./scripts/release.sh

# Create release with specific version
./scripts/release.sh 1.0.0
```

### Using Make

```bash
# Build all platforms
make build-all

# Create release archives
make release VERSION=1.0.0
```

### Manual Steps

1. **Build all platforms:**
   ```bash
   make build-all
   ```

2. **Create release archives:**
   ```bash
   make release VERSION=1.0.0
   ```

3. **Create GitHub release:**
   ```bash
   # Using GitHub CLI
   gh release create v1.0.0 bin/*.tar.gz --title "Release v1.0.0" --notes "Release v1.0.0"
   
   # Or manually upload files from bin/ to GitHub release
   ```

## Supported Platforms

| Platform | Architecture | Binary Name |
|----------|-------------|-------------|
| Linux | AMD64 | `label-mod-linux-amd64` |
| Linux | ARM64 | `label-mod-linux-arm64` |
| macOS | AMD64 | `label-mod-darwin-amd64` |
| macOS | ARM64 | `label-mod-darwin-arm64` |

## Release Archives

Each release includes compressed archives for each platform:

- `label-mod-linux-amd64-v1.0.0.tar.gz`
- `label-mod-linux-arm64-v1.0.0.tar.gz`
- `label-mod-darwin-amd64-v1.0.0.tar.gz`
- `label-mod-darwin-arm64-v1.0.0.tar.gz`

## Installation

### From Release

1. Download the appropriate archive for your platform
2. Extract the binary:
   ```bash
   tar -xzf label-mod-linux-amd64-v1.0.0.tar.gz
   ```
3. Make executable and move to PATH:
   ```bash
   chmod +x label-mod-linux-amd64
   sudo mv label-mod-linux-amd64 /usr/local/bin/label-mod
   ```

### From Source

```bash
git clone https://github.com/your-repo/remove-oci-labels.git
cd remove-oci-labels
make build
sudo make install
```

## Version Management

### Version Format

Releases use semantic versioning: `v1.2.3`

- **Major** (1): Breaking changes
- **Minor** (2): New features, backward compatible
- **Patch** (3): Bug fixes, backward compatible

### Version Sources

The version is determined by:

1. **Git tag** (preferred): `git describe --tags --always --dirty`
2. **Manual override**: `VERSION=1.0.0 make release`
3. **Git commit**: Falls back to git commit hash if no tag

## Release Checklist

Before creating a release:

- [ ] All tests pass: `make test`
- [ ] Code is clean: `git status`
- [ ] Version is updated in documentation
- [ ] Release notes are prepared
- [ ] Tag is created: `git tag -a v1.0.0 -m "Release v1.0.0"`
- [ ] Tag is pushed: `git push origin v1.0.0`

## Troubleshooting

### Build Issues

**Cross-compilation fails:**
```bash
# Ensure Go is properly installed
go version

# Check target platform support
go tool dist list | grep -E "(linux|darwin).*(amd64|arm64)"
```

**Permission denied:**
```bash
# Make script executable
chmod +x scripts/release.sh
```

### Release Issues

**GitHub Actions fails:**
- Check workflow logs in GitHub Actions tab
- Ensure `GITHUB_TOKEN` has release permissions
- Verify tag format: `v1.0.0`

**Manual release fails:**
- Check file permissions
- Verify tar/gzip are available
- Ensure sufficient disk space

## Security

### Binary Verification

Verify binary integrity:

```bash
# Check SHA256 checksums
sha256sum bin/*.tar.gz

# Verify binary was built from source
file bin/label-mod-linux-amd64
```

### Signing (Future Enhancement)

Future releases may include GPG signatures for additional security. 