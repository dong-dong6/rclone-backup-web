# 🚀 Rclone Backup Web V2.0

一个强大的分布式备份管理系统，基于 Hub-and-Spoke 架构设计。

## ✨ 核心特性

- 🌐 **分布式架构** - 中央 Hub 管理，多 Agent 执行
- 📅 **智能调度** - Cron 表达式支持，防重复执行机制
- 🔄 **实时监控** - WebSocket 双向通信 + SSE 实时日志流
- 🛡️ **高可用性** - Agent 本地回退机制，离线可用
- 🎨 **现代 UI** - React 18 + TypeScript + Vite，响应式设计
- 🔒 **安全加密** - JWT 认证 + AES-256 加密存储
- 📦 **容器化部署** - Docker Compose 一键部署
- 🔌 **一键 OAuth** - Google Drive / OneDrive 快速授权
- 📂 **远程浏览** - 通过 WebSocket 远程浏览 Agent 文件系统
- 📊 **指标监控** - 实时 CPU/内存/磁盘监控

## 🏗️ 技术栈

### 后端
- **语言**: Go 1.23+
- **框架**: Gin
- **数据库**: PostgreSQL 15
- **认证**: JWT (golang-jwt/jwt)
- **加密**: AES-256 (golang.org/x/crypto)
- **调度**: Cron (robfig/cron)
- **通信**: WebSocket (gorilla/websocket)

### 前端
- **框架**: React 18
- **语言**: TypeScript 5.3
- **构建**: Vite 5
- **图标**: Tabler Icons / Lucide React
- **图表**: Recharts
- **国际化**: i18next

### 基础设施
- **容器**: Docker + Docker Compose
- **反向代理**: Nginx（可选）
- **部署**: 本地构建，无需外部镜像仓库

## 🏗️ 架构

```
┌─────────────────────────────────────────────┐
│                Hub (中央节点)                 │
│  ┌─────────┐  ┌─────────┐  ┌─────────────┐ │
│  │ Web UI  │  │  API    │  │ PostgreSQL  │ │
│  │ (内嵌)  │  │ (Go)    │  │             │ │
│  └─────────┘  └─────────┘  └─────────────┘ │
└───────────────────┬─────────────────────────┘
                    │ WebSocket (双向)
    ┌───────────────┴───────────────┐
    │                               │
┌───▼────┐                    ┌────▼───┐
│ Agent  │                    │ Agent  │
│   +    │                    │   +    │
│ Rclone │                    │ Rclone │
└────────┘                    └────────┘
```

### 通信机制
- **WebSocket**: Agent 与 Hub 通过 `/api/v1/agent/ws` 建立长连接
  - 任务下发/取消
  - 心跳与状态上报
  - 日志实时流传输
  - 文件系统远程浏览
  - 配置同步
- **SSE**: 前端通过 `/events` 接收实时事件推送
- **HTTP**: 注册、下载、OAuth 回调等无状态操作

## 🚀 快速开始

### 方式一：使用部署脚本（推荐）

```bash
# 1. 克隆仓库
git clone https://github.com/yourusername/rclone-backup-web.git
cd rclone-backup-web

# 2. 部署 Hub 服务
./deploy.sh hub

# 或部署 Hub + 本地 Agent（Hub 自我备份）
./deploy.sh hub-with-agent
```

### 方式二：使用 Makefile

```bash
# 1. 初始化环境（生成 .env）
make init

# 2. 生成安全密钥
make gen-keys >> .env

# 3. 构建镜像
make build

# 4. 启动服务
make up                # 仅 Hub
```

## 📋 部署脚本命令

```bash
./deploy.sh [命令] [选项]

命令：
  hub              # 部署 Hub 服务
  hub-with-agent   # 部署 Hub 和本地 Agent
  agent            # 部署独立 Agent
  build            # 构建镜像
  status           # 查看服务状态
  logs [service]   # 查看日志
  stop             # 停止服务
  restart          # 重启服务
  clean            # 交互式清理数据（自动备份）
  backup           # 备份数据库
  restore <file>   # 恢复备份
  help             # 显示帮助

选项：
  --clean          # 部署前清理数据
```

