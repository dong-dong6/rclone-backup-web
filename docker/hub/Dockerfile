# Build stage for Hub API
FROM golang:1.21-alpine AS builder

# Install build dependencies
RUN apk add --no-cache git

WORKDIR /build

# Copy go mod files
COPY go.mod go.sum ./
RUN go mod download

# Copy source code
COPY . .

# Build the application
RUN CGO_ENABLED=0 GOOS=linux go build -a -installsuffix cgo -o hub-api .

# Final stage
FROM alpine:3.18

# Install runtime dependencies
RUN apk --no-cache add ca-certificates tzdata

WORKDIR /app

# Copy the binary from builder
COPY --from=builder /build/hub-api .

# Create non-root user
RUN addgroup -g 1001 -S rclone && \
    adduser -u 1001 -S rclone -G rclone

# Change ownership
RUN chown -R rclone:rclone /app

USER rclone

EXPOSE 8080

CMD ["./hub-api"]