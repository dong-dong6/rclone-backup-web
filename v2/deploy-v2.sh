#!/bin/bash

# ============================================
# Rclone Backup Web V2.0 - 智能部署脚本
# 特性：
# - 透明的数据持久化
# - 交互式数据清理
# - 自动备份旧数据
# ============================================

set -e

# 颜色定义
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
BLUE='\033[0;34m'
CYAN='\033[0;36m'
MAGENTA='\033[0;35m'
NC='\033[0m' # No Color

# 打印函数
print_info() {
    echo -e "${BLUE}[INFO]${NC} $1"
}

print_success() {
    echo -e "${GREEN}[SUCCESS]${NC} $1"
}

print_warning() {
    echo -e "${YELLOW}[WARNING]${NC} $1"
}

print_error() {
    echo -e "${RED}[ERROR]${NC} $1"
}

print_prompt() {
    echo -e "${CYAN}[INPUT]${NC} $1"
}

# Docker Compose命令检测
detect_docker_compose() {
    if docker compose version &> /dev/null; then
        DOCKER_COMPOSE="docker compose"
    elif docker-compose --version &> /dev/null; then
        DOCKER_COMPOSE="docker-compose"
    else
        print_error "Docker Compose未安装"
        exit 1
    fi
}

# 显示数据目录信息
show_data_info() {
    echo ""
    echo -e "${MAGENTA}━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━${NC}"
    echo -e "${MAGENTA}📂 数据目录结构${NC}"
    echo -e "${MAGENTA}━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━${NC}"
    echo ""
    echo "所有持久化数据将存储在 ./data 目录中："
    echo ""
    echo "  ./data/"
    echo "  ├── postgres/        # PostgreSQL数据库"
    echo "  ├── redis/           # Redis缓存"
    echo "  ├── hub/"
    echo "  │   ├── config/      # Hub配置文件"
    echo "  │   ├── data/        # Hub运行数据"
    echo "  │   └── logs/        # Hub日志"
    echo "  ├── agent/           # Agent数据 (如果启用)"
    echo "  └── backups/         # 数据库备份"
    echo ""
    echo -e "${MAGENTA}━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━${NC}"
    echo ""
}

# 检查现有数据
check_existing_data() {
    if [ -d "./data" ]; then
        local size=$(du -sh ./data 2>/dev/null | cut -f1)
        print_warning "检测到现有数据目录 (大小: ${size})"
        
        # 列出主要数据目录
        echo ""
        echo "现有数据内容："
        for dir in postgres redis hub agent backups; do
            if [ -d "./data/$dir" ]; then
                local dir_size=$(du -sh ./data/$dir 2>/dev/null | cut -f1)
                echo "  - $dir: $dir_size"
            fi
        done
        return 0
    else
        return 1
    fi
}

# 备份现有数据
backup_existing_data() {
    local backup_dir="./data.backup.$(date +%Y%m%d-%H%M%S)"
    print_info "备份现有数据到 $backup_dir ..."
    
    # 停止容器以确保数据一致性
    print_info "停止运行中的容器..."
    $DOCKER_COMPOSE -f docker-compose.prod.yml down 2>/dev/null || true
    
    # 创建备份
    if mv ./data "$backup_dir"; then
        print_success "数据已备份到: $backup_dir"
        echo ""
        echo "  提示：您可以通过以下命令恢复数据："
        echo "  mv $backup_dir ./data"
        echo ""
        return 0
    else
        print_error "备份失败"
        return 1
    fi
}

