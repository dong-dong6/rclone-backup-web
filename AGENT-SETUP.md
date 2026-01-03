# Agent 设置指南

本文档说明如何设置和运行 Rclone Backup Agent。

## 🚀 快速开始

### 1. 启动 Hub 服务

```bash
cd /workspace/v2/hub
go run main.go
```

或者使用 Docker：

```bash
cd /workspace/v2
docker compose up hub
```

### 2. 获取注册令牌

1. 访问 Hub Web 界面：http://localhost:8080
2. 登录到管理界面
3. 进入 "Agents" 页面
4. 点击 "注册新节点" 按钮
5. 复制生成的注册令牌

### 3. 运行 Agent

使用简单的启动脚本：

```bash
cd /workspace/v2
./start-agent.sh --token YOUR_TOKEN --hub-url http://localhost:8080
```

## 📋 启动脚本选项

```bash
./start-agent.sh [选项]

选项:
  -u, --hub-url URL         Hub服务URL (默认: http://localhost:8080)
  -n, --agent-name NAME     Agent名称 (默认: agent-主机名)
  -t, --token TOKEN         注册令牌 (必需)
  -w, --work-dir DIR        工作目录 (默认: /opt/rclone-agent)
  -c, --max-concurrent NUM  最大并发数 (默认: 3)
  -i, --heartbeat-interval NUM  心跳间隔秒数 (默认: 30)
  -p, --platform PLATFORM  目标平台 (linux/amd64, linux/arm64, darwin/amd64, darwin/arm64, windows/amd64)
  -a, --arch ARCH           目标架构 (amd64, arm64, arm)
  -d, --daemon              以守护进程模式运行
  -h, --help               显示帮助信息
```

## 🌍 多平台支持

Agent 支持以下平台：

| 平台 | 架构 | 文件名 |
|------|------|--------|
| Linux | amd64 | rclone-backup-agent-linux-amd64 |
| Linux | arm64 | rclone-backup-agent-linux-arm64 |
| Linux | arm | rclone-backup-agent-linux-arm |
| macOS | amd64 | rclone-backup-agent-darwin-amd64 |
| macOS | arm64 | rclone-backup-agent-darwin-arm64 |
| Windows | amd64 | rclone-backup-agent-windows-amd64.exe |

## 📁 生成的文件

启动脚本会在工作目录中创建：

```
/opt/rclone-agent/
├── rclone-backup-agent          # Agent 二进制文件
├── agent-config.json           # 配置文件
├── .env                        # 环境变量文件
└── agent.log                   # 日志文件（守护进程模式）
```

## 🔧 使用示例

### 基本使用
```bash
./start-agent.sh --token abc123
```

### 指定 Hub 地址
```bash
./start-agent.sh --token abc123 --hub-url http://hub.example.com
```

### 指定 Agent 名称
```bash
./start-agent.sh --token abc123 --agent-name my-backup-agent
```

### 守护进程模式
```bash
./start-agent.sh --token abc123 --daemon
```

### 指定平台（如果自动检测失败）
```bash
./start-agent.sh --token abc123 --platform linux --arch arm64
```

### 完整配置
```bash
./start-agent.sh \
  --token abc123 \
  --hub-url http://hub.example.com \
  --agent-name prod-backup-agent \
  --work-dir /opt/backup-agent \
  --max-concurrent 5 \
  --heartbeat-interval 60 \
  --daemon
```

## 🔍 故障排除

### 常见问题

1. **下载失败**
   - 检查 Hub 服务是否正在运行
   - 验证 Hub URL 是否正确
   - 确认网络连接正常

2. **注册失败**
   - 检查注册令牌是否正确
   - 确认令牌未过期
   - 验证 Hub URL 可访问

3. **平台检测失败**
   - 使用 `--platform` 和 `--arch` 参数手动指定
   - 检查系统架构：`uname -m`

### 调试命令

```bash
# 检查下载的文件
ls -la /opt/rclone-agent/

# 测试二进制文件
/opt/rclone-agent/rclone-backup-agent --help

# 查看配置文件
cat /opt/rclone-agent/agent-config.json

# 查看日志（守护进程模式）
tail -f /opt/rclone-agent/agent.log
```

## 🐳 Docker 部署

### 构建包含 Agent 的 Hub 镜像

```bash
cd /workspace/v2
docker compose build hub-api
```

或者直接构建：

```bash
cd /workspace/v2
docker build -t rclone-backup-hub:latest -f hub/Dockerfile .
```

### 运行 Hub 容器

```bash
docker run -d \
  --name rclone-backup-hub \
  -p 8080:8080 \
  -p 9090:9090 \
  rclone-backup-hub:latest
```

### 从容器中下载 Agent

```bash
# 下载 Linux amd64 版本
curl -L "http://localhost:8080/api/v1/agent/download?platform=linux&arch=amd64" -o agent

# 下载 macOS arm64 版本
curl -L "http://localhost:8080/api/v1/agent/download?platform=darwin&arch=arm64" -o agent

# 下载 Windows amd64 版本
curl -L "http://localhost:8080/api/v1/agent/download?platform=windows&arch=amd64" -o agent.exe
```

## 📚 相关文档

- [README.md](./README.md) - 项目主文档
- [agent/README.md](./agent/README.md) - Agent 详细说明

## 🆘 获取帮助

```bash
# 查看启动脚本帮助
./start-agent.sh --help

# 查看 Agent 帮助
./rclone-backup-agent --help
```

---

**提示：** 建议在正式使用前，先在测试环境中验证配置和连接。