#!/bin/bash
set -e

# Clean up any existing container first
./clean.sh

# Run the Python version container
docker run -d --name el-chrony-suite-python \
  --cap-add=SYS_TIME \
  -p 123:123/udp \
  -p 8291:8291 \
  el/chrony-suite-python 