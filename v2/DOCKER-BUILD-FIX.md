# Docker 构建问题修复

## 问题描述

在运行 `./deploy.sh hub` 时，Docker 构建失败，错误信息：

```
failed to solve: failed to compute cache key: failed to calculate checksum of ref 559c1cb4-2668-42bb-bb7b-42def7378924::no0g4k8b5olsejbjfxrlzy5d3: "/agent": not found
```

## 问题原因

Docker 构建上下文设置为 `./hub`，但 agent 目录位于 `../agent`，超出了构建上下文范围。

## 解决方案

### 1. 修改 docker-compose.yml

将构建上下文从 `./hub` 改为 `.`（v2 目录）：

```yaml
hub-api:
  build:
    context: .                    # 改为 v2 目录
    dockerfile: ./hub/Dockerfile  # 指定 Dockerfile 路径
```

### 2. 修改 Dockerfile 路径

更新 Dockerfile 中的所有 COPY 命令：

```dockerfile
# Hub 构建阶段
COPY hub/go.mod hub/go.sum ./
COPY hub/ ./

# Agent 构建阶段  
COPY agent/go.mod agent/go.sum ./
COPY agent/ ./

# 运行时阶段
COPY hub/database/migrations /app/database/migrations
```

## 验证修复

运行以下命令验证修复：

```bash
cd /workspace/v2
docker compose build hub-api
```

## 构建结果

修复后的 Docker 镜像将包含：

- **Hub API 服务**：Go 二进制文件
- **多平台 Agent 二进制文件**：
  - `rclone-backup-agent-linux-amd64`
  - `rclone-backup-agent-linux-arm64` 
  - `rclone-backup-agent-linux-arm`
  - `rclone-backup-agent-darwin-amd64`
  - `rclone-backup-agent-darwin-arm64`
  - `rclone-backup-agent-windows-amd64.exe`

## 使用方式

### 启动 Hub 服务

```bash
cd /workspace/v2
./deploy.sh hub
```

### 下载 Agent

```bash
# 自动检测平台
curl -L "http://localhost:8080/api/v1/agent/download" -o agent

# 指定平台
curl -L "http://localhost:8080/api/v1/agent/download?platform=linux&arch=amd64" -o agent
```

### 使用启动脚本

```bash
cd /workspace/v2
./start-agent.sh --token YOUR_TOKEN
```

## 优势

1. **一次构建**：Hub 镜像包含所有平台的 Agent 二进制文件
2. **无需本地构建**：Agent 二进制文件已预构建在 Docker 镜像中
3. **版本一致**：确保 Agent 和 Hub 版本完全匹配
4. **简化部署**：用户只需下载并运行，无需编译

---

**状态**：✅ 已修复并验证