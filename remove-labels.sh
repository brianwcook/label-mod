#!/bin/bash

set -e

# Colors for output
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
NC='\033[0m' # No Color

# Default values
REGISTRY=""
USERNAME=""
PASSWORD=""
REPOSITORY=""
TAG=""

# Function to print usage
usage() {
    echo "Usage: $0 <command> [options]"
    echo ""
    echo "Commands:"
    echo "  remove-labels <image> <label1> [label2] ..."
    echo "  update-labels <image> <key=value> [key=value] ..."
    echo "  test <image>"
    echo ""
    echo "Examples:"
    echo "  $0 remove-labels quay.io/bcook/labeltest/test:latest quay.expires-after"
    echo "  $0 update-labels quay.io/bcook/labeltest/test:latest quay.expires-after=2024-12-31"
    echo "  $0 test quay.io/bcook/labeltest/test:latest"
    echo ""
    echo "Environment variables:"
    echo "  QUAY_USERNAME - Quay username"
    echo "  QUAY_PASSWORD - Quay password"
    echo "  REGISTRY_USERNAME - Registry username (fallback)"
    echo "  REGISTRY_PASSWORD - Registry password (fallback)"
}

# Function to extract registry info from image reference
parse_image() {
    local image="$1"
    
    # Extract registry
    if [[ "$image" == *"/"* ]]; then
        REGISTRY=$(echo "$image" | cut -d'/' -f1)
        REPOSITORY=$(echo "$image" | cut -d'/' -f2- | cut -d':' -f1)
    else
        REGISTRY="docker.io"
        REPOSITORY="$image"
    fi
    
    # Extract tag
    if [[ "$image" == *":"* ]]; then
        TAG=$(echo "$image" | cut -d':' -f2)
    else
        TAG="latest"
    fi
    
    echo "Registry: $REGISTRY"
    echo "Repository: $REPOSITORY"
    echo "Tag: $TAG"
}

# Function to get authentication
get_auth() {
    # Try environment variables first
    if [[ -n "$QUAY_USERNAME" && -n "$QUAY_PASSWORD" ]]; then
        USERNAME="$QUAY_USERNAME"
        PASSWORD="$QUAY_PASSWORD"
    elif [[ -n "$REGISTRY_USERNAME" && -n "$REGISTRY_PASSWORD" ]]; then
        USERNAME="$REGISTRY_USERNAME"
        PASSWORD="$REGISTRY_PASSWORD"
    else
        echo -e "${YELLOW}Warning: No authentication credentials found${NC}"
        echo "Set QUAY_USERNAME/QUAY_PASSWORD or REGISTRY_USERNAME/REGISTRY_PASSWORD environment variables"
        return 1
    fi
}

# Function to get manifest
get_manifest() {
    local image="$1"
    local auth_header=""
    
    if [[ -n "$USERNAME" && -n "$PASSWORD" ]]; then
        auth_header="-H \"Authorization: Basic $(echo -n "$USERNAME:$PASSWORD" | base64)\""
    fi
    
    local manifest_url="https://$REGISTRY/v2/$REPOSITORY/manifests/$TAG"
    
    echo "Getting manifest from: $manifest_url"
    
    local manifest=$(curl -s -f $auth_header \
        -H "Accept: application/vnd.docker.distribution.manifest.v2+json" \
        "$manifest_url")
    
    if [[ $? -ne 0 ]]; then
        echo -e "${RED}Error: Failed to get manifest${NC}"
        return 1
    fi
    
    echo "$manifest"
}

# Function to get config blob
get_config() {
    local config_digest="$1"
    local auth_header=""
    
    if [[ -n "$USERNAME" && -n "$PASSWORD" ]]; then
        auth_header="-H \"Authorization: Basic $(echo -n "$USERNAME:$PASSWORD" | base64)\""
    fi
    
    local config_url="https://$REGISTRY/v2/$REPOSITORY/blobs/$config_digest"
    
    echo "Getting config from: $config_url"
    
    local config=$(curl -s -f $auth_header "$config_url")
    
    if [[ $? -ne 0 ]]; then
        echo -e "${RED}Error: Failed to get config${NC}"
        return 1
    fi
    
    echo "$config"
}

