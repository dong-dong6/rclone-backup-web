# API 接口清单（静态 Review）

本清单基于代码静态检查整理（未本地运行）。默认 Hub 基址为 `http(s)://<hub-host>`。

## 约定

- Hub API 前缀：`/api/v1`
- 认证
  - Admin：`Authorization: Bearer <jwt>`（或 SSE 使用 `/events?token=<jwt>`）
  - Agent：`Authorization: Bearer <agent-id>:<api-key>`
  - Local Agent API（用于 Hub 测试 Remote）：`Authorization: Bearer <LOCAL_AGENT_TOKEN>`（可选）
- 错误返回：通常为 `{"error":"..."}`（HTTP 4xx/5xx）

---

## 前端（Admin Web）→ Hub（Admin API）

> 统一前缀：`/api/v1/admin`（除 `/events`、`/health`）

### Auth

- `POST /api/v1/admin/login`（无鉴权）
  - Body：`{ "username": string, "password": string }`
  - 200：`{ "token": string, "user": { "id": string, "name": string, "role": "admin" } }`
- `POST /api/v1/admin/logout`（Admin JWT）
  - 200：`{ "message": "..." }`

### Agents

- `GET /api/v1/admin/agents`（Admin JWT）
  - 200：`Agent[]`
- `DELETE /api/v1/admin/agents/:id`（Admin JWT）
  - 204：空
- `POST /api/v1/admin/agents/:id/sync`（Admin JWT）
  - 200：`{ "message": "Config sync scheduled" }`
- `POST /api/v1/admin/agents/registration-token`（Admin JWT）
  - 201：`{ id, token, used, used_by_agent_id?, expires_at, created_at }`

### Tasks

- `GET /api/v1/admin/tasks`（Admin JWT）
  - 200：`BackupTask[]`（包含 `assigned_agents`；兼容字段 `assigned_agent_ids`）
- `GET /api/v1/admin/tasks/:id`（Admin JWT）
  - 200：`BackupTask`
- `POST /api/v1/admin/tasks`（Admin JWT）
  - Body：`BackupTask`（前端使用 `assigned_agents: string[]`；后端也接受旧字段 `assigned_agent_ids`）
  - 201：`BackupTask`
- `PUT /api/v1/admin/tasks/:id`（Admin JWT）
  - Body：`BackupTask`（如未显式传 `assigned_agents/assigned_agent_ids`，将保持原有分配）
  - 200：`BackupTask`
- `DELETE /api/v1/admin/tasks/:id`（Admin JWT）
  - 204：空

### Remotes

- `GET /api/v1/admin/remotes`（Admin JWT）
  - 200：`RcloneRemote[]`（不包含 `config_data`）
- `GET /api/v1/admin/remotes/:id`（Admin JWT）
  - 200：`{ id, name, type?, config_data, created_at, updated_at }`（`config_data` 为解密明文，用于回填编辑）
- `POST /api/v1/admin/remotes`（Admin JWT）
  - Body：`{ "name": string, "config_data": string }`
  - 201：`RcloneRemote`（不包含 `config_data`）
- `PUT /api/v1/admin/remotes/:id`（Admin JWT）
  - Body：`{ "name": string, "config_data": string }`
  - 200：`{ "status": "updated" }`
- `DELETE /api/v1/admin/remotes/:id`（Admin JWT）
  - 204：空
- `POST /api/v1/admin/remotes/:id/test`（Admin JWT）
  - 200：`{ success, message, remote_id, remote_name, duration_ms, output?, error? }`
  - 503：`{ error, message }`（本地 Agent 不可用）

### Executions

- `GET /api/v1/admin/executions`（Admin JWT）
  - Query：`page`/`limit`/`status`/`task_id`/`agent_id`
  - 200：`{ executions, items(兼容), page, limit, total, total_pages }`
- `GET /api/v1/admin/executions/stats`（Admin JWT）
  - 200：`{ total, running, success, failed, avg_duration_seconds, success_rate_24h }`
- `GET /api/v1/admin/executions/:id`（Admin JWT）
  - 200：`TaskExecution`（包含 `log_output`、`error_message`）
