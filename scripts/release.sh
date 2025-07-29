#!/bin/bash

# Release script for label-mod

set -e

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

# Check if we're in a git repository
if ! git rev-parse --git-dir > /dev/null 2>&1; then
    print_error "Not in a git repository"
    exit 1
fi

# Check if there are uncommitted changes
if ! git diff-index --quiet HEAD --; then
    print_warning "There are uncommitted changes. Please commit or stash them first."
    read -p "Continue anyway? (y/N): " -n 1 -r
    echo
    if [[ ! $REPLY =~ ^[Yy]$ ]]; then
        exit 1
    fi
fi

# Get version from argument or git
VERSION=${1:-$(git describe --tags --always --dirty)}

if [[ "$VERSION" == *"dirty"* ]]; then
    print_error "Cannot create release with dirty working directory"
    exit 1
fi

print_status "Creating release for version: $VERSION"

# Clean previous builds
print_status "Cleaning previous builds..."
make clean

# Build all platforms
print_status "Building for all platforms..."
make build-release VERSION="$VERSION"

# Create release archives
print_status "Creating release archives..."
make release VERSION="$VERSION"

# List created files
print_status "Created release files:"
ls -la bin/*.tar.gz

# Show file sizes
print_status "Release file sizes:"
for file in bin/*.tar.gz; do
    size=$(du -h "$file" | cut -f1)
    echo "  $(basename "$file"): $size"
done

# Show checksums
print_status "SHA256 checksums:"
for file in bin/*.tar.gz; do
    checksum=$(sha256sum "$file" | cut -d' ' -f1)
    echo "  $(basename "$file"): $checksum"
done

print_success "Release created successfully!"
print_status "To create a GitHub release:"
echo "  1. Create a tag: git tag -a v$VERSION -m 'Release v$VERSION'"
echo "  2. Push the tag: git push origin v$VERSION"
echo "  3. Upload the files from bin/ to the GitHub release"
echo ""
print_status "Or use the GitHub CLI:"
echo "  gh release create v$VERSION bin/*.tar.gz --title 'Release v$VERSION' --notes 'Release v$VERSION'" 