# CycleTLS-Proxy Implementation Requirements

## Overview
CycleTLS-Proxy is a lightweight HTTP proxy server that provides TLS fingerprinting capabilities via CycleTLS. It accepts HTTP requests with special headers to control TLS behavior and proxies them with specified fingerprints.

## Core Functionality (Based on VowTLS Design)

The proxy operates on a simple principle:
1. Listen on a local port (default: 8080)
2. Accept HTTP requests (POST or GET)
3. Extract configuration from request headers
4. Forward request using CycleTLS with specified fingerprint
5. Return response to client

### Required Headers
```
X-URL: https://example.com           # Target URL (required)
X-PROXY: http://user:pass@ip:port    # Upstream proxy (optional)
X-IDENTIFIER: chrome|firefox|safari_ios|safari|okhttp  # Browser profile
X-SESSION-ID: unique-session-id      # Session tracking for connection reuse
```

### Optional Headers
Any additional headers will be forwarded to the target URL.

## Implementation Checklist for LLM

### Phase 1: Core Proxy Server
- [ ] Create main.go in cmd/proxy/
- [ ] Set up FastHTTP server listening on port 8080 (configurable via PORT env)
- [ ] Implement request handler that extracts X-* headers
- [ ] Parse X-URL header as target destination
- [ ] Parse X-IDENTIFIER to select TLS fingerprint profile
- [ ] Forward request using CycleTLS with appropriate settings
- [ ] Return response maintaining status code and headers
- [ ] Handle errors gracefully with appropriate HTTP status codes

### Phase 2: CycleTLS Integration
- [ ] Import CycleTLS as dependency
- [ ] Create fingerprint profiles map for each identifier:
  - chrome: Latest Chrome fingerprints (JA3, JA4, HTTP/2)
  - firefox: Latest Firefox fingerprints
  - safari_ios: iOS Safari fingerprints
  - safari: macOS Safari fingerprints
  - okhttp: Android OkHttp fingerprints
- [ ] Implement session management using X-SESSION-ID for connection reuse
- [ ] Pass through X-PROXY to CycleTLS proxy configuration
- [ ] Handle timeouts (default 30s, configurable)

### Phase 3: Request Processing
- [ ] Strip X-* configuration headers before forwarding
- [ ] Forward all other headers to target
- [ ] Handle both GET and POST methods
- [ ] Support request body forwarding for POST
- [ ] Preserve Content-Type and Content-Length
- [ ] Handle response streaming for large responses
- [ ] Automatic decompression of gzip/deflate/br

### Phase 4: GitHub Actions & Releases

Create `.github/workflows/release.yml`:

```yaml
name: Release

on:
  push:
    tags:
      - 'v*'

jobs:
  release:
    runs-on: ubuntu-latest
    steps:
      - uses: actions/checkout@v4
      
      - name: Set up Go
        uses: actions/setup-go@v5
        with:
          go-version: '1.21'
      
      - name: Build binaries
        run: |
          # Build for all platforms
          GOOS=linux GOARCH=amd64 go build -o dist/cycletls-proxy-linux-amd64 ./cmd/proxy
          GOOS=linux GOARCH=arm64 go build -o dist/cycletls-proxy-linux-arm64 ./cmd/proxy
          GOOS=darwin GOARCH=amd64 go build -o dist/cycletls-proxy-darwin-amd64 ./cmd/proxy
          GOOS=darwin GOARCH=arm64 go build -o dist/cycletls-proxy-darwin-arm64 ./cmd/proxy
          GOOS=windows GOARCH=amd64 go build -o dist/cycletls-proxy-windows-amd64.exe ./cmd/proxy
          
          # Create archives
          cd dist
          tar -czf cycletls-proxy-linux-amd64.tar.gz cycletls-proxy-linux-amd64
          tar -czf cycletls-proxy-linux-arm64.tar.gz cycletls-proxy-linux-arm64
          tar -czf cycletls-proxy-darwin-amd64.tar.gz cycletls-proxy-darwin-amd64
          tar -czf cycletls-proxy-darwin-arm64.tar.gz cycletls-proxy-darwin-arm64
          zip cycletls-proxy-windows-amd64.zip cycletls-proxy-windows-amd64.exe
          
          # Generate checksums
          sha256sum *.tar.gz *.zip > checksums.txt
      
      - name: Create Release
        uses: softprops/action-gh-release@v1
        with:
          files: |
            dist/*.tar.gz
            dist/*.zip
            dist/checksums.txt
          generate_release_notes: true
          draft: false
          prerelease: false
```

