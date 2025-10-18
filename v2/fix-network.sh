#!/bin/bash

# ============================================
# 修复Docker网络冲突
# ============================================

set -e

# 颜色定义
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
BLUE='\033[0;34m'
NC='\033[0m'

echo -e "${BLUE}🔧 修复Docker网络冲突...${NC}"
echo ""

# 检测Docker Compose命令
if docker compose version &> /dev/null; then
    DOCKER_COMPOSE="docker compose"
else
    DOCKER_COMPOSE="docker-compose"
fi

# 1. 停止所有容器
echo -e "${BLUE}1️⃣ 停止所有相关容器...${NC}"
$DOCKER_COMPOSE --profile local-agent --profile db-backup down 2>/dev/null || true

# 2. 列出现有网络
echo ""
echo -e "${BLUE}2️⃣ 检查现有Docker网络...${NC}"
docker network ls | grep -E "backend|v2" || echo "没有发现冲突网络"

# 3. 删除冲突的网络
echo ""
echo -e "${BLUE}3️⃣ 清理冲突网络...${NC}"

# 删除v2_backend网络
if docker network ls | grep -q "v2_backend"; then
    echo -e "${YELLOW}删除 v2_backend 网络...${NC}"
    docker network rm v2_backend 2>/dev/null || true
    echo -e "${GREEN}✅ v2_backend 网络已删除${NC}"
fi

# 删除其他可能冲突的backend网络
for network in $(docker network ls --format "{{.Name}}" | grep backend); do
    echo -e "${YELLOW}发现网络: $network${NC}"
    read -p "是否删除此网络？(y/n) " -n 1 -r
    echo
    if [[ $REPLY =~ ^[Yy]$ ]]; then
        docker network rm "$network" 2>/dev/null || echo -e "${RED}无法删除 $network (可能正在使用)${NC}"
    fi
done

# 4. 创建新网络（使用特定子网避免冲突）
echo ""
echo -e "${BLUE}4️⃣ 创建新的Docker网络...${NC}"

# 检查是否需要创建网络
if ! docker network ls | grep -q "v2_backend"; then
    # 使用特定的子网范围避免冲突
    docker network create \
        --driver bridge \
        --subnet=172.30.0.0/16 \
        --gateway=172.30.0.1 \
        v2_backend
    echo -e "${GREEN}✅ 创建了新的 v2_backend 网络 (172.30.0.0/16)${NC}"
else
    echo -e "${GREEN}✅ v2_backend 网络已存在${NC}"
fi

# 5. 显示网络信息
echo ""
echo -e "${BLUE}5️⃣ 当前网络配置：${NC}"
docker network inspect v2_backend --format='Subnet: {{range .IPAM.Config}}{{.Subnet}}{{end}}'

echo ""
echo -e "${GREEN}✨ 网络问题已修复！${NC}"
echo ""
echo "现在可以重新运行部署命令："
echo "  ./deploy.sh hub-with-agent"