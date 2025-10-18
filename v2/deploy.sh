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
        exit 1
    fi
    
    # 检查Docker Compose
    if ! command -v docker-compose &> /dev/null; then
        print_error "Docker Compose未安装，请先安装Docker Compose"
        exit 1
    fi
    
    print_success "依赖检查通过"
}

# 生成安全密钥
generate_keys() {
    print_info "生成安全密钥..."
    
    if [ ! -f .env ]; then
        cp .env.example .env
        
        # 生成随机密钥
        JWT_SECRET=$(openssl rand -hex 32)
        ENCRYPTION_KEY=$(openssl rand -hex 16)
        DB_PASSWORD=$(openssl rand -base64 20 | tr -d "=+/" | cut -c1-20)
        
        # 替换默认值
        sed -i.bak "s/JWT_SECRET=.*/JWT_SECRET=$JWT_SECRET/" .env
        sed -i.bak "s/ENCRYPTION_KEY=.*/ENCRYPTION_KEY=$ENCRYPTION_KEY/" .env
        sed -i.bak "s/DB_PASSWORD=.*/DB_PASSWORD=$DB_PASSWORD/" .env
        
        rm -f .env.bak
        print_success "安全密钥已生成并保存到 .env"
    else
        print_warning ".env 文件已存在，跳过密钥生成"
    fi
}

# 构建镜像
build_images() {
    print_info "开始构建Docker镜像..."
    
    # 构建Hub镜像
    print_info "构建Hub API镜像..."
    docker-compose build hub-api
    
    print_info "构建Web UI镜像..."
    docker-compose build web-ui
    
    # 构建Agent镜像
    print_info "构建Agent镜像..."
    docker-compose --profile local-agent build local-agent
    
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
    generate_keys
    
    # 构建镜像
    print_info "构建Hub镜像..."
    docker-compose build hub-api web-ui
    
    # 启动服务
    print_info "启动Hub服务..."
    docker-compose up -d postgres redis hub-api web-ui
    
    # 等待服务启动
    print_info "等待服务启动..."
    sleep 10
    
    # 检查服务状态
    if curl -f http://localhost:8080/health &> /dev/null; then
        print_success "Hub服务部署成功！"
        echo ""
        echo "访问地址:"
        echo "  Web UI: http://localhost:3000"
        echo "  API: http://localhost:8080"
        echo "  Metrics: http://localhost:9090/metrics"
    else
        print_error "Hub服务启动失败，请检查日志"
        docker-compose logs hub-api
        exit 1
    fi
}

# 部署Hub和本地Agent
deploy_hub_with_agent() {
    print_info "开始部署Hub服务（含本地Agent）..."
    
    check_requirements
    generate_keys
    
    # 先部署Hub
    deploy_hub
    
    # 提示获取令牌
    echo ""
    print_warning "请完成以下步骤："
    echo "1. 访问 http://localhost:3000"
    echo "2. 登录系统"
    echo "3. 进入 Agents 页面"
    echo "4. 点击 '生成注册令牌'"
    echo "5. 复制令牌"
    echo ""
    read -p "请输入注册令牌: " token
    
    if [ -z "$token" ]; then
        print_error "令牌不能为空"
        exit 1
    fi
    
    # 更新.env文件
    echo "LOCAL_AGENT_REGISTRATION_TOKEN=$token" >> .env
    
    # 构建Agent镜像
    print_info "构建Agent镜像..."
    docker-compose --profile local-agent build local-agent
    
    # 重启服务（含Agent）
    print_info "启动本地Agent..."
    docker-compose --profile local-agent up -d local-agent local-rclone-sidecar
    
    # 检查Agent状态
    sleep 5
    if docker-compose ps | grep -q "local-agent.*Up"; then
        print_success "Hub和本地Agent部署成功！"
    else
        print_error "本地Agent启动失败，请检查日志"
        docker-compose logs local-agent
    fi
}

# 部署独立Agent
deploy_agent() {
    print_info "开始部署独立Agent..."
    
    check_requirements
    
    # 创建Agent环境文件
    if [ ! -f .env.agent ]; then
        cat > .env.agent << EOF
# Agent配置
HUB_URL=http://your-hub-server:8080
REGISTRATION_TOKEN=
AGENT_NAME=$(hostname)

# 备份源路径
BACKUP_SOURCE_1=/var/www
BACKUP_SOURCE_2=/home
BACKUP_SOURCE_3=/etc

# Rclone配置
RCLONE_VERSION=latest
RCLONE_CPU_LIMIT=2.0
RCLONE_MEMORY_LIMIT=1G
EOF
        
        print_warning "请编辑 .env.agent 文件，设置HUB_URL和REGISTRATION_TOKEN"
        exit 0
    fi
    
    # 加载配置
    source .env.agent
    
    if [ -z "$HUB_URL" ] || [ -z "$REGISTRATION_TOKEN" ]; then
        print_error "请先配置HUB_URL和REGISTRATION_TOKEN"
        exit 1
    fi
    
    # 构建Agent镜像
    print_info "构建Agent镜像..."
    docker-compose -f docker-compose.agent.yml build
    
    # 启动Agent
    print_info "启动Agent服务..."
    docker-compose -f docker-compose.agent.yml up -d
    
    print_success "Agent部署成功！"
}

# 清理所有数据
clean_all() {
    print_warning "此操作将删除所有数据和镜像，是否继续？(y/N)"
    read -r confirm
    
    if [[ ! "$confirm" =~ ^[Yy]$ ]]; then
        print_info "操作已取消"
        exit 0
    fi
    
    print_info "停止所有容器..."
    docker-compose --profile local-agent --profile db-backup down -v
    
    print_info "删除镜像..."
    docker images | grep rclone-backup | awk '{print $3}' | xargs -r docker rmi -f
    
    print_info "删除备份文件..."
    rm -rf backups/*
    
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
            build_images
            ;;
        clean)
            clean_all
            ;;
        help|--help|-h)
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