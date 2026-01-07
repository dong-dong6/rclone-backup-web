#!/bin/bash

# ============================================
# Rclone Backup Web V2.0 - 智能部署脚本
# 简化版：PostgreSQL + Hub (内嵌前端)
# ============================================

set -e

# 颜色定义
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
BLUE='\033[0;34m'
CYAN='\033[0;36m'
NC='\033[0m'

print_info() { echo -e "${BLUE}[INFO]${NC} $1"; }
print_success() { echo -e "${GREEN}[SUCCESS]${NC} $1"; }
print_warning() { echo -e "${YELLOW}[WARNING]${NC} $1"; }
print_error() { echo -e "${RED}[ERROR]${NC} $1"; }
print_prompt() { echo -e "${CYAN}[INPUT]${NC} $1"; }

DOCKER_COMPOSE=""
COMPOSE_FILE="docker-compose.yml"

# 检测Docker Compose版本
detect_docker_compose() {
    if docker compose version &> /dev/null; then
        DOCKER_COMPOSE="docker compose"
    elif docker-compose --version &> /dev/null; then
        DOCKER_COMPOSE="docker-compose"
    else
        print_error "未找到 Docker Compose"
        exit 1
    fi
}

# 显示帮助信息
show_help() {
    cat << EOF
Rclone Backup Web V2.0 - 部署脚本 (简化版)

用法:
    ./deploy.sh [命令]

命令:
    deploy           一键部署 Hub + 本地Agent
    hub              仅部署Hub (Docker)
    agent            仅安装本地Agent
    uninstall-agent  卸载本地Agent
    status           查看服务状态
    logs [service]   查看服务日志
    stop             停止所有服务
    restart          重启服务
    clean            交互式清理数据
    help             显示此帮助信息

架构说明:
    Hub (Docker容器):
        - postgres: 数据库
        - hub: Go API + React前端
    
    本地Agent (原生安装):
        - 安装到 /opt/rclone-agent
        - 自动下载rclone
        - 提供TestRemote等功能

示例:
    ./deploy.sh deploy    # 一键部署所有组件
    ./deploy.sh hub       # 仅部署Hub
    ./deploy.sh agent     # 仅安装Agent

EOF
}

# 检查依赖
check_requirements() {
    print_info "检查系统依赖..."
    
    if ! command -v docker &> /dev/null; then
        print_error "Docker未安装"
        exit 1
    fi
    
    detect_docker_compose
    
    if ! docker info &> /dev/null; then
        print_error "Docker服务未运行"
        exit 1
    fi
    
    print_success "依赖检查通过"
}

# 设置环境配置
setup_env() {
    if [ ! -f .env ]; then
        print_info "生成环境配置..."
        
        # 生成随机密钥
        JWT_SECRET=$(openssl rand -hex 32)
        ENCRYPTION_KEY=$(openssl rand -hex 16)
        DB_PASSWORD=$(openssl rand -base64 20 | tr -d "=+/" | cut -c1-20)
        AGENT_API_TOKEN=$(openssl rand -hex 24)
        
        cat > .env << EOF
# Rclone Backup Web - 环境配置
# 生成时间: $(date)

# 数据库
DB_NAME=rclone_backup
DB_USER=rclone
DB_PASSWORD=$DB_PASSWORD
DB_PORT=5432
ALLOW_INSECURE_DB_SSL=true

# 安全密钥
JWT_SECRET=$JWT_SECRET
ENCRYPTION_KEY=$ENCRYPTION_KEY

# 服务端口
WEB_PORT=3000

# 应用设置
GIN_MODE=release
LOG_LEVEL=info
VERSION=latest

# 本地Agent配置
LOCAL_AGENT_URL=http://host.docker.internal:9092
AGENT_API_TOKEN=$AGENT_API_TOKEN
LOCAL_AGENT_TOKEN=$AGENT_API_TOKEN

# 安全配置（可按需调整）
ALLOW_JWT_WITHOUT_SESSION=false
EOF
        
        print_success "配置文件已生成"
    else
        print_info "使用现有配置"
    fi
}

# 设置目录
setup_directories() {
    mkdir -p ./data/{postgres,hub/{data,logs},backups,local-agent}
    chmod 700 ./data/postgres 2>/dev/null || true
}

