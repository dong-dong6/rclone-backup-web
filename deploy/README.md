# 📦 Rclone Backup Web V2.0 - 部署指南

## 🚀 快速开始

### 1. 准备环境配置

```bash
# 复制环境配置模板
cp .env.example .env

# 生成安全密钥
echo "JWT_SECRET=$(openssl rand -hex 32)" >> .env
echo "ENCRYPTION_KEY=$(openssl rand -hex 16)" >> .env
echo "DB_PASSWORD=$(openssl rand -base64 20)" >> .env
```

### 2. 启动Hub（中央节点）

```bash
# 拉取/构建镜像
docker-compose build

# 启动所有服务
docker-compose up -d

# 查看服务状态
docker-compose ps

# 查看日志
docker-compose logs -f hub-api
```

服务将在以下端口启动：
- **Hub API**: http://localhost:8080
- **Web UI**: http://localhost:3000
- **Metrics**: http://localhost:9090/metrics

### 3. 部署Agent（执行节点）

在每个需要备份的服务器上：

```bash
# 设置Hub连接信息
export HUB_URL=http://your-hub-server:8080

# 方式1：使用注册令牌（推荐）
export REGISTRATION_TOKEN=your-token-from-web-ui

# 方式2：使用已有凭证
export AGENT_ID=your-agent-id
export AGENT_API_KEY=your-api-key

# 启动Agent
docker-compose -f docker-compose.agent.yml up -d
```

## 🔧 配置说明

### Hub配置

| 环境变量 | 说明 | 默认值 | 必需 |
|---------|------|--------|------|
| `DB_PASSWORD` | PostgreSQL密码 | - | ✅ |
| `JWT_SECRET` | JWT签名密钥（64字符） | - | ✅ |
| `ENCRYPTION_KEY` | AES-256加密密钥（32字符） | - | ✅ |
| `HUB_PORT` | Hub API端口 | 8080 | ❌ |
| `WEB_PORT` | Web UI端口 | 3000 | ❌ |
| `LOG_LEVEL` | 日志级别 | info | ❌ |

### Agent配置

| 环境变量 | 说明 | 默认值 | 必需 |
|---------|------|--------|------|
| `HUB_URL` | Hub地址 | - | ✅ |
| `REGISTRATION_TOKEN` | 注册令牌（首次） | - | ⚠️ |
| `AGENT_ID` | Agent ID（已注册） | - | ⚠️ |
| `AGENT_API_KEY` | API密钥（已注册） | - | ⚠️ |
| `HEARTBEAT_INTERVAL` | 心跳间隔 | 30s | ❌ |
| `ENABLE_LOCAL_FALLBACK` | 启用本地回退 | true | ❌ |

## 🏗️ 生产部署

### 1. 使用Docker Swarm

```bash
# 初始化Swarm
docker swarm init

# 部署Stack
docker stack deploy -c docker-compose.yml rclone-backup

# 扩展服务
docker service scale rclone-backup_hub-api=3
```

### 2. 使用Kubernetes

```yaml
# k8s-deployment.yaml
apiVersion: apps/v1
kind: Deployment
metadata:
  name: rclone-backup-hub
spec:
  replicas: 3
  selector:
    matchLabels:
      app: rclone-backup-hub
  template:
    metadata:
      labels:
        app: rclone-backup-hub
    spec:
      containers:
      - name: hub-api
        image: rclone-backup-hub:latest
        ports:
        - containerPort: 8080
        env:
        - name: DATABASE_URL
          valueFrom:
            secretKeyRef:
              name: rclone-secrets
              key: database-url
```

### 3. 反向代理配置（可选）

如果需要使用Nginx/Traefik等反向代理：

```nginx
# nginx.conf
upstream rclone_hub {
    server localhost:8080;
}

upstream rclone_web {
    server localhost:3000;
}

server {
    listen 443 ssl;
    server_name backup.example.com;
    
    # API路由
    location /api {
        proxy_pass http://rclone_hub;
        proxy_set_header Host $host;
        proxy_set_header X-Real-IP $remote_addr;
    }
    
    # SSE事件流
    location /events {
        proxy_pass http://rclone_hub/events;
        proxy_buffering off;
        proxy_read_timeout 86400s;
    }
    
    # Web UI
    location / {
        proxy_pass http://rclone_web;
    }
}
```

## 📊 监控与维护

### 健康检查

```bash
# 检查Hub健康状态
curl http://localhost:8080/health

# 检查数据库连接
docker-compose exec postgres pg_isready

# 查看服务资源使用
docker stats
```

### 备份数据库

```bash
# 创建备份
docker-compose exec postgres pg_dump -U rclone rclone_backup | gzip > backup-$(date +%Y%m%d).sql.gz

# 恢复备份
gunzip < backup-20240101.sql.gz | docker-compose exec -T postgres psql -U rclone rclone_backup
```

### 日志管理

```bash
# 查看实时日志
docker-compose logs -f

# 导出日志
docker-compose logs > rclone-backup-$(date +%Y%m%d).log

# 清理旧日志
docker-compose exec hub-api sh -c 'find /logs -name "*.log" -mtime +30 -delete'
```

## 🔒 安全建议

### 1. 使用强密码

```bash
# 生成强密码
openssl rand -base64 32
```

### 2. 限制端口访问

```bash
# 只允许特定IP访问
iptables -A INPUT -p tcp --dport 8080 -s 10.0.0.0/8 -j ACCEPT
iptables -A INPUT -p tcp --dport 8080 -j DROP
```

### 3. 定期更新

```bash
# 更新镜像
docker-compose pull
docker-compose up -d
```

## 🐛 故障排除

### Hub无法启动

```bash
# 检查数据库连接
docker-compose logs postgres

# 验证环境变量
docker-compose config

# 重置数据库（谨慎！）
docker-compose down -v
docker-compose up -d
```

### Agent无法连接Hub

```bash
# 检查网络连接
docker-compose exec agent curl http://hub:8080/health

# 查看Agent日志
docker-compose -f docker-compose.agent.yml logs agent

# 重新注册Agent
docker-compose -f docker-compose.agent.yml down
rm -rf agent_data
docker-compose -f docker-compose.agent.yml up -d
```

### 性能问题

```bash
# 增加资源限制
docker-compose down
# 编辑docker-compose.yml添加：
# deploy:
#   resources:
#     limits:
#       cpus: '4'
#       memory: 4G
docker-compose up -d
```

## 📝 常用命令

```bash
# 启动服务
make up

# 停止服务
make down

# 查看日志
make logs

# 进入容器
make shell service=hub-api

# 运行测试
make test

# 清理所有数据（危险！）
make clean
```

## 🆘 获取帮助

- 查看文档：`/docs`目录
- 提交Issue：GitHub Issues
- 查看日志：`docker-compose logs`
- 健康检查：`curl http://localhost:8080/health`