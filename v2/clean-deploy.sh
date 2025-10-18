#!/bin/bash

# ============================================
# 清理并重新部署脚本
# ============================================

set -e

# 颜色定义
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
BLUE='\033[0;34m'
NC='\033[0m'

echo -e "${BLUE}🧹 清理并重新部署 Rclone Backup Web${NC}"
echo ""

# 检测Docker Compose命令
if docker compose version &> /dev/null; then
    DOCKER_COMPOSE="docker compose"
    echo -e "${GREEN}✅ 使用 Docker Compose V2${NC}"
else
    DOCKER_COMPOSE="docker-compose"
    echo -e "${GREEN}✅ 使用 Docker Compose V1${NC}"
fi

# 1. 停止所有容器
echo ""
echo -e "${BLUE}1️⃣ 停止所有容器...${NC}"
$DOCKER_COMPOSE --profile local-agent --profile db-backup down -v 2>/dev/null || true
echo -e "${GREEN}✅ 容器已停止${NC}"

# 2. 清理Docker网络
echo ""
echo -e "${BLUE}2️⃣ 清理Docker网络...${NC}"

# 获取项目名称（用于识别网络）
PROJECT_NAME=$(basename $(pwd))

# 删除可能存在的旧网络
networks_to_remove=(
    "${PROJECT_NAME}_backend"
    "v2_backend"
    "rclone-backup_backend"
)

for network in "${networks_to_remove[@]}"; do
    if docker network ls | grep -q "$network"; then
        echo -e "${YELLOW}删除网络: $network${NC}"
        docker network rm "$network" 2>/dev/null || true
    fi
done

echo -e "${GREEN}✅ 网络已清理${NC}"

# 3. 清理未使用的Docker资源
echo ""
echo -e "${BLUE}3️⃣ 清理未使用的Docker资源...${NC}"
docker system prune -f --volumes 2>/dev/null || true
echo -e "${GREEN}✅ Docker资源已清理${NC}"

# 4. 确保.env文件存在
echo ""
echo -e "${BLUE}4️⃣ 检查配置文件...${NC}"
if [ ! -f .env ]; then
    echo -e "${YELLOW}创建默认配置...${NC}"
    cp .env.example .env 2>/dev/null || cat > .env << 'EOF'
# 数据库配置
DB_NAME=rclone_backup
DB_USER=rclone
DB_PASSWORD=rclone_password_$(openssl rand -hex 8)

# 安全密钥
JWT_SECRET=$(openssl rand -hex 32)
ENCRYPTION_KEY=$(openssl rand -hex 16)

# 端口配置
HUB_PORT=8080
WEB_PORT=3000
METRICS_PORT=9090

# 其他配置
VERSION=latest
GIN_MODE=release
LOG_LEVEL=info
EOF
    echo -e "${GREEN}✅ 配置文件已创建${NC}"
else
    echo -e "${GREEN}✅ 配置文件已存在${NC}"
fi

# 5. 重新构建镜像
echo ""
echo -e "${BLUE}5️⃣ 构建Docker镜像...${NC}"
$DOCKER_COMPOSE build --no-cache hub-api web-ui
echo -e "${GREEN}✅ 镜像构建完成${NC}"

# 6. 启动服务
echo ""
echo -e "${BLUE}6️⃣ 启动服务...${NC}"
$DOCKER_COMPOSE up -d postgres redis hub-api web-ui

# 7. 等待服务就绪
echo ""
echo -e "${BLUE}7️⃣ 等待服务启动...${NC}"
sleep 5

# 检查服务状态
echo ""
echo -e "${BLUE}📊 服务状态：${NC}"
$DOCKER_COMPOSE ps

# 8. 健康检查
echo ""
echo -e "${BLUE}🏥 健康检查：${NC}"

# 检查Hub API
if curl -sf http://localhost:8080/health &> /dev/null; then
    echo -e "${GREEN}✅ Hub API: 健康${NC}"
else
    echo -e "${RED}❌ Hub API: 未响应${NC}"
    echo ""
    echo -e "${YELLOW}查看日志：${NC}"
    $DOCKER_COMPOSE logs hub-api | tail -20
fi

# 检查Web UI
if curl -sf http://localhost:3000 &> /dev/null; then
    echo -e "${GREEN}✅ Web UI: 健康${NC}"
else
    echo -e "${RED}❌ Web UI: 未响应${NC}"
    echo ""
    echo -e "${YELLOW}查看日志：${NC}"
    $DOCKER_COMPOSE logs web-ui | tail -20
fi

echo ""
echo -e "${GREEN}━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━${NC}"
echo -e "${GREEN}✨ 部署完成！${NC}"
echo ""
echo "访问地址："
echo "  Web UI: http://localhost:3000"
echo "  API: http://localhost:8080"
echo "  Metrics: http://localhost:9090/metrics"
echo ""
echo "默认管理员账号："
echo "  用户名: admin"
echo "  密码: admin"
echo -e "${GREEN}━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━${NC}"