# 构建并启动Hub
deploy_hub() {
    print_info "部署Hub..."
    
    check_requirements
    setup_env
    setup_directories
    
    # 加载环境变量并导出
    set -a  # 自动导出所有变量
    source .env
    set +a
    
    # 构建镜像
    print_info "构建Hub镜像 (包含前端)..."
    $DOCKER_COMPOSE -f $COMPOSE_FILE build hub
    
    # 启动数据库
    print_info "启动数据库..."
    $DOCKER_COMPOSE -f $COMPOSE_FILE up -d postgres
    
    # 等待数据库就绪
    print_info "等待数据库初始化..."
    for i in {1..30}; do
        if $DOCKER_COMPOSE -f $COMPOSE_FILE exec -T postgres pg_isready -U ${DB_USER:-rclone} &> /dev/null; then
            break
        fi
        echo -n "."
        sleep 2
    done
    echo ""
    
    # 创建管理员账户
    create_admin_account
    
    # 启动Hub
    print_info "启动Hub服务..."
    $DOCKER_COMPOSE -f $COMPOSE_FILE up -d hub
    
    # 等待服务就绪
    wait_for_hub
    
    print_success "Hub部署完成!"
}

# 创建管理员账户
create_admin_account() {
    source .env
    
    local admin_exists=$($DOCKER_COMPOSE -f $COMPOSE_FILE exec -T postgres psql -U ${DB_USER} -d ${DB_NAME} -tAc "SELECT COUNT(*) FROM users WHERE username='admin';" 2>/dev/null || echo "0")
    
    if [ "$admin_exists" -gt 0 ]; then
        print_info "管理员账户已存在"
        return 0
    fi
    
    print_info "创建管理员账户..."
    
    local ADMIN_PASSWORD=$(openssl rand -base64 16 | tr -d "=+/" | cut -c1-16)
    
    # 使用Docker生成bcrypt哈希
    local ADMIN_PASSWORD_HASH=$(docker run --rm alpine sh -c "apk add --no-cache py3-bcrypt > /dev/null 2>&1 && python3 -c \"import bcrypt; print(bcrypt.hashpw(b'$ADMIN_PASSWORD', bcrypt.gensalt(10)).decode())\"" 2>/dev/null)
    
    if [ -z "$ADMIN_PASSWORD_HASH" ]; then
        ADMIN_PASSWORD="changeme123"
        ADMIN_PASSWORD_HASH='$2a$10$rLJHvVQzMmKGvE5xF5xLN.XqYvHZYF7CdJxvCp7qP6vhqZWYxZKK6'
    fi
    
    $DOCKER_COMPOSE -f $COMPOSE_FILE exec -T postgres psql -U ${DB_USER} -d ${DB_NAME} <<-EOSQL
        INSERT INTO users (username, email, password_hash, full_name, role, is_active, is_admin)
        VALUES ('admin', 'admin@local', '$ADMIN_PASSWORD_HASH', 'Administrator', 'admin', true, true)
        ON CONFLICT (username) DO NOTHING;
EOSQL
    
    # 保存密码
    mkdir -p ./data/hub
    echo "$ADMIN_PASSWORD" > ./data/hub/admin_password.txt
    chmod 600 ./data/hub/admin_password.txt
    
    export INITIAL_ADMIN_PASSWORD="$ADMIN_PASSWORD"
    print_success "管理员账户已创建"
}

# 等待Hub就绪
wait_for_hub() {
    print_info "等待Hub启动..."
    
    for i in {1..30}; do
        if curl -sf http://localhost:${WEB_PORT:-3000}/health &> /dev/null; then
            print_success "Hub已就绪"
            return 0
        fi
        echo -n "."
        sleep 2
    done
    echo ""
    
    print_warning "Hub启动超时，请检查日志: ./deploy.sh logs hub"
}

