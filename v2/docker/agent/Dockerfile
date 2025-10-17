# Build stage for Agent
FROM golang:1.21-alpine AS builder

WORKDIR /build

# Copy go mod files
COPY go.mod go.sum ./
RUN go mod download

# Copy source code
COPY . .

# Build the application
RUN CGO_ENABLED=0 GOOS=linux go build -a -installsuffix cgo -o agent .

# Final stage
FROM alpine:3.18

# Install runtime dependencies including rclone
RUN apk --no-cache add ca-certificates tzdata curl bash && \
    curl https://rclone.org/install.sh | bash

WORKDIR /app

# Copy the binary from builder
COPY --from=builder /build/agent .

# Create directories
RUN mkdir -p /var/lib/rclone-agent

# Create non-root user
RUN addgroup -g 1001 -S rclone && \
    adduser -u 1001 -S rclone -G rclone

# Change ownership
RUN chown -R rclone:rclone /app /var/lib/rclone-agent

USER rclone

CMD ["./agent"]