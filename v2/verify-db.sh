#!/bin/bash

# 数据库验证脚本 - 检查初始化是否成功

set -e

# 颜色定义
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
BLUE='\033[0;34m'
NC='\033[0m'

echo -e "${BLUE}━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━${NC}"
echo -e "${BLUE}   数据库初始化验证${NC}"
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

# 加载环境变量
if [ -f .env ]; then
    source .env
fi

# 1. 检查数据库容器状态
echo -e "${BLUE}1️⃣ 检查数据库容器状态...${NC}"
if $DC -f $CF ps | grep -q "postgres.*running\|postgres.*Up"; then
    echo -e "${GREEN}✅ 数据库容器运行中${NC}"
else
    echo -e "${RED}❌ 数据库容器未运行${NC}"
    echo "请先运行: ./deploy.sh hub"
    exit 1
fi

# 2. 检查数据库连接
echo ""
echo -e "${BLUE}2️⃣ 测试数据库连接...${NC}"
if $DC -f $CF exec -T postgres pg_isready -U ${DB_USER:-rclone} &> /dev/null; then
    echo -e "${GREEN}✅ 数据库连接正常${NC}"
else
    echo -e "${RED}❌ 无法连接到数据库${NC}"
    exit 1
fi

# 3. 检查数据库是否存在
echo ""
echo -e "${BLUE}3️⃣ 检查数据库...${NC}"
db_exists=$($DC -f $CF exec -T postgres psql -U ${DB_USER:-rclone} -lqt 2>/dev/null | cut -d \| -f 1 | grep -w ${DB_NAME:-rclone_backup} | wc -l)
if [ "$db_exists" -gt 0 ]; then
    echo -e "${GREEN}✅ 数据库 '${DB_NAME:-rclone_backup}' 存在${NC}"
else
    echo -e "${RED}❌ 数据库 '${DB_NAME:-rclone_backup}' 不存在${NC}"
    exit 1
fi

# 4. 检查必需的表
echo ""
echo -e "${BLUE}4️⃣ 验证数据库表...${NC}"

required_tables=(
    "agents"
    "rclone_remotes"
    "backup_tasks"
    "task_agent_assignments"
    "task_executions"
    "registration_tokens"
    "users"
    "sessions"
)

missing_tables=()
for table in "${required_tables[@]}"; do
    if $DC -f $CF exec -T postgres psql -U ${DB_USER:-rclone} -d ${DB_NAME:-rclone_backup} -c "SELECT 1 FROM $table LIMIT 1" &> /dev/null; then
        echo -e "  ${GREEN}✅ $table${NC}"
    else
        echo -e "  ${RED}❌ $table (缺失)${NC}"
        missing_tables+=("$table")
    fi
done

# 5. 检查初始数据
echo ""
echo -e "${BLUE}5️⃣ 检查初始数据...${NC}"

# 检查admin用户
admin_exists=$($DC -f $CF exec -T postgres psql -U ${DB_USER:-rclone} -d ${DB_NAME:-rclone_backup} -t -c "SELECT COUNT(*) FROM users WHERE username='admin'" 2>/dev/null | xargs)
if [ "$admin_exists" = "1" ]; then
    echo -e "${GREEN}✅ Admin用户存在${NC}"
else
    echo -e "${YELLOW}⚠️  Admin用户不存在${NC}"
fi

# 6. 显示数据库统计
echo ""
echo -e "${BLUE}6️⃣ 数据库统计...${NC}"
echo ""
$DC -f $CF exec -T postgres psql -U ${DB_USER:-rclone} -d ${DB_NAME:-rclone_backup} -c "
SELECT 
    schemaname,
    tablename,
    pg_size_pretty(pg_total_relation_size(schemaname||'.'||tablename)) AS size
FROM pg_tables
WHERE schemaname = 'public'
ORDER BY tablename;
" 2>/dev/null || echo "无法获取统计信息"

# 7. 结果总结
echo ""
echo -e "${BLUE}━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━${NC}"
if [ ${#missing_tables[@]} -eq 0 ]; then
    echo -e "${GREEN}✨ 数据库初始化成功！${NC}"
    echo ""
    echo "所有必需的表都已创建。"
    echo ""
    echo "现在可以：" 
    echo "  1. 访问 Web UI: http://localhost:3000"
    echo "  2. 使用 admin/admin 登录"
else
    echo -e "${RED}❌ 数据库初始化不完整${NC}"
    echo ""
    echo "缺失的表: ${missing_tables[*]}"
    echo ""
    echo "请尝试："
    echo "  1. 运行: ./fix-db.sh"
    echo "  2. 或手动检查日志: $DC -f $CF logs postgres"
fi
echo -e "${BLUE}━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━${NC}"