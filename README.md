# Brick Clock Service

A high-precision Network Time Protocol (NTP) service built with Go, providing both client and server capabilities for time synchronization in distributed systems.

## 🚀 Features

- **NTP Client Mode**: Synchronize with upstream NTP servers
- **NTP Server Mode**: Act as a time source for other devices
- **RESTful API**: Full HTTP API for monitoring and management
- **Real-time Status**: Live tracking of synchronization status
- **Server Management**: Add, remove, and configure NTP servers
- **Activity Monitoring**: Track success/failure statistics
- **Docker Ready**: Containerized deployment with Alpine Linux
- **Health Monitoring**: Built-in health checks and status endpoints
- **Caching Layer**: In-memory caching for improved performance
- **Dynamic Configuration**: Runtime server and mode configuration

## 📋 Prerequisites

- Docker and Docker Compose
- Linux environment (for NTP compatibility)
- Network access to NTP servers
- Port 123/UDP available for NTP traffic
- Port 17003/TCP available for API
- `jq` (optional, for JSON formatting in tests)

## 🛠️ Quick Start

### Option 1: One-Command Setup (Recommended)

```bash
./scripts/quick_start.sh
```

This script performs a complete build → run → test cycle.

### Option 2: Step-by-Step Setup

```bash
# Build the Docker image
./scripts/build.sh

# Run the container
./scripts/run.sh

# Test the API endpoints
./scripts/test.sh
```

## 📚 Scripts Reference

### Main Management Script

```bash
./scripts/quick_start.sh [command] [version]
```

**Commands:**
- `build` - Build Docker image only
- `run` - Run container only  
- `test` - Test API endpoints only
- `clean` - Stop and remove containers
- `logs` - Show container logs
- `status` - Check container status
- `all` - Full cycle (default)

**Examples:**
```bash
./scripts/quick_start.sh                    # Full cycle with default version
./scripts/quick_start.sh test               # Test only
./scripts/quick_start.sh all 1.0.0         # Full cycle with specific version
```

### Individual Scripts

| Script | Purpose | Usage |
|--------|---------|-------|
| `build.sh` | Build Docker image | `./scripts/build.sh [version]` |
| `run.sh` | Start container | `./scripts/run.sh [version]` |
| `test.sh` | Test API endpoints | `./scripts/test.sh [host:port]` |
| `clean.sh` | Clean up resources | `./scripts/clean.sh [--image]` |
| `config.sh` | Configuration management | `./scripts/config.sh` |

## 🔌 API Reference

### Core Endpoints

| Method | Endpoint | Description |
|--------|----------|-------------|
| `GET` | `/health` | Health check endpoint |
| `GET` | `/version` | Application version and build info |
| `GET` | `/app-version` | Application version info |
| `GET` | `/status` | Current synchronization status |
| `GET` | `/status/tracking` | Detailed tracking information |
| `GET` | `/status/sources` | NTP source information |
| `GET` | `/status/activity` | Activity statistics |
| `GET` | `/status/clients` | Connected client information |
| `GET` | `/servers` | List configured NTP servers |
| `PUT` | `/servers` | Configure NTP servers |
| `DELETE` | `/servers` | Reset to default servers |
| `PUT` | `/servers/default` | Set default NTP servers |
| `GET` | `/server-mode` | Get server mode status |
| `PUT` | `/server-mode` | Enable/disable server mode |

### Status Endpoint Parameters

The `/status` endpoint supports query parameters to control which data is returned:

| Parameter | Value | Description |
|-----------|-------|-------------|
| `flags` | `1` | Include tracking data only |
| `flags` | `2` | Include sources data only |
| `flags` | `4` | Include activity data only |
| `flags` | `8` | Include clients data only |
| `flags` | `16` | Include server mode data only |
| `flags` | `23` | Include tracking + sources + activity + server mode (excludes clients) |
| `flags` | `31` | Include all data (default) |

### Request/Response Examples

