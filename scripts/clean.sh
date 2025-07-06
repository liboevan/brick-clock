#!/bin/bash
set -e

# Source shared configuration
SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
source "$SCRIPT_DIR/config.sh"

print_header "Cleanup"

# Check if container exists and is running
if docker ps -a --filter name=$CONTAINER_NAME --format "table {{.Names}}" | grep -q $CONTAINER_NAME; then
    print_info "Found $CONTAINER_NAME container, stopping and removing..."
    cleanup_container
    print_info "Container cleaned up successfully!"
else
    print_warning "No $CONTAINER_NAME container found."
fi

# Optional: Remove all images if requested
if [ "$1" = "--image" ]; then
    print_info "Removing all brick-clock images..."
    docker images --filter "reference=$IMAGE_NAME" --format "{{.Repository}}:{{.Tag}}" | while read image; do
        print_info "Removing image: $image"
        docker rmi "$image" 2>/dev/null || true
    done
    print_info "All images removed successfully!"
fi

print_info "Cleanup completed!" 