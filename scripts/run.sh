#!/bin/bash
set -e

# Source shared configuration
SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
source "$SCRIPT_DIR/config.sh"

print_header "Run"

VERSION_ARG=$1
cleanup_container
run_container $VERSION_ARG 