**Health Check:**
```bash
curl http://localhost:17003/health
# Response: OK
```

**Version Information:**
```bash
curl http://localhost:17003/version
```

**Response:**
```json
{
  "version": "0.1.0-dev",
  "buildInfo": {
    "version": "0.1.0-dev",
    "buildDateTime": "2024-03-18T10:30:45Z",
    "buildTimestamp": 1710759045,
    "environment": "production",
    "service": "brick-clock",
    "description": "Brick Clock NTP Service"
  }
}
```

**Status Information:**
```bash
curl http://localhost:17003/status
```

**Response:**
```json
{
  "tracking": {
    "Reference ID": "202.118.1.130",
    "Stratum": "3",
    "Ref time (UTC)": "Mon Mar 18 10:30:45 2024",
    "System time": "0.000000000 seconds slow of NTP time",
    "Last offset": "+0.000123456 seconds",
    "RMS offset": "0.000123456 seconds",
    "Frequency": "+0.000 ppm",
    "Residual freq": "+0.000 ppm",
    "Skew": "0.000 ppm",
    "Root delay": "0.001234567 seconds",
    "Root dispersion": "0.000123456 seconds",
    "Update interval": "64.0 seconds",
    "Leap status": "Normal"
  },
  "sources": [
    {
      "state": "^",
      "name": "202.118.1.130",
      "stratum": "2",
      "poll": "6",
      "reach": "377",
      "lastrx": "19",
      "offset": "+625ms"
    }
  ],
  "activity": {
    "ok_count": "1234",
    "failed_count": "5",
    "bogus_count": "0",
    "timeout_count": "2"
  },
  "clients": [],
  "server_mode_enabled": true
}
```

**Configure Servers:**
```bash
curl -X PUT http://localhost:17003/servers \
  -H "Content-Type: application/json" \
  -d '{"servers": ["pool.ntp.org", "time.google.com"]}'
```

**Server Mode Control:**
```bash
# Enable server mode
curl -X PUT http://localhost:17003/server-mode \
  -H "Content-Type: application/json" \
  -d '{"enabled": true}'

# Disable server mode
curl -X PUT http://localhost:17003/server-mode \
  -H "Content-Type: application/json" \
  -d '{"enabled": false}'
```

**Response:**
```json
{
  "success": true,
  "server_mode_enabled": true
}
```

## 🔧 Configuration

### NTP Configuration

The service uses a custom NTP configuration with these key settings:

```conf
# Upstream NTP server
server pool.ntp.org iburst

# Allow all clients (server mode)
allow 0.0.0.0/0

# Local stratum for fallback
local stratum 10

# NTP port
port 123

# Log settings
log measurements statistics tracking
```

### Environment Variables

| Variable | Default | Description |
|----------|---------|-------------|
| `VERSION` | `0.1.0-dev` | Application version |
| `BUILD_DATETIME` | Current time | Build timestamp |
| `IMAGE_NAME` | `el/brick-clock` | Docker image name |
| `CONTAINER_NAME` | `el-brick-clock` | Docker container name |
| `API_PORT` | `17003` | API server port |
| `NTP_PORT` | `123` | NTP server port |

## 🌐 Network Ports

| Port | Protocol | Purpose |
|------|----------|---------|
| `123` | UDP | NTP server/client traffic |
| `17003` | TCP | HTTP API server |

## 🐳 Docker Deployment

### Build Image

```bash
./scripts/build.sh [version]
```

**Examples:**
```bash
./scripts/build.sh                    # Build with default version (0.1.0-dev)
./scripts/build.sh 1.0.0             # Build with specific version
```

### Run Container

```bash
./scripts/run.sh [version]
```

**Examples:**
```bash
./scripts/run.sh                     # Run with default version
./scripts/run.sh 1.0.0              # Run with specific version
```

### Docker Compose

