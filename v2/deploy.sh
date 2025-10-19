#!/bin/bash

# ============================================
# Rclone Backup Web V2.0 - 智能部署脚本
# 特性：
# - 自动检测Docker Compose版本
# - 交互式配置生成
# - 透明的数据持久化（./data目录）
# - 智能数据清理与备份
# - 本地镜像构建
# ============================================

set -e

# 颜色定义
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
BLUE='\033[0;34m'
CYAN='\033[0;36m'
MAGENTA='\033[0;35m'
NC='\033[0m' # No Color

# 打印带颜色的消息
print_info() {
    echo -e "${BLUE}[INFO]${NC} $1"
}

print_success() {
    echo -e "${GREEN}[SUCCESS]${NC} $1"
}

print_warning() {
    echo -e "${YELLOW}[WARNING]${NC} $1"
}

print_error() {
    echo -e "${RED}[ERROR]${NC} $1"
}

print_prompt() {
    echo -e "${CYAN}[INPUT]${NC} $1"
}

# Docker Compose命令兼容性处理
DOCKER_COMPOSE=""

# 检测Docker Compose版本
detect_docker_compose() {
    if docker compose version &> /dev/null; then
        DOCKER_COMPOSE="docker compose"
        print_info "检测到 Docker Compose V2 (docker compose)"
    elif docker-compose --version &> /dev/null; then
        DOCKER_COMPOSE="docker-compose"
        print_info "检测到 Docker Compose V1 (docker-compose)"
    else
        print_error "未找到 Docker Compose，请安装 Docker Compose"
        echo "安装方法："
        echo "  - 新版Docker已内置: 确保Docker版本 >= 20.10"
        echo "  - 独立安装: https://docs.docker.com/compose/install/"
        exit 1
    fi
}

# 显示帮助信息
show_help() {
    cat << EOF
Rclone Backup Web V2.0 - 部署脚本

用法:
    ./deploy.sh [命令] [选项]

命令:
    hub              部署Hub（不含Agent）
    hub-with-agent   部署Hub（含本地Agent）
    agent            部署独立Agent
    build            构建所有镜像
    status           查看服务状态
    logs [service]   查看服务日志
    stop             停止所有服务
    restart          重启服务
    clean            交互式清理数据
    backup           备份数据目录
    restore <file>   从备份恢复
    help             显示此帮助信息

选项:
    --clean          部署前清理数据
    --force          跳过确认提示

数据目录:
    所有数据存储在 ./data 目录中：
    ./data/postgres  - 数据库
    ./data/redis     - 缓存
    ./data/hub       - Hub数据
    ./data/agent     - Agent数据
    ./data/backups   - 自动备份

示例:
    ./deploy.sh hub              # 部署Hub服务
    ./deploy.sh hub-with-agent   # 部署Hub和本地Agent
    ./deploy.sh backup           # 备份数据
    ./deploy.sh logs hub-api     # 查看Hub日志

EOF
}

# 检查依赖
check_requirements() {
    print_info "检查系统依赖..."
    
    # 检查Docker
    if ! command -v docker &> /dev/null; then
        print_error "Docker未安装，请先安装Docker"
        echo "安装指南: https://docs.docker.com/get-docker/"
        exit 1
    fi
    
    # 检测Docker Compose
    detect_docker_compose
    
    # 检查Docker服务是否运行
    if ! docker info &> /dev/null; then
        print_error "Docker服务未运行，请启动Docker"
        exit 1
    fi
    
    print_success "依赖检查通过"
}

# 显示数据目录信息
show_data_info() {
    echo ""
    echo -e "${MAGENTA}━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━${NC}"
    echo -e "${MAGENTA}📂 数据目录结构${NC}"
    echo -e "${MAGENTA}━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━${NC}"
    echo ""
    echo "所有持久化数据将存储在 ./data 目录中："
    echo ""
    echo "  ./data/"
    echo "  ├── postgres/        # PostgreSQL数据库"
    echo "  ├── redis/           # Redis缓存"
    echo "  ├── hub/"
    echo "  │   ├── config/      # Hub配置文件"
    echo "  │   ├── data/        # Hub运行数据"
    echo "  │   └── logs/        # Hub日志"
    echo "  ├── agent/           # Agent数据 (如果启用)"
    echo "  └── backups/         # 数据库备份"
    echo ""
}

