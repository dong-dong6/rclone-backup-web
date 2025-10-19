#!/bin/bash

# 修复版部署脚本 - 更稳定的健康检查

set +e  # 不要在错误时立即退出

# 颜色定义
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
BLUE='\033[0;34m'
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
if command -v "docker" &> /dev/null && docker compose version &> /dev/null; then
    DOCKER_COMPOSE="docker compose"
else
    DOCKER_COMPOSE="docker-compose"
fi

print_info "等待Hub API服务启动..."

max_attempts=30
attempt=0
success=false

while [ $attempt -lt $max_attempts ] && [ "$success" = "false" ]; do
    # 显示进度
    echo -n "."
    
    # 方法1: 测试localhost
    if curl -sf -m 2 http://localhost:8080/health > /dev/null 2>&1; then
        echo ""
        print_success "Hub服务可通过localhost:8080访问"
        success=true
        break
    fi
    
    # 方法2: 测试容器内部
    if $DOCKER_COMPOSE exec -T hub-api sh -c 'curl -sf http://localhost:8080/health' > /dev/null 2>&1; then
        echo ""
        print_success "Hub服务运行正常（容器内部验证）"
        success=true
        break
    fi
    
    # 方法3: 获取容器IP并测试
    # 使用更简单的方法获取IP
    CONTAINER_ID=$($DOCKER_COMPOSE ps -q hub-api 2>/dev/null)
    if [ -n "$CONTAINER_ID" ]; then
        # 获取容器在backend网络中的IP
        CONTAINER_IP=$(docker inspect $CONTAINER_ID --format='{{range .NetworkSettings.Networks}}{{if eq .NetworkID ""}}{{else}}{{.IPAddress}}{{end}}{{end}}' 2>/dev/null | head -c 15)
        
        if [ -n "$CONTAINER_IP" ] && [[ "$CONTAINER_IP" =~ ^[0-9]+\.[0-9]+\.[0-9]+\.[0-9]+$ ]]; then
            if curl -sf -m 2 http://${CONTAINER_IP}:8080/health > /dev/null 2>&1; then
                echo ""
                print_success "Hub服务可通过容器IP访问: ${CONTAINER_IP}:8080"
                print_warning "注意: localhost可能无法访问，请使用容器IP或配置端口转发"
                success=true
                break
            fi
        fi
    fi
    
    sleep 2
    ((attempt++))
done

echo ""

if [ "$success" = "true" ]; then
    print_success "部署成功！"
    echo ""
    echo "━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━"
    echo "访问信息"
    echo "━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━"
    
    # 尝试获取实际可访问的地址
    if curl -sf -m 2 http://localhost:8080/health > /dev/null 2>&1; then
        echo "Hub API: http://localhost:8080"
        echo "Web UI:  http://localhost:3000"
    else
        # 获取容器IP作为备选
        CONTAINER_ID=$($DOCKER_COMPOSE ps -q hub-api 2>/dev/null)
        if [ -n "$CONTAINER_ID" ]; then
            CONTAINER_IP=$(docker inspect $CONTAINER_ID --format='{{range .NetworkSettings.Networks}}{{.IPAddress}}{{end}}' 2>/dev/null | head -1)
            if [ -n "$CONTAINER_IP" ]; then
                echo "Hub API: http://${CONTAINER_IP}:8080"
                echo "Web UI:  http://localhost:3000"
                echo ""
                print_warning "localhost:8080不可访问，请使用容器IP或检查防火墙设置"
            fi
        fi
    fi
    
    echo ""
    echo "默认管理员账号:"
    echo "  用户名: admin"
    echo "  密码: admin123"
    echo ""
    echo "查看日志:"
    echo "  $DOCKER_COMPOSE logs -f hub-api"
    echo "━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━"
else
    print_error "服务启动超时"
    echo ""
    print_info "诊断信息:"
    echo "1. 容器状态:"
    $DOCKER_COMPOSE ps
    
    echo ""
    echo "2. Hub API日志（最后20行）:"
    $DOCKER_COMPOSE logs --tail=20 hub-api
    
    echo ""
    echo "3. 可能的问题:"
    echo "   - 数据库连接失败"
    echo "   - 端口被占用"
    echo "   - 配置文件错误"
    echo ""
    echo "建议操作:"
    echo "   1. 检查完整日志: $DOCKER_COMPOSE logs hub-api"
    echo "   2. 重启服务: $DOCKER_COMPOSE restart hub-api"
    echo "   3. 检查.env配置文件"
    
    exit 1
fi