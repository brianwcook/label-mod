#!/bin/bash

# Test script for remove-oci-labels
# This script helps test the functionality with your Quay credentials

set -e

# Colors for output
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
NC='\033[0m' # No Color

echo -e "${GREEN}Setting up test environment...${NC}"

# Set credentials from the password file
export QUAY_USERNAME="bcook"
export QUAY_PASSWORD="eboatumbil"

echo -e "${GREEN}Credentials set:${NC}"
echo "Username: $QUAY_USERNAME"
echo "Password: [hidden]"

echo -e "\n${YELLOW}Testing Go version:${NC}"
echo "Testing help output..."
./remove-labels

echo -e "\n${YELLOW}Testing shell script version:${NC}"
echo "Testing help output..."
./remove-labels.sh

echo -e "\n${GREEN}Both versions are ready for testing!${NC}"
echo ""
echo "To test with your target image:"
echo "  ./remove-labels remove-labels quay.io/redhat-user-workloads/bcook-tenant/simple-container-a9695:tree-b83d54e2488749d83c2bd81dfcb9ed4bef28656d quay.expires-after"
echo ""
echo "To test with your test images:"
echo "  ./remove-labels test quay.io/bcook/labeltest/test:latest"
echo "  ./remove-labels remove-labels quay.io/bcook/labeltest/test:latest quay.expires-after"
echo ""
echo "To update labels:"
echo "  ./remove-labels update-labels quay.io/bcook/labeltest/test:latest quay.expires-after=2024-12-31" 