### Phase 5: Installation Methods

#### Install Script (install.sh)
Create an installation script that detects platform and downloads appropriate binary:

```bash
#!/bin/bash
# Add to repository root as install.sh

VERSION="${1:-latest}"
INSTALL_DIR="${2:-/usr/local/bin}"

# Detect OS and architecture
OS=$(uname -s | tr '[:upper:]' '[:lower:]')
ARCH=$(uname -m)

case "$ARCH" in
    x86_64) ARCH="amd64" ;;
    aarch64|arm64) ARCH="arm64" ;;
    *) echo "Unsupported architecture: $ARCH"; exit 1 ;;
esac

# Construct download URL
if [ "$VERSION" = "latest" ]; then
    VERSION=$(curl -s https://api.github.com/repos/yourusername/cycletls-proxy/releases/latest | grep '"tag_name"' | cut -d'"' -f4)
fi

BINARY="cycletls-proxy-${OS}-${ARCH}"
if [ "$OS" = "windows" ]; then
    BINARY="${BINARY}.exe"
fi

URL="https://github.com/yourusername/cycletls-proxy/releases/download/${VERSION}/${BINARY}.tar.gz"

# Download and install
echo "Downloading CycleTLS-Proxy ${VERSION} for ${OS}/${ARCH}..."
curl -L "$URL" | tar -xz -C /tmp
sudo mv "/tmp/${BINARY}" "${INSTALL_DIR}/cycletls-proxy"
sudo chmod +x "${INSTALL_DIR}/cycletls-proxy"

echo "CycleTLS-Proxy installed successfully!"
```

#### Platform-Specific Installation

**macOS (Homebrew formula to be added later):**
```bash
# Future implementation
brew tap yourusername/cycletls-proxy
brew install cycletls-proxy
```

**Linux (One-liner):**
```bash
curl -sSL https://raw.githubusercontent.com/yourusername/cycletls-proxy/main/install.sh | bash
```

**Windows (PowerShell):**
```powershell
# Create install.ps1
$version = "latest"
$installDir = "$env:LOCALAPPDATA\cycletls-proxy"

# Get latest version
if ($version -eq "latest") {
    $release = Invoke-RestMethod "https://api.github.com/repos/yourusername/cycletls-proxy/releases/latest"
    $version = $release.tag_name
}

# Download
$url = "https://github.com/yourusername/cycletls-proxy/releases/download/$version/cycletls-proxy-windows-amd64.zip"
$output = "$env:TEMP\cycletls-proxy.zip"

Invoke-WebRequest -Uri $url -OutFile $output
Expand-Archive -Path $output -DestinationPath $installDir -Force

# Add to PATH
$path = [Environment]::GetEnvironmentVariable("Path", "User")
if ($path -notlike "*$installDir*") {
    [Environment]::SetEnvironmentVariable("Path", "$path;$installDir", "User")
}

Write-Host "CycleTLS-Proxy installed successfully!"
```

**Docker:**
```dockerfile
# Dockerfile
FROM golang:1.21-alpine AS builder
WORKDIR /app
COPY . .
RUN go mod download
RUN go build -o cycletls-proxy ./cmd/proxy

FROM alpine:latest
RUN apk --no-cache add ca-certificates
COPY --from=builder /app/cycletls-proxy /usr/local/bin/
EXPOSE 8080
CMD ["cycletls-proxy"]
```

### Phase 6: Project Structure

```
CycleTLS-Proxy/
├── README.md                 # User documentation with examples
├── REQUIREMENTS.md          # This file
├── install.sh              # Unix installation script
├── install.ps1             # Windows installation script
├── Dockerfile              # Docker build file
├── go.mod                  # Go module file
├── go.sum                  # Go dependencies
├── cmd/
│   └── proxy/
│       └── main.go        # Entry point
├── internal/
│   ├── proxy/
│   │   └── handler.go     # Request handler
│   ├── fingerprints/
│   │   └── profiles.go    # TLS fingerprint profiles
│   └── cycletls/
│       └── client.go      # CycleTLS wrapper
├── .github/
│   └── workflows/
│       ├── release.yml    # Release automation
│       └── test.yml       # Test on PR
└── examples/
    ├── curl.sh            # cURL examples
    ├── python.py          # Python examples
    └── node.js            # Node.js examples
```

