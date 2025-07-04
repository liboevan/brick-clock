#!/bin/sh
# Start chronyd in the background
chronyd -f /etc/chrony/chrony.conf &

# Start the API app in the foreground
# Check if Go binary exists, otherwise use Python
if [ -f "/chrony-api-app" ]; then
    echo "Starting Go API server..."
    exec /chrony-api-app
else
    echo "Starting Python API server (fallback)..."
    exec python3 /chrony_api_app.py
fi

tail -f /dev/null