## 🔧 环境配置

### 必需的环境变量

创建 `.env` 文件（或复制 `.env.example`）：

```bash
# 数据库配置
DB_PASSWORD=your_strong_password
DB_USER=rclone
DB_NAME=rclone_backup

# 安全密钥（使用 make gen-keys 生成）
JWT_SECRET=your_64_character_random_string
ENCRYPTION_KEY=your_32_character_random_string

# 服务端口
WEB_PORT=3000

# 可选：日志级别
LOG_LEVEL=info
GIN_MODE=release
```

### 生成安全密钥

```bash
# 自动生成并追加到 .env
make gen-keys >> .env

# 或手动生成
openssl rand -hex 32  # JWT_SECRET
openssl rand -hex 16  # ENCRYPTION_KEY
```

## 🌐 访问系统

启动后访问：

- **Web UI**: http://localhost:3000
- **API 文档**: [docs/api-interface-catalog.md](docs/api-interface-catalog.md)
- **健康检查**: http://localhost:3000/health

### 默认管理员账号

- **用户名**: `admin`
- **密码**: `admin`

⚠️ **重要**: 首次登录后必须立即修改默认密码！

## 📦 部署方案

### 方案 1：仅 Hub 部署

适合集中管理，Agent 部署在其他服务器：

```bash
# 1. 配置环境变量
cp .env.example .env
vim .env  # 编辑配置

# 2. 构建并启动
make build
make up

# 3. 访问 Web UI
# http://localhost:3000
```

### 方案 2：Hub + 本地 Agent

Hub 可以备份自身数据：

```bash
# 使用部署脚本（推荐）
./deploy.sh hub-with-agent

# 或使用 Makefile
make build
# 在 .env 中添加 LOCAL_AGENT_REGISTRATION_TOKEN
make up-with-agent
```

### 方案 3：远程 Agent 部署

在远程服务器部署 Agent：

```bash
# 方式 A：使用安装脚本（推荐）
# 1. 在 Hub Web UI 生成注册令牌
# 2. 在远程服务器执行：
curl -fsSL http://your-hub:3000/api/v1/agent/install.sh | bash -s -- \
  --token YOUR_TOKEN \
  --hub-url http://your-hub:3000

# 方式 B：手动部署
# 详见 AGENT-SETUP.md
./start-agent.sh --token YOUR_TOKEN --hub-url http://your-hub:3000
```

## 📊 数据管理

### 数据存储位置

所有数据透明存储在 `./data` 目录：

```
data/
├── postgres/         # PostgreSQL 数据库文件
├── hub/
│   ├── data/        # Hub 运行时数据
│   └── logs/        # Hub 日志
├── local-agent/     # 本地 Agent 数据（如果启用）
└── backups/         # 数据库备份文件
```

### 备份与恢复

```bash
# 手动备份
make backup
# 或
./deploy.sh backup

# 自动备份（每 24 小时）
make backup-auto

# 恢复备份
make restore FILE=data/backups/backup-20260117-120000.sql.gz
# 或
./deploy.sh restore data/backups/backup-20260117-120000.sql.gz
```

## 🔧 维护与监控

### 查看服务状态

```bash
# 使用部署脚本
./deploy.sh status

# 使用 Makefile
make status

# 直接使用 Docker Compose
docker compose ps
```

### 查看日志

```bash
# 所有服务日志
make logs

# 特定服务日志
make logs-hub
docker compose logs -f hub

# Hub 内部日志文件
tail -f data/hub/logs/hub.log
```

### 健康检查

```bash
# Hub 健康检查
curl http://localhost:3000/health

# 响应示例：
# {"status":"healthy","time":1705493200}
```

## 🔄 更新升级

