#!/bin/bash
set -e

echo "Building el/chrony-suite (Go API)..."
docker build -f Dockerfile -t el/chrony-suite . 