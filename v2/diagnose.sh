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

# 7. 测试健康检查端点
print_info "测试健康检查端点..."

# 使用固定IP进行内部测试
print_info "内部网络测试（固定IP）:"
echo -n "  PostgreSQL (172.30.0.10): "
docker exec v2-postgres-1 pg_isready &> /dev/null && print_success "正常" || print_error "失败"

echo -n "  Redis (172.30.0.11): "
docker exec v2-redis-1 redis-cli ping &> /dev/null && print_success "正常" || print_error "失败"

echo -n "  Hub API (172.30.0.20:8080): "
curl -sf http://172.30.0.20:8080/health &> /dev/null && print_success "正常" || print_error "失败"

echo -n "  Web UI (172.30.0.30:80): "
curl -sf http://172.30.0.30:80/health &> /dev/null && print_success "正常" || print_error "失败"

# 测试外部访问
WEB_PORT="${WEB_PORT:-3000}"
print_info "外部访问测试:"
echo -n "  Web UI (localhost:${WEB_PORT}): "
if curl -sf http://localhost:${WEB_PORT}/health &> /dev/null; then
    print_success "正常"
else
    print_error "失败"
fi

echo -n "  API via Web UI (localhost:${WEB_PORT}/api): "
if curl -sf http://localhost:${WEB_PORT}/api/health &> /dev/null; then
    HEALTH_RESPONSE=$(curl -s http://localhost:${WEB_PORT}/api/health)
    print_success "正常 - $HEALTH_RESPONSE"
else
    print_error "失败"
fi

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