# 检查现有数据
check_existing_data() {
    if [ -d "./data" ]; then
        local size=$(du -sh ./data 2>/dev/null | cut -f1)
        print_warning "检测到现有数据目录 (大小: ${size})"
        
        # 列出主要数据目录
        echo ""
        echo "现有数据内容："
        for dir in postgres redis hub agent backups; do
            if [ -d "./data/$dir" ]; then
                local dir_size=$(du -sh ./data/$dir 2>/dev/null | cut -f1)
                echo "  - $dir: $dir_size"
            fi
        done
        return 0
    else
        return 1
    fi
}

# 备份现有数据
backup_existing_data() {
    local backup_dir="./data.backup.$(date +%Y%m%d-%H%M%S)"
    print_info "备份现有数据到 $backup_dir ..."
    
    # 停止容器以确保数据一致性
    print_info "停止运行中的容器..."
    $DOCKER_COMPOSE down 2>/dev/null || true
    
    # 创建备份
    if mv ./data "$backup_dir"; then
        print_success "数据已备份到: $backup_dir"
        echo ""
        echo "  提示：您可以通过以下命令恢复数据："
        echo "  mv $backup_dir ./data"
        echo ""
        return 0
    else
        print_error "备份失败"
        return 1
    fi
}

# 清理数据函数（交互式）
cleanup_data_interactive() {
    echo ""
    if check_existing_data; then
        echo ""
        print_warning "⚠️  警告：清理数据是不可逆的操作！"
        echo ""
        echo "请选择操作："
        echo "  1) 保留现有数据（推荐）"
        echo "  2) 备份后清理"
        echo "  3) 直接清理（危险！）"
        echo "  4) 取消操作"
        echo ""
        
        print_prompt "请输入选择 [1-4]: "
        read -r choice
        
        case "$choice" in
            1)
                print_info "保留现有数据"
                ;;
            2)
                if backup_existing_data; then
                    mkdir -p ./data
                    print_success "数据清理完成（已备份）"
                else
                    print_error "备份失败，已取消清理"
                    exit 1
                fi
                ;;
            3)
                print_prompt "请输入 'DELETE' 确认删除所有数据: "
                read -r confirm
                if [ "$confirm" = "DELETE" ]; then
                    print_info "正在清理数据..."
                    $DOCKER_COMPOSE down -v 2>/dev/null || true
                    rm -rf ./data
                    mkdir -p ./data
                    print_success "数据已清理"
                else
                    print_info "已取消清理"
                fi
                ;;
            4)
                print_info "操作已取消"
                exit 0
                ;;
            *)
                print_error "无效的选择"
                exit 1
                ;;
        esac
    else
        print_info "未检测到现有数据"
        mkdir -p ./data
    fi
}

