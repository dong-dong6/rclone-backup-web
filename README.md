# 🚀 Rclone Backup Web V2.0

一个强大的分布式备份管理系统，基于Hub-and-Spoke架构设计。

## ✨ 特性

- 🌐 **分布式架构** - 中央Hub管理，多Agent执行
- 📅 **智能调度** - Cron表达式支持，防重复执行
- 🔄 **实时监控** - SSE推送，实时日志流
- 🛡️ **高可用性** - Agent本地回退机制
- 🎨 **现代UI** - React + TypeScript，响应式设计
- 🔒 **安全加密** - JWT认证，AES-256加密存储
- 📦 **容器化部署** - Docker Compose一键部署

## 🏗️ 架构

```
┌─────────────────────────────────────────────┐
│                  Hub (中央节点)               │
│  ┌─────────┐  ┌─────────┐  ┌─────────────┐ │
│  │ Web UI  │  │ API     │  │ PostgreSQL  │ │
│  └─────────┘  └─────────┘  └─────────────┘ │
└───────────────────┬─────────────────────────┘
                    │ HTTPS/WSS
    ┌───────────────┴───────────────┐
    │                               │
┌───▼────┐                    ┌────▼───┐
│ Agent  │                    │ Agent  │
│   +    │                    │   +    │
│Sidecar │                    │Sidecar │
└────────┘                    └────────┘
```

Agent 与 Hub 默认通过 WebSocket（`/api/v1/agent/ws`）建立双向长连接，用于任务下发/取消、日志与状态回传、远程目录浏览；HTTP 仅用于下载/注册等必要引导流程。

如果 Hub 部署在 Nginx 等反向代理后，需要开启 WebSocket Upgrade（示例已包含在 `hub/web/nginx.conf`、`hub/web/nginx.conf.template`、`docker/hub/nginx.conf`）。

## 🚀 快速开始

### 使用部署脚本（推荐）

```bash
# 克隆仓库
git clone https://github.com/yourusername/rclone-backup-web.git
cd rclone-backup-web/v2

# 部署Hub服务
./deploy.sh hub

# 或部署Hub + 本地Agent（自我备份）
./deploy.sh hub-with-agent
```

### 部署脚本功能

```bash
./deploy.sh [命令] [选项]

命令：
  hub              # 部署Hub服务
  hub-with-agent   # 部署Hub和本地Agent
  agent            # 部署独立Agent
  build            # 构建镜像
  status           # 查看状态
  logs [service]   # 查看日志
  stop             # 停止服务
  restart          # 重启服务
  clean            # 交互式清理数据
  backup           # 备份数据
  restore <file>   # 恢复备份
  help             # 显示帮助

选项：
  --clean          # 部署前清理数据
```

### 核心特性

- ✅ **透明数据管理** - 所有数据存储在 `./data` 目录
- ✅ **智能清理** - 交互式数据清理，自动备份
- ✅ **版本兼容** - 自动检测Docker Compose V1/V2
- ✅ **本地构建** - 所有镜像本地构建，无需外部仓库
- ✅ **交互配置** - 首次运行自动生成配置

### 使用Makefile（可选）

```bash
# 初始化环境
make init

# 构建镜像
make build

# 启动服务
make up                # 仅Hub
make up-with-agent     # Hub + 本地Agent
```

## 📋 部署方案

### 方案1：仅Hub部署

适合集中管理，Agent部署在其他服务器：

```bash
# 1. 配置环境变量
cp .env.example .env
vim .env  # 编辑配置

# 2. 构建并启动
make build
make up

# 3. 访问Web UI
# http://localhost:3000
```

### 方案2：Hub + 本地Agent

Hub可以备份自身数据：

```bash
# 1. 先启动Hub
make up

# 2. 获取注册令牌
# 在Web UI的Agents页面生成令牌

# 3. 配置本地Agent
echo "LOCAL_AGENT_REGISTRATION_TOKEN=xxx" >> .env

# 4. 重启服务
make down
make up-with-agent
```

### 方案3：独立Agent部署

在远程服务器部署Agent：

```bash
# 1. 复制Agent配置
scp -r v2/agent v2/docker-compose.agent.yml remote-server:

# 2. 在远程服务器
cd agent
docker-compose -f docker-compose.agent.yml up -d
```

## 🔧 配置说明

### 必需的环境变量

| 变量 | 说明 | 示例 |
|------|------|------|
| `DB_PASSWORD` | 数据库密码 | `strong_password` |
| `JWT_SECRET` | JWT签名密钥 | 64位随机字符串 |
| `ENCRYPTION_KEY` | 加密密钥 | 32位随机字符串 |

### 生成安全密钥

```bash
# 自动生成所有密钥
make gen-keys >> .env

# 或手动生成
openssl rand -hex 32  # JWT_SECRET
openssl rand -hex 16  # ENCRYPTION_KEY
```

## 📦 镜像构建

所有镜像都在本地构建，无需依赖外部镜像仓库：

```bash
# 构建所有镜像
make build

# 查看构建的镜像
docker images | grep rclone-backup

# 输出:
# rclone-backup-hub      latest    xxx    1.2GB
# rclone-backup-web      latest    xxx    45MB
# rclone-backup-agent    latest    xxx    850MB
```

## 🌐 访问地址

启动后可以访问：

- **Web UI**: http://localhost:3000
- **API**: http://localhost:8080
- **Metrics**: http://localhost:9090/metrics

默认管理员账号：
- 用户名：`admin`
- 密码：`admin` （首次登录后请修改）

## 📊 监控与维护

### 查看服务状态

```bash
make status           # 服务状态
make logs            # 查看日志
make local-agent-logs # 本地Agent日志
```

### 数据库备份

```bash
make backup          # 手动备份
make backup-auto     # 启动自动备份
make restore FILE=backups/xxx.sql.gz  # 恢复
```

### 健康检查

```bash
curl http://localhost:8080/health
```

## 🔄 更新升级

```bash
# 更新代码
git pull

# 重新构建并更新
make update              # 更新Hub
make update-with-agent   # 更新Hub和Agent
```

## 🛠️ 故障排查

### Hub无法启动

```bash
# 检查日志
docker-compose logs hub-api

# 检查数据库连接
docker-compose exec postgres pg_isready

# 重置环境（谨慎）
make clean
```

### Agent无法连接

```bash
# 检查网络
docker-compose exec local-agent curl http://hub-api:8080/health

# 重新注册
make down
rm -rf agent_data
make up-with-agent
```

## 📚 文档

- [部署指南](deploy/README.md)
- [本地Agent配置](deploy/LOCAL_AGENT.md)
- [API文档](docs/API.md)
- [架构设计](docs/ARCHITECTURE.md)

## 🤝 贡献

欢迎提交Issue和Pull Request！

## 📄 许可

MIT License

## 🙏 致谢

- [Rclone](https://rclone.org/) - 强大的云存储工具
- [Docker](https://www.docker.com/) - 容器化平台
- 所有贡献者

---

**Made with ❤️ by the Community**
