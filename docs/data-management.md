# 数据管理指南

## 📂 数据目录结构

所有持久化数据都透明地存储在 `./data` 目录中，而不是Docker的隐藏卷中：

```
v2/
├── data/                      # 所有持久化数据
│   ├── postgres/             # PostgreSQL数据库文件
│   │   ├── base/
│   │   ├── global/
│   │   └── pg_wal/
│   ├── redis/                # Redis持久化数据
│   │   └── appendonly.aof
│   ├── hub/                  # Hub服务数据
│   │   ├── config/          # 配置文件
│   │   ├── data/            # 运行时数据
│   │   └── logs/            # 日志文件
│   ├── agent/               # Agent数据（如果启用）
│   │   ├── cache/          # 任务缓存
│   │   ├── rclone-config/  # Rclone配置
│   │   └── logs/           # Agent日志
│   └── backups/             # 数据库自动备份
│       └── backup-*.sql.gz
├── docker-compose.yml
└── .env
```

## 🎯 为什么使用绑定挂载？

### 传统Docker卷的问题

```yaml
# 传统方式 - 不透明
volumes:
  postgres_data:  # 数据在哪？/var/lib/docker/volumes/...
```

- ❌ 数据位置不透明
- ❌ 需要root权限访问
- ❌ 备份困难
- ❌ 迁移复杂

### 绑定挂载的优势

```yaml
# 新方式 - 完全透明
volumes:
  - ./data/postgres:/var/lib/postgresql/data
```

- ✅ 数据位置清晰可见
- ✅ 普通用户可访问
- ✅ 备份简单（tar/rsync）
- ✅ 迁移方便

## 🔧 数据操作

### 1. 查看数据大小

```bash
# 查看总大小
du -sh ./data

# 查看各组件大小
du -sh ./data/*

# 详细信息
du -h --max-depth=2 ./data
```

### 2. 备份数据

```bash
# 方法1：使用部署脚本
./deploy-v2.sh backup

# 方法2：手动备份
tar czf backup-$(date +%Y%m%d).tar.gz ./data

# 方法3：增量备份（使用rsync）
rsync -avz ./data/ /backup/location/
```

### 3. 恢复数据

```bash
# 方法1：使用部署脚本
./deploy-v2.sh restore backup-20241018.tar.gz

# 方法2：手动恢复
tar xzf backup-20241018.tar.gz
```

### 4. 迁移到新服务器

```bash
# 在旧服务器
tar czf migration.tar.gz ./v2

# 在新服务器
tar xzf migration.tar.gz
cd v2
./deploy-v2.sh hub
```

## 🛡️ 安全清理机制

新的部署脚本提供了智能的数据清理选项：

```bash
./deploy-v2.sh clean
```

提供4种选项：
1. **保留数据**（默认）- 不做任何改动
2. **备份后清理** - 自动备份到带时间戳的目录
3. **直接清理** - 需要输入'DELETE'确认
4. **取消** - 退出脚本

### 自动备份

如果选择"备份后清理"，旧数据会被移动到：
```
data.backup.20241018-123456/
```

可以随时恢复：
```bash
mv data.backup.20241018-123456 data
```

## 📊 数据库管理

### 查看数据库

```bash
# 进入数据库
docker compose -f docker-compose.prod.yml exec postgres psql -U rclone rclone_backup

# 查看表
\dt

# 查看数据量
SELECT COUNT(*) FROM agents;
SELECT COUNT(*) FROM backup_tasks;
SELECT COUNT(*) FROM task_executions;
```

### 手动备份数据库

```bash
# 备份
docker compose -f docker-compose.prod.yml exec postgres \
  pg_dump -U rclone rclone_backup | gzip > db-backup.sql.gz

# 恢复
gunzip < db-backup.sql.gz | \
  docker compose -f docker-compose.prod.yml exec -T postgres \
  psql -U rclone rclone_backup
```

## 🔄 自动备份服务

启用自动备份：

```bash
# 启动自动备份（每24小时）
docker compose -f docker-compose.prod.yml --profile db-backup up -d

# 查看备份
ls -lh ./data/backups/

# 备份保留30天
```

## 🚨 重要提示

### 权限问题

某些服务需要特定权限：

```bash
# PostgreSQL需要700权限
chmod 700 ./data/postgres

# 如果遇到权限问题
sudo chown -R $(whoami):$(whoami) ./data
```

### 磁盘空间

监控磁盘使用：

```bash
# 检查可用空间
df -h .

# 清理旧备份
find ./data/backups -name "*.sql.gz" -mtime +30 -delete

# 清理日志
find ./data/hub/logs -name "*.log" -mtime +7 -delete
```

### 数据安全

1. **定期备份**：设置cron任务
   ```bash
   0 2 * * * cd /path/to/v2 && ./deploy-v2.sh backup
   ```

2. **异地备份**：使用rclone同步到云存储
   ```bash
   rclone sync ./data remote:backup/rclone-backup-web
   ```

3. **权限控制**：
   ```bash
   # 限制数据目录访问
   chmod 750 ./data
   chmod 700 ./data/postgres
   ```

## 📈 监控

### 数据增长

创建监控脚本：

```bash
#!/bin/bash
# monitor-data.sh

echo "=== 数据使用报告 ==="
echo "时间: $(date)"
echo ""
echo "总大小: $(du -sh ./data | cut -f1)"
echo ""
echo "详细:"
for dir in postgres redis hub agent backups; do
    if [ -d "./data/$dir" ]; then
        size=$(du -sh ./data/$dir 2>/dev/null | cut -f1)
        count=$(find ./data/$dir -type f 2>/dev/null | wc -l)
        echo "  $dir: $size ($count 文件)"
    fi
done
```

### 健康检查

```bash
# 检查所有服务
docker compose -f docker-compose.prod.yml ps

# 检查数据目录
ls -la ./data/

# 检查最近备份
ls -lt ./data/backups/ | head -5
```

## 🎯 最佳实践

1. **使用生产配置文件**
   ```bash
   # 使用 docker-compose.prod.yml 而不是默认的
   docker compose -f docker-compose.prod.yml up -d
   ```

2. **环境分离**
   ```bash
   # 开发环境
   cp docker-compose.yml docker-compose.dev.yml
   
   # 生产环境
   cp docker-compose.prod.yml docker-compose.yml
   ```

3. **版本控制**
   ```
   .gitignore:
   data/
   *.backup/
   *.tar.gz
   .env
   ```

4. **监控脚本**
   ```bash
   # 添加到crontab
   */5 * * * * /path/to/check-health.sh
   0 * * * * /path/to/monitor-data.sh >> /var/log/rclone-backup.log
   ```

通过这种透明的数据管理方式，您可以完全掌控应用的数据，轻松进行备份、恢复和迁移操作。