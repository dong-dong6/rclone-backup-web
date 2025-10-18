# ============================================
# Agent Service - Multi-stage Build
# ============================================

# Build stage
FROM golang:1.23-alpine AS builder

# Install build dependencies
RUN apk add --no-cache git make gcc musl-dev

# Set working directory
WORKDIR /build

# Copy go mod files
COPY go.mod go.sum ./

# Download dependencies
RUN go mod download

# Copy source code
COPY . .

# Build the binary
RUN CGO_ENABLED=1 GOOS=linux go build \
    -ldflags="-w -s -X main.Version=${VERSION:-dev}" \
    -o agent \
    ./main.go

# ============================================
# Runtime stage
# ============================================
FROM alpine:3.18

# Install runtime dependencies
RUN apk add --no-cache \
    ca-certificates \
    tzdata \
    curl \
    wget \
    bash

# Create non-root user
RUN addgroup -g 1000 -S rclone && \
    adduser -u 1000 -S rclone -G rclone

# Set working directory
WORKDIR /app

# Copy binary from builder
COPY --from=builder /build/agent /app/agent

# Create necessary directories
RUN mkdir -p /app/config /app/data /app/logs /app/cache && \
    chown -R rclone:rclone /app

# Create data directory for configuration persistence
RUN mkdir -p /data && chown -R rclone:rclone /data

# Switch to non-root user
USER rclone

# Expose health check port
EXPOSE 8081

# Health check
HEALTHCHECK --interval=30s --timeout=10s --start-period=60s --retries=3 \
    CMD wget -q --spider http://localhost:8081/health || exit 1

# Run the application
ENTRYPOINT ["/app/agent"]