# 安装本地Agent
install_agent() {
    print_info "安装本地Agent..."
    
    local AGENT_DIR="/opt/rclone-agent"
    local AGENT_BIN="$AGENT_DIR/rclone-backup-agent"
    
    # 检查是否需要sudo
    local SUDO=""
    if [ "$(id -u)" != "0" ]; then
        SUDO="sudo"
    fi
    
    # 创建目录
    $SUDO mkdir -p $AGENT_DIR/{bin,configs,tasks}
    
    # 检测平台
    local OS=$(uname -s | tr '[:upper:]' '[:lower:]')
    local ARCH=$(uname -m)
    case $ARCH in
        x86_64) ARCH="amd64" ;;
        aarch64) ARCH="arm64" ;;
        armv7l) ARCH="arm" ;;
    esac
    
    local BINARY_NAME="rclone-backup-agent-${OS}-${ARCH}"
    
    # 编译Agent (如果源码存在)
    if [ -d "./agent" ] && command -v go &> /dev/null; then
        print_info "从源码编译Agent..."
        cd agent
        go build -o $BINARY_NAME ./main_standalone.go
        $SUDO cp $BINARY_NAME $AGENT_BIN
        $SUDO chmod +x $AGENT_BIN
        cd ..
    else
        # 从Hub下载
        print_info "从Hub下载Agent..."
        local HUB_URL="http://localhost:${WEB_PORT:-3000}"
        curl -sf "${HUB_URL}/api/v1/agent/download?platform=${OS}&arch=${ARCH}" -o /tmp/rclone-backup-agent
        $SUDO mv /tmp/rclone-backup-agent $AGENT_BIN
        $SUDO chmod +x $AGENT_BIN
    fi
    
    # 创建配置文件
    source .env 2>/dev/null || true
    
    local HUB_URL="http://localhost:${WEB_PORT:-3000}"

    # 确保本地Agent HTTP API Token 存在（用于Hub调用 /api/test-remote）
    local LOCAL_API_TOKEN=""
    if [ -n "${LOCAL_AGENT_TOKEN:-}" ]; then
        LOCAL_API_TOKEN="$LOCAL_AGENT_TOKEN"
    elif [ -n "${AGENT_API_TOKEN:-}" ]; then
        LOCAL_API_TOKEN="$AGENT_API_TOKEN"
    else
        LOCAL_API_TOKEN=$(openssl rand -hex 24)
        print_warning "未检测到 LOCAL_AGENT_TOKEN/AGENT_API_TOKEN，已生成新的本地Agent API Token"
    fi

    # 尝试回写/补全 .env，确保Hub容器可拿到Token
    if [ -f .env ]; then
        if grep -q '^AGENT_API_TOKEN=' .env; then
            sed -i "s/^AGENT_API_TOKEN=.*/AGENT_API_TOKEN=$LOCAL_API_TOKEN/" .env
        else
            echo "AGENT_API_TOKEN=$LOCAL_API_TOKEN" >> .env
        fi
        if grep -q '^LOCAL_AGENT_TOKEN=' .env; then
            sed -i "s/^LOCAL_AGENT_TOKEN=.*/LOCAL_AGENT_TOKEN=$LOCAL_API_TOKEN/" .env
        else
            echo "LOCAL_AGENT_TOKEN=$LOCAL_API_TOKEN" >> .env
        fi
    fi
    
    # 生成注册 token 并注册 Agent
    print_info "生成Agent注册令牌..."
    
    # 获取管理员登录 token
    local ADMIN_PASSWORD=""
    if [ -f "./data/hub/admin_password.txt" ]; then
        ADMIN_PASSWORD=$(cat ./data/hub/admin_password.txt)
    else
        ADMIN_PASSWORD="changeme123"
    fi
    
    # 登录获取 session token
    local LOGIN_RESPONSE=$(curl -sf -X POST "${HUB_URL}/api/v1/admin/login" \
        -H "Content-Type: application/json" \
        -d "{\"username\": \"admin\", \"password\": \"${ADMIN_PASSWORD}\"}" 2>/dev/null || echo "")
    
    local AUTH_TOKEN=$(echo "$LOGIN_RESPONSE" | grep -o '"token":"[^"]*"' | sed 's/"token":"//;s/"$//' || echo "")
    
    local REG_TOKEN=""
    if [ -n "$AUTH_TOKEN" ]; then
        # 生成注册令牌
        local TOKEN_RESPONSE=$(curl -sf -X POST "${HUB_URL}/api/v1/admin/agents/registration-token" \
            -H "Authorization: Bearer ${AUTH_TOKEN}" 2>/dev/null || echo "")
        
        REG_TOKEN=$(echo "$TOKEN_RESPONSE" | grep -o '"token":"[^"]*"' | sed 's/"token":"//;s/"$//' || echo "")
    fi
    
    if [ -z "$REG_TOKEN" ]; then
        print_warning "无法自动生成注册令牌，请手动注册Agent"
        print_info "1. 登录Web界面"
        print_info "2. 在节点管理页面生成注册令牌"
        print_info "3. 完成注册（看到 Successfully registered 后 Ctrl+C 退出）："
        print_info "   sudo $AGENT_BIN --config $AGENT_DIR/agent.json --token <TOKEN>"
        print_info "4. 启用并启动服务："
        print_info "   sudo systemctl enable --now rclone-backup-agent"

        $SUDO tee $AGENT_DIR/agent.json > /dev/null << EOF
{
    "hub_url": "$HUB_URL",
    "agent_name": "local-agent",
    "work_dir": "$AGENT_DIR",
    "max_concurrent": 3,
    "heartbeat_interval": 30,
    "is_local": true,
    "enable_api": true,
    "api_bind_addr": "0.0.0.0",
    "api_port": 9092,
    "api_token": "$LOCAL_API_TOKEN",
    "registration_token": ""
}
EOF

        print_success "Agent已安装到 $AGENT_DIR（未注册/未启动）"

        # 创建systemd服务（不启动）
        if command -v systemctl &> /dev/null; then
            create_agent_service false
        else
            print_info "手动启动（完成注册后保持运行即可）: $AGENT_BIN --config $AGENT_DIR/agent.json --token <TOKEN>"
        fi
        return 0
    fi

    print_success "注册令牌已生成"
    
    $SUDO tee $AGENT_DIR/agent.json > /dev/null << EOF
{
    "hub_url": "$HUB_URL",
    "agent_name": "local-agent",
    "work_dir": "$AGENT_DIR",
    "max_concurrent": 3,
    "heartbeat_interval": 30,
    "is_local": true,
    "enable_api": true,
    "api_bind_addr": "0.0.0.0",
    "api_port": 9092,
    "api_token": "$LOCAL_API_TOKEN",
    "registration_token": "$REG_TOKEN"
}
EOF
    
    print_success "Agent已安装到 $AGENT_DIR"
    
    # 创建systemd服务
    if command -v systemctl &> /dev/null; then
        create_agent_service true
    else
        print_info "启动Agent: $AGENT_BIN --config $AGENT_DIR/agent.json"
    fi
}

