# Makefile for Rclone Backup Agent (Standalone)

# Variables
VERSION ?= $(shell git describe --tags --always --dirty 2>/dev/null || echo "dev")
BUILD_TIME = $(shell date -u '+%Y-%m-%d %H:%M:%S UTC')
GIT_COMMIT = $(shell git rev-parse --short HEAD 2>/dev/null || echo "unknown")
GO = go
BINARY = rclone-agent
MAIN = main.go
OUTPUT_DIR = ./dist

# Build flags
LDFLAGS = -X 'main.Version=$(VERSION)'
LDFLAGS += -X 'main.BuildTime=$(BUILD_TIME)'
LDFLAGS += -X 'main.GitCommit=$(GIT_COMMIT)'
LDFLAGS += -s -w

# Platforms
PLATFORMS = linux/amd64 linux/arm64 linux/arm darwin/amd64 darwin/arm64 windows/amd64

.PHONY: all build clean test run install dev release help

# Default target
all: build

# Build for current platform
build:
	@echo "Building $(BINARY) for current platform..."
	@$(GO) build -ldflags "$(LDFLAGS)" -o $(BINARY) $(MAIN)
	@echo "✓ Built $(BINARY)"

# Development build with race detector
dev:
	@echo "Building development version with race detector..."
	@$(GO) build -race -ldflags "$(LDFLAGS)" -o $(BINARY)-dev $(MAIN)
	@echo "✓ Built $(BINARY)-dev"

# Run tests
test:
	@echo "Running tests..."
	@$(GO) test -v -race -coverprofile=coverage.out ./...
	@echo "✓ Tests passed"

# Run tests with coverage report
test-coverage: test
	@echo "Generating coverage report..."
	@$(GO) tool cover -html=coverage.out -o coverage.html
	@echo "✓ Coverage report generated: coverage.html"

# Run the agent locally
run: build
	@echo "Running agent..."
	@./$(BINARY) -config agent.json

# Install to system
install: build
	@echo "Installing to /usr/local/bin..."
	@sudo cp $(BINARY) /usr/local/bin/
	@sudo chmod 755 /usr/local/bin/$(BINARY)
	@echo "✓ Installed to /usr/local/bin/$(BINARY)"

# Build for all platforms
release:
	@echo "Building release for all platforms..."
	@chmod +x build.sh
	@VERSION=$(VERSION) ./build.sh
	@echo "✓ Release build complete"

# Build Docker image
docker:
	@echo "Building Docker image..."
	@docker build -t rclone-agent:$(VERSION) -f Dockerfile.standalone .
	@echo "✓ Docker image built: rclone-agent:$(VERSION)"

# Clean build artifacts
clean:
	@echo "Cleaning..."
	@rm -f $(BINARY) $(BINARY)-dev
	@rm -rf $(OUTPUT_DIR)
	@rm -f coverage.out coverage.html
	@$(GO) clean -cache
	@echo "✓ Cleaned"

# Format code
fmt:
	@echo "Formatting code..."
	@$(GO) fmt ./...
	@echo "✓ Code formatted"

# Lint code
lint:
	@echo "Running linters..."
	@if ! command -v golangci-lint &> /dev/null; then \
		echo "Installing golangci-lint..."; \
		go install github.com/golangci/golangci-lint/cmd/golangci-lint@latest; \
	fi
	@golangci-lint run ./...
	@echo "✓ Linting complete"

# Update dependencies
deps:
	@echo "Updating dependencies..."
	@$(GO) mod tidy
	@$(GO) mod download
	@$(GO) mod verify
	@echo "✓ Dependencies updated"

# Generate code (if needed)
generate:
	@echo "Generating code..."
	@$(GO) generate ./...
	@echo "✓ Code generation complete"

# Run benchmarks
bench:
	@echo "Running benchmarks..."
	@$(GO) test -bench=. -benchmem ./...

# Check for security vulnerabilities
security:
	@echo "Checking for vulnerabilities..."
	@if ! command -v gosec &> /dev/null; then \
		echo "Installing gosec..."; \
		go install github.com/securego/gosec/v2/cmd/gosec@latest; \
	fi
	@gosec -fmt json -out security-report.json ./...
	@echo "✓ Security check complete. Report: security-report.json"

# Create systemd service file
systemd:
	@echo "Creating systemd service file..."
	@echo "[Unit]" > rclone-agent.service
	@echo "Description=Rclone Backup Agent" >> rclone-agent.service
	@echo "After=network-online.target" >> rclone-agent.service
	@echo "" >> rclone-agent.service
	@echo "[Service]" >> rclone-agent.service
	@echo "Type=simple" >> rclone-agent.service
	@echo "User=rclone-agent" >> rclone-agent.service
	@echo "ExecStart=/usr/local/bin/rclone-agent -config /etc/rclone-agent/agent.json" >> rclone-agent.service
	@echo "Restart=always" >> rclone-agent.service
	@echo "RestartSec=10" >> rclone-agent.service
	@echo "" >> rclone-agent.service
	@echo "[Install]" >> rclone-agent.service
	@echo "WantedBy=multi-user.target" >> rclone-agent.service
	@echo "✓ Created rclone-agent.service"

# Show version
version:
	@echo "Version: $(VERSION)"
	@echo "Build Time: $(BUILD_TIME)"
	@echo "Git Commit: $(GIT_COMMIT)"

# Help
help:
	@echo "Rclone Backup Agent - Makefile Targets"
	@echo ""
	@echo "Building:"
	@echo "  make build       - Build binary for current platform"
	@echo "  make dev         - Build with race detector for development"
	@echo "  make release     - Build for all platforms"
	@echo "  make docker      - Build Docker image"
	@echo ""
	@echo "Testing:"
	@echo "  make test        - Run tests"
	@echo "  make test-coverage - Run tests with coverage"
	@echo "  make bench       - Run benchmarks"
	@echo "  make security    - Check for vulnerabilities"
	@echo ""
	@echo "Running:"
	@echo "  make run         - Build and run locally"
	@echo "  make install     - Install to /usr/local/bin"
	@echo ""
	@echo "Development:"
	@echo "  make fmt         - Format code"
	@echo "  make lint        - Run linters"
	@echo "  make deps        - Update dependencies"
	@echo "  make generate    - Generate code"
	@echo ""
	@echo "Utilities:"
	@echo "  make clean       - Remove build artifacts"
	@echo "  make systemd     - Create systemd service file"
	@echo "  make version     - Show version info"
	@echo "  make help        - Show this help"