# 生成环境配置
generate_env_config() {
    print_info "开始配置环境变量..."
    echo ""
    
    # 数据库配置
    print_prompt "请输入数据库名称 [默认: rclone_backup]: "
    read -r DB_NAME
    DB_NAME=${DB_NAME:-rclone_backup}
    
    print_prompt "请输入数据库用户名 [默认: rclone]: "
    read -r DB_USER
    DB_USER=${DB_USER:-rclone}
    
    print_prompt "请输入数据库密码 [自动生成]: "
    read -r -s DB_PASSWORD
    echo ""
    if [ -z "$DB_PASSWORD" ]; then
        DB_PASSWORD=$(openssl rand -base64 20 | tr -d "=+/" | cut -c1-20)
        print_info "已自动生成数据库密码"
    fi
    
    # 端口配置
    print_prompt "请输入Hub API端口 [默认: 8080]: "
    read -r HUB_PORT
    HUB_PORT=${HUB_PORT:-8080}
    
    print_prompt "请输入Web UI端口 [默认: 3000]: "
    read -r WEB_PORT
    WEB_PORT=${WEB_PORT:-3000}
    
    # 生成安全密钥
    print_info "生成安全密钥..."
    JWT_SECRET=$(openssl rand -hex 32)
    ENCRYPTION_KEY=$(openssl rand -hex 16)
    
    # 创建.env文件
    cat > .env << EOF
# ============================================
# Rclone Backup Web V2.0 - 环境配置
# 生成时间: $(date)
# ============================================

# 数据库配置
DB_NAME=$DB_NAME
DB_USER=$DB_USER
DB_PASSWORD=$DB_PASSWORD

# 安全密钥（自动生成）
JWT_SECRET=$JWT_SECRET
ENCRYPTION_KEY=$ENCRYPTION_KEY

# 服务端口
HUB_PORT=$HUB_PORT
WEB_PORT=$WEB_PORT
METRICS_PORT=9090

# 应用设置
GIN_MODE=release
LOG_LEVEL=info
VERSION=latest

# Hub配置
SESSION_TIMEOUT=24h
API_KEY_EXPIRY=365d
ENABLE_METRICS=true
ENABLE_PROFILING=false

# Redis配置
REDIS_MAX_MEMORY=256mb

# 数据库备份间隔（秒）
DB_BACKUP_INTERVAL=86400

# 本地Agent配置（可选）
LOCAL_AGENT_REGISTRATION_TOKEN=
LOCAL_AGENT_NAME=hub-local-agent
HEARTBEAT_INTERVAL=30s
TASK_TIMEOUT=2h
ENABLE_LOCAL_FALLBACK=true
LOCAL_FALLBACK_THRESHOLD=5m

# Rclone配置
RCLONE_VERSION=latest
RCLONE_LOG_LEVEL=INFO
RCLONE_CPU_LIMIT=2.0
RCLONE_MEMORY_LIMIT=1G
EOF
    
    print_success "配置文件 .env 已生成"
}

# 设置环境配置
setup_env() {
    if [ ! -f .env ]; then
        print_warning "未找到 .env 文件"
        echo ""
        echo "请选择配置方式："
        echo "  1) 交互式配置（推荐）"
        echo "  2) 使用默认配置"
        echo "  3) 退出"
        echo ""
        print_prompt "请选择 [1-3]: "
        read -r choice
        
        case $choice in
            1)
                generate_env_config
                ;;
            2)
                print_info "使用默认配置..."
                cp .env.example .env 2>/dev/null || generate_env_config
                
                # 生成随机密钥
                JWT_SECRET=$(openssl rand -hex 32)
                ENCRYPTION_KEY=$(openssl rand -hex 16)
                DB_PASSWORD=$(openssl rand -base64 20 | tr -d "=+/" | cut -c1-20)
                
                # 替换默认值
                if [[ "$OSTYPE" == "darwin"* ]]; then
                    # macOS
                    sed -i '' "s/JWT_SECRET=.*/JWT_SECRET=$JWT_SECRET/" .env
                    sed -i '' "s/ENCRYPTION_KEY=.*/ENCRYPTION_KEY=$ENCRYPTION_KEY/" .env
                    sed -i '' "s/DB_PASSWORD=.*/DB_PASSWORD=$DB_PASSWORD/" .env
                else
                    # Linux
                    sed -i "s/JWT_SECRET=.*/JWT_SECRET=$JWT_SECRET/" .env
                    sed -i "s/ENCRYPTION_KEY=.*/ENCRYPTION_KEY=$ENCRYPTION_KEY/" .env
                    sed -i "s/DB_PASSWORD=.*/DB_PASSWORD=$DB_PASSWORD/" .env
                fi
                
                print_success "已使用默认配置并生成安全密钥"
                ;;
            3)
                print_info "退出安装"
                exit 0
                ;;
            *)
                print_error "无效选择"
                exit 1
                ;;
        esac
    else
        print_info "使用现有的 .env 配置文件"
    fi
}

