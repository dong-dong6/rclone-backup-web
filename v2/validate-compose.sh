#!/bin/bash

# ============================================
# 验证Docker Compose配置
# ============================================

set -e

# 颜色定义
GREEN='\033[0;32m'
RED='\033[0;31m'
BLUE='\033[0;34m'
NC='\033[0m'

echo -e "${BLUE}🔍 验证Docker Compose配置...${NC}"
echo ""

# 检测Docker Compose命令
if docker compose version &> /dev/null; then
    DOCKER_COMPOSE="docker compose"
else
    DOCKER_COMPOSE="docker-compose"
fi

# 验证配置文件
if $DOCKER_COMPOSE config > /dev/null 2>&1; then
    echo -e "${GREEN}✅ docker-compose.yml 格式正确${NC}"
    echo ""
    
    # 显示服务列表
    echo -e "${BLUE}📦 定义的服务：${NC}"
    $DOCKER_COMPOSE config --services
    
    echo ""
    echo -e "${BLUE}🌐 定义的网络：${NC}"
    $DOCKER_COMPOSE config | grep -A 5 "^networks:" | head -10
    
    echo ""
    echo -e "${GREEN}✅ 配置验证通过！${NC}"
else
    echo -e "${RED}❌ docker-compose.yml 有语法错误${NC}"
    echo ""
    echo -e "${RED}错误详情：${NC}"
    $DOCKER_COMPOSE config
    exit 1
fi