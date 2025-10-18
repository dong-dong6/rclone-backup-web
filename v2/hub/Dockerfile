# ============================================
# Hub API Server - Multi-stage Build
# ============================================

# Build stage
FROM golang:1.21-alpine AS builder

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
    -o hub-api \
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
    wget

# Create non-root user
RUN addgroup -g 1000 -S rclone && \
    adduser -u 1000 -S rclone -G rclone

# Set working directory
WORKDIR /app

# Copy binary from builder
COPY --from=builder /build/hub-api /app/hub-api

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