# ============================================
# Rclone Backup Web V2.0 - Makefile
# ============================================

.PHONY: help up down restart logs build clean test shell backup restore

# Default target
help:
	@echo "Rclone Backup Web V2.0 - 可用命令"
	@echo ""
	@echo "基础命令:"
	@echo "  make up              - 启动Hub服务（不含Agent）"
	@echo "  make up-with-agent   - 启动Hub服务（含本地Agent）"
	@echo "  make down            - 停止所有服务"
	@echo "  make restart         - 重启所有服务"
	@echo "  make logs            - 查看服务日志"
	@echo "  make status          - 查看服务状态"
	@echo ""
	@echo "构建命令:"
	@echo "  make build           - 构建所有镜像"
	@echo "  make rebuild         - 强制重新构建镜像"
	@echo ""
	@echo "Agent命令:"
	@echo "  make agent-up        - 启动远程Agent"
	@echo "  make agent-down      - 停止远程Agent"
	@echo "  make agent-logs      - 查看远程Agent日志"
	@echo "  make local-agent-logs - 查看本地Agent日志"
	@echo ""
	@echo "维护命令:"
	@echo "  make backup          - 备份数据库"
	@echo "  make backup-auto     - 启动自动备份服务"
	@echo "  make restore         - 恢复数据库"
	@echo "  make clean           - 清理所有数据（危险！）"
	@echo "  make test            - 运行测试"
	@echo ""
	@echo "调试命令:"
	@echo "  make shell           - 进入Hub容器"
	@echo "  make db-shell        - 进入数据库"

# ============================================
# 基础命令
# ============================================

# Profile变量（用于控制启动哪些服务）
COMPOSE_PROFILES ?=

# 启动Hub服务（不含Agent）
up:
	@echo "🚀 启动Rclone Backup Hub服务..."
	@docker-compose $(if $(COMPOSE_PROFILES),--profile $(COMPOSE_PROFILES)) up -d
	@echo "✅ Hub服务已启动！"
	@echo ""
	@echo "访问地址:"
	@echo "  Web UI: http://localhost:3000"
	@echo "  API: http://localhost:8080"
	@echo "  Metrics: http://localhost:9090/metrics"
	@if [ "$(COMPOSE_PROFILES)" = "local-agent" ]; then \
		echo "  本地Agent: 已启动"; \
	fi

# 启动Hub服务（含本地Agent）
up-with-agent:
	@echo "🚀 启动Rclone Backup服务（含本地Agent）..."
	@if [ -z "$${LOCAL_AGENT_REGISTRATION_TOKEN}" ] && [ -z "$${LOCAL_AGENT_API_KEY}" ]; then \
		echo "⚠️  警告: 未配置LOCAL_AGENT_REGISTRATION_TOKEN"; \
		echo "请先:"; \
		echo "  1. 运行 'make up' 启动Hub"; \
		echo "  2. 在Web UI中生成注册令牌"; \
		echo "  3. 将令牌添加到.env文件的LOCAL_AGENT_REGISTRATION_TOKEN"; \
		echo "  4. 再次运行 'make up-with-agent'"; \
		exit 1; \
	fi
	@$(MAKE) COMPOSE_PROFILES=local-agent up

# 停止所有服务
down:
	@echo "🛑 停止Rclone Backup服务..."
	@docker-compose --profile local-agent --profile db-backup down

# 重启所有服务
restart: down up

# 查看日志
logs:
	@docker-compose logs -f --tail=100

# 查看特定服务日志
logs-%:
	@docker-compose logs -f --tail=100 $*

# 查看服务状态
status:
	@echo "📊 服务状态:"
	@docker-compose ps
	@echo ""
	@echo "🏥 健康检查:"
	@curl -s http://localhost:8080/health | jq '.' || echo "Hub API未响应"

# ============================================
# 构建命令
# ============================================

# 构建所有镜像（本地构建）
build:
	@echo "🔨 本地构建Docker镜像..."
	@docker-compose --profile local-agent build
	@echo "✅ 镜像构建完成！"
	@docker images | grep rclone-backup

# 强制重新构建（清除缓存）
rebuild:
	@echo "🔨 强制重新构建镜像..."
	@docker-compose --profile local-agent build --no-cache
	@echo "✅ 镜像重新构建完成！"

# 构建并启动
build-and-up: build up

# 构建并启动（含本地Agent）
build-and-up-with-agent: build up-with-agent

# 更新服务（本地构建）
update:
	@echo "🔄 更新服务..."
	@$(MAKE) build
	@$(MAKE) COMPOSE_PROFILES=$(COMPOSE_PROFILES) up

# 更新服务（含本地Agent）
update-with-agent:
	@$(MAKE) COMPOSE_PROFILES=local-agent update