# 创建systemd服务
create_agent_service() {
    local AUTO_START="${1:-true}"
    local SUDO=""
    if [ "$(id -u)" != "0" ]; then
        SUDO="sudo"
    fi
    
    $SUDO tee /etc/systemd/system/rclone-backup-agent.service > /dev/null << EOF
[Unit]
Description=Rclone Backup Agent
After=network.target

[Service]
Type=simple
Environment="RCLONE_CHECKSUM_LINUX_AMD64=07c23d21a94d70113d949253478e13261c54d14d72023bb14d96a8da5f3e7722"
ExecStart=/opt/rclone-agent/rclone-backup-agent --config /opt/rclone-agent/agent.json
Restart=always
RestartSec=10
WorkingDirectory=/opt/rclone-agent

[Install]
WantedBy=multi-user.target
EOF
    
    $SUDO systemctl daemon-reload

    if [ "$AUTO_START" = "true" ]; then
        $SUDO systemctl enable rclone-backup-agent
        $SUDO systemctl start rclone-backup-agent
        print_success "Agent服务已启动"
    else
        print_info "Agent服务已创建（未启动）"
    fi
}

# 一键部署
deploy_all() {
    print_info "开始一键部署..."
    echo ""
    echo "部署架构:"
    echo "  ┌─────────────────────────────────────┐"
    echo "  │  Docker                             │"
    echo "  │  ├── postgres (数据库)              │"
    echo "  │  └── hub (API + 前端)               │"
    echo "  └─────────────────────────────────────┘"
    echo "  ┌─────────────────────────────────────┐"
    echo "  │  本地                               │"
    echo "  │  └── agent (备份执行 + TestRemote)  │"
    echo "  └─────────────────────────────────────┘"
    echo ""
    
    # 部署Hub
    deploy_hub
    
    # 安装Agent
    install_agent
    
    # 显示访问信息
    show_access_info
}

# 显示访问信息
show_access_info() {
    source .env 2>/dev/null || true
    
    echo ""
    echo -e "${GREEN}━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━${NC}"
    echo -e "${GREEN}✨ 部署成功！${NC}"
    echo ""
    echo "📍 访问地址: http://localhost:${WEB_PORT:-3000}"
    echo ""
    
    if [ -f "./data/hub/admin_password.txt" ]; then
        local PASSWORD=$(cat ./data/hub/admin_password.txt)
        echo "🔑 管理员账号:"
        echo "   用户名: admin"
        echo "   密码: $PASSWORD"
        echo ""
        echo "   ⚠️  请立即修改密码!"
    fi
    
    echo ""
    echo "📝 常用命令:"
    echo "   查看状态: ./deploy.sh status"
    echo "   查看日志: ./deploy.sh logs hub"
    echo "   停止服务: ./deploy.sh stop"
    echo -e "${GREEN}━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━${NC}"
}

