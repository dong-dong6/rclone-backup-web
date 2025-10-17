# Rclone-Backup-Web V2.0 - 分布式备份系统

## 概述

Rclone-Backup-Web V2.0 是一个基于 Hub-and-Spoke 架构的分布式备份解决方案，实现了集中管理、分布式执行的备份策略。系统支持多节点部署，具有高可用性和资源高效的特点。

### 主要特性

- **集中管理**: 通过单一 Web 界面管理所有备份节点和任务
- **分布式执行**: 轻量级 Agent 在各 VPS 节点上执行备份任务
- **高可用性**: 支持本地回退机制，即使中央节点离线也能继续执行
- **实时监控**: 通过 SSE 实时查看备份状态和日志
- **安全加密**: 敏感数据使用 AES-256 加密，通信使用 TLS
- **灵活调度**: 支持 Cron 表达式的灵活任务调度

## 系统架构

```
┌─────────────────────────────────────────────────────────┐
│                     中央节点 (Hub)                        │
│  ┌─────────┐  ┌──────────┐  ┌────────────┐            │
│  │ Web UI  │──│ API Server│──│ PostgreSQL │            │
│  └─────────┘  └──────────┘  └────────────┘            │
└──────────────────────┬──────────────────────────────────┘
                       │ HTTPS/WSS
        ┌──────────────┼──────────────┐
        │              │              │
┌───────▼─────┐ ┌──────▼─────┐ ┌─────▼──────┐
│  Agent #1   │ │  Agent #2   │ │  Agent #3  │
│  ┌────────┐ │ │  ┌────────┐ │ │ ┌────────┐ │
│  │ Agent  │ │ │  │ Agent  │ │ │ │ Agent  │ │
│  └───┬────┘ │ │  └───┬────┘ │ │ └───┬────┘ │
│      │      │ │      │      │ │     │      │
│  ┌───▼────┐ │ │  ┌───▼────┐ │ │ ┌───▼────┐ │
│  │Rclone  │ │ │  │Rclone  │ │ │ │Rclone  │ │
│  │Sidecar │ │ │  │Sidecar │ │ │ │Sidecar │ │
│  └────────┘ │ │  └────────┘ │ │ └────────┘ │
└─────────────┘ └─────────────┘ └─────────────┘
   子节点 #1       子节点 #2       子节点 #3
```

## 快速开始

### 1. 部署中央节点

```bash
# 克隆仓库
git clone https://github.com/your-repo/rclone-backup-web.git
cd rclone-backup-web/v2

# 配置环境变量
cp docker/hub/.env.example docker/hub/.env
# 编辑 .env 文件，设置数据库密码、JWT密钥、加密密钥等

# 启动中央节点
cd docker/hub
docker-compose up -d

# 查看日志
docker-compose logs -f
```

### 2. 注册并部署 Agent 节点

#### 在 Web UI 中创建注册令牌

1. 访问 http://your-hub-domain
2. 登录管理界面
3. 进入 "Agents" 页面
4. 点击 "Create Registration Token"
5. 复制生成的令牌

#### 在目标 VPS 上部署 Agent

```bash
# 复制 agent 部署文件到目标 VPS
scp -r docker/agent/ user@vps-ip:/opt/rclone-agent/

# SSH 到目标 VPS
ssh user@vps-ip

# 配置 Agent
cd /opt/rclone-agent
cp .env.example .env

# 首次注册（使用注册令牌）
curl -X POST http://your-hub-domain/api/v1/agent/register \
  -H "Content-Type: application/json" \
  -d '{
    "token": "your-registration-token",
    "name": "vps-node-01"
  }'

# 将返回的 agent_id 和 api_key 填入 .env 文件

# 启动 Agent
docker-compose up -d
```

### 3. 配置备份任务

#### 通过 Web UI

1. **添加 Rclone 远程存储**
   - 进入 "Remotes" 页面
   - 点击 "Add Remote"
   - 输入远程存储配置（S3、Google Drive、OneDrive 等）

