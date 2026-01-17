# API 接口清单

本清单基于代码静态分析整理（v2-dev 分支）。Hub 基址：`http(s)://<hub-host>`

## 约定

- Hub API 前缀：`/api/v1`
- 认证方式：
  - **Admin**：`Authorization: Bearer <jwt>`
  - **Agent**：`Authorization: Bearer <agent-id>:<api-key>`
  - **SSE**：`/events?token=<jwt>` 或 Header `Authorization: Bearer <jwt>`
  - **Local Agent API**：`Authorization: Bearer <LOCAL_AGENT_TOKEN>`（可选）
- 错误返回：`{"error":"..."}`（HTTP 4xx/5xx）

---

## Admin API

> 前缀：`/api/v1/admin`（除 `/events` 和 `/health`）

### 认证

#### 登录
```http
POST /api/v1/admin/login
```
- **认证**：无需认证
- **请求体**：
  ```json
  {
    "username": "string",
    "password": "string"
  }
  ```
- **响应 200**：
  ```json
  {
    "token": "jwt-token",
    "user": {
      "id": "uuid",
      "name": "string",
      "role": "admin"
    }
  }
  ```

#### 登出
```http
POST /api/v1/admin/logout
```
- **认证**：Admin JWT
- **响应 200**：`{"message":"Logged out successfully"}`

---

### Agents 管理

#### 列出所有 Agents
```http
GET /api/v1/admin/agents
```
- **认证**：Admin JWT
- **响应 200**：`Agent[]`

#### 获取 Agent 最新指标
```http
GET /api/v1/admin/agents/:id/metrics/latest
```
- **认证**：Admin JWT
- **响应 200**：
  ```json
  {
    "cpu_percent": 25.5,
    "memory_percent": 60.2,
    "disk_percent": 45.0,
    "recorded_at": "2026-01-17T12:00:00Z"
  }
  ```

#### 获取 Agent 历史指标
```http
GET /api/v1/admin/agents/:id/metrics/history
```
- **认证**：Admin JWT
- **查询参数**：
  - `hours`：小时数（默认 24）
- **响应 200**：`AgentMetrics[]`

#### 列出 Agent 文件系统目录
```http
GET /api/v1/admin/agents/:id/fs/list
```
- **认证**：Admin JWT
- **查询参数**：
  - `path`：目录路径
  - `limit`：返回数量限制（默认 200，最大 2000）
- **响应 200**：
  ```json
  {
    "path": "/path/to/dir",
    "entries": [
      {
        "name": "filename",
        "path": "/full/path",
        "is_dir": true,
        "size": 1024,
        "mode": "0755",
        "mod_time": "2026-01-17T12:00:00Z"
      }
    ],
    "total": 100
  }
  ```

#### 更新 Agent
```http
PUT /api/v1/admin/agents/:id
```
- **认证**：Admin JWT
- **请求体**：
  ```json
  {
    "name": "new-name"
  }
  ```
- **响应 200**：Updated `Agent`

#### 删除 Agent
```http
DELETE /api/v1/admin/agents/:id
```
- **认证**：Admin JWT
- **响应 204**：无内容

#### 创建注册令牌
```http
POST /api/v1/admin/agents/registration-token
```
- **认证**：Admin JWT
- **响应 201**：
  ```json
  {
    "id": "uuid",
    "token": "registration-token",
    "used": false,
    "expires_at": "2026-01-24T12:00:00Z",
    "created_at": "2026-01-17T12:00:00Z"
  }
  ```

---

### Tasks 管理

#### 列出所有任务
```http
GET /api/v1/admin/tasks
```
- **认证**：Admin JWT
- **响应 200**：
  ```json
  [
    {
      "id": "uuid",
      "name": "Backup MySQL",
      "description": "Daily MySQL backup",
      "source_path": "/var/lib/mysql",
      "remote_id": "uuid",
      "remote_path": "/backups/mysql",
      "schedule": "0 2 * * *",
      "enabled": true,
      "assigned_agents": ["agent-id-1", "agent-id-2"],
      "rclone_args": "--verbose",
      "created_at": "2026-01-01T00:00:00Z",
      "updated_at": "2026-01-10T00:00:00Z"
    }
  ]
  ```

#### 获取单个任务
```http
GET /api/v1/admin/tasks/:id
```
- **认证**：Admin JWT
- **响应 200**：`BackupTask`

