# 📦 本地Agent部署指南

## 概述

本指南说明如何在Hub服务器上部署一个本地Agent，用于备份Hub自身的数据。这是一个**可选**功能，提供了自我备份的能力。

## 🏗️ 架构优势

通过使用Docker Compose的`profiles`功能，我们实现了：

1. **架构清晰** - Hub和Agent仍然是独立的服务
2. **部署灵活** - 可以选择是否启用本地Agent
3. **维护简单** - 使用统一的docker-compose文件管理
4. **向后兼容** - 不影响现有的独立部署方式

## 🚀 部署方案

### 方案A：标准部署（仅Hub）

这是最简单的部署方式，只启动管理中心。

```bash
# 1. 复制并编辑配置文件
cp .env.example .env
vim .env  # 设置必要的密钥

# 2. 启动Hub服务
make up

# 3. 访问Web UI
# http://localhost:3000
```

### 方案B：Hub + 本地Agent部署

如果您希望备份系统能够备份自身，请使用此方案。

#### 步骤1：首次启动Hub

```bash
# 确保配置文件已设置
make up
```

#### 步骤2：获取注册令牌

1. 登录Web UI：http://localhost:3000
2. 进入 **Agents** 页面
3. 点击 **生成注册令牌**
4. 复制生成的令牌

#### 步骤3：配置本地Agent

编辑`.env`文件，添加令牌：

```ini
# Optional: Local Agent Configuration
LOCAL_AGENT_REGISTRATION_TOKEN=你复制的令牌
LOCAL_AGENT_NAME=hub-backup-agent  # 可选：自定义名称
```

#### 步骤4：重启服务（含本地Agent）

```bash
# 停止当前服务
make down

# 启动Hub和本地Agent
make up-with-agent
```

#### 步骤5：验证部署

1. 访问Web UI
2. 进入 **Agents** 页面
3. 您应该看到一个名为`hub-backup-agent`的Agent已上线

## 📁 备份内容

本地Agent会自动挂载并可备份以下内容：

| 目录/卷 | 路径 | 内容 |
|---------|------|------|
| PostgreSQL数据 | `/backup/postgres` | 数据库原始文件 |
| 数据库备份 | `/backup/db-backups` | SQL备份文件 |
| Hub数据 | `/backup/hub-data` | Hub运行数据 |
| Redis数据 | `/backup/redis` | 缓存数据 |
| Hub配置 | `/backup/hub-config` | 配置文件 |
| 项目代码 | `/backup/project` | 完整源代码 |

## 🔧 高级配置

### 自动数据库备份

启用定期数据库备份服务：

```bash
# 启动自动备份（每24小时）
make backup-auto

# 自定义备份间隔（编辑.env）
DB_BACKUP_INTERVAL=3600  # 每小时备份
```

### 调整Agent资源限制

编辑`.env`文件：

```ini
# Rclone资源限制
RCLONE_CPU_LIMIT=1.0      # 最多使用1个CPU核心
RCLONE_MEMORY_LIMIT=512M  # 最多使用512MB内存
```

### 查看本地Agent日志

```bash
# 查看本地Agent日志
make local-agent-logs

# 查看本地Agent状态
make local-agent-status
```

## 📊 创建备份任务

在Web UI中为本地Agent创建备份任务：

1. **创建Remote**
   - 类型：S3/GCS/Azure等
   - 配置您的云存储凭证

2. **创建Task**
   - 名称：`Hub PostgreSQL Backup`
   - 源路径：`/backup/db-backups`
   - 目标路径：`your-bucket/hub-backups/db`
   - 计划：`0 3 * * *`（每天凌晨3点）
   - 分配给：`hub-backup-agent`

3. **创建其他任务**
   - Redis备份：`/backup/redis`
   - Hub数据备份：`/backup/hub-data`
   - 配置备份：`/backup/hub-config`

## 🔄 更新流程

### 更新Hub（不含Agent）

```bash
docker-compose pull
make up
```

### 更新Hub（含本地Agent）

```bash
docker-compose --profile local-agent pull
make up-with-agent
```

## ⚠️ 注意事项

1. **首次部署**：必须先启动Hub获取注册令牌
2. **令牌安全**：注册令牌仅使用一次，注册后Agent会保存凭证
3. **网络隔离**：本地Agent使用内部网络通信，不需要暴露额外端口
4. **备份策略**：建议将不同类型的数据备份到不同的目标路径

## 🐛 故障排除

### 本地Agent无法注册

```bash
# 检查令牌是否正确
grep LOCAL_AGENT_REGISTRATION_TOKEN .env

# 查看Agent日志
make local-agent-logs

# 重新生成令牌并更新.env
```

### 本地Agent离线

```bash
# 检查服务状态
docker-compose ps

# 重启本地Agent
docker-compose restart local-agent local-rclone-sidecar

# 查看健康检查
docker inspect rclone-backup-local-agent | grep -A 10 Health
```

### 备份任务失败

```bash
# 检查挂载路径
docker-compose exec local-agent ls -la /backup/

# 检查Sidecar连接
docker-compose exec local-agent curl http://local-rclone-sidecar:5572/rc/noop

# 查看任务执行日志（在Web UI的Executions页面）
```

## 📈 最佳实践

1. **定期测试恢复**：定期从备份恢复数据以验证备份有效性
2. **监控备份状态**：在Dashboard查看备份任务的成功率
3. **多地备份**：配置多个Remote，实现异地容灾
4. **增量备份**：使用`--checksum`参数实现增量备份
5. **加密传输**：为Remote配置加密，保护数据安全

## 🎯 总结

通过本地Agent，您可以：

✅ **自我备份** - Hub能够备份自身数据  
✅ **统一管理** - 在同一界面管理所有备份  
✅ **灵活部署** - 可选择是否启用本地Agent  
✅ **零额外成本** - 复用现有的Hub服务器资源  

这个方案既保持了架构的优雅性，又提供了极致的便利性！