# 查看状态
show_status() {
    check_requirements
    
    echo ""
    print_info "Docker服务状态:"
    $DOCKER_COMPOSE -f $COMPOSE_FILE ps
    
    echo ""
    print_info "本地Agent状态:"
    if command -v systemctl &> /dev/null; then
        systemctl status rclone-backup-agent --no-pager 2>/dev/null || echo "Agent服务未安装"
    else
        pgrep -f rclone-backup-agent &>/dev/null && echo "Agent正在运行" || echo "Agent未运行"
    fi
}

# 查看日志
show_logs() {
    check_requirements
    
    if [ -n "$1" ]; then
        $DOCKER_COMPOSE -f $COMPOSE_FILE logs -f --tail=100 "$1"
    else
        $DOCKER_COMPOSE -f $COMPOSE_FILE logs -f --tail=100
    fi
}

# 停止服务
stop_services() {
    check_requirements
    
    print_info "停止Docker服务..."
    $DOCKER_COMPOSE -f $COMPOSE_FILE down
    
    print_info "停止本地Agent..."
    if command -v systemctl &> /dev/null; then
        sudo systemctl stop rclone-backup-agent 2>/dev/null || true
    else
        pkill -f rclone-backup-agent 2>/dev/null || true
    fi
    
    print_success "所有服务已停止"
}

# 重启服务
restart_services() {
    check_requirements
    
    print_info "重启服务..."
    $DOCKER_COMPOSE -f $COMPOSE_FILE restart
    
    if command -v systemctl &> /dev/null; then
        sudo systemctl restart rclone-backup-agent 2>/dev/null || true
    fi
    
    print_success "服务已重启"
}

# 清理数据
clean_data() {
    check_requirements
    
    print_warning "⚠️  此操作将删除所有数据并卸载本地Agent!"
    print_prompt "输入 'DELETE' 确认: "
    read -r confirm
    
    if [ "$confirm" = "DELETE" ]; then
        stop_services
        rm -rf ./data
        print_success "数据已清理"
        
        # 卸载本地Agent（跳过确认）
        uninstall_agent_force
    else
        print_info "操作已取消"
    fi
}

# 卸载本地Agent（带确认）
uninstall_agent() {
    print_warning "⚠️  此操作将完全删除本地Agent!"
    print_prompt "输入 'UNINSTALL' 确认: "
    read -r confirm
    
    if [ "$confirm" != "UNINSTALL" ]; then
        print_info "操作已取消"
        return 0
    fi
    
    uninstall_agent_force
}

# 卸载本地Agent（不需要确认，供内部调用）
uninstall_agent_force() {
    local SUDO=""
    if [ "$(id -u)" != "0" ]; then
        SUDO="sudo"
    fi
    
    local AGENT_DIR="/opt/rclone-agent"
    
    # 停止并禁用服务
    if command -v systemctl &> /dev/null; then
        print_info "停止Agent服务..."
        $SUDO systemctl stop rclone-backup-agent 2>/dev/null || true
        $SUDO systemctl disable rclone-backup-agent 2>/dev/null || true
        $SUDO rm -f /etc/systemd/system/rclone-backup-agent.service
        $SUDO systemctl daemon-reload
        print_success "服务已停止并移除"
    else
        # 非systemd系统，直接杀进程
        pkill -f rclone-backup-agent 2>/dev/null || true
    fi
    
    # 删除安装目录
    if [ -d "$AGENT_DIR" ]; then
        print_info "删除Agent目录: $AGENT_DIR"
        $SUDO rm -rf "$AGENT_DIR"
        print_success "Agent目录已删除"
    else
        print_info "Agent目录不存在: $AGENT_DIR"
    fi
    
    print_success "本地Agent已完全卸载"
}

# 主函数
main() {
    case "$1" in
        deploy)
            deploy_all
            ;;
        hub)
            deploy_hub
            show_access_info
            ;;
        agent)
            install_agent
            ;;
        uninstall-agent)
            uninstall_agent
            ;;
        status)
            show_status
            ;;
        logs)
            show_logs "$2"
            ;;
        stop)
            stop_services
            ;;
        restart)
            restart_services
            ;;
        clean)
            clean_data
            ;;
        help|--help|-h|"")
            show_help
            ;;
        *)
            print_error "未知命令: $1"
            show_help
            exit 1
            ;;
    esac
}

main "$@"