```yaml
version: '3.8'
services:
  brick-clock:
    image: el/brick-clock:latest
    container_name: el-brick-clock
    ports:
      - "123:123/udp"
      - "17003:17003"
    restart: unless-stopped
    privileged: true
    volumes:
      - /etc/localtime:/etc/localtime:ro
    environment:
      - VERSION=0.1.0-dev
```

## 🔍 Monitoring & Troubleshooting

### Check Service Status

```bash
# Container status
./scripts/quick_start.sh status

# View logs
./scripts/quick_start.sh logs

# Test API
curl http://localhost:17003/health
curl http://localhost:17003/status
```

### Common Issues

1. **Port Conflicts**: Ensure ports 123/UDP and 17003/TCP are available
2. **Network Access**: Verify connectivity to NTP servers
3. **Permissions**: Container needs root access for NTP operations
4. **Time Sync**: Check if system time is reasonably accurate
5. **API Not Responding**: Wait for service to fully start (up to 30 seconds)

### Log Locations

- **Application Logs**: Docker container logs
- **NTP Logs**: `/var/log/ntp/` (inside container)

### Health Checks

```bash
# Basic health check
curl http://localhost:17003/health

# Detailed status check
curl http://localhost:17003/status?flags=23

# Test all endpoints
./scripts/test.sh
```

## 🏗️ Architecture

### Service Components

- **API Server**: Go HTTP server on port 17003
- **NTP Daemon**: Background NTP service on port 123
- **Configuration Management**: Dynamic server configuration
- **Caching Layer**: In-memory cache for performance (30s TTL)
- **Health Monitoring**: Built-in health checks

### Data Flow

1. **Client Requests** → API Server (port 17003)
2. **API Server** → NTP Daemon (internal communication)
3. **NTP Daemon** → Upstream NTP servers (port 123)
4. **Response** → Client via API

### Caching Strategy

- **Tracking Data**: 30-second TTL
- **Sources Data**: 30-second TTL
- **Activity Data**: 30-second TTL
- **Server Mode**: 5-second TTL
- **Clients Data**: 30-second TTL

## 🔒 Security Considerations

- **Network**: Use VPN for secure NTP communication
- **Authentication**: Consider implementing API authentication
- **Updates**: Regularly update NTP for security patches
- **Firewall**: Restrict access to necessary ports only
- **Container Security**: Run with minimal required privileges

## 🚀 Performance

- **Response Time**: < 100ms for API calls
- **Memory Usage**: ~50MB container footprint
- **CPU Usage**: Minimal during normal operation
- **Network**: Efficient NTP packet handling
- **Caching**: Reduces NTP command overhead

## 🧪 Testing

### Automated Testing

```bash
# Run all tests
./scripts/test.sh

# Test with custom host
./scripts/test.sh localhost:17003

# Test with remote host
./scripts/test.sh api.example.com:17003
```

### Manual Testing

```bash
# Health check
curl http://localhost:17003/health

# Version info
curl http://localhost:17003/version

# Status with specific flags
curl "http://localhost:17003/status?flags=23"

# Configure servers
curl -X PUT http://localhost:17003/servers \
  -H "Content-Type: application/json" \
  -d '{"servers": ["pool.ntp.org"]}'
```

## 🤝 Contributing

1. Fork the repository
2. Create a feature branch
3. Make your changes
4. Add tests if applicable
5. Run the test suite: `./scripts/test.sh`
6. Submit a pull request

## 📄 License

This project is licensed under the MIT License - see the LICENSE file for details.

## 🆘 Support

For issues and questions:
- Check the troubleshooting section above
- Review the logs: `./scripts/quick_start.sh logs`
- Test the API endpoints manually
- Open an issue on GitHub

## 📝 Changelog

### Version 0.1.0-dev
- Initial release
- NTP client and server capabilities
- RESTful API for management
- Docker containerization
- Caching layer for performance
- Comprehensive testing suite

---

**Version**: 0.1.0-dev  
**Last Updated**: July 2025