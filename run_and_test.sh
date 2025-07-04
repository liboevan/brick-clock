#!/bin/bash
set -e

echo "======================================"
echo "Chrony Suite - Build, Run & Test"
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

# Step 1: Stop and remove existing container if running
print_status "Checking for existing container..."
if docker ps -a --filter name=el-chrony-suite --format "table {{.Names}}" | grep -q el-chrony-suite; then
    print_warning "Found existing el-chrony-suite container, removing..."
    docker stop el-chrony-suite 2>/dev/null || true
    docker rm el-chrony-suite 2>/dev/null || true
fi

# Step 2: Build the image
print_status "Building el/chrony-suite image..."
./build.sh

# Step 3: Run the container
print_status "Starting el-chrony-suite container..."
./run.sh

# Step 4: Wait for container to be ready
print_status "Waiting for container to be ready..."
sleep 5

# Step 5: Check if container is running
if docker ps --filter name=el-chrony-suite --format "table {{.Names}}" | grep -q el-chrony-suite; then
    print_status "Container is running successfully!"
else
    print_error "Container failed to start!"
    docker logs el-chrony-suite
    exit 1
fi

# Step 6: Wait a bit more for services to fully initialize
print_status "Waiting for services to initialize..."
sleep 3

# Step 7: Run API tests
print_status "Running API tests..."
./test.sh

print_status "======================================"
print_status "All done! Chrony Suite is ready to use."
print_status "API available at: http://localhost:8291"
print_status "NTP service available on UDP port 123"
print_status "======================================" 