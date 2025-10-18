#!/bin/bash

# ============================================
# Rclone Backup Web V2.0 - 快速部署脚本
# ============================================

set -e

# 颜色定义
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
BLUE='\033[0;34m'
CYAN='\033[0;36m'
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
    ./deploy.sh [选项]

选项:
    hub              部署Hub（不含Agent）
    hub-with-agent   部署Hub（含本地Agent）
    agent            部署独立Agent
    build            构建所有镜像
    status           查看服务状态
    logs             查看服务日志
    clean            清理所有数据和镜像
    help             显示此帮助信息

示例:
    ./deploy.sh hub              # 部署Hub服务
    ./deploy.sh hub-with-agent   # 部署Hub和本地Agent
    ./deploy.sh agent            # 在远程服务器部署Agent

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

# 交互式生成配置
generate_env_interactive() {
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
VERSION=1.0.0

# Hub配置
SESSION_TIMEOUT=24h
API_KEY_EXPIRY=365d
ENABLE_METRICS=true
ENABLE_PROFILING=false

# 前端配置
API_URL=http://localhost:$HUB_PORT
SSE_URL=http://localhost:$HUB_PORT

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

# 生成或更新配置
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
                generate_env_interactive
                ;;
            2)
                print_info "使用默认配置..."
                cp .env.example .env
                
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

# 构建镜像
build_images() {
    print_info "开始构建Docker镜像..."
    
    # 构建Hub镜像
    print_info "构建Hub API镜像..."
    $DOCKER_COMPOSE build hub-api
    
    print_info "构建Web UI镜像..."
    $DOCKER_COMPOSE build web-ui
    
    # 构建Agent镜像
    print_info "构建Agent镜像..."
    $DOCKER_COMPOSE --profile local-agent build local-agent
    
    print_success "所有镜像构建完成"
    
    # 显示构建的镜像
    echo ""
    print_info "构建的镜像列表:"
    docker images | grep -E "rclone-backup|REPOSITORY" | head -5
}

# 部署Hub
deploy_hub() {
    print_info "开始部署Hub服务..."
    
    check_requirements
    setup_env
    
    # 构建镜像
    print_info "构建Hub镜像..."
    $DOCKER_COMPOSE build hub-api web-ui
    
    # 启动服务
    print_info "启动Hub服务..."
    $DOCKER_COMPOSE up -d postgres redis hub-api web-ui
    
    # 等待服务启动
    print_info "等待服务启动..."
    local max_attempts=30
    local attempt=0
    
    while [ $attempt -lt $max_attempts ]; do
        if curl -f http://localhost:${HUB_PORT:-8080}/health &> /dev/null; then
            print_success "Hub服务部署成功！"
            echo ""
            echo "━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━"
            echo "访问地址:"
            echo "  Web UI: http://localhost:${WEB_PORT:-3000}"
            echo "  API: http://localhost:${HUB_PORT:-8080}"
            echo "  Metrics: http://localhost:${METRICS_PORT:-9090}/metrics"
            echo ""
            echo "默认管理员账号:"
            echo "  用户名: admin"
            echo "  密码: admin"
            echo "  (首次登录后请立即修改密码)"
            echo "━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━"
            return 0
        fi
        
        echo -n "."
        sleep 2
        ((attempt++))
    done
    
    print_error "Hub服务启动超时，请检查日志"
    $DOCKER_COMPOSE logs hub-api
    exit 1
}

# 部署Hub和本地Agent
deploy_hub_with_agent() {
    print_info "开始部署Hub服务（含本地Agent）..."
    
    check_requirements
    setup_env
    
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
    print_info "构建Agent镜像..."
    $DOCKER_COMPOSE --profile local-agent build local-agent
    
    # 启动本地Agent
    print_info "启动本地Agent..."
    $DOCKER_COMPOSE --profile local-agent up -d local-agent local-rclone-sidecar
    
    # 检查Agent状态
    sleep 5
    if $DOCKER_COMPOSE ps | grep -q "local-agent.*Up\|local-agent.*running"; then
        print_success "Hub和本地Agent部署成功！"
        echo ""
        echo "本地Agent已启动，请在Web UI的Agents页面查看"
    else
        print_error "本地Agent启动失败，请检查日志"
        $DOCKER_COMPOSE logs local-agent
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

# 查看服务状态
show_status() {
    check_requirements
    
    print_info "服务状态:"
    $DOCKER_COMPOSE ps
    
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
}

# 查看日志
show_logs() {
    check_requirements
    
    if [ -n "$1" ]; then
        print_info "查看 $1 服务日志..."
        $DOCKER_COMPOSE logs -f --tail=100 "$1"
    else
        print_info "查看所有服务日志..."
        $DOCKER_COMPOSE logs -f --tail=100
    fi
}

# 清理所有数据
clean_all() {
    check_requirements
    
    print_warning "⚠️  此操作将删除所有数据和镜像！"
    print_prompt "请输入 'yes' 确认删除: "
    read -r confirm
    
    if [ "$confirm" != "yes" ]; then
        print_info "操作已取消"
        exit 0
    fi
    
    print_info "停止所有容器..."
    $DOCKER_COMPOSE --profile local-agent --profile db-backup down -v
    
    print_info "删除镜像..."
    docker images | grep rclone-backup | awk '{print $3}' | xargs -r docker rmi -f 2>/dev/null || true
    
    print_info "删除配置文件..."
    rm -f .env .env.agent
    
    print_info "删除备份文件..."
    rm -rf backups/* 2>/dev/null || true
    
    print_success "清理完成"
}

# 主函数
main() {
    case "$1" in
        hub)
            deploy_hub
            ;;
        hub-with-agent)
            deploy_hub_with_agent
            ;;
        agent)
            deploy_agent
            ;;
        build)
            check_requirements
            setup_env
            build_images
            ;;
        status)
            show_status
            ;;
        logs)
            show_logs "$2"
            ;;
        clean)
            clean_all
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