# 设置目录权限
setup_permissions() {
    print_info "设置目录权限..."
    
    # 创建必要的目录
    mkdir -p ./data/{postgres,redis,hub/{config,data,logs},agent,backups}
    
    # 设置权限（某些服务需要特定的权限）
    # PostgreSQL需要700权限
    chmod 700 ./data/postgres 2>/dev/null || true
    
    print_success "目录权限已设置"
}

# 处理网络冲突
handle_network_conflicts() {
    print_info "处理Docker网络..."
    
    # 删除可能冲突的网络
    NETWORK_NAME="v2_backend"
    if docker network ls | grep -q "$NETWORK_NAME"; then
        print_warning "删除现有网络 $NETWORK_NAME..."
        docker network rm "$NETWORK_NAME" 2>/dev/null || true
    fi
    
    print_success "网络准备就绪"
}

# 构建镜像
build_images() {
    print_info "开始构建Docker镜像..."
    
    # 检查使用哪个配置文件
    if [ -f "docker-compose.prod.yml" ]; then
        COMPOSE_FILE="docker-compose.prod.yml"
    else
        COMPOSE_FILE="docker-compose.yml"
    fi
    
    # 构建Hub镜像
    print_info "构建Hub API镜像..."
    $DOCKER_COMPOSE -f $COMPOSE_FILE build hub-api
    
    print_info "构建Web UI镜像..."
    $DOCKER_COMPOSE -f $COMPOSE_FILE build web-ui
    
    # 构建Agent镜像（如果需要）
    if [[ "$1" == *"agent"* ]]; then
        print_info "构建Agent镜像..."
        $DOCKER_COMPOSE -f $COMPOSE_FILE --profile local-agent build local-agent 2>/dev/null || true
    fi
    
    print_success "所有镜像构建完成"
}

# 部署Hub
deploy_hub() {
    print_info "开始部署Hub服务..."
    
    check_requirements
    setup_env
    
    # 询问是否清理数据
    if [[ "$2" == "--clean" ]]; then
        cleanup_data_interactive
    fi
    
    # 设置权限
    setup_permissions
    
    # 处理网络冲突
    handle_network_conflicts
    
    # 构建镜像
    build_images "hub"
    
    # 选择配置文件
    if [ -f "docker-compose.prod.yml" ]; then
        COMPOSE_FILE="docker-compose.prod.yml"
    else
        COMPOSE_FILE="docker-compose.yml"
    fi
    
    # 启动服务（分步启动以确保依赖顺序）
    print_info "启动数据库服务..."
    $DOCKER_COMPOSE -f $COMPOSE_FILE up -d postgres
    
    # 等待数据库完全就绪
    print_info "等待数据库初始化..."
    local db_ready=0
    for i in {1..30}; do
        if $DOCKER_COMPOSE -f $COMPOSE_FILE exec -T postgres pg_isready -U ${DB_USER:-rclone} &> /dev/null; then
            db_ready=1
            break
        fi
        echo -n "."
        sleep 2
    done
    echo ""
    
    if [ $db_ready -eq 0 ]; then
        print_error "数据库启动失败"
        print_info "查看数据库日志："
        $DOCKER_COMPOSE -f $COMPOSE_FILE logs postgres | tail -20
        exit 1
    fi
    
    print_success "数据库已就绪"
    
    # 启动其他服务
    print_info "启动Hub服务..."
    $DOCKER_COMPOSE -f $COMPOSE_FILE up -d redis hub-api web-ui
    
    # 等待服务启动
    print_info "等待服务启动..."
    local max_attempts=30
    local attempt=0
    
    while [ $attempt -lt $max_attempts ]; do
        # 方法1: 直接访问容器IP（更可靠）
        CONTAINER_IP=$(docker inspect -f '{{range .NetworkSettings.Networks}}{{.IPAddress}}{{end}}' v2-hub-api-1 2>/dev/null || echo "")
        
        if [ -n "$CONTAINER_IP" ]; then
            # 使用容器IP进行健康检查
            HEALTH_CHECK_URL="http://${CONTAINER_IP}:8080/health"
            CURL_RESPONSE=$(curl -s -o /dev/null -w "%{http_code}" $HEALTH_CHECK_URL 2>/dev/null || echo "000")
            
            if [ "$CURL_RESPONSE" = "200" ]; then
                print_success "Hub服务部署成功！（通过容器IP: ${CONTAINER_IP}）"
                # 额外检查localhost映射
                if curl -sf http://localhost:${HUB_PORT:-8080}/health &> /dev/null; then
                    print_info "端口映射正常 (localhost:${HUB_PORT:-8080} 可访问)"
                else
                    print_warning "注意: localhost:${HUB_PORT:-8080} 不可访问，但服务运行正常"
                    print_info "可通过容器IP访问: http://${CONTAINER_IP}:8080"
                fi
                show_access_info
                return 0
            elif [ "$CURL_RESPONSE" = "000" ]; then
                echo -n "."
            else
                echo -n "[$CURL_RESPONSE]"
            fi
        else
            # 方法2: 如果无法获取容器IP，使用docker exec
            if docker compose exec -T hub-api wget -q -O /dev/null http://localhost:8080/health 2>/dev/null; then
                print_success "Hub服务部署成功！（通过容器内部验证）"
                show_access_info
                return 0
            else
                echo -n "."
            fi
        fi
        
        sleep 2
        ((attempt++))
    done
    
    print_error "Hub服务启动超时，请检查日志"
    $DOCKER_COMPOSE -f $COMPOSE_FILE logs hub-api
    exit 1
}

