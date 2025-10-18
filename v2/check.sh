#!/bin/bash

# 快速检查脚本 - 用于验证部署状态

# 颜色定义
GREEN='\033[0;32m'
RED='\033[0;31m'
YELLOW='\033[1;33m'
BLUE='\033[0;34m'
NC='\033[0m'

echo -e "${BLUE}━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━${NC}"
echo -e "${BLUE}   Rclone Backup Web - 系统检查${NC}"
echo -e "${BLUE}━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━${NC}"
echo ""

# 检测Docker Compose
if docker compose version &> /dev/null 2>&1; then
    DC="docker compose"
elif docker-compose --version &> /dev/null 2>&1; then
    DC="docker-compose"
else
    echo -e "${RED}❌ Docker Compose未安装${NC}"
    exit 1
fi

# 选择配置文件
if [ -f "docker-compose.prod.yml" ]; then
    CF="docker-compose.prod.yml"
else
    CF="docker-compose.yml"
fi

# 检查服务状态
echo -e "${BLUE}📊 服务状态：${NC}"
$DC -f $CF ps --format "table {{.Name}}\t{{.Status}}\t{{.Ports}}"

echo ""
echo -e "${BLUE}🏥 健康检查：${NC}"

# Hub API
if curl -sf http://localhost:8080/health &> /dev/null; then
    echo -e "${GREEN}✅ Hub API${NC}: http://localhost:8080"
else
    echo -e "${RED}❌ Hub API${NC}: 未响应"
fi

# Web UI
if curl -sf http://localhost:3000 &> /dev/null; then
    echo -e "${GREEN}✅ Web UI${NC}: http://localhost:3000"
else
    echo -e "${RED}❌ Web UI${NC}: 未响应"
fi

# 数据目录
if [ -d "./data" ]; then
    echo ""
    echo -e "${BLUE}📂 数据目录：${NC}"
    du -sh ./data/* 2>/dev/null | head -10
else
    echo ""
    echo -e "${YELLOW}⚠️  数据目录不存在${NC}"
fi

echo ""
echo -e "${BLUE}━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━${NC}"