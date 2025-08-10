# CycleTLS-Proxy Deployment Guide

This document provides comprehensive instructions for installing and deploying CycleTLS-Proxy using the provided installation scripts and Docker configurations.

## Installation Scripts

### Linux/macOS Installation (`install.sh`)

The `install.sh` script provides automatic installation for Linux and macOS systems.

#### Features
- Auto-detects OS (Linux, macOS) and architecture (amd64, arm64, arm, 386)
- Downloads latest release from GitHub automatically
- Installs binary to `/usr/local/bin/` with proper permissions
- Handles errors gracefully with user-friendly messages
- Supports custom version and install directory
- Adds install directory to PATH if needed

#### Usage

```bash
# Install latest version
./install.sh

# Install specific version
./install.sh -v v1.2.3

# Install to custom directory
./install.sh -d ~/bin

# Use GitHub token for private repos
./install.sh -t ghp_xxxxxxxxxxxx

# Show help
./install.sh -h
```

#### Requirements
- Linux or macOS
- `curl` and `tar` commands available
- Internet connection
- Write permissions to install directory (may need sudo)

### Windows Installation (`install.ps1`)

The `install.ps1` script provides installation for Windows systems using PowerShell.

#### Features
- Downloads and extracts Windows binary automatically
- Installs to `%LOCALAPPDATA%\cycletls-proxy`
- Adds to PATH environment variable
- Handles Windows-specific operations
- Supports PowerShell 5.0+ and PowerShell Core

#### Usage

```powershell
# Install latest version
.\install.ps1

# Install specific version
.\install.ps1 -Version v1.2.3

# Install to custom directory
.\install.ps1 -InstallDir C:\Tools

# Use GitHub token
.\install.ps1 -GitHubToken ghp_xxxxxxxxxxxx

# Show help
.\install.ps1 -Help
```

#### Requirements
- Windows PowerShell 5.0+ or PowerShell Core 6.0+
- Internet connection
- Administrator privileges (recommended for PATH modification)

## Docker Deployment

### Basic Docker Usage

Build and run the container:

```bash
# Build the image
docker build -t cycletls-proxy .

# Run the container
docker run -d -p 8080:8080 --name cycletls-proxy cycletls-proxy
```

### Docker Compose Deployment

The `docker-compose.yml` provides multiple deployment scenarios:

#### Development Mode

```bash
# Start development environment with hot reload
docker-compose --profile dev up -d cycletls-proxy-dev

# View logs
docker-compose logs -f cycletls-proxy-dev
```

#### Production Mode

```bash
# Start production environment with nginx reverse proxy
docker-compose --profile production up -d cycletls-proxy nginx

# With SSL certificates (place cert.pem and key.pem in ./ssl/)
docker-compose --profile production up -d
```

#### With Monitoring

```bash
# Start with Prometheus and Grafana monitoring
docker-compose --profile production --profile monitoring up -d

# Access Grafana at http://localhost:3000 (admin/admin)
# Access Prometheus at http://localhost:9090
```

#### Basic Usage

```bash
# Start just the proxy service
docker-compose up -d cycletls-proxy

# Build and start
docker-compose up --build -d

# Stop all services
docker-compose down

# Stop and remove volumes
docker-compose down -v
```

### Docker Configuration

#### Environment Variables

| Variable | Default | Description |
|----------|---------|-------------|
| `PORT` | `8080` | Port to listen on |
| `LOG_LEVEL` | `info` | Logging level (debug, info, warn, error) |

#### Health Check

The container includes a health check endpoint at `/health` that can be used by orchestrators like Docker Compose, Kubernetes, or load balancers.

#### Security Features

- Runs as non-root user (uid: 1000)
- Uses minimal Alpine Linux base image
- Includes security options in docker-compose
- Resource limits configured

## Production Deployment

### SSL/TLS Configuration

For production deployments with SSL:

1. Place your SSL certificates in a `./ssl/` directory:
   - `cert.pem` - SSL certificate
   - `key.pem` - Private key

2. Update `nginx.conf` with your domain name

3. Start with the production profile:
   ```bash
   docker-compose --profile production up -d
   ```

### Reverse Proxy Setup

The included `nginx.conf` provides:
- SSL termination
- HTTP to HTTPS redirect
- Rate limiting
- Security headers
- Gzip compression
- Health check endpoints

### Monitoring Setup

Optional monitoring stack includes:
- **Prometheus** - Metrics collection
- **Grafana** - Visualization dashboard

Access after deployment:
- Grafana: http://localhost:3000 (admin/admin)
- Prometheus: http://localhost:9090

### Resource Requirements

#### Minimum Requirements
- CPU: 0.5 cores
- Memory: 256MB
- Disk: 100MB

#### Recommended for Production
- CPU: 2 cores
- Memory: 1GB
- Disk: 1GB (for logs and monitoring)

## Development

### Hot Reload Development

For development with automatic code reloading:

1. Install Air (Go hot reload tool):
   ```bash
   go install github.com/cosmtrek/air@latest
   ```

2. Start development environment:
   ```bash
   docker-compose --profile dev up -d cycletls-proxy-dev
   ```

3. Code changes will automatically trigger rebuilds

### Local Development Without Docker

```bash
# Install dependencies
go mod download

# Run locally
go run ./cmd/proxy

# Or build and run
go build -o cycletls-proxy ./cmd/proxy
./cycletls-proxy
```

## Troubleshooting

### Common Issues

1. **Port already in use**
   ```bash
   # Change port in docker-compose.yml or use different port
   docker-compose up -d -p 8081:8080
   ```

2. **Permission denied during installation**
   ```bash
   # Run with sudo or choose different install directory
   sudo ./install.sh
   # OR
   ./install.sh -d ~/bin
   ```

3. **SSL certificate issues**
   - Ensure certificates are in PEM format
   - Check file permissions (readable by nginx container)
   - Verify certificate chain is complete

4. **Memory issues**
   - Increase Docker memory limits in docker-compose.yml
   - Monitor resource usage with `docker stats`

### Logs

View application logs:
```bash
# Docker Compose
docker-compose logs -f cycletls-proxy

# Docker
docker logs -f cycletls-proxy

# Follow logs in real-time
docker-compose logs -f --tail=100 cycletls-proxy
```

### Health Checks

Test the health endpoint:
```bash
# Direct connection
curl http://localhost:8080/health

# Through nginx (production)
curl https://your-domain.com/health
```

## Security Considerations

1. **Use HTTPS in production** - The nginx configuration enforces HTTPS
2. **Rate limiting** - Configured in nginx to prevent abuse
3. **Non-root execution** - Container runs as non-privileged user
4. **Security headers** - HSTS, XSS protection, etc. configured
5. **Regular updates** - Keep base images and dependencies updated

## Support

For issues and questions:
- Check the main README.md for basic usage
- Review logs for error messages  
- Open an issue on GitHub with detailed information

## File Structure

After deployment, your directory should look like:

```
CycleTLS-Proxy/
├── install.sh              # Linux/macOS installer
├── install.ps1             # Windows installer  
├── Dockerfile              # Multi-stage Docker build
├── docker-compose.yml      # Compose configuration
├── .dockerignore          # Docker build ignore rules
├── nginx.conf             # Nginx reverse proxy config
├── .air.toml              # Hot reload configuration
├── DEPLOYMENT.md          # This file
└── ssl/                   # SSL certificates (create if needed)
    ├── cert.pem
    └── key.pem
```