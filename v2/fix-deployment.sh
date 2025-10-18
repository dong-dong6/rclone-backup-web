#!/bin/bash

# 修复部署问题的脚本
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

echo "================================================"
echo "   Rclone Backup Web V2 - 部署问题修复工具"
echo "================================================"
echo ""

# 检测 Docker Compose 命令
if command -v docker-compose &> /dev/null; then
    DOCKER_COMPOSE="docker-compose"
elif docker compose version &> /dev/null 2>&1; then
    DOCKER_COMPOSE="docker compose"
else
    print_error "未找到 Docker Compose"
    exit 1
fi

print_info "使用的 Docker Compose 命令: $DOCKER_COMPOSE"

# 步骤1: 停止所有容器并清理
print_info "停止现有容器..."
$DOCKER_COMPOSE down -v 2>/dev/null || true

# 步骤2: 清理数据目录（可选）
if [ -d "./data" ]; then
    print_warning "检测到数据目录 ./data"
    read -p "是否要清理数据目录? 这将删除所有数据! [y/N]: " choice
    case "$choice" in
        y|Y )
            print_info "清理数据目录..."
            sudo rm -rf ./data
            print_success "数据目录已清理"
            ;;
        * )
            print_info "保留现有数据"
            ;;
    esac
fi

# 步骤3: 创建必要的目录
print_info "创建数据目录..."
mkdir -p data/{postgres,hub,agent,redis,web-ui}
mkdir -p data/backups/{postgres,hub}

# 步骤4: 设置权限
print_info "设置目录权限..."
chmod -R 755 data

# 步骤5: 检查并验证 .env 文件
if [ ! -f ".env" ]; then
    print_error "未找到 .env 文件"
    print_info "请先运行 './deploy.sh hub' 生成配置文件"
    exit 1
fi

# 步骤6: 验证数据库迁移文件
MIGRATIONS_DIR="./hub/database/migrations"
if [ ! -d "$MIGRATIONS_DIR" ]; then
    print_error "未找到 migrations 目录: $MIGRATIONS_DIR"
    exit 1
fi

print_info "检查数据库迁移文件..."
if [ -f "$MIGRATIONS_DIR/000_complete_schema.sql" ]; then
    print_success "找到完整的初始化脚本"
    
    # 清理冗余文件
    if [ -f "$MIGRATIONS_DIR/001_initial_schema.sql" ] || [ -f "$MIGRATIONS_DIR/schema.sql" ]; then
        print_warning "发现冗余的 SQL 文件，将清理..."
        rm -f "$MIGRATIONS_DIR/001_initial_schema.sql"
        rm -f "$MIGRATIONS_DIR/002_user_auth.sql"
        rm -f "$MIGRATIONS_DIR/schema.sql"
        print_success "冗余文件已清理"
    fi
else
    print_error "未找到 000_complete_schema.sql"
    print_info "需要创建完整的数据库初始化脚本"
    exit 1
fi

# 步骤7: 重新构建并启动服务
print_info "重新构建镜像..."
$DOCKER_COMPOSE build

print_info "启动服务..."
$DOCKER_COMPOSE up -d postgres

# 等待数据库完全就绪
print_info "等待数据库初始化..."
sleep 5

# 检查数据库是否正常
for i in {1..30}; do
    if $DOCKER_COMPOSE exec -T postgres pg_isready -U ${DB_USER:-rclone} &> /dev/null; then
        print_success "数据库已就绪"
        break
    fi
    echo -n "."
    sleep 2
done

# 启动其他服务
print_info "启动 Hub 服务..."
$DOCKER_COMPOSE up -d hub-api web-ui redis

# 如果需要本地 Agent
if [[ "$1" == "with-agent" ]]; then
    print_info "启动本地 Agent..."
    $DOCKER_COMPOSE --profile local-agent up -d
fi

# 等待服务启动
print_info "等待服务启动..."
sleep 10

# 检查服务状态
print_info "检查服务状态..."
$DOCKER_COMPOSE ps

# 测试健康检查
print_info "测试 Hub API 健康检查..."
HUB_PORT=$(grep HUB_PORT .env | cut -d= -f2 | tr -d ' ' || echo "8080")
if curl -s "http://localhost:${HUB_PORT}/health" > /dev/null; then
    print_success "Hub API 健康检查通过"
else
    print_warning "Hub API 健康检查失败，请检查日志"
fi

print_success "修复完成！"
print_info ""
print_info "你可以使用以下命令检查日志："
print_info "  $DOCKER_COMPOSE logs -f postgres    # 查看数据库日志"
print_info "  $DOCKER_COMPOSE logs -f hub-api      # 查看 Hub API 日志"
print_info "  $DOCKER_COMPOSE logs -f web-ui       # 查看 Web UI 日志"
print_info ""
print_info "访问地址："
print_info "  Web UI: http://localhost:3000"
print_info "  Hub API: http://localhost:${HUB_PORT}"