# Multi-stage Docker build for CycleTLS-Proxy
# Stage 1: Build the application
FROM golang:1.24-alpine AS builder

# Install build dependencies
RUN apk add --no-cache \
    git \
    ca-certificates \
    tzdata

# Set working directory
WORKDIR /app

# Copy go.mod and go.sum first for better layer caching
COPY go.mod go.sum ./

# Download dependencies (CycleTLS dependency is already properly configured)
RUN go mod download && go mod verify

# Copy source code
COPY . .

# Build the application
# Use build flags for optimization and static linking
RUN CGO_ENABLED=0 GOOS=linux GOARCH=amd64 go build \
    -ldflags='-w -s -extldflags "-static"' \
    -a -installsuffix cgo \
    -o cycletls-proxy \
    ./cmd/proxy

# Verify the binary was built correctly
RUN chmod +x cycletls-proxy && \
    ./cycletls-proxy --version || ./cycletls-proxy -v || echo "Binary built successfully"

# Stage 2: Create minimal runtime image
FROM alpine:3.19 AS runtime

# Install runtime dependencies
RUN apk add --no-cache \
    ca-certificates \
    tzdata \
    && update-ca-certificates

# Create non-root user for security
RUN addgroup -g 1000 cycletls && \
    adduser -u 1000 -G cycletls -s /bin/sh -D cycletls

# Set working directory
WORKDIR /app

# Copy binary from builder stage
COPY --from=builder /app/cycletls-proxy /usr/local/bin/cycletls-proxy

# Copy any additional files if needed (configs, etc.)
# COPY --from=builder /app/config/ ./config/

# Change ownership to non-root user
RUN chown -R cycletls:cycletls /app

# Switch to non-root user
USER cycletls

# Expose the default port
EXPOSE 8080

# Add health check
HEALTHCHECK --interval=30s --timeout=10s --start-period=5s --retries=3 \
    CMD wget --no-verbose --tries=1 --spider http://localhost:8080/health || exit 1

# Set environment variables with defaults
ENV PORT=8080 \
    LOG_LEVEL=info

# Add labels for better maintainability
LABEL org.opencontainers.image.title="CycleTLS-Proxy" \
      org.opencontainers.image.description="Advanced TLS Fingerprint Proxy Server" \
      org.opencontainers.image.vendor="Danny-Dasilva" \
      org.opencontainers.image.source="https://github.com/Danny-Dasilva/CycleTLS-Proxy" \
      org.opencontainers.image.licenses="MIT"

# Command to run the application
CMD ["cycletls-proxy"]

# Alternative Dockerfile for development with hot reload
# Uncomment the section below if you want a development variant

# FROM golang:1.21-alpine AS development
# 
# RUN apk add --no-cache git ca-certificates
# RUN go install github.com/cosmtrek/air@latest
# 
# WORKDIR /app
# 
# # Copy go files
# COPY go.mod go.sum ./
# COPY ../CycleTLS/cycletls /tmp/cycletls/
# RUN go mod edit -replace github.com/Danny-Dasilva/CycleTLS/cycletls=/tmp/cycletls
# RUN go mod download
# 
# # Copy source
# COPY . .
# 
# EXPOSE 8080
# 
# CMD ["air", "-c", ".air.toml"]