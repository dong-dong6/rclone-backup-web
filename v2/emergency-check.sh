#!/bin/bash

echo "紧急检查 - Hub API状态"
echo "======================"

# 1. 容器是否存在
echo ""
echo "1. 检查容器是否存在:"
docker ps -a | grep hub-api || echo "  ✗ 没有找到hub-api容器"

# 2. 查看容器日志（最重要）
echo ""
echo "2. Hub API容器日志（最后50行）:"
echo "--------------------------------"
docker compose logs --tail=50 hub-api 2>&1 || docker logs v2-hub-api-1 2>&1 | tail -50

# 3. 检查容器是否真的在运行
echo ""
echo "3. 容器内进程:"
docker compose exec hub-api ps aux 2>&1 || echo "  ✗ 无法连接到容器（可能已崩溃）"

# 4. 查看容器退出代码
echo ""
echo "4. 容器状态详情:"
docker inspect v2-hub-api-1 --format='{{.State.Status}} - ExitCode: {{.State.ExitCode}} - Error: {{.State.Error}}' 2>/dev/null || echo "无法获取状态"

# 5. 尝试手动运行看错误
echo ""
echo "5. 尝试手动启动Hub API查看错误:"
echo "--------------------------------"
docker compose run --rm hub-api /app/hub-api 2>&1 | head -30 || echo "手动启动失败"

# 6. 检查数据库连接
echo ""
echo "6. 测试数据库连接:"
docker compose exec postgres psql -U ${DB_USER:-rclone} -d ${DB_NAME:-rclone_backup} -c "SELECT 1;" 2>&1 || echo "  ✗ 数据库连接失败"

# 7. 检查环境变量
echo ""
echo "7. 检查关键环境变量:"
if [ -f .env ]; then
    echo "  JWT_SECRET: $(grep JWT_SECRET .env | head -1)"
    echo "  ENCRYPTION_KEY: $(grep ENCRYPTION_KEY .env | head -1)"
    echo "  DB_HOST: $(grep DB_HOST .env | head -1 || echo 'DB_HOST=未设置（使用默认值）')"
    echo "  HUB_PORT: $(grep HUB_PORT .env | head -1 || echo 'HUB_PORT=未设置（使用默认8080）')"
else
    echo "  ✗ 未找到.env文件"
fi