# 清理数据函数
cleanup_data() {
    echo ""
    if check_existing_data; then
        echo ""
        print_warning "⚠️  警告：清理数据是不可逆的操作！"
        echo ""
        echo "请选择操作："
        echo "  1) 保留现有数据（推荐）"
        echo "  2) 备份后清理"
        echo "  3) 直接清理（危险！）"
        echo "  4) 取消部署"
        echo ""
        
        print_prompt "请输入选择 [1-4]: "
        read -r choice
        
        case "$choice" in
            1)
                print_info "保留现有数据"
                ;;
            2)
                if backup_existing_data; then
                    mkdir -p ./data
                    print_success "数据清理完成（已备份）"
                else
                    print_error "备份失败，已取消清理"
                    exit 1
                fi
                ;;
            3)
                print_prompt "请输入 'DELETE' 确认删除所有数据: "
                read -r confirm
                if [ "$confirm" = "DELETE" ]; then
                    print_info "正在清理数据..."
                    $DOCKER_COMPOSE -f docker-compose.prod.yml down -v 2>/dev/null || true
                    rm -rf ./data
                    mkdir -p ./data
                    print_success "数据已清理"
                else
                    print_info "已取消清理"
                fi
                ;;
            4)
                print_info "部署已取消"
                exit 0
                ;;
            *)
                print_error "无效的选择"
                exit 1
                ;;
        esac
    else
        print_info "未检测到现有数据"
        mkdir -p ./data
    fi
}

# 设置目录权限
setup_permissions() {
    print_info "设置目录权限..."
    
    # 创建必要的目录
    mkdir -p ./data/{postgres,redis,hub/{config,data,logs},agent,backups}
    
    # 设置权限（某些服务需要特定的权限）
    # PostgreSQL需要700权限
    chmod 700 ./data/postgres 2>/dev/null || true
    
    print_success "目录权限已设置"
}

# 主部署函数
deploy() {
    local deployment_type=$1
    
    echo ""
    echo -e "${CYAN}━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━${NC}"
    echo -e "${CYAN}   Rclone Backup Web V2.0 - 智能部署${NC}"
    echo -e "${CYAN}━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━${NC}"
    echo ""
    
    # 检测Docker Compose
    detect_docker_compose
    print_success "Docker Compose已就绪 ($DOCKER_COMPOSE)"
    
    # 显示数据目录信息
    show_data_info
    
    # 处理数据清理
    cleanup_data
    
    # 设置权限
    setup_permissions
    
    # 检查.env文件
    if [ ! -f .env ]; then
        print_warning ".env文件不存在"
        print_info "创建默认配置..."
        cat > .env << 'EOF'
# ============================================
# Rclone Backup Web - 环境配置
# ============================================

# 数据库配置
DB_NAME=rclone_backup
DB_USER=rclone
DB_PASSWORD=changeme_$(openssl rand -hex 8)

# 安全密钥（请修改！）
JWT_SECRET=$(openssl rand -hex 32)
ENCRYPTION_KEY=$(openssl rand -hex 16)

# 服务端口
HUB_PORT=8080
WEB_PORT=3000
METRICS_PORT=9090

# 应用配置
VERSION=latest
GIN_MODE=release
LOG_LEVEL=info
SESSION_TIMEOUT=24h
API_KEY_EXPIRY=365d
ENABLE_METRICS=true

# 数据库备份间隔（秒）
DB_BACKUP_INTERVAL=86400

# 本地Agent配置（可选）
LOCAL_AGENT_REGISTRATION_TOKEN=
LOCAL_AGENT_NAME=hub-local-agent
EOF
        print_success ".env文件已创建"
        print_warning "请编辑.env文件设置安全的密码！"
    else
        print_success "使用现有.env文件"
    fi
    
    # 构建镜像
    print_info "构建Docker镜像..."
    if [ "$deployment_type" = "hub-with-agent" ]; then
        $DOCKER_COMPOSE -f docker-compose.prod.yml --profile local-agent build
    else
        $DOCKER_COMPOSE -f docker-compose.prod.yml build hub-api web-ui
    fi
    print_success "镜像构建完成"
    
    # 启动服务
    print_info "启动服务..."
    if [ "$deployment_type" = "hub-with-agent" ]; then
        $DOCKER_COMPOSE -f docker-compose.prod.yml --profile local-agent up -d
    else
        $DOCKER_COMPOSE -f docker-compose.prod.yml up -d postgres redis hub-api web-ui
    fi
    
    # 等待服务就绪
    print_info "等待服务启动..."
    local max_attempts=30
    local attempt=0
    
    while [ $attempt -lt $max_attempts ]; do
        if curl -sf http://localhost:${HUB_PORT:-8080}/health > /dev/null 2>&1; then
            echo ""
            print_success "服务已就绪！"
            break
        fi
        echo -n "."
        sleep 2
        ((attempt++))
    done
    
    if [ $attempt -eq $max_attempts ]; then
        echo ""
        print_error "服务启动超时"
        print_info "查看日志："
        $DOCKER_COMPOSE -f docker-compose.prod.yml logs hub-api | tail -20
        exit 1
    fi
    
    # 显示服务状态
    echo ""
    print_info "服务状态："
    $DOCKER_COMPOSE -f docker-compose.prod.yml ps
    
    # 显示访问信息
    echo ""
    echo -e "${GREEN}━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━${NC}"
    echo -e "${GREEN}✨ 部署成功！${NC}"
    echo ""
    echo "📍 访问地址："
    echo "   Web UI: http://localhost:${WEB_PORT:-3000}"
    echo "   API: http://localhost:${HUB_PORT:-8080}"
    echo "   Metrics: http://localhost:${METRICS_PORT:-9090}/metrics"
    echo ""
    echo "🔑 默认管理员："
    echo "   用户名: admin"
    echo "   密码: admin"
    echo ""
    echo "📂 数据位置："
    echo "   ./data/"
    echo ""
    echo "📝 有用的命令："
    echo "   查看日志: $DOCKER_COMPOSE -f docker-compose.prod.yml logs -f"
    echo "   停止服务: $DOCKER_COMPOSE -f docker-compose.prod.yml down"
    echo "   备份数据: tar czf backup.tar.gz ./data"
    echo -e "${GREEN}━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━${NC}"
}