# Function to upload config blob
upload_config() {
    local config_data="$1"
    local auth_header=""
    
    if [[ -n "$USERNAME" && -n "$PASSWORD" ]]; then
        auth_header="-H \"Authorization: Basic $(echo -n "$USERNAME:$PASSWORD" | base64)\""
    fi
    
    local upload_url="https://$REGISTRY/v2/$REPOSITORY/blobs/uploads/"
    
    echo "Initiating upload to: $upload_url"
    
    # Start upload
    local upload_response=$(curl -s -f $auth_header -X POST "$upload_url")
    
    if [[ $? -ne 0 ]]; then
        echo -e "${RED}Error: Failed to initiate upload${NC}"
        return 1
    fi
    
    # Extract location from response headers
    local location=$(echo "$upload_response" | grep -i "location:" | cut -d' ' -f2 | tr -d '\r')
    
    if [[ -z "$location" ]]; then
        echo -e "${RED}Error: No upload location in response${NC}"
        return 1
    fi
    
    echo "Uploading config to: $location"
    
    # Upload the config data
    local upload_result=$(curl -s -f $auth_header \
        -X PUT \
        -H "Content-Type: application/vnd.docker.container.image.v1+json" \
        -H "Content-Length: $(echo -n "$config_data" | wc -c)" \
        --data-binary "$config_data" \
        "$location")
    
    if [[ $? -ne 0 ]]; then
        echo -e "${RED}Error: Failed to upload config${NC}"
        return 1
    fi
    
    # Extract digest from response headers
    local digest=$(echo "$upload_result" | grep -i "docker-content-digest:" | cut -d' ' -f2 | tr -d '\r')
    
    if [[ -z "$digest" ]]; then
        echo -e "${RED}Error: No digest in upload response${NC}"
        return 1
    fi
    
    echo "$digest"
}

# Function to upload manifest
upload_manifest() {
    local manifest_data="$1"
    local auth_header=""
    
    if [[ -n "$USERNAME" && -n "$PASSWORD" ]]; then
        auth_header="-H \"Authorization: Basic $(echo -n "$USERNAME:$PASSWORD" | base64)\""
    fi
    
    local manifest_url="https://$REGISTRY/v2/$REPOSITORY/manifests/$TAG"
    
    echo "Uploading manifest to: $manifest_url"
    
    local result=$(curl -s -f $auth_header \
        -X PUT \
        -H "Content-Type: application/vnd.docker.distribution.manifest.v2+json" \
        -H "Content-Length: $(echo -n "$manifest_data" | wc -c)" \
        --data-binary "$manifest_data" \
        "$manifest_url")
    
    if [[ $? -ne 0 ]]; then
        echo -e "${RED}Error: Failed to upload manifest${NC}"
        return 1
    fi
    
    echo -e "${GREEN}Successfully uploaded manifest${NC}"
}

# Function to remove labels
remove_labels() {
    local image="$1"
    shift
    local labels_to_remove=("$@")
    
    echo -e "${YELLOW}Removing labels from $image: ${labels_to_remove[*]}${NC}"
    
    # Parse image
    parse_image "$image"
    
    # Get authentication
    get_auth
    
    # Get manifest
    local manifest=$(get_manifest "$image")
    if [[ $? -ne 0 ]]; then
        exit 1
    fi
    
    # Extract config digest
    local config_digest=$(echo "$manifest" | jq -r '.config.digest')
    if [[ "$config_digest" == "null" ]]; then
        echo -e "${RED}Error: No config digest found in manifest${NC}"
        exit 1
    fi
    
    echo "Config digest: $config_digest"
    
    # Get config
    local config=$(get_config "$config_digest")
    if [[ $? -ne 0 ]]; then
        exit 1
    fi
    
    # Remove labels
    local updated_config="$config"
    local removed=false
    
    for label in "${labels_to_remove[@]}"; do
        if echo "$config" | jq -e ".Labels.\"$label\"" > /dev/null 2>&1; then
            updated_config=$(echo "$updated_config" | jq "del(.Labels.\"$label\")")
            echo -e "${GREEN}Removed label: $label${NC}"
            removed=true
        else
            echo -e "${YELLOW}Label not found: $label${NC}"
        fi
    done
    
    if [[ "$removed" == "false" ]]; then
        echo -e "${YELLOW}No labels were removed${NC}"
        return 0
    fi
    
    # Upload new config
    local new_config_digest=$(upload_config "$updated_config")
    if [[ $? -ne 0 ]]; then
        exit 1
    fi
    
    echo "New config digest: $new_config_digest"
    
    # Update manifest
    local updated_manifest=$(echo "$manifest" | jq --arg digest "$new_config_digest" --arg size "$(echo -n "$updated_config" | wc -c)" '.config.digest = $digest | .config.size = ($size | tonumber)')
    
    # Upload new manifest
    upload_manifest "$updated_manifest"
    if [[ $? -ne 0 ]]; then
        exit 1
    fi
    
    echo -e "${GREEN}Successfully updated image labels${NC}"
}

