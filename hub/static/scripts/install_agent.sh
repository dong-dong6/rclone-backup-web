#!/bin/bash
#
# Rclone Backup Agent - Universal Installer Script
#
# This script automatically detects the OS and architecture, downloads the
# appropriate agent binary, and installs it as a system service.
#
# Usage:
#   curl -fsSL https://your-hub-url/api/v1/agent/install.sh | sudo bash -s -- \
#     --hub-url "https://your-hub-url" \
#     --token "YOUR_REGISTRATION_TOKEN" \
#     --name "your-agent-name"
#

set -e

# --- Configuration ---
AGENT_USER="rclone-agent"
AGENT_HOME="/opt/rclone-agent"
AGENT_BIN_DIR="$AGENT_HOME/bin"
AGENT_BIN="$AGENT_BIN_DIR/rclone-agent"
AGENT_CONFIG="$AGENT_HOME/agent.json"
SERVICE_NAME="rclone-agent"
SERVICE_FILE="/etc/systemd/system/${SERVICE_NAME}.service"
UNINSTALL_SCRIPT="$AGENT_HOME/uninstall.sh"

# --- Colors ---
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
BLUE='\033[0;34m'
NC='\033[0m'

# --- Helper Functions ---
info() {
    echo -e "${BLUE}[INFO]${NC} $1"
}

success() {
    echo -e "${GREEN}[SUCCESS]${NC} $1"
}

warn() {
    echo -e "${YELLOW}[WARNING]${NC} $1"
}

error() {
    echo -e "${RED}[ERROR]${NC} $1" >&2
}

# --- Pre-flight Checks ---
check_root() {
    if [[ $EUID -ne 0 ]]; then
        error "This script must be run as root. Please use 'sudo'."
        exit 1
    fi
}

check_systemd() {
    if ! command -v systemctl &> /dev/null; then
        error "This installer requires systemd. Your system is not supported."
        exit 1
    fi
}

# --- OS & Arch Detection ---
detect_os_arch() {
    OS=$(uname -s | tr '[:upper:]' '[:lower:]')
    ARCH=$(uname -m)

    case $ARCH in
        x86_64 | amd64) 
            ARCH="amd64"
            ;;
        aarch64 | arm64) 
            ARCH="arm64"
            ;;
        armv7l) 
            ARCH="arm"
            ;;
        *) 
            error "Unsupported architecture: $ARCH"
            exit 1
            ;;
    esac

    info "Detected OS: $OS, Arch: $ARCH"
}

# --- Main Functions ---
parse_args() {
    RUN_AS_ROOT="false"
    LOG_LEVEL="info"
    ENABLE_API="false"
    API_PORT="9092"
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
            --run-as-root)
                RUN_AS_ROOT="true"
                shift
                ;;
            --log-level)
                LOG_LEVEL="$2"
                shift 2
                ;;
            --enable-api)
                ENABLE_API="true"
                shift
                ;;
            --api-port)
                API_PORT="$2"
                shift 2
                ;;
            uninstall) 
                ACTION="uninstall"
                shift
                ;;
            *) 
                error "Unknown option: $1"
                exit 1
                ;;
        esac
    done

    if [[ "$ACTION" != "uninstall" ]]; then
        if [[ -z "$HUB_URL" || -z "$REGISTRATION_TOKEN" ]]; then
            error "Missing required arguments: --hub-url and --token are required."
            echo "Usage: $0 --hub-url <URL> --token <TOKEN> [--name <NAME>] [--run-as-root]"
            exit 1
        fi
        if [[ -z "$AGENT_NAME" ]]; then
            AGENT_NAME=$(hostname)
            info "Agent name not provided, using hostname: $AGENT_NAME"
        fi
        if [[ "$RUN_AS_ROOT" == "true" ]]; then
            warn "⚠️  Running as root - agent will have full filesystem access!"
        fi
    fi
}

setup_user_and_dirs() {
    info "Setting up user and directories..."
    
    if [[ "$RUN_AS_ROOT" == "true" ]]; then
        AGENT_USER="root"
        info "Running as root user."
    else
        if ! id "$AGENT_USER" &>/dev/null; then
            useradd -r -m -d "$AGENT_HOME" -s /usr/sbin/nologin "$AGENT_USER"
            success "Created dedicated user '$AGENT_USER'."
        else
            info "User '$AGENT_USER' already exists."
        fi
    fi

    mkdir -p "$AGENT_BIN_DIR" "$AGENT_HOME/logs" "$AGENT_HOME/tasks"
    chown -R "$AGENT_USER:$AGENT_USER" "$AGENT_HOME"
    chmod 750 "$AGENT_HOME"
    success "Directory structure created at $AGENT_HOME."
}

download_agent() {
    info "Downloading agent binary for $OS/$ARCH..."
    DOWNLOAD_URL="${HUB_URL}/api/v1/agent/download?platform=${OS}&arch=${ARCH}"
    
    if command -v curl &> /dev/null; then
        curl -sSLf -o "$AGENT_BIN" "$DOWNLOAD_URL"
    elif command -v wget &> /dev/null; then
        wget -qO "$AGENT_BIN" "$DOWNLOAD_URL"
    else
        error "Neither curl nor wget is available. Please install one."
        exit 1
    fi

    if [[ ! -s "$AGENT_BIN" ]]; then
        error "Failed to download agent binary. Please check the Hub URL and network connection."
        exit 1
    fi

    chmod 755 "$AGENT_BIN"
    chown "$AGENT_USER:$AGENT_USER" "$AGENT_BIN"
    success "Agent binary downloaded to $AGENT_BIN."
}

