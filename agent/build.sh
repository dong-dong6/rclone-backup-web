#!/bin/bash
# Build script for standalone Rclone Backup Agent

set -e

# Colors
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
NC='\033[0m'

# Variables
VERSION=${VERSION:-"dev"}
BUILD_TIME=$(date -u '+%Y-%m-%d %H:%M:%S UTC')
GIT_COMMIT=$(git rev-parse --short HEAD 2>/dev/null || echo "unknown")
OUTPUT_DIR="./dist"

echo "========================================="
echo "Building Rclone Backup Agent"
echo "========================================="
echo "Version:    $VERSION"
echo "Build Time: $BUILD_TIME"
echo "Git Commit: $GIT_COMMIT"
echo ""

# Create output directory
mkdir -p "$OUTPUT_DIR"

# Build for multiple platforms
PLATFORMS=("linux/amd64" "linux/arm64" "linux/arm" "darwin/amd64" "darwin/arm64" "windows/amd64")

for PLATFORM in "${PLATFORMS[@]}"; do
    GOOS=${PLATFORM%/*}
    GOARCH=${PLATFORM#*/}
    OUTPUT_NAME="rclone-agent-${VERSION}-${GOOS}-${GOARCH}"
    
    if [ "$GOOS" = "windows" ]; then
        OUTPUT_NAME="${OUTPUT_NAME}.exe"
    fi
    
    echo -e "${YELLOW}Building for ${GOOS}/${GOARCH}...${NC}"
    
    # Set build flags
    LDFLAGS="-X main.Version=${VERSION}"
    LDFLAGS="${LDFLAGS} -X 'main.BuildTime=${BUILD_TIME}'"
    LDFLAGS="${LDFLAGS} -X main.GitCommit=${GIT_COMMIT}"
    LDFLAGS="${LDFLAGS} -s -w" # Strip debug info for smaller binary
    
    # Build
    GOOS=$GOOS GOARCH=$GOARCH go build \
        -ldflags "${LDFLAGS}" \
        -o "${OUTPUT_DIR}/${OUTPUT_NAME}" \
        main.go
    
    # Compress with UPX if available (except for macOS)
    if command -v upx &> /dev/null && [ "$GOOS" != "darwin" ]; then
        echo "  Compressing with UPX..."
        upx --best --lzma "${OUTPUT_DIR}/${OUTPUT_NAME}" || true
    fi
    
    # Calculate file size
    SIZE=$(du -h "${OUTPUT_DIR}/${OUTPUT_NAME}" | cut -f1)
    echo -e "${GREEN}  ✓ Built ${OUTPUT_NAME} (${SIZE})${NC}"
done

# Create archives
echo -e "\n${YELLOW}Creating distribution archives...${NC}"

cd "$OUTPUT_DIR"
for FILE in rclone-agent-*; do
    if [[ "$FILE" == *.exe ]]; then
        # Windows: Create ZIP
        ZIP_NAME="${FILE%.exe}.zip"
        zip -q "$ZIP_NAME" "$FILE"
        echo -e "${GREEN}  ✓ Created ${ZIP_NAME}${NC}"
    else
        # Unix: Create tar.gz
        TAR_NAME="${FILE}.tar.gz"
        tar -czf "$TAR_NAME" "$FILE"
        echo -e "${GREEN}  ✓ Created ${TAR_NAME}${NC}"
    fi
done
cd ..

# Create SHA256 checksums
echo -e "\n${YELLOW}Generating checksums...${NC}"
cd "$OUTPUT_DIR"
sha256sum rclone-agent-* > SHA256SUMS
echo -e "${GREEN}  ✓ Created SHA256SUMS${NC}"
cd ..

# Create sample configuration
echo -e "\n${YELLOW}Creating sample configuration...${NC}"
cat > "${OUTPUT_DIR}/agent.json.sample" <<EOF
{
  "hub_url": "https://hub.example.com",
  "registration_token": "YOUR_REGISTRATION_TOKEN",
  "agent_name": "agent-1",
  "work_dir": "/opt/rclone-agent",
  "max_concurrent": 3,
  "heartbeat_interval": 30,
  "enable_local_fallback": true,
  "run_as_service": false,
  "log_level": "info",
  "log_file": "",
  "pid_file": ""
}
EOF
echo -e "${GREEN}  ✓ Created agent.json.sample${NC}"

# Copy installation script
cp scripts/install_agent.sh "${OUTPUT_DIR}/"
chmod +x "${OUTPUT_DIR}/install_agent.sh"
echo -e "${GREEN}  ✓ Copied install_agent.sh${NC}"

# Create README
cat > "${OUTPUT_DIR}/README.md" <<EOF
# Rclone Backup Agent (Standalone)

Version: ${VERSION}
Built: ${BUILD_TIME}
Commit: ${GIT_COMMIT}

## Quick Start

### 1. Manual Run
\`\`\`bash
# Create configuration
cp agent.json.sample agent.json
# Edit agent.json with your hub URL and registration token

# Run agent
./rclone-agent -config agent.json
\`\`\`

### 2. Install as System Service (Linux)
\`\`\`bash
sudo ./install_agent.sh --hub-url https://hub.example.com --token YOUR_TOKEN
\`\`\`

### 3. Command Line Options
\`\`\`bash
# Show version
./rclone-agent -version

# Specify work directory
./rclone-agent -work-dir /path/to/workdir

# Override hub URL
./rclone-agent -hub-url https://hub.example.com

# Provide registration token
./rclone-agent -token YOUR_REGISTRATION_TOKEN
\`\`\`

## Features
- ✅ Self-contained binary (includes rclone management)
- ✅ Automatic rclone download and updates
- ✅ Sandboxed task execution
- ✅ Local fallback when hub is unreachable
- ✅ System service integration
- ✅ Automatic retries and error recovery

## System Requirements
- Linux: kernel 3.10+, glibc 2.17+
- macOS: 10.12+
- Windows: Windows 7+
- RAM: 256MB minimum
- Disk: 100MB for agent + space for backups

## Security
- Tasks run in isolated environments
- Sensitive configurations are encrypted
- API key authentication with hub
- Restricted file permissions

## Support
GitHub: https://github.com/your-repo/rclone-backup-web
EOF
echo -e "${GREEN}  ✓ Created README.md${NC}"

# Summary
echo -e "\n========================================="
echo -e "${GREEN}Build Complete!${NC}"
echo -e "========================================="
echo ""
echo "Output directory: ${OUTPUT_DIR}"
echo ""
echo "Files created:"
ls -lh "$OUTPUT_DIR" | grep -E "rclone-agent|SHA256|README|agent.json|install-service"
echo ""
echo -e "${GREEN}Next steps:${NC}"
echo "1. Test the binary: ${OUTPUT_DIR}/rclone-agent-${VERSION}-$(go env GOOS)-$(go env GOARCH) -version"
echo "2. Deploy to target systems"
echo "3. Run install-service.sh for system service installation"