# 部署Hub和本地Agent
deploy_hub_with_agent() {
    print_info "开始部署Hub服务（含本地Agent）..."
    
    check_requirements
    setup_env
    
    # 询问是否清理数据
    if [[ "$2" == "--clean" ]]; then
        cleanup_data_interactive
    fi
    
    # 设置权限
    setup_permissions
    
    # 处理网络冲突
    handle_network_conflicts
    
    # 先部署Hub
    deploy_hub
    
    # 检查是否已配置Agent令牌
    source .env
    if [ -n "$LOCAL_AGENT_REGISTRATION_TOKEN" ] && [ "$LOCAL_AGENT_REGISTRATION_TOKEN" != "" ]; then
        print_info "检测到已配置的Agent令牌，跳过令牌获取步骤"
    else
        # 提示获取令牌
        echo ""
        print_warning "需要配置本地Agent"
        echo ""
        echo "请按以下步骤操作："
        echo "  1. 打开浏览器访问: http://localhost:${WEB_PORT:-3000}"
        echo "  2. 使用 admin/admin 登录系统"
        echo "  3. 进入 Agents 页面"
        echo "  4. 点击 '生成注册令牌'"
        echo "  5. 复制生成的令牌"
        echo ""
        print_prompt "请输入注册令牌: "
        read -r token
        
        if [ -z "$token" ]; then
            print_error "令牌不能为空"
            exit 1
        fi
        
        # 更新.env文件
        if [[ "$OSTYPE" == "darwin"* ]]; then
            sed -i '' "s/LOCAL_AGENT_REGISTRATION_TOKEN=.*/LOCAL_AGENT_REGISTRATION_TOKEN=$token/" .env
        else
            sed -i "s/LOCAL_AGENT_REGISTRATION_TOKEN=.*/LOCAL_AGENT_REGISTRATION_TOKEN=$token/" .env
        fi
    fi
    
    # 构建Agent镜像
    build_images "agent"
    
    # 选择配置文件
    if [ -f "docker-compose.prod.yml" ]; then
        COMPOSE_FILE="docker-compose.prod.yml"
    else
        COMPOSE_FILE="docker-compose.yml"
    fi
    
    # 启动本地Agent
    print_info "启动本地Agent..."
    $DOCKER_COMPOSE -f $COMPOSE_FILE --profile local-agent up -d local-agent local-rclone-sidecar
    
    # 检查Agent状态
    sleep 5
    if $DOCKER_COMPOSE -f $COMPOSE_FILE ps | grep -q "local-agent.*Up\|local-agent.*running"; then
        print_success "Hub和本地Agent部署成功！"
        echo ""
        echo "本地Agent已启动，请在Web UI的Agents页面查看"
    else
        print_error "本地Agent启动失败，请检查日志"
        $DOCKER_COMPOSE -f $COMPOSE_FILE logs local-agent
    fi
}

