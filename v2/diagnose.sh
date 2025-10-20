#!/bin/bash

# ============================================
# Rclone Backup Web V2.0 - 诊断脚本
# 用于诊断部署问题
# ============================================

set -e

# 颜色定义
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
BLUE='\033[0;34m'
NC='\033[0m' # No Color

print_info() {
    echo -e "${BLUE}[INFO]${NC} $1"
}

print_success() {
    echo -e "${GREEN}[✓]${NC} $1"
}

print_warning() {
    echo -e "${YELLOW}[⚠]${NC} $1"
}

print_error() {
    echo -e "${RED}[✗]${NC} $1"
}

# 检测Docker Compose命令
detect_docker_compose() {
    if command -v "docker" &> /dev/null && docker compose version &> /dev/null; then
        echo "docker compose"
    elif command -v "docker-compose" &> /dev/null; then
        echo "docker-compose"
    else
        echo ""
    fi
}

echo "========================================="
echo "Rclone Backup Web V2.0 - 系统诊断"
echo "========================================="
echo ""

# 1. 检查Docker
print_info "检查Docker环境..."
if command -v docker &> /dev/null; then
    DOCKER_VERSION=$(docker --version)
    print_success "Docker已安装: $DOCKER_VERSION"
else
    print_error "Docker未安装"
    exit 1
fi

# 2. 检查Docker Compose
DOCKER_COMPOSE=$(detect_docker_compose)
if [ -n "$DOCKER_COMPOSE" ]; then
    COMPOSE_VERSION=$($DOCKER_COMPOSE version)
    print_success "Docker Compose已安装: $COMPOSE_VERSION"
else
    print_error "Docker Compose未安装"
    exit 1
fi

# 3. 检查配置文件
print_info "检查配置文件..."
if [ -f ".env" ]; then
    print_success "找到.env配置文件"
    # 检查关键配置
    source .env
    if [ -z "$JWT_SECRET" ] || [ -z "$ENCRYPTION_KEY" ]; then
        print_warning "缺少关键配置（JWT_SECRET或ENCRYPTION_KEY）"
    fi
else
    print_error "未找到.env配置文件"
    print_info "请运行: cp .env.example .env 并编辑配置"
fi

# 4. 检查容器状态
print_info "检查容器状态..."
CONTAINERS=$($DOCKER_COMPOSE ps --format json 2>/dev/null || echo "[]")

if [ "$CONTAINERS" = "[]" ]; then
    print_warning "没有运行中的容器"
else
    echo "$CONTAINERS" | jq -r '.[] | "\(.Service): \(.State)"' 2>/dev/null || \
    $DOCKER_COMPOSE ps
fi

# 5. 检查网络
print_info "检查Docker网络..."
if docker network ls | grep -q "v2_backend"; then
    print_success "找到backend网络"
else
    print_warning "未找到backend网络"
fi

# 6. 检查端口占用
print_info "检查端口占用..."
check_port() {
    local port=$1
    local service=$2
    if nc -z localhost $port 2>/dev/null; then
        print_success "端口 $port ($service) 已监听"
    else
        print_warning "端口 $port ($service) 未监听"
    fi
}

check_port 5432 "PostgreSQL"
check_port 8080 "Hub API"
check_port 3000 "Web UI"

# 7. Docker 健康检查状态
print_info "Docker 健康检查状态..."

# 获取所有容器的健康状态
get_health_status() {
    local container=$1
    local health=$(docker inspect $container --format='{{.State.Health.Status}}' 2>/dev/null || echo "not_found")
    local state=$(docker inspect $container --format='{{.State.Status}}' 2>/dev/null || echo "not_found")
    
    if [ "$health" = "not_found" ]; then
        echo "容器不存在"
    elif [ "$health" = "healthy" ]; then
        print_success "$health"
    elif [ "$health" = "unhealthy" ]; then
        print_error "$health"
        # 显示最近的健康检查日志
        local log=$(docker inspect $container --format='{{range .State.Health.Log}}{{.Output}}{{end}}' 2>/dev/null | tail -1)
        if [ -n "$log" ]; then
            echo "    最近日志: $log"
        fi
    elif [ "$health" = "starting" ]; then
        print_warning "$health"
    else
        echo "$state (无健康检查)"
    fi
}