```bash
# 1. 备份数据
./deploy.sh backup

# 2. 拉取最新代码
git pull

# 3. 重新构建并更新
./deploy.sh build
./deploy.sh restart
```

## 🛠️ 故障排查

完整的故障排查指南请参考：[docs/troubleshooting.md](docs/troubleshooting.md)

### 常见问题

#### Hub 无法启动

```bash
# 查看日志
docker compose logs hub

# 检查数据库连接
docker compose exec postgres pg_isready

# 检查环境变量
docker compose config
```

#### Agent 无法连接

```bash
# 检查 WebSocket 连接
# 查看 Hub 日志中的 [AgentWS] 相关信息
docker compose logs hub | grep AgentWS

# 检查网络连通性
docker compose exec hub ping agent-hostname

# 重新注册 Agent
# 删除 Agent 数据并使用新令牌重新注册
```

#### 前端无法加载

```bash
# 检查静态文件是否存在
ls -la hub/static/web/

# 重新构建前端
cd hub/web
npm install
npm run build
```

## 📚 文档

- [Agent 设置指南](AGENT-SETUP.md)
- [API 接口清单](docs/api-interface-catalog.md)
- [Rclone 配置模板](docs/rclone-remote-templates.md)
- [故障排查指南](docs/troubleshooting.md)
- [部署指南](deploy/README.md)
- [本地 Agent 配置](deploy/LOCAL_AGENT.md)

## 🔐 安全建议

1. **修改默认密码**: 首次登录后立即修改 admin 密码
2. **使用强密钥**: JWT_SECRET 和 ENCRYPTION_KEY 应使用随机生成
3. **限制 CORS**: 生产环境修改 `hub/main.go` 中的 CORS 配置
4. **HTTPS 部署**: 生产环境使用 Nginx + Let's Encrypt
5. **定期备份**: 启用自动备份或设置定时任务
6. **网络隔离**: 使用防火墙限制 Hub 和 Agent 的访问

## 🌟 功能清单

### 已实现

- ✅ Hub-and-Spoke 分布式架构
- ✅ WebSocket 双向通信
- ✅ JWT 认证与 AES-256 加密
- ✅ Cron 调度与防重复执行
- ✅ 实时日志流（SSE）
- ✅ Agent 健康监控（CPU/内存/磁盘）
- ✅ 远程文件系统浏览
- ✅ Remote 配置管理与测试
- ✅ 任务触发与取消
- ✅ 执行历史与统计
- ✅ Dashboard 可视化
- ✅ Google Drive / OneDrive 一键 OAuth
- ✅ 多语言支持（中文/英文）
- ✅ 配置导入/导出
- ✅ 自动备份与恢复
- ✅ Docker 一键部署

### 计划中

- 🔲 多用户权限管理
- 🔲 Webhook 通知（企业微信/钉钉/Slack）
- 🔲 S3 兼容存储直接备份
- 🔲 备份版本管理
- 🔲 增量备份支持
- 🔲 Prometheus 指标导出
- 🔲 高可用 Hub 部署方案

## 🤝 贡献

欢迎提交 Issue 和 Pull Request！

### 开发环境

```bash
# 后端开发
cd hub
go run main.go

# 前端开发
cd hub/web
npm install
npm run dev

# Agent 开发
cd agent
go run main.go
```

## 📄 许可证

MIT License - 详见 [LICENSE](LICENSE)

## 🙏 致谢

- [Rclone](https://rclone.org/) - 强大的云存储同步工具
- [Gin](https://github.com/gin-gonic/gin) - 高性能 Go Web 框架
- [React](https://reactjs.org/) - 用户界面库
- [PostgreSQL](https://www.postgresql.org/) - 强大的开源数据库
- [Docker](https://www.docker.com/) - 容器化平台
- 所有贡献者和用户

---

**项目版本**: 2.0
**最后更新**: 2026-01-17
**开发分支**: v2-dev

**Made with ❤️ by the Community**