# 部署独立Agent
deploy_agent() {
    print_info "开始配置独立Agent..."
    
    check_requirements
    
    # 创建Agent配置
    if [ ! -f .env.agent ]; then
        print_info "配置Agent连接信息..."
        
        print_prompt "请输入Hub服务器地址 (例如: http://hub.example.com:8080): "
        read -r HUB_URL
        
        if [ -z "$HUB_URL" ]; then
            print_error "Hub地址不能为空"
            exit 1
        fi
        
        print_prompt "请输入Agent注册令牌: "
        read -r REGISTRATION_TOKEN
        
        if [ -z "$REGISTRATION_TOKEN" ]; then
            print_error "注册令牌不能为空"
            exit 1
        fi
        
        print_prompt "请输入Agent名称 [默认: $(hostname)]: "
        read -r AGENT_NAME
        AGENT_NAME=${AGENT_NAME:-$(hostname)}
        
        # 创建配置文件
        cat > .env.agent << EOF
# Agent配置
HUB_URL=$HUB_URL
REGISTRATION_TOKEN=$REGISTRATION_TOKEN
AGENT_NAME=$AGENT_NAME

# 备份源路径
BACKUP_SOURCE_1=/var/www
BACKUP_SOURCE_2=/home
BACKUP_SOURCE_3=/etc

# Rclone配置
RCLONE_VERSION=latest
RCLONE_CPU_LIMIT=2.0
RCLONE_MEMORY_LIMIT=1G
EOF
        
        print_success "Agent配置文件已生成"
    fi
    
    # 加载配置
    source .env.agent
    
    # 构建Agent镜像
    print_info "构建Agent镜像..."
    $DOCKER_COMPOSE -f docker-compose.agent.yml build
    
    # 启动Agent
    print_info "启动Agent服务..."
    $DOCKER_COMPOSE -f docker-compose.agent.yml up -d
    
    print_success "Agent部署成功！"
}

# 显示访问信息
show_access_info() {
    echo ""
    echo -e "${GREEN}━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━${NC}"
    echo -e "${GREEN}✨ 部署成功！${NC}"
    echo ""
    echo "📍 访问地址:"
    echo "  Web UI: http://localhost:${WEB_PORT:-3000}"
    echo "  API: http://localhost:${HUB_PORT:-8080}"
    echo "  Metrics: http://localhost:${METRICS_PORT:-9090}/metrics"
    echo ""
    echo "🔑 默认管理员账号:"
    echo "  用户名: admin"
    echo "  密码: admin"
    echo "  (首次登录后请立即修改密码)"
    echo ""
    echo "📂 数据目录:"
    echo "  ./data/"
    echo ""
    echo "📝 有用的命令:"
    echo "  查看日志: ./deploy.sh logs"
    echo "  查看状态: ./deploy.sh status"
    echo "  停止服务: ./deploy.sh stop"
    echo "  备份数据: ./deploy.sh backup"
    echo -e "${GREEN}━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━${NC}"
}

# 备份数据
backup_data() {
    if [ -d "./data" ]; then
        local backup_file="data-backup-$(date +%Y%m%d-%H%M%S).tar.gz"
        print_info "创建备份: $backup_file"
        tar czf "$backup_file" ./data
        print_success "备份完成: $backup_file ($(du -h $backup_file | cut -f1))"
    else
        print_error "数据目录不存在"
        exit 1
    fi
}

