# Chrony Suite

A Docker container that provides both a chrony NTP server/client and a RESTful API for managing chrony configuration.

## Overview

Chrony Suite is an all-in-one solution that runs:
- **chrony** as both NTP server and client (with `--cap-add=SYS_TIME` to set host time)
- **Flask API** for remote management of chrony configuration
- **Health checks** to ensure the service is running properly

## Features

- 🕐 **NTP Server/Client**: Synchronizes time from upstream NTP servers and serves time to other clients
- 🔧 **RESTful API**: Manage chrony configuration remotely via HTTP endpoints
- 🐳 **Docker Ready**: Easy deployment with Docker
- 🔒 **Secure**: Based on Alpine Linux for smaller attack surface
- 📊 **Health Monitoring**: Built-in health checks

## Quick Start

### Option 1: All-in-One (Recommended)
```bash
./run_and_test.sh
```
This script will build, run, and test everything automatically.

### Option 2: Step by Step
```bash
# Build the image
./build.sh

# Run the container
./run.sh

# Test the API
./test.sh
```

## API Documentation

### Base URL
```
http://localhost:8291
```

### Endpoints

#### 1. Get Chrony Version
```http
GET /chrony/version
```

**Response:**
```json
{
  "version": "chrony version 4.6.1",
  "error": null
}
```

#### 2. Get Current Status
```http
GET /chrony/status
```

**Response:**
```json
{
  "server_mode_enabled": true,
  "tracking": {
    "Reference ID": "7F7F0101 ()",
    "Stratum": "10",
    "Ref time (UTC)": "Thu Jul 03 15:41:24 2025",
    "System time": "0.000000000 seconds fast of NTP time"
  },
  "tracking_error": null,
  "sources": [
    {
      "name": "pool.ntp.org",
      "raw": "^? pool.ntp.org 0 7 0 - +0ns[   +0ns] +/- 0ns"
    }
  ],
  "sources_error": null
}
```

#### 3. Get Current Sources
```http
GET /chrony/servers
```

**Response:**
```json
{
  "servers": "MS Name/IP address         Stratum Poll Reach LastRx Last sample\n===============================================================================\n^? pool.ntp.org 0 7 0 - +0ns[   +0ns] +/- 0ns",
  "error": null
}
```

#### 4. Set NTP Servers
```http
PUT /chrony/servers
Content-Type: application/json

{
  "servers": ["pool.ntp.org", "time.google.com", "time.windows.com"]
}
```

**Response:**
```json
{
  "result": [
    {
      "server": "pool.ntp.org",
      "output": "",
      "error": null
    },
    {
      "server": "time.google.com",
      "output": "",
      "error": null
    }
  ]
}
```

#### 5. Reset Servers (Delete All)
```http
DELETE /chrony/servers
```

**Response:**
```json
{
  "output": "",
  "error": null
}
```

#### 6. Set Default Servers
```http
PUT /chrony/servers/default
```

**Response:**
```json
{
  "result": [
    {
      "server": "pool.ntp.org",
      "output": "",
      "error": null
    }
  ]
}
```

#### 7. Get Server Mode Status
```http
GET /chrony/server-mode
```

**Response:**
```json
{
  "server_mode_enabled": true
}
```

#### 8. Set Server Mode
```http
PUT /chrony/server-mode
Content-Type: application/json

{
  "enabled": true
}
```

**Response:**
```json
{
  "success": true,
  "server_mode_enabled": true
}
```

## Configuration

### Default NTP Servers
The container uses `pool.ntp.org` as the default NTP server.

### Ports
- **UDP 123**: NTP service (chrony)
- **TCP 8291**: REST API (Flask)

### Docker Capabilities
- `--cap-add=SYS_TIME`: Allows chrony to set the host system time

## File Structure

```
.
├── Dockerfile              # Docker image definition
├── chrony.conf             # Chrony configuration
├── chrony_api_app.py       # Flask API application
├── entrypoint.sh           # Container startup script
├── build.sh                # Build script
├── run.sh                  # Run script (includes cleanup)
├── test.sh                 # API testing script
├── clean.sh                # Cleanup script
├── run_and_test.sh         # All-in-one build, run & test script
└── README.md               # This file
```

## Development

### Prerequisites
- Docker
- curl (for testing)
- jq (optional, for pretty JSON output)

### Building from Source
```bash
# Build the image
docker build -f Dockerfile -t el/chrony-suite .

# Run the container (automatically cleans up existing container)
./run.sh

# Clean up manually if needed
./clean.sh              # Remove container only
./clean.sh --image      # Remove container and image
```

### Testing
```bash
# Run comprehensive API tests
./test.sh

# Test individual endpoints
curl http://localhost:8291/chrony/status
curl http://localhost:8291/chrony/version
```

## Troubleshooting

### Container Won't Start
- Check if port 8291 is available
- Ensure Docker has permission to add SYS_TIME capability
- Check container logs: `docker logs el-chrony-suite`

### API Not Responding
- Verify container is running: `docker ps`
- Check if chrony is running: `docker exec el-chrony-suite chronyc tracking`
- Check API logs: `docker logs el-chrony-suite`

### Time Sync Issues
- Verify NTP servers are reachable
- Check chrony configuration: `docker exec el-chrony-suite cat /etc/chrony/chrony.conf`
- Monitor chrony sources: `docker exec el-chrony-suite chronyc sources`

## Security Considerations

- The container runs with `--cap-add=SYS_TIME` which allows setting system time
- Only use this container in trusted environments
- Consider firewall rules to restrict API access
- The API allows all clients by default (`allow 0.0.0.0/0`)

## License

This project is provided as-is for educational and development purposes.

## Contributing

Feel free to submit issues and enhancement requests! 