#### 创建任务
```http
POST /api/v1/admin/tasks
```
- **认证**：Admin JWT
- **请求体**：
  ```json
  {
    "name": "string",
    "description": "string",
    "source_path": "/path/to/source",
    "remote_id": "uuid",
    "remote_path": "/remote/path",
    "schedule": "0 2 * * *",
    "enabled": true,
    "assigned_agents": ["agent-id-1"],
    "rclone_args": "--verbose"
  }
  ```
- **响应 201**：Created `BackupTask`

#### 更新任务
```http
PUT /api/v1/admin/tasks/:id
```
- **认证**：Admin JWT
- **请求体**：同创建任务
- **响应 200**：Updated `BackupTask`

#### 删除任务
```http
DELETE /api/v1/admin/tasks/:id
```
- **认证**：Admin JWT
- **响应 204**：无内容

---

### Remotes 管理

#### 列出所有 Remotes
```http
GET /api/v1/admin/remotes
```
- **认证**：Admin JWT
- **响应 200**：`RcloneRemote[]`（不含 `config_data`）

#### 获取单个 Remote（含配置）
```http
GET /api/v1/admin/remotes/:id
```
- **认证**：Admin JWT
- **响应 200**：
  ```json
  {
    "id": "uuid",
    "name": "s3-backup",
    "type": "s3",
    "config_data": "[s3-backup]\ntype = s3\n...",
    "created_at": "2026-01-01T00:00:00Z",
    "updated_at": "2026-01-10T00:00:00Z"
  }
  ```
  注：`config_data` 为解密后的明文

#### 创建 Remote
```http
POST /api/v1/admin/remotes
```
- **认证**：Admin JWT
- **请求体**：
  ```json
  {
    "name": "s3-backup",
    "config_data": "[s3-backup]\ntype = s3\naccess_key_id = xxx\n..."
  }
  ```
- **响应 201**：Created `RcloneRemote`（不含 `config_data`）

#### 更新 Remote
```http
PUT /api/v1/admin/remotes/:id
```
- **认证**：Admin JWT
- **请求体**：同创建
- **响应 200**：`{"status":"updated"}`

#### 删除 Remote
```http
DELETE /api/v1/admin/remotes/:id
```
- **认证**：Admin JWT
- **响应 204**：无内容

#### 测试 Remote 连接
```http
POST /api/v1/admin/remotes/:id/test
```
- **认证**：Admin JWT
- **请求体**（可选）：
  ```json
  {
    "test_path": "/optional/path"
  }
  ```
- **响应 200**（成功）：
  ```json
  {
    "success": true,
    "message": "Remote connection successful",
    "remote_id": "uuid",
    "remote_name": "s3-backup",
    "duration_ms": 1234,
    "output": "rclone lsd output..."
  }
  ```
- **响应 503**（Agent 不可用）：
  ```json
  {
    "success": false,
    "message": "Failed to connect to local Agent",
    "error": "connection refused"
  }
  ```

---

### Executions 管理

#### 列出执行记录
```http
GET /api/v1/admin/executions
```
- **认证**：Admin JWT
- **查询参数**：
  - `page`：页码（默认 1）
  - `limit`：每页数量（默认 20）
  - `status`：过滤状态（pending/running/success/failed/cancelled）
  - `task_id`：过滤任务 ID
  - `agent_id`：过滤 Agent ID
- **响应 200**：
  ```json
  {
    "executions": [],
    "page": 1,
    "limit": 20,
    "total": 100,
    "total_pages": 5
  }
  ```

#### 获取执行统计
```http
GET /api/v1/admin/executions/stats
```
- **认证**：Admin JWT
- **响应 200**：
  ```json
  {
    "total": 1000,
    "running": 5,
    "success": 850,
    "failed": 145,
    "avg_duration_seconds": 125.5,
    "success_rate_24h": 0.92
  }
  ```

#### 获取执行详情
```http
GET /api/v1/admin/executions/:id
```
- **认证**：Admin JWT
- **响应 200**：
  ```json
  {
    "id": "uuid",
    "task_id": "uuid",
    "agent_id": "uuid",
    "status": "success",
    "started_at": "2026-01-17T02:00:00Z",
    "ended_at": "2026-01-17T02:05:30Z",
    "log_output": "rclone logs...",
    "error_message": null
  }
  ```

#### 触发任务执行
```http
POST /api/v1/admin/executions/trigger
```
- **认证**：Admin JWT
- **请求体**：
  ```json
  {
    "task_id": "uuid",
    "agent_id": "uuid"
  }
  ```
- **响应 201**：Created `TaskExecution`（status=pending）