# 恢复数据
restore_data() {
    if [ -z "$1" ]; then
        print_error "请指定备份文件"
        echo "用法: ./deploy.sh restore <backup-file.tar.gz>"
        exit 1
    fi
    
    if [ -f "$1" ]; then
        print_info "恢复备份: $1"
        
        # 备份现有数据
        if [ -d "./data" ]; then
            backup_existing_data
        fi
        
        # 恢复数据
        tar xzf "$1"
        print_success "备份已恢复"
    else
        print_error "备份文件不存在: $1"
        exit 1
    fi
}

# 查看服务状态
show_status() {
    check_requirements
    
    # 选择配置文件
    if [ -f "docker-compose.prod.yml" ]; then
        COMPOSE_FILE="docker-compose.prod.yml"
    else
        COMPOSE_FILE="docker-compose.yml"
    fi
    
    print_info "服务状态:"
    $DOCKER_COMPOSE -f $COMPOSE_FILE ps
    
    echo ""
    print_info "健康检查:"
    
    # 检查Hub健康状态
    if curl -sf http://localhost:${HUB_PORT:-8080}/health &> /dev/null; then
        print_success "Hub API: 健康"
    else
        print_warning "Hub API: 未响应"
    fi
    
    # 检查Web UI
    if curl -sf http://localhost:${WEB_PORT:-3000}/ &> /dev/null; then
        print_success "Web UI: 健康"
    else
        print_warning "Web UI: 未响应"
    fi
    
    # 显示数据目录信息
    if [ -d "./data" ]; then
        echo ""
        print_info "数据目录使用情况:"
        du -sh ./data/* 2>/dev/null || echo "  无数据"
    fi
}

# 查看日志
show_logs() {
    check_requirements
    
    # 选择配置文件
    if [ -f "docker-compose.prod.yml" ]; then
        COMPOSE_FILE="docker-compose.prod.yml"
    else
        COMPOSE_FILE="docker-compose.yml"
    fi
    
    if [ -n "$1" ]; then
        print_info "查看 $1 服务日志..."
        $DOCKER_COMPOSE -f $COMPOSE_FILE logs -f --tail=100 "$1"
    else
        print_info "查看所有服务日志..."
        $DOCKER_COMPOSE -f $COMPOSE_FILE logs -f --tail=100
    fi
}

# 主函数
main() {
    case "$1" in
        hub)
            show_data_info
            deploy_hub "$@"
            ;;
        hub-with-agent)
            show_data_info
            deploy_hub_with_agent "$@"
            ;;
        agent)
            deploy_agent
            ;;
        build)
            check_requirements
            setup_env
            build_images "all"
            ;;
        status)
            show_status
            ;;
        logs)
            show_logs "$2"
            ;;
        stop)
            check_requirements
            if [ -f "docker-compose.prod.yml" ]; then
                COMPOSE_FILE="docker-compose.prod.yml"
            else
                COMPOSE_FILE="docker-compose.yml"
            fi
            print_info "停止所有服务..."
            $DOCKER_COMPOSE -f $COMPOSE_FILE --profile local-agent --profile db-backup down
            print_success "服务已停止"
            ;;
        restart)
            check_requirements
            if [ -f "docker-compose.prod.yml" ]; then
                COMPOSE_FILE="docker-compose.prod.yml"
            else
                COMPOSE_FILE="docker-compose.yml"
            fi
            print_info "重启服务..."
            $DOCKER_COMPOSE -f $COMPOSE_FILE --profile local-agent --profile db-backup restart
            print_success "服务已重启"
            ;;
        clean)
            check_requirements
            cleanup_data_interactive
            ;;
        backup)
            backup_data
            ;;
        restore)
            restore_data "$2"
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

# 执行主函数
main "$@"