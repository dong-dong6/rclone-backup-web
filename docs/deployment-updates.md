# 部署系统更新说明

## 📋 更新内容

### 1. Docker Compose兼容性

系统现已完全兼容新版Docker的内置`docker compose`命令：

- **自动检测**：脚本会自动检测可用的Docker Compose版本
  - 优先使用新版：`docker compose`（Docker 20.10+内置）
  - 回退到旧版：`docker-compose`（独立安装）

- **Makefile更新**：所有Make命令已更新为动态检测并使用正确的命令格式

```makefile
# 自动检测Docker Compose命令
DOCKER_COMPOSE := $(shell command -v docker-compose 2> /dev/null)
ifeq ($(DOCKER_COMPOSE),)
    DOCKER_COMPOSE := docker compose
endif
```

### 2. 交互式配置

新的`deploy.sh`脚本提供完全交互式的配置体验：

```bash
./deploy.sh
```

首次运行时，脚本会：
1. 检查系统依赖（Docker、Docker Compose）
2. 交互式询问配置信息
3. 自动生成安全密钥
4. 创建`.env`配置文件
5. 构建并启动服务

### 3. 配置生成流程

当系统检测到没有`.env`文件时：

```
[WARNING] 未找到 .env 文件

请选择配置方式：
  1) 交互式配置（推荐）
  2) 使用默认配置
  3) 退出

[INPUT] 请选择 [1-3]: 
```

选择交互式配置后，会依次询问：
- 数据库名称（默认：rclone_backup）
- 数据库用户名（默认：rclone）
- 数据库密码（可自动生成）
- Hub API端口（默认：8080）
- Web UI端口（默认：3000）

### 4. 本地镜像构建

所有Docker镜像现在都在本地构建，无需依赖外部镜像仓库：

```yaml
# docker-compose.yml
services:
  hub-api:
    build:
      context: ./hub
      dockerfile: Dockerfile
    image: rclone-backup-hub:${VERSION:-latest}
    
  web-ui:
    build:
      context: ./hub/web
      dockerfile: Dockerfile
    image: rclone-backup-web:${VERSION:-latest}
    
  local-agent:
    build:
      context: ./agent
      dockerfile: Dockerfile
    image: rclone-backup-agent:${VERSION:-latest}
```

### 5. 命令对比

| 功能 | 旧版本 | 新版本 |
|-----|--------|--------|
| 启动Hub | `docker-compose up -d` | `./deploy.sh hub` 或 `make up` |
| 启动Hub+Agent | 手动配置 | `./deploy.sh hub-with-agent` |
| 构建镜像 | `docker-compose build` | 自动在部署时构建 |
| 配置环境 | 手动编辑.env | 交互式生成 |
| Docker Compose | 仅支持docker-compose | 自动检测并适配 |

### 6. 安全增强

自动生成的安全密钥：
- **JWT_SECRET**: 32字节随机十六进制
- **ENCRYPTION_KEY**: 16字节随机十六进制  
- **DB_PASSWORD**: 20字符随机字符串

```bash
JWT_SECRET=$(openssl rand -hex 32)
ENCRYPTION_KEY=$(openssl rand -hex 16)
DB_PASSWORD=$(openssl rand -base64 20 | tr -d "=+/" | cut -c1-20)
```

### 7. 健康检查

部署脚本包含自动健康检查：

```bash
# 等待服务启动并检查健康状态
while [ $attempt -lt $max_attempts ]; do
    if curl -f http://localhost:${HUB_PORT:-8080}/health &> /dev/null; then
        print_success "Hub服务部署成功！"
        break
    fi
    sleep 2
done
```

## 🔧 故障排除

### Docker Compose未找到

如果看到"未找到 Docker Compose"错误：

1. **新版Docker（推荐）**：
   ```bash
   # 检查Docker版本
   docker --version
   # 如果版本 < 20.10，请升级Docker
   ```

2. **独立安装Docker Compose**：
   ```bash
   # Linux
   sudo curl -L "https://github.com/docker/compose/releases/latest/download/docker-compose-$(uname -s)-$(uname -m)" -o /usr/local/bin/docker-compose
   sudo chmod +x /usr/local/bin/docker-compose
   ```

### 端口冲突

如果默认端口已被占用，可在配置时指定其他端口：
- Hub API: 8080（可改）
- Web UI: 3000（可改）  
- Metrics: 9090（可改）

### macOS兼容性

脚本已针对macOS进行优化：
```bash
if [[ "$OSTYPE" == "darwin"* ]]; then
    # macOS特定的sed命令格式
    sed -i '' "s/OLD/NEW/" .env
else
    # Linux
    sed -i "s/OLD/NEW/" .env
fi
```

## 📝 总结

这次更新大幅简化了部署流程：
- 无需手动创建和编辑配置文件
- 自动适配不同版本的Docker Compose
- 本地构建所有镜像，无外部依赖
- 交互式配置，用户友好
- 自动健康检查，确保部署成功

现在，从零开始部署整个系统只需要一个命令：
```bash
./deploy.sh
```