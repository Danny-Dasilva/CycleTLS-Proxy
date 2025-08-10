# CycleTLS-Proxy Usage Guide

This document explains how to use the CycleTLS-Proxy server with its enhanced request handler and session management capabilities.

## Overview

CycleTLS-Proxy is a FastHTTP-based proxy server that enables TLS fingerprint spoofing using the CycleTLS library. It provides:

- **Multiple TLS Fingerprint Profiles**: Chrome, Firefox, Safari, Edge, and more
- **Session Management**: Connection reuse with automatic cleanup
- **Streaming Support**: Handle large responses efficiently  
- **Comprehensive Error Handling**: Detailed validation and logging
- **Configurable Timeouts**: Per-request and default timeout settings

## Quick Start

1. **Start the server:**
   ```bash
   go run ./cmd/proxy/main.go
   ```

2. **Make a request:**
   ```bash
   curl -H "X-URL: https://httpbin.org/get" \
        -H "X-IDENTIFIER: chrome" \
        "http://localhost:8080"
   ```

## Request Headers

### Required Headers

- **X-URL**: Target URL to proxy the request to
  ```bash
  -H "X-URL: https://example.com/api/endpoint"
  ```

### Optional Headers

- **X-IDENTIFIER**: TLS fingerprint profile to use (default: `chrome`)
  ```bash
  -H "X-IDENTIFIER: firefox"
  ```

- **X-SESSION-ID**: Session identifier for connection reuse
  ```bash
  -H "X-SESSION-ID: my-session-123"
  ```

- **X-PROXY**: Proxy server to use
  ```bash
  -H "X-PROXY: http://proxy.example.com:8080"
  ```

- **X-TIMEOUT**: Request timeout in seconds (1-300, default: 30)
  ```bash
  -H "X-TIMEOUT: 60"
  ```

## Available Profiles

| Profile | Description | Platform |
|---------|-------------|----------|
| `chrome` | Chrome 120 (default) | Linux |
| `chrome_windows` | Chrome 120 | Windows |
| `firefox` | Firefox 121 | Linux |
| `firefox_windows` | Firefox 121 | Windows |
| `safari` | Safari 17 | macOS |
| `safari_ios` | Safari 17.1.1 | iOS |
| `edge` | Edge 120 | Windows |
| `okhttp` | OkHttp 4.12.0 | Android |
| `chrome_legacy_tls12` | Chrome 91 with TLS 1.2 | Windows |

## Example Requests

### Basic GET Request
```bash
curl -H "X-URL: https://httpbin.org/get" \
     -H "X-IDENTIFIER: firefox" \
     "http://localhost:8080"
```

### POST Request with JSON Body
```bash
curl -X POST \
     -H "X-URL: https://httpbin.org/post" \
     -H "X-IDENTIFIER: chrome" \
     -H "Content-Type: application/json" \
     -d '{"key": "value"}' \
     "http://localhost:8080"
```

### Request with Custom Headers
```bash
curl -H "X-URL: https://api.example.com/data" \
     -H "X-IDENTIFIER: safari" \
     -H "Authorization: Bearer token123" \
     -H "Custom-Header: custom-value" \
     "http://localhost:8080"
```

### Session Reuse Example
```bash
# First request creates session
curl -H "X-URL: https://httpbin.org/cookies/set?session=abc123" \
     -H "X-SESSION-ID: my-session" \
     "http://localhost:8080"

# Second request reuses the same session (maintains cookies)
curl -H "X-URL: https://httpbin.org/cookies" \
     -H "X-SESSION-ID: my-session" \
     "http://localhost:8080"
```

### Request with Proxy
```bash
curl -H "X-URL: https://httpbin.org/ip" \
     -H "X-IDENTIFIER: chrome" \
     -H "X-PROXY: http://proxy.example.com:8080" \
     "http://localhost:8080"
```

### Custom Timeout
```bash
curl -H "X-URL: https://httpbin.org/delay/5" \
     -H "X-TIMEOUT: 10" \
     "http://localhost:8080"
```

## Response Format

The proxy returns responses exactly as received from the target server, including:

- **Status Code**: Same as target server
- **Headers**: All headers except connection-specific ones
- **Body**: Complete response body with streaming support

## Session Management

### Automatic Session Creation
- Sessions are created automatically when you provide an `X-SESSION-ID`
- Each session maintains its own connection pool and cookies

### Session Reuse Benefits
- **Connection Pooling**: Faster subsequent requests
- **Cookie Persistence**: Maintains login state
- **TLS Session Resumption**: Reduced handshake overhead

