# Agent 部署选项总结

本文档总结了所有可用的 Agent 部署选项和脚本。

## 📋 部署脚本概览

| 脚本名称 | 类型 | 适用场景 | 配置方式 |
|---------|------|----------|----------|
| `deploy-agent.sh` | 基础 | 快速测试 | 默认配置 |
| `deploy-agent-interactive.sh` | 交互式 | 用户友好 | 交互式输入 |
| `deploy-agent-with-params.sh` | 参数化 | 自动化部署 | 命令行参数 |

## 🚀 快速选择指南

### 我是新手，想要简单易用的方式
**推荐：** `deploy-agent-interactive.sh`
```bash
cd /workspace/v2
./deploy-agent-interactive.sh
```
- ✅ 交互式引导输入
- ✅ 自动生成配置文件
- ✅ 详细的运行说明
- ✅ 适合学习和测试

### 我需要自动化部署
**推荐：** `deploy-agent-with-params.sh`
```bash
cd /workspace/v2
./deploy-agent-with-params.sh --token YOUR_TOKEN --hub-url http://hub.example.com
```
- ✅ 支持命令行参数
- ✅ 适合 CI/CD 集成
- ✅ 可脚本化调用
- ✅ 支持所有配置选项

### 我只需要快速测试
**推荐：** `deploy-agent.sh`
```bash
cd /workspace/v2
./deploy-agent.sh
```
- ✅ 最简单的方式
- ✅ 使用默认配置
- ✅ 快速验证功能

## 🔧 配置参数详解

### 必需参数
- **注册令牌** (`--token`): 从 Hub Web 界面生成的令牌

### 可选参数
- **Hub URL** (`--hub-url`): Hub 服务地址，默认 `http://localhost:8080`
- **Agent 名称** (`--agent-name`): Agent 显示名称，默认 `agent-主机名`
- **工作目录** (`--work-dir`): Agent 工作目录，默认 `/opt/rclone-agent`
- **最大并发数** (`--max-concurrent`): 同时执行的最大任务数，默认 `3`
- **心跳间隔** (`--heartbeat-interval`): 心跳间隔秒数，默认 `30`

## 📁 生成的文件

所有部署脚本都会生成以下文件：

```
v2/
├── agent/
│   ├── rclone-backup-agent.json    # Agent 配置文件
│   └── .env                        # 环境变量文件
└── hub/
    └── static/binaries/
        ├── rclone-backup-agent     # 二进制文件
        ├── rclone-backup-agent-latest -> rclone-backup-agent
        └── agent-config.json       # 配置文件副本
```

## 🎯 使用场景示例

### 场景1：本地开发测试
```bash
# 使用交互式脚本，输入本地 Hub 地址
./deploy-agent-interactive.sh
# 输入: http://localhost:8080
# 输入: test-agent
# 输入: 从 Web 界面获取的令牌
```

### 场景2：生产环境部署
```bash
# 使用参数化脚本，指定生产环境配置
./deploy-agent-with-params.sh \
  --token "prod-token-123" \
  --hub-url "https://hub.company.com" \
  --agent-name "prod-backup-agent" \
  --work-dir "/opt/backup-agent" \
  --max-concurrent 5 \
  --heartbeat-interval 60
```

### 场景3：CI/CD 自动化
```bash
# 在 CI/CD 脚本中使用
export HUB_URL="https://hub.company.com"
export REGISTRATION_TOKEN="ci-token-456"

./deploy-agent-with-params.sh \
  --token "$REGISTRATION_TOKEN" \
  --hub-url "$HUB_URL" \
  --agent-name "ci-agent-$(date +%s)"
```

### 场景4：多环境部署
```bash
# 开发环境
./deploy-agent-with-params.sh --token dev-token --hub-url http://dev-hub:8080

# 测试环境
./deploy-agent-with-params.sh --token test-token --hub-url http://test-hub:8080

# 生产环境
./deploy-agent-with-params.sh --token prod-token --hub-url https://prod-hub.com
```

## 🔍 故障排除

### 常见问题

1. **注册令牌无效**
   - 检查令牌是否从正确的 Hub 实例获取
   - 确认令牌未过期
   - 验证 Hub URL 是否正确

2. **构建失败**
   - 检查 Go 环境：`go version`
   - 清理模块缓存：`go clean -modcache`
   - 重新下载依赖：`go mod download`

3. **下载失败**
   - 确认 Hub 服务正在运行
   - 检查防火墙设置
   - 验证 URL 路径：`/api/v1/agent/download`

### 调试命令

```bash
# 检查二进制文件
ls -la /workspace/v2/hub/static/binaries/

# 测试下载
curl -I http://localhost:8080/api/v1/agent/download

# 检查配置文件
cat /workspace/v2/agent/rclone-backup-agent.json

# 查看环境变量
cat /workspace/v2/agent/.env
```

## 📚 相关文档

- [AGENT-DEPLOYMENT.md](./AGENT-DEPLOYMENT.md) - 详细部署指南
- [README.md](./README.md) - 项目主文档
- [agent/README.md](./agent/README.md) - Agent 使用说明

## 🆘 获取帮助

```bash
# 查看参数化脚本帮助
./deploy-agent-with-params.sh --help

# 查看测试脚本
./test-agent-deployment.sh

# 查看项目文档
cat AGENT-DEPLOYMENT.md
```

---

**提示：** 建议在正式使用前，先使用交互式脚本熟悉配置流程，然后根据实际需求选择合适的部署方式。