# 拉取最新镜像
pull:
	@echo "📥 拉取最新镜像..."
	@docker-compose pull

# ============================================
# Agent命令
# ============================================

# 启动Agent
agent-up:
	@echo "🚀 启动Agent..."
	@docker-compose -f docker-compose.agent.yml up -d

# 停止Agent
agent-down:
	@echo "🛑 停止Agent..."
	@docker-compose -f docker-compose.agent.yml down

# 查看Agent日志
agent-logs:
	@docker-compose -f docker-compose.agent.yml logs -f --tail=100

# Agent状态
agent-status:
	@docker-compose -f docker-compose.agent.yml ps

# 查看本地Agent日志
local-agent-logs:
	@docker-compose logs -f --tail=100 local-agent local-rclone-sidecar

# 本地Agent状态
local-agent-status:
	@docker-compose ps local-agent local-rclone-sidecar

# ============================================
# 维护命令
# ============================================

# 备份数据库
backup:
	@echo "💾 备份数据库..."
	@mkdir -p backups
	@docker-compose exec -T postgres pg_dump -U $${DB_USER:-rclone} $${DB_NAME:-rclone_backup} | gzip > backups/backup-$$(date +%Y%m%d-%H%M%S).sql.gz
	@echo "✅ 备份完成: backups/backup-$$(date +%Y%m%d-%H%M%S).sql.gz"

# 启动自动备份服务
backup-auto:
	@echo "🔄 启动自动备份服务..."
	@docker-compose --profile db-backup up -d db-backup
	@echo "✅ 自动备份服务已启动（每24小时备份一次）"

# 恢复数据库 (使用: make restore FILE=backups/backup-xxx.sql.gz)
restore:
	@if [ -z "$(FILE)" ]; then \
		echo "❌ 请指定备份文件: make restore FILE=backups/backup-xxx.sql.gz"; \
		exit 1; \
	fi
	@echo "🔄 恢复数据库..."
	@gunzip < $(FILE) | docker-compose exec -T postgres psql -U rclone rclone_backup
	@echo "✅ 恢复完成"

# 清理所有数据（危险！）
clean:
	@echo "⚠️  警告: 这将删除所有数据！"
	@echo "按 Ctrl+C 取消，或按 Enter 继续..."
	@read confirm
	@docker-compose down -v
	@rm -rf backups/*
	@echo "🗑️  所有数据已清理"

# ============================================
# 测试命令
# ============================================

# 运行测试
test:
	@echo "🧪 运行测试..."
	@cd test && ./e2e_test.sh

# API测试
test-api:
	@echo "🧪 测试API..."
	@curl -s http://localhost:8080/health | jq '.'

# 调度测试
test-scheduling:
	@echo "🧪 测试调度系统..."
	@cd test && ./test_scheduling.sh

# ============================================
# 调试命令
# ============================================

# 进入Hub容器
shell:
	@docker-compose exec hub-api sh

# 进入数据库
db-shell:
	@docker-compose exec postgres psql -U rclone rclone_backup

# 查看数据库表
db-tables:
	@docker-compose exec postgres psql -U rclone -d rclone_backup -c "\dt"

# 查看环境变量
env:
	@docker-compose config

# ============================================
# 开发命令
# ============================================

# 开发模式（实时重载）
dev:
	@echo "🔧 启动开发模式..."
	@GIN_MODE=debug LOG_LEVEL=debug docker-compose up

# 监视文件变化
watch:
	@echo "👁️  监视文件变化..."
	@docker-compose up -d
	@docker-compose logs -f hub-api

# 格式化代码
fmt:
	@echo "✨ 格式化Go代码..."
	@cd hub && go fmt ./...
	@cd agent && go fmt ./...

# 代码检查
lint:
	@echo "🔍 检查代码..."
	@cd hub && golangci-lint run
	@cd agent && golangci-lint run

# ============================================
# 生产部署
# ============================================

# 生产环境启动
prod:
	@echo "🚀 启动生产环境..."
	@docker-compose -f docker-compose.yml -f docker-compose.prod.yml up -d

# 生成密钥
gen-keys:
	@echo "🔐 生成安全密钥..."
	@echo "JWT_SECRET=$$(openssl rand -hex 32)"
	@echo "ENCRYPTION_KEY=$$(openssl rand -hex 16)"
	@echo "DB_PASSWORD=$$(openssl rand -base64 20)"

# 初始化环境
init:
	@echo "🎯 初始化环境..."
	@if [ ! -f .env ]; then \
		cp .env.example .env; \
		echo "✅ 已创建 .env 文件"; \
		echo "⚠️  请编辑 .env 文件并设置安全密钥"; \
	else \
		echo "⚠️  .env 文件已存在"; \
	fi