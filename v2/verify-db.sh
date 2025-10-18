#!/bin/bash

# 验证数据库初始化状态
set -e

# 颜色输出
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
BLUE='\033[0;34m'
NC='\033[0m'

print_info() { echo -e "${BLUE}[INFO]${NC} $1"; }
print_success() { echo -e "${GREEN}[SUCCESS]${NC} $1"; }
print_warning() { echo -e "${YELLOW}[WARNING]${NC} $1"; }
print_error() { echo -e "${RED}[ERROR]${NC} $1"; }

# 检测 Docker Compose 命令
if command -v docker-compose &> /dev/null; then
    DOCKER_COMPOSE="docker-compose"
elif docker compose version &> /dev/null 2>&1; then
    DOCKER_COMPOSE="docker compose"
else
    print_error "未找到 Docker Compose"
    exit 1
fi

# 加载环境变量
if [ -f ".env" ]; then
    export $(grep -v '^#' .env | xargs)
else
    print_error "未找到 .env 文件"
    exit 1
fi

DB_NAME="${DB_NAME:-rclone_backup}"
DB_USER="${DB_USER:-rclone}"

echo "================================================"
echo "   数据库验证工具"
echo "================================================"
echo ""

print_info "数据库名称: $DB_NAME"
print_info "数据库用户: $DB_USER"
echo ""

# 检查容器是否运行
print_info "检查 PostgreSQL 容器状态..."
if $DOCKER_COMPOSE ps | grep -q "postgres.*Up"; then
    print_success "PostgreSQL 容器正在运行"
else
    print_error "PostgreSQL 容器未运行"
    exit 1
fi

# 测试数据库连接
print_info "测试数据库连接..."
if $DOCKER_COMPOSE exec -T postgres pg_isready -U "$DB_USER" -d "$DB_NAME" &> /dev/null; then
    print_success "数据库连接成功"
else
    print_error "无法连接到数据库"
    exit 1
fi

# 检查表是否存在
print_info "检查数据库表..."
TABLES=$($DOCKER_COMPOSE exec -T postgres psql -U "$DB_USER" -d "$DB_NAME" -t -c "SELECT tablename FROM pg_tables WHERE schemaname='public';" 2>/dev/null)

if [ -z "$TABLES" ]; then
    print_error "数据库中没有找到任何表"
    exit 1
fi

echo ""
print_success "找到以下表:"
echo "$TABLES" | sed 's/^/  - /'

# 检查关键表
REQUIRED_TABLES=("agents" "backup_tasks" "task_executions" "rclone_remotes" "users")
for table in "${REQUIRED_TABLES[@]}"; do
    if echo "$TABLES" | grep -q "$table"; then
        print_success "✓ 表 '$table' 存在"
    else
        print_error "✗ 表 '$table' 不存在"
    fi
done

# 检查初始数据
print_info ""
print_info "检查初始数据..."

# 检查默认管理员用户
ADMIN_COUNT=$($DOCKER_COMPOSE exec -T postgres psql -U "$DB_USER" -d "$DB_NAME" -t -c "SELECT COUNT(*) FROM users WHERE username='admin';" 2>/dev/null | tr -d ' ')
if [ "$ADMIN_COUNT" -gt 0 ]; then
    print_success "默认管理员用户已创建"
else
    print_warning "未找到默认管理员用户"
fi

# 检查系统设置
SETTINGS_COUNT=$($DOCKER_COMPOSE exec -T postgres psql -U "$DB_USER" -d "$DB_NAME" -t -c "SELECT COUNT(*) FROM system_settings;" 2>/dev/null | tr -d ' ')
if [ "$SETTINGS_COUNT" -gt 0 ]; then
    print_success "系统设置已初始化 ($SETTINGS_COUNT 条记录)"
else
    print_warning "系统设置未初始化"
fi

echo ""
print_success "数据库验证完成！"
echo ""
print_info "提示："
print_info "  - 使用以下命令连接到数据库："
print_info "    $DOCKER_COMPOSE exec postgres psql -U $DB_USER -d $DB_NAME"
print_info "  - 查看所有表："
print_info "    \\dt"
print_info "  - 退出 psql："
print_info "    \\q"