# 故障排查（Hub ↔ Agent / 监控 / 路径浏览）

## 1) 节点显示在线，但「监控」没有数据/不更新

1. **先看数据库是否有指标入库**
   - `select count(*), max(recorded_at) from agent_metrics where agent_id='AGENT_UUID';`
   - `count=0`：说明 Hub 没收到心跳或写入失败（重点看 Hub 的 WebSocket 日志）。
   - `count>0`：说明后端正常，优先排查前端 SSE 连接是否断开。

2. **看 Hub 是否收到 Agent WebSocket 心跳**
   - Hub 容器日志：`docker logs rclone-backup-hub --since 10m | grep -E \"\\[AgentWS\\]|agent\\.heartbeat\"`

3. **看浏览器 SSE 是否在线**
   - DevTools → Network → `events?token=...` 是否一直保持 `Pending`。
   - 新版前端已支持 SSE 自动重连；若仍异常，优先看 Hub 的 `SSE client connected/disconnected` 日志。

## 2) 创建任务时「浏览源路径」列表无法加载

1. **先看 HTTP 请求返回**
   - DevTools → Network → `GET /api/v1/admin/agents/:id/fs/list?path=...`
   - 常见：
     - `409 Agent not connected`：Hub 认为该 Agent 的 WebSocket 不在线（等待 Agent 重连）。
     - `400 Failed to list directory`：通常是权限/路径不可访问。
     - `504 Directory listing timed out`：Agent 未回包（看 Hub/Agent 两端日志）。

2. **看 Hub ↔ Agent 的 FS List 业务日志**
   - Hub：`docker logs rclone-backup-hub --since 10m | grep \"\\[FSList\\]\"`
   - Agent（默认写文件）：`sudo tail -n 200 /opt/rclone-agent/logs/agent.log | grep \"\\[FSList\\]\"`

3. **确认 Agent 进程能读到该路径**
   - Agent 默认以 `rclone-agent` 用户运行：`sudo -u rclone-agent ls -la /目标路径`
   - 如果提示 `Permission denied`：
     - 给目录授予读/执行权限（或 ACL），或
     - 以更高权限运行 Agent（更不安全，按需取舍）。

4. **systemd ProtectHome 影响**
   - 旧安装脚本使用 `ProtectHome=true` 会直接屏蔽 `/home`、`/root` 等目录。
   - 新安装脚本默认 `ProtectHome=read-only`（允许读取但禁止写入）。
   - 已安装机器可手动改：编辑 `/etc/systemd/system/rclone-agent.service` 后执行：
     - `sudo systemctl daemon-reload`
     - `sudo systemctl restart rclone-agent`

## 3) 如何查看 Agent 运行日志

- 文件日志（默认）：`sudo tail -f /opt/rclone-agent/logs/agent.log`
- systemd：`sudo journalctl -u rclone-agent -n 200 --no-pager`