create_config() {
    info "Creating configuration file..."
    cat > "$AGENT_CONFIG" <<EOF
{
  "hub_url": "$HUB_URL",
  "registration_token": "$REGISTRATION_TOKEN",
  "agent_name": "$AGENT_NAME",
  "work_dir": "$AGENT_HOME",
  "max_concurrent": 3,
  "heartbeat_interval": 30,
  "heartbeat_interval": 30,
  "enable_local_fallback": true,
  "enable_api": $ENABLE_API,
  "api_bind_addr": "0.0.0.0",
  "api_port": $API_PORT,
  "run_as_service": true,
  "log_level": "$LOG_LEVEL",
  "log_file": "$AGENT_HOME/logs/agent.log",
  "pid_file": "/run/$SERVICE_NAME/$SERVICE_NAME.pid"
}
EOF
    chown "$AGENT_USER:$AGENT_USER" "$AGENT_CONFIG"
    chmod 600 "$AGENT_CONFIG"
    success "Configuration file created at $AGENT_CONFIG."
}

create_systemd_service() {
    info "Creating systemd service..."
    
    if [[ "$RUN_AS_ROOT" == "true" ]]; then
        # Root mode: minimal restrictions for full filesystem access
        cat > "$SERVICE_FILE" <<EOF
[Unit]
Description=Rclone Backup Agent
Documentation=https://github.com/rclone-backup-web/rclone-backup-web
After=network-online.target
Wants=network-online.target

[Service]
Type=simple
User=root
Group=root
WorkingDirectory=$AGENT_HOME
ExecStart=$AGENT_BIN --config $AGENT_CONFIG
Restart=always
RestartSec=10
RuntimeDirectory=$SERVICE_NAME
PIDFile=/run/$SERVICE_NAME/$SERVICE_NAME.pid

# Minimal security (root access required)
PrivateTmp=true

[Install]
WantedBy=multi-user.target
EOF
        warn "Systemd service configured for ROOT access."
    else
        # Normal mode: security hardened
        cat > "$SERVICE_FILE" <<EOF
[Unit]
Description=Rclone Backup Agent
Documentation=https://github.com/rclone-backup-web/rclone-backup-web
After=network-online.target
Wants=network-online.target

[Service]
Type=simple
User=$AGENT_USER
Group=$AGENT_USER
WorkingDirectory=$AGENT_HOME
ExecStart=$AGENT_BIN --config $AGENT_CONFIG
Restart=always
RestartSec=10
RuntimeDirectory=$SERVICE_NAME
PIDFile=/run/$SERVICE_NAME/$SERVICE_NAME.pid

# Security Hardening
NoNewPrivileges=true
PrivateTmp=true
ProtectSystem=strict
ProtectHome=read-only
ReadWritePaths=$AGENT_HOME

[Install]
WantedBy=multi-user.target
EOF
        success "Systemd service file created with security hardening."
    fi
    
    success "Systemd service file created at $SERVICE_FILE."
}

create_uninstall_script() {
    info "Creating uninstall script..."
    cat > "$UNINSTALL_SCRIPT" <<EOF
#!/bin/bash
set -e

echo "Stopping and disabling service..."
systemctl stop $SERVICE_NAME
systemctl disable $SERVICE_NAME

echo "Removing files..."
rm -f $SERVICE_FILE
systemctl daemon-reload

rm -rf $AGENT_HOME

read -p "Do you want to remove the '$AGENT_USER' user? [y/N] " -r
if [[ \$REPLY =~ ^[Yy]$ ]]; then
    userdel -r $AGENT_USER
    echo "User '$AGENT_USER' removed."
fi

echo "Uninstallation complete."
EOF
    chmod 750 "$UNINSTALL_SCRIPT"
    chown "$AGENT_USER:$AGENT_USER" "$UNINSTALL_SCRIPT"
    success "Uninstall script created at $UNINSTALL_SCRIPT."
}

start_service() {
    info "Starting and enabling the agent service..."
    systemctl daemon-reload
    systemctl enable "$SERVICE_NAME"
    systemctl restart "$SERVICE_NAME"

    # Give it a moment to start
    sleep 3

    if systemctl is-active --quiet "$SERVICE_NAME"; then
        success "Agent service is running."
    else
        error "Agent service failed to start. Check logs with:"
        echo "  journalctl -u $SERVICE_NAME -n 100"
        exit 1
    fi
}

do_install() {
    info "Starting Rclone Backup Agent installation..."
    check_root
    check_systemd
    detect_os_arch
    setup_user_and_dirs
    download_agent
    create_config
    create_systemd_service
    create_uninstall_script
    start_service
    
    echo
    success "Installation complete!"
    info "To manage the service, use:"
    echo "  systemctl status $SERVICE_NAME"
    echo "  systemctl stop $SERVICE_NAME"
    echo "  systemctl start $SERVICE_NAME"
    info "To uninstall, run: sudo bash $UNINSTALL_SCRIPT"
}

do_uninstall() {
    info "Starting Rclone Backup Agent uninstallation..."
    check_root
    if [[ -f "$UNINSTALL_SCRIPT" ]]; then
        bash "$UNINSTALL_SCRIPT"
    else
        error "Uninstall script not found at $UNINSTALL_SCRIPT. Cannot proceed."
        exit 1
    fi
}

# --- Main Execution ---
main() {
    parse_args "$@"

    if [[ "$ACTION" == "uninstall" ]]; then
        do_uninstall
    else
        do_install
    fi
}

main "$@"