echo -n "  PostgreSQL: "
get_health_status v2-postgres-1

echo -n "  Redis: "
get_health_status v2-redis-1

echo -n "  Hub API: "
get_health_status v2-hub-api-1

echo -n "  Web UI: "
get_health_status v2-web-ui-1

# 显示容器网络信息（动态获取）
print_info "容器网络配置:"
for container in v2-postgres-1 v2-redis-1 v2-hub-api-1 v2-web-ui-1; do
    if docker ps -q -f name=$container &> /dev/null; then
        # 获取容器在backend网络中的IP
        IP=$(docker inspect $container --format='{{range .NetworkSettings.Networks}}{{if eq .NetworkID ""}}{{else}}{{.IPAddress}}{{end}}{{end}}' 2>/dev/null | head -1)
        if [ -n "$IP" ]; then
            case $container in
                v2-postgres-1) echo "  PostgreSQL: $IP:5432" ;;
                v2-redis-1) echo "  Redis: $IP:6379" ;;
                v2-hub-api-1) echo "  Hub API: $IP:8080" ;;
                v2-web-ui-1) echo "  Web UI: $IP:80" ;;
            esac
        fi
    fi
done

# 测试外部访问
WEB_PORT="${WEB_PORT:-3000}"
print_info "外部访问端口:"
echo "  Web应用: http://localhost:${WEB_PORT}"

# 8. 检查数据库连接
print_info "检查数据库连接..."
if docker exec $(docker ps -q -f name=rclone-backup-postgres) pg_isready -U ${DB_USER:-rclone} &> /dev/null; then
    print_success "数据库连接正常"
else
    print_warning "数据库连接失败"
fi

# 9. 查看最近的日志
print_info "查看最近的错误日志..."
echo ""
echo "=== PostgreSQL日志 ==="
$DOCKER_COMPOSE logs --tail=5 postgres 2>&1 | grep -E "ERROR|FATAL|WARNING" || echo "无错误"

echo ""
echo "=== Hub API日志 ==="
$DOCKER_COMPOSE logs --tail=10 hub-api 2>&1 | grep -E "ERROR|FATAL|Failed|404|500" || echo "无错误"

# 10. 建议
echo ""
echo "========================================="
echo "诊断建议"
echo "========================================="

if ! curl -sf http://localhost:8080/health &> /dev/null; then
    echo "1. Hub API服务可能未正确启动，建议："
    echo "   - 检查完整日志: $DOCKER_COMPOSE logs hub-api"
    echo "   - 重新构建镜像: $DOCKER_COMPOSE build hub-api"
    echo "   - 重启服务: $DOCKER_COMPOSE restart hub-api"
fi

if ! docker exec $(docker ps -q -f name=rclone-backup-postgres) pg_isready -U ${DB_USER:-rclone} &> /dev/null 2>&1; then
    echo "2. 数据库连接问题，建议："
    echo "   - 检查数据库日志: $DOCKER_COMPOSE logs postgres"
    echo "   - 确认数据库配置正确"
    echo "   - 重启数据库: $DOCKER_COMPOSE restart postgres"
fi

echo ""
echo "完整的部署命令："
echo "  ./deploy.sh hub          # 仅部署Hub"
echo "  ./deploy.sh hub-with-agent  # 部署Hub和本地Agent"
echo ""
echo "如果问题持续，请运行："
echo "  $DOCKER_COMPOSE down -v  # 清理所有容器和卷"
echo "  ./deploy.sh hub          # 重新部署"