# 显示帮助
show_help() {
    echo "用法: $0 [命令] [选项]"
    echo ""
    echo "命令:"
    echo "  hub              部署Hub服务"
    echo "  hub-with-agent   部署Hub和本地Agent"
    echo "  stop             停止所有服务"
    echo "  clean            清理所有数据（交互式）"
    echo "  backup           备份数据目录"
    echo "  restore          恢复数据"
    echo "  help             显示帮助"
    echo ""
    echo "选项:"
    echo "  --force          跳过确认提示"
    echo "  --clean          部署前清理数据"
}

# 主函数
main() {
    case "$1" in
        hub)
            deploy "hub"
            ;;
        hub-with-agent)
            deploy "hub-with-agent"
            ;;
        stop)
            print_info "停止所有服务..."
            detect_docker_compose
            $DOCKER_COMPOSE -f docker-compose.prod.yml down
            print_success "服务已停止"
            ;;
        clean)
            detect_docker_compose
            cleanup_data
            ;;
        backup)
            if [ -d "./data" ]; then
                local backup_file="data-backup-$(date +%Y%m%d-%H%M%S).tar.gz"
                print_info "创建备份: $backup_file"
                tar czf "$backup_file" ./data
                print_success "备份完成: $backup_file"
            else
                print_error "数据目录不存在"
            fi
            ;;
        restore)
            if [ -z "$2" ]; then
                print_error "请指定备份文件"
                echo "用法: $0 restore <backup-file.tar.gz>"
                exit 1
            fi
            if [ -f "$2" ]; then
                print_info "恢复备份: $2"
                backup_existing_data
                tar xzf "$2"
                print_success "备份已恢复"
            else
                print_error "备份文件不存在: $2"
            fi
            ;;
        help|--help|-h)
            show_help
            ;;
        *)
            print_error "未知命令: $1"
            show_help
            exit 1
            ;;
    esac
}

# 执行主函数
main "$@"