### Phase 7: Core Implementation (main.go)

```go
package main

import (
    "encoding/json"
    "log"
    "os"
    "strings"
    
    "github.com/valyala/fasthttp"
    "github.com/Danny-Dasilva/CycleTLS/cycletls"
)

var (
    port = getEnv("PORT", "8080")
    profiles = map[string]Profile{
        "chrome": {
            JA3: "771,4865-4866-4867-49195-49199-49196-49200-52393-52392-49171-49172-156-157-47-53,0-23-65281-10-11-35-16-5-13-18-51-45-43-27-17513,29-23-24,0",
            JA4: "t13d1517h2_8daaf6152771_7e51fdad25f2",
            UserAgent: "Mozilla/5.0 (Windows NT 10.0; Win64; x64) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/121.0.0.0 Safari/537.36",
        },
        "firefox": {
            JA3: "771,4865-4867-4866-49195-49199-52393-52392-49196-49200-49162-49161-49171-49172-51-57-47-53-10,0-23-65281-10-11-35-16-5-51-43-13-45-28-21,29-23-24-25-256-257,0",
            JA4: "t13d1717h2_5b57614c22b0_f2748d6cd58d",
            UserAgent: "Mozilla/5.0 (X11; Ubuntu; Linux x86_64; rv:141.0) Gecko/20100101 Firefox/141.0",
        },
        // Add more profiles...
    }
    sessions = make(map[string]*cycletls.CycleTLS)
)

type Profile struct {
    JA3       string
    JA4       string
    UserAgent string
}

func main() {
    log.Printf("CycleTLS-Proxy starting on :%s", port)
    
    if err := fasthttp.ListenAndServe(":"+port, requestHandler); err != nil {
        log.Fatalf("Error starting server: %s", err)
    }
}

func requestHandler(ctx *fasthttp.RequestCtx) {
    // Extract required headers
    targetURL := string(ctx.Request.Header.Peek("X-URL"))
    if targetURL == "" {
        ctx.Error("X-URL header is required", fasthttp.StatusBadRequest)
        return
    }
    
    identifier := string(ctx.Request.Header.Peek("X-IDENTIFIER"))
    if identifier == "" {
        identifier = "chrome" // Default
    }
    
    sessionID := string(ctx.Request.Header.Peek("X-SESSION-ID"))
    proxy := string(ctx.Request.Header.Peek("X-PROXY"))
    
    // Get profile
    profile, ok := profiles[identifier]
    if !ok {
        ctx.Error("Invalid identifier", fasthttp.StatusBadRequest)
        return
    }
    
    // Get or create CycleTLS client for session
    client := getClient(sessionID)
    
    // Build request options
    options := cycletls.Options{
        Ja3:       profile.JA3,
        Ja4:       profile.JA4,
        UserAgent: profile.UserAgent,
        Proxy:     proxy,
        Timeout:   30,
    }
    
    // Forward headers (except X-* config headers)
    headers := make(map[string]string)
    ctx.Request.Header.VisitAll(func(key, value []byte) {
        k := string(key)
        if !strings.HasPrefix(strings.ToUpper(k), "X-") {
            headers[k] = string(value)
        }
    })
    options.Headers = headers
    
    // Make request
    method := string(ctx.Method())
    body := string(ctx.Request.Body())
    
    response, err := client.Do(targetURL, options, method)
    if err != nil {
        ctx.Error("Request failed: "+err.Error(), fasthttp.StatusBadGateway)
        return
    }
    
    // Return response
    ctx.SetStatusCode(response.Status)
    for k, v := range response.Headers {
        ctx.Response.Header.Set(k, v)
    }
    ctx.SetBody([]byte(response.Body))
}

func getClient(sessionID string) *cycletls.CycleTLS {
    if sessionID != "" {
        if client, exists := sessions[sessionID]; exists {
            return client
        }
    }
    client := cycletls.Init()
    if sessionID != "" {
        sessions[sessionID] = &client
    }
    return &client
}

func getEnv(key, defaultValue string) string {
    if value := os.Getenv(key); value != "" {
        return value
    }
    return defaultValue
}
```

