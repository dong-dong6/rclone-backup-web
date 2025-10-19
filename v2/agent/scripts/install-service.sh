#!/bin/bash
# Install script for Rclone Backup Agent as systemd service

set -e

# Colors for output
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
NC='\033[0m' # No Color

# Configuration
AGENT_USER="rclone-agent"
AGENT_HOME="/opt/rclone-agent"
AGENT_BIN="$AGENT_HOME/rclone-agent"
AGENT_CONFIG="$AGENT_HOME/agent.json"
SERVICE_NAME="rclone-agent"
SERVICE_FILE="/etc/systemd/system/${SERVICE_NAME}.service"

# Check if running as root
if [[ $EUID -ne 0 ]]; then
   echo -e "${RED}This script must be run as root${NC}"
   exit 1
fi

# Security configuration
CREATE_DEDICATED_USER=${CREATE_DEDICATED_USER:-true}
DEDICATED_USER="rclone-agent"
DEDICATED_GROUP="rclone-agent"

echo "========================================="
echo "Rclone Backup Agent Installation Script"
echo "========================================="

# Parse arguments
HUB_URL=""
REGISTRATION_TOKEN=""
AGENT_NAME=""

while [[ $# -gt 0 ]]; do
    case $1 in
        --hub-url)
            HUB_URL="$2"
            shift 2
            ;;
        --token)
            REGISTRATION_TOKEN="$2"
            shift 2
            ;;
        --name)
            AGENT_NAME="$2"
            shift 2
            ;;
        --help)
            echo "Usage: $0 [options]"
            echo ""
            echo "Options:"
            echo "  --hub-url URL       Hub API URL (e.g., https://hub.example.com)"
            echo "  --token TOKEN       Registration token from hub"
            echo "  --name NAME         Agent name (default: hostname)"
            echo "  --help              Show this help message"
            exit 0
            ;;
        *)
            echo -e "${RED}Unknown option: $1${NC}"
            exit 1
            ;;
    esac
done

# Validate required parameters
if [[ -z "$HUB_URL" ]]; then
    echo -e "${YELLOW}Enter Hub URL (e.g., https://hub.example.com):${NC}"
    read -r HUB_URL
fi

if [[ -z "$REGISTRATION_TOKEN" ]]; then
    echo -e "${YELLOW}Enter Registration Token:${NC}"
    read -r REGISTRATION_TOKEN
fi

if [[ -z "$AGENT_NAME" ]]; then
    AGENT_NAME=$(hostname)
    echo -e "${GREEN}Using hostname as agent name: $AGENT_NAME${NC}"
fi

# Step 1: Create dedicated user (if enabled)
echo -e "\n${YELLOW}Step 1: Setting up user...${NC}"
if [[ "$CREATE_DEDICATED_USER" == "true" ]]; then
    if id "$DEDICATED_USER" &>/dev/null; then
        echo -e "${GREEN}User $DEDICATED_USER already exists${NC}"
    else
        # Create system user with no login shell for security
        useradd -r -m -d "$AGENT_HOME" -s /usr/sbin/nologin "$DEDICATED_USER"
        echo -e "${GREEN}Created dedicated user $DEDICATED_USER (no login)${NC}"
    fi
    AGENT_USER="$DEDICATED_USER"
else
    echo -e "${YELLOW}Running as root (not recommended for production)${NC}"
    AGENT_USER="root"
fi

# Step 2: Create directory structure
echo -e "\n${YELLOW}Step 2: Creating directory structure...${NC}"
mkdir -p "$AGENT_HOME"/{bin,configs,tasks,logs}
chown -R "$AGENT_USER:$AGENT_USER" "$AGENT_HOME"
chmod 755 "$AGENT_HOME"
echo -e "${GREEN}Directory structure created${NC}"

# Step 3: Copy agent binary
echo -e "\n${YELLOW}Step 3: Installing agent binary...${NC}"
if [[ ! -f "./rclone-agent" ]]; then
    echo -e "${RED}Error: rclone-agent binary not found in current directory${NC}"
    echo "Please build the agent first: go build -o rclone-agent main_standalone.go"
    exit 1
fi

cp ./rclone-agent "$AGENT_BIN"
chown "$AGENT_USER:$AGENT_USER" "$AGENT_BIN"
chmod 755 "$AGENT_BIN"
echo -e "${GREEN}Agent binary installed${NC}"

# Step 4: Create initial configuration
echo -e "\n${YELLOW}Step 4: Creating configuration...${NC}"
cat > "$AGENT_CONFIG" <<EOF
{
  "hub_url": "$HUB_URL",
  "registration_token": "$REGISTRATION_TOKEN",
  "agent_name": "$AGENT_NAME",
  "work_dir": "$AGENT_HOME",
  "max_concurrent": 3,
  "heartbeat_interval": 30,
  "enable_local_fallback": true,
  "enable_auto_update": false,
  "enable_metrics": true,
  "metrics_port": 9091,
  "run_as_service": true,
  "log_file": "$AGENT_HOME/logs/agent.log",
  "pid_file": "$AGENT_HOME/agent.pid"
}
EOF