2. **创建备份任务**
   - 进入 "Tasks" 页面
   - 点击 "Create Task"
   - 选择远程存储、源路径、目标路径
   - 设置 Cron 调度表达式
   - 分配到相应的 Agent 节点

3. **监控执行**
   - 在 "Dashboard" 查看整体状态
   - 在 "Executions" 查看详细执行历史
   - 实时日志会自动推送到界面

## API 文档

### Agent API

#### 注册 Agent
```http
POST /api/v1/agent/register
Content-Type: application/json

{
  "token": "registration-token",
  "name": "agent-name"
}
```

#### 发送心跳
```http
POST /api/v1/agent/heartbeat
Authorization: Bearer <api-key>
Content-Type: application/json

{
  "status": "idle"
}
```

### Admin API

#### 登录
```http
POST /api/v1/admin/login
Content-Type: application/json

{
  "username": "admin",
  "password": "password"
}
```

#### 获取 Agent 列表
```http
GET /api/v1/admin/agents
Authorization: Bearer <jwt-token>
```

#### 创建备份任务
```http
POST /api/v1/admin/tasks
Authorization: Bearer <jwt-token>
Content-Type: application/json

{
  "name": "Daily Website Backup",
  "rclone_remote_id": "uuid",
  "source_path": "/var/www/html",
  "destination_path": "backups/website",
  "schedule": "0 2 * * *",
  "assigned_agent_ids": ["agent-uuid"]
}
```

## 配置说明

### 中央节点环境变量

```bash
# 数据库配置
DATABASE_URL=postgres://user:pass@localhost:5432/dbname

# 安全配置
JWT_SECRET=your-jwt-secret-key
ENCRYPTION_KEY=your-encryption-key

# 服务配置
PORT=8080
GIN_MODE=release
```

### Agent 环境变量

```bash
# Hub 连接配置
HUB_URL=http://hub.example.com
AGENT_ID=agent-uuid
AGENT_API_KEY=agent-api-key

# Agent 配置
HEARTBEAT_INTERVAL=30s
CONFIG_CACHE_DIR=/var/lib/rclone-agent
```

## 安全考虑

1. **通信加密**: 生产环境必须使用 HTTPS/TLS
2. **凭证保护**: 
   - API Key 使用 bcrypt 哈希存储
   - Rclone 配置使用 AES-256 加密
3. **访问控制**: 
   - Agent 只能访问分配给自己的任务
   - Admin API 需要 JWT 认证
4. **输入验证**: 所有输入都经过严格验证

## 故障排查

### Agent 无法连接到 Hub

1. 检查网络连接
2. 验证 HUB_URL 配置
3. 确认 API Key 正确
4. 查看 Hub 日志

### 备份任务执行失败

1. 检查 Rclone 配置
2. 验证源路径存在且有读权限
3. 确认远程存储凭证有效
4. 查看执行日志

### 本地回退不工作

1. 确认配置已缓存到本地
2. 检查 Cron 调度器状态
3. 验证本地时间同步

## 开发指南

### 本地开发环境

```bash
# 启动数据库
docker run -d \
  -e POSTGRES_PASSWORD=password \
  -p 5432:5432 \
  postgres:15-alpine

# 运行 Hub API
cd v2/hub
go mod download
go run .

# 运行 Web UI 开发服务器
cd v2/hub/web
npm install
npm run dev

# 运行 Agent
cd v2/agent
go mod download
go run .
```

### 构建生产镜像

```bash
# 构建 Hub
cd v2/hub
docker build -t rclone-hub:v2.0.0 .

# 构建 Agent
cd v2/agent
docker build -t rclone-agent:v2.0.0 .
```

## 许可证

MIT License

## 贡献

欢迎提交 Issue 和 Pull Request！

## 支持

- GitHub Issues: [链接]
- 文档: [链接]
- 社区讨论: [链接]