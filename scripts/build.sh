#!/bin/bash
set -e

# Source shared configuration
SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}" )" && pwd)"
source "$SCRIPT_DIR/config.sh"

print_header "Build"

# Accept version as first argument, or use default
VERSION=${1:-0.1.0-dev}
BUILD_DATETIME=$(date -u +"%Y-%m-%dT%H:%M:%SZ")

print_info "Building with version: $VERSION"

# Build with version and build datetime - Dockerfile is in parent folder
cd "$SCRIPT_DIR"
docker build -f ../Dockerfile --build-arg VERSION=$VERSION --build-arg BUILD_DATETIME=$BUILD_DATETIME -t $IMAGE_NAME:$VERSION -t $IMAGE_NAME:latest ..

print_info "Build completed!"
print_info "Image: $IMAGE_NAME:$VERSION"
print_info "Image: $IMAGE_NAME:latest" 