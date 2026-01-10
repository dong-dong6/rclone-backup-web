# ============================================
# Hub Server - Multi-stage Build
# 包含: Go API + React 前端 + Agent 二进制
# ============================================

# ============================================
# Stage 1: Build React Frontend
# ============================================
FROM node:20-alpine AS frontend-builder

WORKDIR /frontend

# Copy package files
COPY hub/web/package*.json ./

# Install dependencies
RUN npm ci --prefer-offline --no-audit

# Copy source code
COPY hub/web/ ./

# Build production bundle
RUN npm run build

# ============================================
# Stage 2: Build Go Backend
# ============================================
FROM golang:1.23-alpine AS backend-builder

RUN apk add --no-cache git make gcc musl-dev

WORKDIR /build

# Copy and download dependencies
COPY hub/go.mod hub/go.sum ./
RUN go mod download

# Copy source
COPY hub/ ./

# Build Hub binary
RUN CGO_ENABLED=1 GOOS=linux go build \
    -ldflags="-w -s -X main.Version=${VERSION:-dev}" \
    -o hub-api \
    ./main.go

# ============================================
# Stage 3: Build Agent Binaries
# ============================================
FROM golang:1.23-alpine AS agent-builder

RUN apk add --no-cache git make gcc musl-dev

WORKDIR /agent-build

# Copy and download dependencies
COPY agent/go.mod agent/go.sum ./
RUN go mod download

COPY agent/ ./

# Build for multiple platforms
RUN CGO_ENABLED=0 GOOS=linux GOARCH=amd64 go build \
    -ldflags="-w -s -X main.Version=${VERSION:-dev}" \
    -o rclone-backup-agent-linux-amd64 \
    ./main.go

RUN CGO_ENABLED=0 GOOS=linux GOARCH=arm64 go build \
    -ldflags="-w -s -X main.Version=${VERSION:-dev}" \
    -o rclone-backup-agent-linux-arm64 \
    ./main.go

RUN CGO_ENABLED=0 GOOS=darwin GOARCH=amd64 go build \
    -ldflags="-w -s -X main.Version=${VERSION:-dev}" \
    -o rclone-backup-agent-darwin-amd64 \
    ./main.go

RUN CGO_ENABLED=0 GOOS=darwin GOARCH=arm64 go build \
    -ldflags="-w -s -X main.Version=${VERSION:-dev}" \
    -o rclone-backup-agent-darwin-arm64 \
    ./main.go

RUN CGO_ENABLED=0 GOOS=windows GOARCH=amd64 go build \
    -ldflags="-w -s -X main.Version=${VERSION:-dev}" \
    -o rclone-backup-agent-windows-amd64.exe \
    ./main.go

# Organize binaries
RUN mkdir -p /agent-binaries && \
    cp rclone-backup-agent-* /agent-binaries/ && \
    cd /agent-binaries && \
    ln -s rclone-backup-agent-linux-amd64 rclone-backup-agent

# ============================================
# Stage 4: Runtime
# ============================================
FROM alpine:3.18

# Install runtime dependencies
RUN apk add --no-cache \
    ca-certificates \
    tzdata \
    wget

# Create non-root user
RUN addgroup -g 1000 -S rclone && \
    adduser -u 1000 -S rclone -G rclone

WORKDIR /app

# Copy Go binary
COPY --from=backend-builder /build/hub-api /app/hub-api

# Copy frontend build (will be served by Go)
COPY --from=frontend-builder /frontend/dist /app/static/web

# Copy Agent binaries (for download)
COPY --from=agent-builder /agent-binaries /app/static/binaries

# Copy installation scripts
COPY hub/static/scripts /app/static/scripts

# Copy database migrations
COPY hub/database/migrations /app/database/migrations

# Create directories and set permissions
RUN mkdir -p /app/data /app/logs && \
    chown -R rclone:rclone /app

USER rclone

# Expose port (unified: API + Static files)
EXPOSE 8080

# Health check
HEALTHCHECK --interval=30s --timeout=10s --start-period=30s --retries=3 \
    CMD wget -q --spider http://localhost:8080/health || exit 1

ENTRYPOINT ["/app/hub-api"]