#### 取消执行
```http
POST /api/v1/admin/executions/:id/cancel
```
- **认证**：Admin JWT
- **响应 200**：`{"message":"Execution cancelled"}`

---

### Dashboard & Statistics

#### 获取 Dashboard 统计
```http
GET /api/v1/admin/dashboard/stats
```
- **认证**：Admin JWT
- **响应 200**：
  ```json
  {
    "agents": {
      "total": 10,
      "online": 8,
      "offline": 2
    },
    "tasks": {
      "total": 25,
      "active": 20,
      "inactive": 5
    },
    "executions": {
      "total": 5000,
      "success": 4500,
      "failed": 450,
      "running": 5,
      "success_rate": 0.91
    },
    "timestamp": "2026-01-17T12:00:00Z"
  }
  ```

#### 获取最近活动
```http
GET /api/v1/admin/dashboard/recent
```
- **认证**：Admin JWT
- **响应 200**：
  ```json
  {
    "executions": [],
    "timestamp": "2026-01-17T12:00:00Z"
  }
  ```

#### 获取图表数据
```http
GET /api/v1/admin/dashboard/charts
```
- **认证**：Admin JWT
- **查询参数**：
  - `range`：时间范围（24h / 7d / 30d，默认 24h）
- **响应 200**：
  ```json
  {
    "data": [
      {
        "time": "2026-01-17T00:00:00Z",
        "success": 50,
        "failed": 5
      }
    ],
    "range": "24h",
    "days": 1,
    "timestamp": "2026-01-17T12:00:00Z"
  }
  ```

#### 获取统计概览
```http
GET /api/v1/admin/statistics/overview
```
- **认证**：Admin JWT
- **响应 200**：同 `dashboard/stats`

#### 获取 Agent 统计
```http
GET /api/v1/admin/statistics/agents/:id
```
- **认证**：Admin JWT
- **响应 200**：
  ```json
  {
    "total": 500,
    "success": 450,
    "failed": 50,
    "avg_duration": 120.5
  }
  ```

#### 获取 Task 统计
```http
GET /api/v1/admin/statistics/tasks/:id
```
- **认证**：Admin JWT
- **响应 200**：同 Agent 统计格式

---

### OAuth（一键授权）

#### 创建 Google Drive OAuth 流程
```http
POST /api/v1/admin/oauth/drive/flow
```
- **认证**：Admin JWT
- **请求体**：
  ```json
  {
    "client_id": "google-client-id",
    "client_secret": "google-client-secret"
  }
  ```
- **响应 200**：
  ```json
  {
    "flow_id": "uuid",
    "auth_url": "https://accounts.google.com/o/oauth2/auth?..."
  }
  ```

#### 创建 OneDrive OAuth 流程
```http
POST /api/v1/admin/oauth/onedrive/flow
```
- **认证**：Admin JWT
- **请求体**：
  ```json
  {
    "client_id": "onedrive-client-id",
    "client_secret": "onedrive-client-secret"
  }
  ```
- **响应 200**：同 Drive

#### 获取 Google Drive OAuth 流程状态
```http
GET /api/v1/admin/oauth/drive/flow/:flowId
```
- **认证**：Admin JWT
- **响应 200**：
  ```json
  {
    "status": "completed",
    "config_snippet": "[remote]\ntype = drive\ntoken = {...}"
  }
  ```

#### 获取 OneDrive OAuth 流程状态
```http
GET /api/v1/admin/oauth/onedrive/flow/:flowId
```
- **认证**：Admin JWT
- **响应 200**：同 Drive

---

## Agent API

> 前缀：`/api/v1/agent`

### 注册
```http
POST /api/v1/agent/register
```
- **认证**：无需认证
- **请求体**：
  ```json
  {
    "token": "registration-token",
    "name": "agent-name",
    "is_local": false
  }
  ```
- **响应 201**：
  ```json
  {
    "agent_id": "uuid",
    "api_key": "api-key"
  }
  ```

### 下载 Agent 二进制
```http
GET /api/v1/agent/download
```
- **认证**：无需认证
- **查询参数**（可选）：
  - `platform`：操作系统（linux/darwin/windows）
  - `arch`：架构（amd64/arm64）
- **响应 200**：二进制文件（application/octet-stream）

### 下载安装脚本
```http
GET /api/v1/agent/install.sh
```
- **认证**：无需认证
- **响应 200**：Shell 脚本（text/plain）

