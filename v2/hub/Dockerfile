# ============================================
# Hub API Server - Multi-stage Build
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

# Build the Hub binary
RUN CGO_ENABLED=1 GOOS=linux go build \
    -ldflags="-w -s -X main.Version=${VERSION:-dev}" \
    -o hub-api \
    ./main.go

# ============================================
# Agent build stage
# ============================================
FROM golang:1.23-alpine AS agent-builder

# Install build dependencies
RUN apk add --no-cache git make gcc musl-dev

# Set working directory
WORKDIR /agent-build

# Copy agent source code
COPY ../agent/go.mod ../agent/go.sum ./
RUN go mod download

COPY ../agent/ ./

# Build agent binary for multiple platforms
RUN CGO_ENABLED=0 GOOS=linux GOARCH=amd64 go build \
    -ldflags="-w -s -X main.Version=${VERSION:-dev}" \
    -o rclone-backup-agent-linux-amd64 \
    ./main_standalone.go

RUN CGO_ENABLED=0 GOOS=linux GOARCH=arm64 go build \
    -ldflags="-w -s -X main.Version=${VERSION:-dev}" \
    -o rclone-backup-agent-linux-arm64 \
    ./main_standalone.go

RUN CGO_ENABLED=0 GOOS=linux GOARCH=arm go build \
    -ldflags="-w -s -X main.Version=${VERSION:-dev}" \
    -o rclone-backup-agent-linux-arm \
    ./main_standalone.go

RUN CGO_ENABLED=0 GOOS=darwin GOARCH=amd64 go build \
    -ldflags="-w -s -X main.Version=${VERSION:-dev}" \
    -o rclone-backup-agent-darwin-amd64 \
    ./main_standalone.go

RUN CGO_ENABLED=0 GOOS=darwin GOARCH=arm64 go build \
    -ldflags="-w -s -X main.Version=${VERSION:-dev}" \
    -o rclone-backup-agent-darwin-arm64 \
    ./main_standalone.go

RUN CGO_ENABLED=0 GOOS=windows GOARCH=amd64 go build \
    -ldflags="-w -s -X main.Version=${VERSION:-dev}" \
    -o rclone-backup-agent-windows-amd64.exe \
    ./main_standalone.go

# Create agent binaries directory
RUN mkdir -p /agent-binaries && \
    cp rclone-backup-agent-* /agent-binaries/ && \
    # Create symlink for default download
    cd /agent-binaries && \
    ln -s rclone-backup-agent-linux-amd64 rclone-backup-agent

# ============================================
# Runtime stage
# ============================================
FROM alpine:3.18

# Install runtime dependencies
RUN apk add --no-cache \
    ca-certificates \
    tzdata \
    curl \
    wget

# Create non-root user
RUN addgroup -g 1000 -S rclone && \
    adduser -u 1000 -S rclone -G rclone

# Set working directory
WORKDIR /app

# Copy binary from builder
COPY --from=builder /build/hub-api /app/hub-api

# Copy agent binaries from agent-builder
COPY --from=agent-builder /agent-binaries /app/static/binaries

# Copy database migrations
COPY database/migrations /app/database/migrations

# Create necessary directories
RUN mkdir -p /app/config /app/data /app/logs && \
    chown -R rclone:rclone /app

# Switch to non-root user
USER rclone

# Expose ports
EXPOSE 8080 9090

# Health check
HEALTHCHECK --interval=30s --timeout=10s --start-period=60s --retries=3 \
    CMD wget -q --spider http://localhost:8080/health || exit 1

# Run the application
ENTRYPOINT ["/app/hub-api"]