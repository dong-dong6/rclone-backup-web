# 版本兼容性指南

## 🎯 当前版本要求

### Go语言版本
- **Hub**: Go 1.23+ (go.mod要求)
- **Agent**: Go 1.23+ (统一版本)
- **Docker镜像**: `golang:1.23-alpine`

### Node.js版本
- **Web UI**: Node.js 20 LTS
- **Docker镜像**: `node:20-alpine`

### 其他组件版本
- **PostgreSQL**: 15-alpine
- **Redis**: 7-alpine
- **Nginx**: 1.25-alpine
- **Rclone**: latest

## 🐛 常见版本问题

### 1. Go版本不匹配

**错误信息：**
```
go: go.mod requires go >= 1.23 (running go 1.21.13; GOTOOLCHAIN=local)
```

**原因：**
- go.mod文件要求Go 1.23+
- Dockerfile使用的是Go 1.21

**解决方案：**
更新Dockerfile中的Go版本：
```dockerfile
# 修改前
FROM golang:1.21-alpine AS builder

# 修改后
FROM golang:1.23-alpine AS builder
```

### 2. Go版本检测

**查看go.mod要求的版本：**
```bash
# Hub
grep "^go " hub/go.mod

# Agent
grep "^go " agent/go.mod
```

**查看Dockerfile使用的版本：**
```bash
# Hub
grep "FROM golang" hub/Dockerfile

# Agent
grep "FROM golang" agent/Dockerfile
```

## 📝 版本同步建议

### 保持版本一致性

1. **开发环境与构建环境同步**
   - 本地Go版本应与Dockerfile中的版本一致
   - 使用go.mod中指定的版本进行开发

2. **定期更新**
   - Go: 使用最新的稳定版本
   - Node.js: 使用LTS版本
   - 数据库: 使用稳定版本，避免频繁升级

3. **版本固定策略**
   ```dockerfile
   # 推荐：使用特定版本
   FROM golang:1.23-alpine
   
   # 不推荐：使用latest
   FROM golang:latest
   ```

## 🔧 自动版本检查脚本

```bash
#!/bin/bash
# check-versions.sh

echo "检查版本兼容性..."

# 检查Hub Go版本
HUB_GO_REQUIRED=$(grep "^go " hub/go.mod | awk '{print $2}')
HUB_DOCKER_GO=$(grep "FROM golang" hub/Dockerfile | head -1 | sed 's/.*golang:\([0-9.]*\).*/\1/')

if [ "$HUB_GO_REQUIRED" != "$HUB_DOCKER_GO" ]; then
    echo "⚠️  Hub版本不匹配："
    echo "   go.mod要求: $HUB_GO_REQUIRED"
    echo "   Dockerfile: $HUB_DOCKER_GO"
else
    echo "✅ Hub Go版本匹配: $HUB_GO_REQUIRED"
fi

# 检查Agent Go版本
AGENT_GO_REQUIRED=$(grep "^go " agent/go.mod | awk '{print $2}')
AGENT_DOCKER_GO=$(grep "FROM golang" agent/Dockerfile | head -1 | sed 's/.*golang:\([0-9.]*\).*/\1/')

if [ "$AGENT_GO_REQUIRED" != "$AGENT_DOCKER_GO" ]; then
    echo "⚠️  Agent版本不匹配："
    echo "   go.mod要求: $AGENT_GO_REQUIRED"
    echo "   Dockerfile: $AGENT_DOCKER_GO"
else
    echo "✅ Agent Go版本匹配: $AGENT_GO_REQUIRED"
fi
```

## 🚀 版本升级指南

### 升级Go版本

1. **更新go.mod**
   ```bash
   cd hub
   go mod edit -go=1.23
   go mod tidy
   ```

2. **更新Dockerfile**
   ```dockerfile
   FROM golang:1.23-alpine AS builder
   ```

3. **测试构建**
   ```bash
   docker compose build --no-cache hub-api
   ```

### 升级Node.js版本

1. **更新package.json**
   ```json
   {
     "engines": {
       "node": ">=20.0.0"
     }
   }
   ```

2. **更新Dockerfile**
   ```dockerfile
   FROM node:20-alpine AS builder
   ```

## 📊 版本兼容性矩阵

| 组件 | 最低版本 | 推荐版本 | 最高测试版本 |
|------|---------|---------|-------------|
| Go | 1.23 | 1.23 | 1.23 |
| Node.js | 18.0 | 20 LTS | 20 |
| PostgreSQL | 14 | 15 | 15 |
| Redis | 6 | 7 | 7 |
| Docker | 20.10 | 24.0 | 24.0 |
| Docker Compose | 2.0 | 2.20 | 2.20 |

## 💡 最佳实践

1. **使用Alpine镜像**
   - 体积小，安全性高
   - 所有组件统一使用Alpine

2. **多阶段构建**
   - 编译阶段：使用完整的开发镜像
   - 运行阶段：使用精简的运行时镜像

3. **版本锁定**
   - 生产环境使用固定版本
   - 开发环境可以使用latest进行测试

4. **定期更新**
   - 每季度检查一次版本更新
   - 安全补丁立即更新