### WebSocket 连接
```http
GET /api/v1/agent/ws
```
- **认证**：Agent（`Authorization: Bearer <agent-id>:<api-key>`）
- **协议**：WebSocket
- **消息类型**：
  - `HEARTBEAT`：心跳
  - `TASK_DISPATCH`：任务下发
  - `TASK_CANCEL`：任务取消
  - `EXECUTION_UPDATE`：执行状态更新
  - `EXECUTION_LOGS`：日志流
  - `FS_LIST`：文件系统列表请求
  - `FS_LIST_RESULT`：文件系统列表结果
  - `CONFIG_SYNC`：配置同步

详见：`hub/api/agent_ws_protocol.go` 和 `agent/services/ws_protocol.go`

---

## OAuth 回调（Public）

> 前缀：`/api/v1/oauth`

### Google Drive 授权开始
```http
GET /api/v1/oauth/drive/start?flow_id=uuid
```
- **认证**：无需认证
- **功能**：重定向到 Google OAuth 授权页面

### Google Drive 授权回调
```http
GET /api/v1/oauth/drive/callback?code=xxx&state=flowId
```
- **认证**：无需认证
- **功能**：接收 Google 回调，交换 token

### OneDrive 授权开始
```http
GET /api/v1/oauth/onedrive/start?flow_id=uuid
```
- **认证**：无需认证
- **功能**：重定向到 Microsoft OAuth 授权页面

### OneDrive 授权回调
```http
GET /api/v1/oauth/onedrive/callback?code=xxx&state=flowId
```
- **认证**：无需认证
- **功能**：接收 Microsoft 回调，交换 token

---

## SSE & 健康检查

### Server-Sent Events
```http
GET /events?token=<jwt>
```
或
```http
GET /events
Authorization: Bearer <jwt>
```
- **认证**：Admin JWT
- **协议**：Server-Sent Events
- **事件类型**：
  - `connected`：连接建立
  - `agent.registered`：Agent 注册
  - `agent.status.update`：Agent 状态更新
  - `agent.heartbeat`：Agent 心跳
  - `task.created`：任务创建
  - `task.updated`：任务更新
  - `task.deleted`：任务删除
  - `task.dispatched`：任务已派发
  - `execution.status.update`：执行状态更新
  - `execution.log.update`：执行日志更新

### 健康检查
```http
GET /health
HEAD /health
```
- **认证**：无需认证
- **响应 200**（健康）：
  ```json
  {
    "status": "healthy",
    "time": 1705493200
  }
  ```
- **响应 503**（不健康）：
  ```json
  {
    "status": "unhealthy",
    "error": "database connection failed",
    "time": 1705493200
  }
  ```

---

## Local Agent API（用于 Remote 测试）

> Hub 通过环境变量 `LOCAL_AGENT_URL`（默认 `http://localhost:9092`）访问本地 Agent API

### 测试 Remote 连接
```http
POST /api/test-remote
```
- **认证**：Bearer Token（可选，通过 `LOCAL_AGENT_TOKEN` 配置）
- **请求体**：
  ```json
  {
    "remote_name": "s3-backup",
    "config_data": "[s3-backup]\ntype = s3\n...",
    "test_path": "/optional/path"
  }
  ```
- **响应 200**：
  ```json
  {
    "success": true,
    "message": "Remote connection successful",
    "duration_ms": 1234,
    "output": "rclone lsd output..."
  }
  ```

### Agent 健康检查
```http
GET /api/health
```
- **认证**：Bearer Token（可选）
- **响应 200**：
  ```json
  {
    "status": "healthy",
    "time": 1705493200
  }
  ```

### Agent 版本信息
```http
GET /api/version
```
- **认证**：Bearer Token（可选）
- **响应 200**：
  ```json
  {
    "agent_version": "2.0.0",
    "rclone_version": "v1.65.0"
  }
  ```

---

## 注意事项

1. **WebSocket 代理**：如果 Hub 部署在 Nginx 等反向代理后，需要配置 WebSocket Upgrade 支持
2. **OAuth 回调**：OAuth 流程存储在 Hub 进程内存中，多实例部署需要使用 sticky sessions 或共享存储
3. **CORS**：当前配置为允许所有域（`*`），生产环境建议限制为前端域名
4. **认证**：所有 Admin API 都需要 JWT 认证，通过 `AdminAuthMiddleware` 实现
5. **错误格式**：API 错误统一返回 `{"error":"error message"}`
6. **分页**：支持分页的接口统一使用 `page` 和 `limit` 参数

---

**文档版本**：2.0
**最后更新**：2026-01-17
**对应分支**：v2-dev