### Automatic Cleanup
- **Idle Timeout**: Sessions are cleaned up after 5 minutes of inactivity
- **Maximum Age**: Sessions are cleaned up after 1 hour regardless of activity
- **Manual Cleanup**: Server restart clears all sessions

## Error Handling

### Common Errors

**400 Bad Request - Missing X-URL**
```
X-URL header is required
```

**400 Bad Request - Invalid Profile**
```
Invalid identifier 'invalid'. Available profiles: chrome, firefox, safari, ...
```

**400 Bad Request - Invalid Timeout**
```
X-TIMEOUT must be between 1 and 300 seconds, got 500
```

**400 Bad Request - Malformed URL**
```
Invalid target URL: malformed URL: missing host in URL
```

**502 Bad Gateway - Request Failed**
```
Request failed: connection timeout
```

## Configuration

### Environment Variables

- **PORT**: Server port (default: 8080)
  ```bash
  PORT=3000 go run ./cmd/proxy/main.go
  ```

### Default Settings

- **Default Timeout**: 30 seconds
- **Maximum Timeout**: 300 seconds (5 minutes)
- **Session Idle Timeout**: 300 seconds (5 minutes)
- **Session Maximum Age**: 3600 seconds (1 hour)
- **Cleanup Interval**: 60 seconds

## Testing

Run the comprehensive test suite:

```bash
# Start the server
go run ./cmd/proxy/main.go

# Run tests (in another terminal)
./examples/test_requests.sh
```

## Architecture

### Components

1. **Handler** (`internal/proxy/handler.go`)
   - FastHTTP request processing
   - Header extraction and validation
   - Profile management
   - Response streaming

2. **Client Manager** (`internal/cycletls/client.go`)
   - CycleTLS client wrapping
   - Session lifecycle management
   - Automatic cleanup
   - Statistics tracking

3. **Fingerprint Profiles** (`internal/fingerprints/profiles.go`)
   - TLS fingerprint definitions
   - JA3/JA4 configurations
   - User-Agent strings

### Session Lifecycle

1. **Creation**: Client provides `X-SESSION-ID` header
2. **Storage**: Session stored in ClientManager
3. **Reuse**: Subsequent requests with same ID reuse connection
4. **Cleanup**: Automatic removal based on idle time or age
5. **Shutdown**: All sessions closed on server shutdown

### Performance Features

- **Connection Pooling**: Reuse TCP connections where possible
- **Streaming Responses**: Handle large responses without buffering
- **Concurrent Safe**: Thread-safe session management
- **Memory Efficient**: Automatic cleanup prevents memory leaks

## Advanced Usage

### Programmatic Access

```go
// Create handler
logger := log.New(os.Stderr)
handler := proxy.NewHandler(logger)

// Get session statistics
stats := handler.GetSessionStats("my-session")
fmt.Printf("Session age: %s\n", stats.Age)

// Manual session cleanup
handler.RemoveIdleClients(5 * time.Minute)

// Graceful shutdown
handler.Close()
```

### Custom Client Configuration

```go
// Create custom client manager
config := &cycletls.ClientConfig{
    DefaultTimeout:  60 * time.Second,
    MaxIdleTime:     10 * time.Minute,
    MaxSessionAge:   2 * time.Hour,
    EnableCleanup:   true,
}

manager := cycletls.NewClientManagerWithConfig(config)
```

## Troubleshooting

### Common Issues

1. **Connection Refused**: Ensure server is running on correct port
2. **Timeout Errors**: Increase timeout with `X-TIMEOUT` header
3. **Profile Not Found**: Check available profiles in server startup message
4. **Session Issues**: Verify `X-SESSION-ID` consistency across requests

### Debug Mode

Enable debug logging for detailed request information:

```bash
LOG_LEVEL=debug go run ./cmd/proxy/main.go
```

### Monitoring

Check server logs for:
- Request processing times
- Session creation/cleanup events
- Error details and stack traces
- Performance warnings for slow requests

## Best Practices

1. **Use Sessions**: Always provide `X-SESSION-ID` for better performance
2. **Choose Appropriate Profiles**: Match the profile to your target website
3. **Handle Errors**: Check response status codes and handle errors gracefully
4. **Monitor Performance**: Watch for slow request warnings in logs
5. **Cleanup Resources**: Ensure proper session cleanup in long-running applications

## Support

For issues, feature requests, or questions:

1. Check server logs for detailed error information
2. Verify request headers match the required format
3. Test with the provided examples first
4. Review this documentation for configuration options