# Function to update labels
update_labels() {
    local image="$1"
    shift
    local label_updates=("$@")
    
    echo -e "${YELLOW}Updating labels on $image${NC}"
    
    # Parse image
    parse_image "$image"
    
    # Get authentication
    get_auth
    
    # Get manifest
    local manifest=$(get_manifest "$image")
    if [[ $? -ne 0 ]]; then
        exit 1
    fi
    
    # Extract config digest
    local config_digest=$(echo "$manifest" | jq -r '.config.digest')
    if [[ "$config_digest" == "null" ]]; then
        echo -e "${RED}Error: No config digest found in manifest${NC}"
        exit 1
    fi
    
    echo "Config digest: $config_digest"
    
    # Get config
    local config=$(get_config "$config_digest")
    if [[ $? -ne 0 ]]; then
        exit 1
    fi
    
    # Update labels
    local updated_config="$config"
    
    for update in "${label_updates[@]}"; do
        if [[ "$update" == *"="* ]]; then
            local key="${update%%=*}"
            local value="${update#*=}"
            
            updated_config=$(echo "$updated_config" | jq --arg key "$key" --arg value "$value" '.Labels[$key] = $value')
            echo -e "${GREEN}Updated label: $key=$value${NC}"
        else
            echo -e "${RED}Invalid label format: $update (expected key=value)${NC}"
            exit 1
        fi
    done
    
    # Upload new config
    local new_config_digest=$(upload_config "$updated_config")
    if [[ $? -ne 0 ]]; then
        exit 1
    fi
    
    echo "New config digest: $new_config_digest"
    
    # Update manifest
    local updated_manifest=$(echo "$manifest" | jq --arg digest "$new_config_digest" --arg size "$(echo -n "$updated_config" | wc -c)" '.config.digest = $digest | .config.size = ($size | tonumber)')
    
    # Upload new manifest
    upload_manifest "$updated_manifest"
    if [[ $? -ne 0 ]]; then
        exit 1
    fi
    
    echo -e "${GREEN}Successfully updated image labels${NC}"
}

# Function to test image
test_image() {
    local image="$1"
    
    echo -e "${YELLOW}Testing image: $image${NC}"
    
    # Parse image
    parse_image "$image"
    
    # Get authentication
    get_auth
    
    # Get manifest
    local manifest=$(get_manifest "$image")
    if [[ $? -ne 0 ]]; then
        exit 1
    fi
    
    # Extract config digest
    local config_digest=$(echo "$manifest" | jq -r '.config.digest')
    if [[ "$config_digest" == "null" ]]; then
        echo -e "${RED}Error: No config digest found in manifest${NC}"
        exit 1
    fi
    
    echo "Config digest: $config_digest"
    
    # Get config
    local config=$(get_config "$config_digest")
    if [[ $? -ne 0 ]]; then
        exit 1
    fi
    
    echo "Current labels:"
    local labels=$(echo "$config" | jq -r '.Labels // {}')
    
    if [[ "$labels" == "{}" ]]; then
        echo "  No labels found"
    else
        echo "$labels" | jq -r 'to_entries[] | "  \(.key)=\(.value)"'
    fi
}

# Main script logic
if [[ $# -lt 1 ]]; then
    usage
    exit 1
fi

command="$1"
shift

case "$command" in
    "remove-labels")
        if [[ $# -lt 2 ]]; then
            echo -e "${RED}Usage: $0 remove-labels <image> <label1> [label2] ...${NC}"
            exit 1
        fi
        image="$1"
        shift
        remove_labels "$image" "$@"
        ;;
    "update-labels")
        if [[ $# -lt 2 ]]; then
            echo -e "${RED}Usage: $0 update-labels <image> <key=value> [key=value] ...${NC}"
            exit 1
        fi
        image="$1"
        shift
        update_labels "$image" "$@"
        ;;
    "test")
        if [[ $# -lt 1 ]]; then
            echo -e "${RED}Usage: $0 test <image>${NC}"
            exit 1
        fi
        image="$1"
        test_image "$image"
        ;;
    *)
        echo -e "${RED}Unknown command: $command${NC}"
        usage
        exit 1
        ;;
esac 