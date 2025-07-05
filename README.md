# Brick Clock Service

A high-precision Network Time Protocol (NTP) service built with Go and Chrony, providing both client and server capabilities for time synchronization in distributed systems.

## 🚀 Features

- **NTP Client Mode**: Synchronize with upstream NTP servers
- **NTP Server Mode**: Act as a time source for other devices
- **RESTful API**: Full HTTP API for monitoring and management
- **Real-time Status**: Live tracking of synchronization status
- **Server Management**: Add, remove, and configure NTP servers
- **Activity Monitoring**: Track success/failure statistics
- **Docker Ready**: Containerized deployment with Alpine Linux

## 📋 Prerequisites

- Docker and Docker Compose
- Linux environment (for chrony compatibility)
- Network access to NTP servers

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
./scripts/quick_start.sh [command]
```

**Commands:**
- `build` - Build Docker image only
- `run` - Run container only  
- `test` - Test API endpoints only
- `clean` - Stop and remove containers
- `logs` - Show container logs
- `status` - Check container status
- `all` - Full cycle (default)

### Individual Scripts

| Script | Purpose | Usage |
|--------|---------|-------|
| `build.sh` | Build Docker image | `./scripts/build.sh [version]` |
| `run.sh` | Start container | `./scripts/run.sh [version]` |
| `test.sh` | Test API endpoints | `./scripts/test.sh [host:port]` |
| `clean.sh` | Clean up resources | `./scripts/clean.sh` |
| `config.sh` | Configuration management | `./scripts/config.sh` |

## 🔌 API Endpoints

### Core Endpoints

| Method | Endpoint | Description |
|--------|----------|-------------|
| `GET` | `/version` | Application version and build info |
| `GET` | `/chrony/version` | Chrony daemon version |
| `GET` | `/chrony/status` | Current synchronization status |
| `GET` | `/chrony/servers` | List configured NTP servers |
| `PUT` | `/chrony/servers` | Configure NTP servers |
| `DELETE` | `/chrony/servers` | Reset to default servers |
| `PUT` | `/chrony/servers/default` | Set default NTP servers |

### Response Examples

**Status Response:**
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
  }
}
```

## 🔧 Configuration

### Chrony Configuration

The service uses a custom `chrony.conf` with these key settings:

```conf
# Upstream NTP server
server pool.ntp.org iburst

# Allow all clients (server mode)
allow 0.0.0.0/0

# Local stratum for fallback
local stratum 10

# NTP port
port 123
```

### Environment Variables

| Variable | Default | Description |
|----------|---------|-------------|
| `VERSION` | `0.1.0-dev` | Application version |
| `BUILD_DATETIME` | Current time | Build timestamp |
| `IMAGE_NAME` | `el/brick-clock` | Docker image name |
| `CONTAINER_NAME` | `el-brick-clock` | Docker container name |

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

## 🔍 Monitoring & Troubleshooting

### Check Service Status

```bash
# Container status
./scripts/quick_start.sh status

# View logs
./scripts/quick_start.sh logs

# Test API
curl http://localhost:17003/chrony/status
```

### Common Issues

1. **Port Conflicts**: Ensure ports 123/UDP and 17003/TCP are available
2. **Network Access**: Verify connectivity to NTP servers
3. **Permissions**: Container needs root access for chrony operations

### Log Locations

- **Application Logs**: Docker container logs
- **Chrony Logs**: `/var/log/chrony/` (inside container)

## 🏗️ Architecture

```
┌─────────────────┐    ┌─────────────────┐    ┌─────────────────┐
│   HTTP Client   │    │   NTP Client    │    │   NTP Server    │
│   (Port 17003)  │    │   (Port 123)    │    │   (Port 123)    │
└─────────────────┘    └─────────────────┘    └─────────────────┘
         │                       │                       │
         └───────────────────────┼───────────────────────┘
                                 │
                    ┌─────────────────┐
                    │   Go API App    │
                    │  (brick-clock)   │
                    └─────────────────┘
```

## 📈 Performance

- **Accuracy**: Sub-millisecond precision
- **Latency**: Minimal overhead for time queries
- **Scalability**: Supports multiple concurrent clients
- **Reliability**: Automatic failover between NTP servers

## 🔒 Security Considerations

- **Firewall**: Restrict NTP port access in production
- **Authentication**: Consider implementing API authentication
- **Network**: Use VPN for secure NTP communication
- **Updates**: Regularly update chrony for security patches

## 🤝 Contributing

1. Fork the repository
2. Create a feature branch
3. Make your changes
4. Test thoroughly
5. Submit a pull request

## 📄 License

This project is part of the Brick ecosystem. See the main repository for license information.

## 📞 Support

For issues and questions:
- Check the troubleshooting section above
- Review the API documentation
- Open an issue in the repository

---

**Version**: 0.1.0-dev  
**Last Updated**: July 2025