## Usage Examples

### cURL
```bash
# Basic request
curl -X POST http://localhost:8080 \
  -H "X-URL: https://tls.peet.ws/api/all" \
  -H "X-IDENTIFIER: chrome"

# With proxy
curl -X POST http://localhost:8080 \
  -H "X-URL: https://example.com" \
  -H "X-PROXY: http://user:pass@proxy:8080" \
  -H "X-IDENTIFIER: firefox" \
  -H "X-SESSION-ID: session123"

# With custom headers
curl -X POST http://localhost:8080 \
  -H "X-URL: https://api.example.com/data" \
  -H "X-IDENTIFIER: okhttp" \
  -H "Authorization: Bearer token123" \
  -H "Content-Type: application/json" \
  -d '{"key": "value"}'
```

### Python
```python
import requests

def make_request(url, identifier="chrome", proxy=None, session_id=None):
    headers = {
        "X-URL": url,
        "X-IDENTIFIER": identifier
    }
    
    if proxy:
        headers["X-PROXY"] = proxy
    if session_id:
        headers["X-SESSION-ID"] = session_id
    
    response = requests.post("http://localhost:8080", headers=headers)
    return response

# Example usage
resp = make_request("https://example.com", identifier="firefox")
print(resp.text)
```

### Node.js
```javascript
const axios = require('axios');

async function makeRequest(url, options = {}) {
    const headers = {
        'X-URL': url,
        'X-IDENTIFIER': options.identifier || 'chrome',
        ...options.headers
    };
    
    if (options.proxy) headers['X-PROXY'] = options.proxy;
    if (options.sessionId) headers['X-SESSION-ID'] = options.sessionId;
    
    const response = await axios({
        method: options.method || 'POST',
        url: 'http://localhost:8080',
        headers: headers,
        data: options.data
    });
    
    return response.data;
}

// Example usage
makeRequest('https://example.com', {
    identifier: 'safari',
    sessionId: 'my-session'
}).then(console.log);
```

## Testing Checklist

- [ ] Test basic proxy functionality with each identifier
- [ ] Verify TLS fingerprints at https://tls.peet.ws/api/all
- [ ] Test session persistence with X-SESSION-ID
- [ ] Verify upstream proxy support
- [ ] Test POST requests with body
- [ ] Test header forwarding
- [ ] Verify error handling for invalid URLs
- [ ] Test timeout behavior
- [ ] Benchmark performance (target: 1000+ RPS)

## Release Process

1. **Version Tagging**: 
   ```bash
   git tag -a v1.0.0 -m "Release version 1.0.0"
   git push origin v1.0.0
   ```

2. **GitHub Actions** automatically:
   - Builds binaries for all platforms
   - Creates tar.gz/zip archives
   - Generates SHA256 checksums
   - Creates GitHub release with artifacts
   - Publishes Docker image

3. **Users can install via**:
   - Direct download from GitHub releases
   - Install script: `curl -sSL install.sh | bash`
   - Docker: `docker run -p 8080:8080 cycletls-proxy`
   - Platform package managers (future)

## Success Criteria

1. **Functionality**: Matches VowTLS behavior exactly
2. **Performance**: Handles 1000+ requests/second
3. **Simplicity**: Single binary, no dependencies
4. **Compatibility**: Works on Linux, macOS, Windows
5. **Size**: Binary under 20MB
6. **Installation**: One-command install on all platforms

## Notes for Implementation

- Use FastHTTP for maximum performance
- Keep dependencies minimal (only CycleTLS and FastHTTP)
- Ensure binary is statically linked for portability
- Include ASCII art banner on startup (like VowTLS)
- Default to Chrome fingerprint if X-IDENTIFIER not provided
- Log requests in Apache Combined Log Format
- Graceful shutdown on SIGINT/SIGTERM
- Support both HTTP and HTTPS proxy modes in future versions

---

This document provides complete requirements for implementing CycleTLS-Proxy as a simple, efficient proxy server with TLS fingerprinting capabilities. The implementation should be straightforward and focus on core functionality rather than complex features.