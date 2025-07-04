#!/bin/bash
set -e

echo "Building el/chrony-suite (Python API)..."
docker build -f Dockerfile.python -t el/chrony-suite-python . 