chown "$AGENT_USER:$AGENT_USER" "$AGENT_CONFIG"
chmod 600 "$AGENT_CONFIG"
echo -e "${GREEN}Configuration created${NC}"

# Step 5: Create systemd service file
echo -e "\n${YELLOW}Step 5: Creating systemd service...${NC}"
cat > "$SERVICE_FILE" <<EOF
[Unit]
Description=Rclone Backup Agent
Documentation=https://github.com/your-repo/rclone-backup-web
After=network-online.target
Wants=network-online.target

[Service]
Type=simple
User=$AGENT_USER
Group=$AGENT_USER
WorkingDirectory=$AGENT_HOME

# Service configuration
ExecStart=$AGENT_BIN -config $AGENT_CONFIG
ExecReload=/bin/kill -USR1 \$MAINPID
Restart=always
RestartSec=10

# Security hardening
NoNewPrivileges=true
PrivateTmp=true
ProtectSystem=strict
ProtectHome=true
ReadWritePaths=$AGENT_HOME

# Resource limits
LimitNOFILE=65536
TasksMax=4096

# Logging
StandardOutput=journal
StandardError=journal
SyslogIdentifier=$SERVICE_NAME

[Install]
WantedBy=multi-user.target
EOF

echo -e "${GREEN}Systemd service created${NC}"

# Step 6: Enable and start service
echo -e "\n${YELLOW}Step 6: Enabling and starting service...${NC}"
systemctl daemon-reload
systemctl enable "$SERVICE_NAME"
systemctl start "$SERVICE_NAME"

# Wait a moment for service to start
sleep 2

# Check service status
if systemctl is-active --quiet "$SERVICE_NAME"; then
    echo -e "${GREEN}Service started successfully!${NC}"
else
    echo -e "${RED}Service failed to start. Check logs with: journalctl -u $SERVICE_NAME${NC}"
    exit 1
fi

# Step 7: Setup log rotation
echo -e "\n${YELLOW}Step 7: Setting up log rotation...${NC}"
cat > "/etc/logrotate.d/$SERVICE_NAME" <<EOF
$AGENT_HOME/logs/*.log {
    daily
    rotate 14
    missingok
    notifempty
    compress
    delaycompress
    create 0640 $AGENT_USER $AGENT_USER
    sharedscripts
    postrotate
        systemctl reload $SERVICE_NAME > /dev/null 2>&1 || true
    endscript
}
EOF
echo -e "${GREEN}Log rotation configured${NC}"

# Step 8: Create uninstall script
echo -e "\n${YELLOW}Step 8: Creating uninstall script...${NC}"
cat > "$AGENT_HOME/uninstall.sh" <<'EOF'
#!/bin/bash

echo "Uninstalling Rclone Backup Agent..."

# Stop and disable service
systemctl stop rclone-agent
systemctl disable rclone-agent

# Remove service file
rm -f /etc/systemd/system/rclone-agent.service
systemctl daemon-reload

# Remove logrotate config
rm -f /etc/logrotate.d/rclone-agent

# Remove user and files (optional)
read -p "Remove agent user and all data? (y/N): " -n 1 -r
echo
if [[ $REPLY =~ ^[Yy]$ ]]; then
    userdel -r rclone-agent
    rm -rf /opt/rclone-agent
fi

echo "Uninstallation complete"
EOF

chmod 755 "$AGENT_HOME/uninstall.sh"
chown "$AGENT_USER:$AGENT_USER" "$AGENT_HOME/uninstall.sh"
echo -e "${GREEN}Uninstall script created${NC}"

# Summary
echo -e "\n========================================="
echo -e "${GREEN}Installation Complete!${NC}"
echo -e "========================================="
echo ""
echo "Agent Status:"
systemctl status "$SERVICE_NAME" --no-pager | head -n 5
echo ""
echo "Useful commands:"
echo "  Check status:  systemctl status $SERVICE_NAME"
echo "  View logs:     journalctl -u $SERVICE_NAME -f"
echo "  Restart:       systemctl restart $SERVICE_NAME"
echo "  Stop:          systemctl stop $SERVICE_NAME"
echo "  Uninstall:     $AGENT_HOME/uninstall.sh"
echo ""
echo "Agent home:      $AGENT_HOME"
echo "Configuration:   $AGENT_CONFIG"
echo "Logs:           $AGENT_HOME/logs/"
echo ""
echo -e "${GREEN}The agent is now running and will register with the hub.${NC}"