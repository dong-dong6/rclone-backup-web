# Agent 部署指南

本文档说明如何在开发阶段构建和部署 Rclone Backup Agent 二进制文件。

## 概述

在开发阶段，我们使用本地构建的二进制文件来测试 agent 功能，而不是使用 Docker 镜像。这提供了更快的迭代速度和更简单的调试过程。

## 文件结构

```
v2/
├── agent/                    # Agent 源代码
│   ├── main_standalone.go   # 独立运行的主程序
│   ├── Makefile            # 构建配置
│   └── ...
├── hub/                     # Hub 服务
│   ├── static/binaries/    # 二进制文件存储目录
│   │   ├── rclone-backup-agent
│   │   └── rclone-backup-agent-latest -> rclone-backup-agent
│   └── ...
├── deploy-agent.sh         # Agent 部署脚本
├── test-agent-deployment.sh # 测试脚本
└── AGENT-DEPLOYMENT.md     # 本文档
```

## 快速开始

### 1. 部署 Agent 二进制文件

```bash
cd /workspace/v2
./deploy-agent.sh
```

这个脚本会：
- 构建 agent 二进制文件
- 将二进制文件复制到 hub 的静态文件目录
- 更新 DownloadAgent API 以提供实际文件
- 测试构建是否成功

### 2. 测试部署

```bash
cd /workspace/v2
./test-agent-deployment.sh
```

### 3. 启动 Hub 服务

```bash
cd /workspace/v2/hub
go run main.go
```

### 4. 测试下载功能

在另一个终端中：

```bash
# 下载 agent 二进制文件
curl -L http://localhost:8080/api/v1/agent/download -o test-agent
chmod +x test-agent

# 测试 agent 功能
./test-agent --help
```

## 详细说明

### deploy-agent.sh 脚本功能

1. **环境检查**: 验证 Go 环境是否可用
2. **依赖管理**: 运行 `go mod tidy` 和 `go mod download`
3. **构建二进制**: 使用 `go build` 构建当前平台的二进制文件
4. **文件部署**: 将二进制文件复制到 `hub/static/binaries/` 目录
5. **API 更新**: 修改 `DownloadAgent` 函数以提供实际文件
6. **构建测试**: 验证 Hub 服务可以正常构建

### 构建配置

- **版本号**: 使用时间戳格式 `dev-YYYYMMDD-HHMMSS`
- **构建标志**: 包含版本、构建时间和 Git 提交信息
- **优化**: 使用 `-s -w` 标志减小二进制文件大小
- **平台**: 当前只构建当前平台，支持 Linux/AMD64

### API 端点

- **下载端点**: `GET /api/v1/agent/download`
- **文件路径**: `./static/binaries/rclone-backup-agent`
- **内容类型**: `application/octet-stream`
- **文件名**: `rclone-backup-agent`

## 开发工作流

### 修改 Agent 代码后

1. 运行部署脚本：
   ```bash
   ./deploy-agent.sh
   ```

2. 重启 Hub 服务（如果需要）：
   ```bash
   cd /workspace/v2/hub
   go run main.go
   ```

3. 测试新功能：
   ```bash
   curl -L http://localhost:8080/api/v1/agent/download -o test-agent
   chmod +x test-agent
   ./test-agent --help
   ```

### 清理构建文件

```bash
# 清理 agent 目录中的临时文件
cd /workspace/v2/agent
make clean

# 清理 hub 中的二进制文件
rm -rf /workspace/v2/hub/static/binaries/*
```

## 故障排除

### 构建失败

1. **Go 模块问题**:
   ```bash
   cd /workspace/v2/agent
   go mod tidy
   go mod download
   ```

2. **依赖冲突**:
   ```bash
   go mod graph | grep conflict
   ```

3. **未使用的导入**:
   检查编译错误，移除未使用的导入

### 下载失败

1. **文件不存在**:
   ```bash
   ls -la /workspace/v2/hub/static/binaries/
   ```

2. **权限问题**:
   ```bash
   chmod +x /workspace/v2/hub/static/binaries/rclone-backup-agent
   ```

3. **API 问题**:
   检查 `DownloadAgent` 函数是否正确更新

### Hub 构建失败

1. **导入问题**:
   检查 `admin_handlers.go` 中的导入是否正确

2. **语法错误**:
   使用 `go fmt` 和 `go vet` 检查代码

## 生产部署

在生产环境中，建议：

1. 使用 CI/CD 管道自动构建
2. 构建多平台二进制文件
3. 使用版本标签而不是时间戳
4. 添加数字签名验证
5. 使用 CDN 分发二进制文件

## 相关文件

- `agent/Makefile`: Agent 构建配置
- `agent/build.sh`: 多平台构建脚本
- `hub/api/admin_handlers.go`: DownloadAgent API 实现
- `hub/web/src/pages/Agents.tsx`: 前端注册界面

## 支持

如果遇到问题，请检查：

1. Go 版本兼容性
2. 文件权限设置
3. 网络连接状态
4. 磁盘空间充足

更多信息请参考项目主 README 文件。