#!/bin/bash
set -e

echo "======================================"
echo "Chrony Suite - Cleanup"
echo "======================================"

# Colors for output
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
NC='\033[0m' # No Color

# Function to print colored output
print_status() {
    echo -e "${GREEN}[INFO]${NC} $1"
}

print_warning() {
    echo -e "${YELLOW}[WARN]${NC} $1"
}

print_error() {
    echo -e "${RED}[ERROR]${NC} $1"
}

# Check if container exists and is running
if docker ps -a --filter name=el-chrony-suite --format "table {{.Names}}" | grep -q el-chrony-suite; then
    print_status "Found el-chrony-suite container, stopping and removing..."
    docker stop el-chrony-suite 2>/dev/null || true
    docker rm el-chrony-suite 2>/dev/null || true
    print_status "Container cleaned up successfully!"
else
    print_warning "No el-chrony-suite container found."
fi

# Optional: Remove the image if requested
if [ "$1" = "--image" ]; then
    print_status "Removing el/chrony-suite image..."
    docker rmi el/chrony-suite 2>/dev/null || true
    print_status "Image removed successfully!"
fi

print_status "Cleanup completed!" 