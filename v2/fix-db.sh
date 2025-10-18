#!/bin/bash

# 数据库连接修复脚本

set -e

# 颜色定义
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
BLUE='\033[0;34m'
NC='\033[0m'

echo -e "${BLUE}━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━${NC}"
echo -e "${BLUE}   数据库连接问题修复脚本${NC}"
echo -e "${BLUE}━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━${NC}"
echo ""

# 检测Docker Compose
if docker compose version &> /dev/null; then
    DC="docker compose"
else
    DC="docker-compose"
fi

# 选择配置文件
if [ -f "docker-compose.prod.yml" ]; then
    CF="docker-compose.prod.yml"
else
    CF="docker-compose.yml"
fi

echo -e "${YELLOW}⚠️  警告：此脚本将清理所有容器和数据卷！${NC}"
echo ""
read -p "是否继续？(y/n) " -n 1 -r
echo
if [[ ! $REPLY =~ ^[Yy]$ ]]; then
    echo "已取消"
    exit 0
fi

# 1. 停止并清理所有容器和卷
echo ""
echo -e "${BLUE}1️⃣ 停止并清理容器...${NC}"
$DC -f $CF down -v
echo -e "${GREEN}✅ 清理完成${NC}"

# 2. 清理旧的数据目录
echo ""
echo -e "${BLUE}2️⃣ 清理数据目录...${NC}"
if [ -d "./data/postgres" ]; then
    echo "删除旧的PostgreSQL数据..."
    sudo rm -rf ./data/postgres
fi
mkdir -p ./data/postgres
chmod 700 ./data/postgres
echo -e "${GREEN}✅ 数据目录准备就绪${NC}"

# 3. 检查.env文件
echo ""
echo -e "${BLUE}3️⃣ 检查环境配置...${NC}"
if [ ! -f .env ]; then
    echo -e "${RED}❌ .env文件不存在${NC}"
    echo "请先运行: ./deploy.sh hub"
    exit 1
fi

# 加载环境变量
source .env

# 4. 验证数据库配置
echo ""
echo -e "${BLUE}4️⃣ 数据库配置：${NC}"
echo "  DB_NAME: ${DB_NAME:-rclone_backup}"
echo "  DB_USER: ${DB_USER:-rclone}"
echo "  DB_HOST: postgres (Docker内部)"

# 5. 启动数据库
echo ""
echo -e "${BLUE}5️⃣ 启动数据库服务...${NC}"
$DC -f $CF up -d postgres

# 6. 等待数据库就绪
echo ""
echo -e "${BLUE}6️⃣ 等待数据库初始化...${NC}"
max_attempts=30
attempt=0

while [ $attempt -lt $max_attempts ]; do
    if $DC -f $CF exec -T postgres pg_isready -U ${DB_USER:-rclone} &> /dev/null; then
        echo ""
        echo -e "${GREEN}✅ 数据库已就绪${NC}"
        break
    fi
    echo -n "."
    sleep 2
    ((attempt++))
done

if [ $attempt -eq $max_attempts ]; then
    echo ""
    echo -e "${RED}❌ 数据库启动超时${NC}"
    echo "查看日志："
    $DC -f $CF logs postgres | tail -20
    exit 1
fi

# 7. 验证数据库表
echo ""
echo -e "${BLUE}7️⃣ 验证数据库表...${NC}"

# 等待一下让初始化脚本执行
sleep 3

# 检查核心表是否存在
tables=$($DC -f $CF exec -T postgres psql -U ${DB_USER:-rclone} -d ${DB_NAME:-rclone_backup} -c "\dt" 2>/dev/null | grep -E "agents|backup_tasks|rclone_remotes|task_executions|users" | wc -l)

if [ "$tables" -gt 0 ]; then
    echo -e "${GREEN}✅ 数据库表创建成功${NC}"
    echo ""
    echo "已创建的表："
    $DC -f $CF exec -T postgres psql -U ${DB_USER:-rclone} -d ${DB_NAME:-rclone_backup} -c "\dt" | grep -E "public\."
else
    echo -e "${RED}❌ 数据库表未创建${NC}"
    echo "检查初始化日志："
    $DC -f $CF logs postgres | grep -E "ERROR|FATAL"
    exit 1
fi

# 8. 启动Hub API
echo ""
echo -e "${BLUE}8️⃣ 启动Hub API...${NC}"
$DC -f $CF up -d hub-api

# 9. 等待Hub API就绪
echo ""
echo -e "${BLUE}9️⃣ 等待Hub API启动...${NC}"
sleep 5

if curl -sf http://localhost:${HUB_PORT:-8080}/health &> /dev/null; then
    echo -e "${GREEN}✅ Hub API已启动${NC}"
else
    echo -e "${RED}❌ Hub API启动失败${NC}"
    echo "查看日志："
    $DC -f $CF logs hub-api | tail -20
    exit 1
fi

# 10. 启动其他服务
echo ""
echo -e "${BLUE}🔟 启动其他服务...${NC}"
$DC -f $CF up -d

echo ""
echo -e "${GREEN}━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━${NC}"
echo -e "${GREEN}✨ 数据库问题已修复！${NC}"
echo ""
echo "访问地址："
echo "  Web UI: http://localhost:${WEB_PORT:-3000}"
echo "  API: http://localhost:${HUB_PORT:-8080}"
echo ""
echo "查看服务状态："
echo "  ./check.sh"
echo -e "${GREEN}━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━${NC}"