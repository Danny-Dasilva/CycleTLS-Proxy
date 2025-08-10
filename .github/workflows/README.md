# GitHub Actions Workflows

This directory contains GitHub Actions workflows for the CycleTLS-Proxy project. These workflows provide comprehensive CI/CD, testing, and release automation.

## Workflows Overview

### 🧪 test.yml - CI/CD Testing Pipeline
Runs comprehensive tests on pull requests and pushes to main/develop branches.

**Triggers:**
- Push to `main` or `develop` branches
- Pull requests targeting `main` or `develop`
- Manual dispatch

**Features:**
- Multi-platform testing (Linux, macOS, Windows)
- Multi-version Go testing (1.21, 1.22, 1.23, 1.24)
- Code linting with golangci-lint
- Security vulnerability scanning
- Integration tests with real network calls
- Build verification for all target platforms
- Dependency checks and validation
- Performance benchmarking (main branch only)
- Code coverage reporting

**Jobs:**
- `lint` - Code quality and security analysis
- `test` - Unit tests across OS and Go version matrix
- `build-test` - Cross-compilation verification
- `integration-test` - Real-world functionality testing
- `security-scan` - Vulnerability detection
- `dependency-check` - Module integrity verification
- `benchmark` - Performance analysis (main branch)

### 🚀 release.yml - Automated Release Pipeline
Creates production releases with binaries and Docker images when version tags are pushed.

**Triggers:**
- Version tags matching `v*` (e.g., `v1.0.0`)
- Manual dispatch with version input

**Features:**
- Multi-platform binary builds (Linux, macOS, Windows × amd64/arm64)
- Proper handling of local CycleTLS dependency
- Compressed archives with SHA256 checksums
- Multi-architecture Docker images
- GitHub release creation with detailed notes
- Docker Hub and GitHub Container Registry publishing

**Artifacts:**
- `cycletls-proxy-VERSION-PLATFORM-ARCH.tar.gz` (Unix-like)
- `cycletls-proxy-VERSION-PLATFORM-ARCH.zip` (Windows)
- `dannydasilva/cycletls-proxy:VERSION` (Docker Hub)
- `ghcr.io/danny-dasilva/cycletls-proxy:VERSION` (GHCR)

### 🐳 docker.yml - Docker Build & Publish
Dedicated Docker image building and testing workflow.

**Triggers:**
- Push to `main` or `develop` branches
- Version tags matching `v*`
- Pull requests (build only, no push)
- Manual dispatch with push option

**Features:**
- Multi-platform Docker builds (linux/amd64, linux/arm64)
- Security scanning with Trivy
- Image testing and validation
- Automated cleanup of old images
- Dual registry publishing (Docker Hub + GHCR)

**Image Tags:**
- `latest` (main branch)
- `develop` (develop branch)
- `VERSION` (version tags)
- `BRANCH-SHA` (all branches)

## Required Secrets

The workflows require the following repository secrets:

### Docker Registry Access
- `DOCKER_USERNAME` - Docker Hub username
- `DOCKER_PASSWORD` - Docker Hub password/token

### Optional Enhancements
- `CODECOV_TOKEN` - For enhanced coverage reporting

### Automatic Secrets
- `GITHUB_TOKEN` - Automatically provided by GitHub Actions

## Workflow Dependencies

All workflows handle the local CycleTLS dependency properly by:
1. Checking out the main repository
2. Checking out the CycleTLS dependency to `./CycleTLS`
3. Updating `go.mod` to use the local path: `go mod edit -replace github.com/Danny-Dasilva/CycleTLS/cycletls=./CycleTLS/cycletls`
4. Running `go mod tidy` to resolve dependencies

## Local Development

To run similar checks locally:

```bash
# Install golangci-lint
go install github.com/golangci/golangci-lint/cmd/golangci-lint@latest

# Run linting
golangci-lint run

# Run tests with race detection
go test -race ./...

# Run tests with coverage
go test -cover -coverprofile=coverage.out ./...

# Build for different platforms
GOOS=linux GOARCH=amd64 go build -o dist/cycletls-proxy-linux-amd64 ./cmd/proxy
GOOS=darwin GOARCH=arm64 go build -o dist/cycletls-proxy-darwin-arm64 ./cmd/proxy
```

## Workflow Status

You can monitor workflow status in the GitHub Actions tab of the repository. Each workflow provides detailed logs and summaries for troubleshooting.

### Success Criteria

**Test Workflow:**
- All linting checks pass
- Unit tests pass on all OS/Go version combinations
- Integration tests complete successfully
- Security scans find no critical vulnerabilities
- Build verification succeeds for all platforms

**Release Workflow:**
- Binaries build successfully for all platforms
- Docker images build and push to registries
- GitHub release created with all assets

**Docker Workflow:**
- Multi-platform images build successfully
- Security scans pass
- Image functionality tests complete