- `POST /api/v1/admin/executions/trigger`（Admin JWT）
  - Body：`{ "task_id": string, "agent_id": string }`
  - 201：`TaskExecution`（status=pending）
- `POST /api/v1/admin/executions/:id/cancel`（Admin JWT）
  - 200：`{ "message": "Execution cancelled" }`

### Dashboard / Statistics

- `GET /api/v1/admin/dashboard/stats`（Admin JWT）
  - 200：`{ agents:{total,online,offline}, tasks:{total,active,inactive}, executions:{total,success,failed,running,success_rate}, timestamp }`
- `GET /api/v1/admin/dashboard/recent`（Admin JWT）
  - 200：`{ executions, timestamp }`
- `GET /api/v1/admin/dashboard/charts`（Admin JWT）
  - Query：`range=24h|7d|30d`
  - 200：`{ data:[{time,success,failed}], range, days, timestamp }`
- `GET /api/v1/admin/statistics/overview`（Admin JWT）
  - 200：同 `dashboard/stats`
- `GET /api/v1/admin/statistics/agents/:id`（Admin JWT）
  - 200：`{ total, success, failed, avg_duration }`
- `GET /api/v1/admin/statistics/tasks/:id`（Admin JWT）
  - 200：`{ total, success, failed, avg_duration }`

### Settings

- `GET /api/v1/admin/settings`（Admin JWT）
  - 200：`{ hub_name, session_timeout, log_level, enable_metrics }`
- `PUT /api/v1/admin/settings`（Admin JWT）
  - Body：同上
  - 200：同上

### Import / Export

- `GET /api/v1/admin/export/config`（Admin JWT）
  - 200：下载 `application/json` 文件：`{ version, exported_at, settings, remotes[], tasks[] }`
- `POST /api/v1/admin/import/config`（Admin JWT）
  - `multipart/form-data`：`file`（≤5MB）
  - 200：`{ imported_remotes, imported_tasks }`

---

## 前端（Admin Web）→ Hub（非 /api/v1）

- `GET /events`（SSE，需要 Admin JWT；支持 Header Bearer 或 Query `token`）
  - 事件类型：
    - `connected`
    - `agent.registered`
    - `agent.status.update`
    - `agent.heartbeat`
    - `task.created` / `task.updated` / `task.deleted`
    - `task.dispatched`
    - `execution.status.update`
    - `execution.log.update`
- `GET|HEAD /health`（无鉴权）
  - 200：`{ status:"healthy", time:<unix> }`

---

## Agent → Hub（Agent API）

> 统一前缀：`/api/v1/agent`

### 注册/下载（无鉴权）

- `POST /api/v1/agent/register`
  - Body：`{ "token": string, "name": string, "is_local"?: boolean }`
  - 201：`{ "agent_id": string, "api_key": string }`
- `GET /api/v1/agent/download`
  - Query（可选）：`platform`、`arch`
  - 200：下载二进制（`application/octet-stream`）
- `GET /api/v1/agent/install.sh`
  - 200：安装脚本（`text/plain`）

### WebSocket（Agent 鉴权）

- `GET /api/v1/agent/ws`（WebSocket Upgrade）
  - Header：`Authorization: Bearer <agent-id>:<api-key>`
  - Hub ↔ Agent 通过 WS 传输心跳、任务下发/取消、日志与状态上报、远程目录浏览、配置同步等消息（消息协议见代码：`hub/api/agent_ws_protocol.go`、`agent/services/ws_protocol.go`）。

---

## Hub → Local Agent API（用于 Remote Test）

> Hub 通过环境变量 `LOCAL_AGENT_URL`（默认 `http://localhost:9092`）访问本地 Agent API。

- `POST /api/test-remote`（Bearer 可选）
  - Body：`{ remote_name, config_data }`（明文 rclone 配置）
  - 200：`{ success, message, duration_ms, output?, error? }`
- `GET /api/health`（Bearer 可选）
  - 200：`{ status:"healthy", time:<unix> }`
- `GET /api/version`（Bearer 可选）
  - 200：`{ agent_version, rclone_version }`
