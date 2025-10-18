#!/bin/bash

# ============================================
# 安全部署脚本 - 处理网络冲突
# ============================================

set -e

# 颜色定义
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
BLUE='\033[0;34m'
CYAN='\033[0;36m'
NC='\033[0m'

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

# 检测Docker Compose命令
detect_docker_compose() {
    if docker compose version &> /dev/null; then
        DOCKER_COMPOSE="docker compose"
        print_info "使用 Docker Compose V2"
    elif docker-compose --version &> /dev/null; then
        DOCKER_COMPOSE="docker-compose"
        print_info "使用 Docker Compose V1"
    else
        print_error "未找到 Docker Compose"
        exit 1
    fi
}

# 主函数
main() {
    echo -e "${CYAN}━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━${NC}"
    echo -e "${CYAN}   Rclone Backup Web - 安全部署${NC}"
    echo -e "${CYAN}━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━${NC}"
    echo ""
    
    # 1. 检测Docker Compose
    print_info "检查Docker环境..."
    detect_docker_compose
    print_success "Docker环境就绪"
    echo ""
    
    # 2. 验证配置文件
    print_info "验证docker-compose.yml..."
    if $DOCKER_COMPOSE config > /dev/null 2>&1; then
        print_success "配置文件格式正确"
    else
        print_error "配置文件有错误："
        $DOCKER_COMPOSE config
        exit 1
    fi
    echo ""
    
    # 3. 处理网络冲突
    print_info "处理Docker网络..."
    
    # 停止现有容器
    $DOCKER_COMPOSE down 2>/dev/null || true
    
    # 删除可能冲突的网络
    NETWORK_NAME="v2_backend"
    if docker network ls | grep -q "$NETWORK_NAME"; then
        print_warning "删除现有网络 $NETWORK_NAME..."
        docker network rm "$NETWORK_NAME" 2>/dev/null || true
    fi
    
    print_success "网络准备就绪"
    echo ""
    
    # 4. 检查.env文件
    print_info "检查环境配置..."
    if [ ! -f .env ]; then
        print_warning ".env 文件不存在，创建默认配置..."
        cat > .env << 'EOF'
# 数据库配置
DB_NAME=rclone_backup
DB_USER=rclone
DB_PASSWORD=changeme123

# 安全密钥
JWT_SECRET=your-jwt-secret-key-here
ENCRYPTION_KEY=your-encryption-key

# 端口配置
HUB_PORT=8080
WEB_PORT=3000
METRICS_PORT=9090

# 其他配置
VERSION=latest
GIN_MODE=release
LOG_LEVEL=info
SESSION_TIMEOUT=24h
API_KEY_EXPIRY=365d
ENABLE_METRICS=true
EOF
        print_success "已创建默认 .env 文件"
        print_warning "请编辑 .env 文件设置安全密钥"
    else
        print_success "使用现有 .env 文件"
    fi
    echo ""
    
    # 5. 构建镜像
    print_info "构建Docker镜像..."
    print_info "这可能需要几分钟，请耐心等待..."
    
    if $DOCKER_COMPOSE build hub-api web-ui; then
        print_success "镜像构建成功"
    else
        print_error "镜像构建失败"
        exit 1
    fi
    echo ""
    
    # 6. 启动服务
    print_info "启动服务..."
    
    # 先启动数据库
    print_info "启动数据库..."
    $DOCKER_COMPOSE up -d postgres redis
    sleep 5
    
    # 启动Hub和Web UI
    print_info "启动Hub API和Web UI..."
    $DOCKER_COMPOSE up -d hub-api web-ui
    
    print_success "所有服务已启动"
    echo ""
    
    # 7. 等待服务就绪
    print_info "等待服务就绪..."
    local max_attempts=30
    local attempt=0
    
    while [ $attempt -lt $max_attempts ]; do
        if curl -sf http://localhost:8080/health > /dev/null 2>&1; then
            print_success "Hub API已就绪"
            break
        fi
        echo -n "."
        sleep 2
        ((attempt++))
    done
    
    if [ $attempt -eq $max_attempts ]; then
        print_error "Hub API启动超时"
        print_info "查看日志："
        $DOCKER_COMPOSE logs hub-api | tail -20
        exit 1
    fi
    echo ""
    
    # 8. 显示服务状态
    print_info "服务状态："
    echo ""
    $DOCKER_COMPOSE ps
    echo ""
    
    # 9. 完成
    echo -e "${GREEN}━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━${NC}"
    echo -e "${GREEN}✨ 部署成功！${NC}"
    echo ""
    echo "访问地址："
    echo "  Web UI: http://localhost:3000"
    echo "  API: http://localhost:8080"
    echo "  Metrics: http://localhost:9090/metrics"
    echo ""
    echo "默认管理员账号："
    echo "  用户名: admin"
    echo "  密码: admin"
    echo ""
    echo "查看日志："
    echo "  $DOCKER_COMPOSE logs -f"
    echo -e "${GREEN}━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━${NC}"
}

# 运行主函数
main "$@"