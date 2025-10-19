#!/bin/bash

echo "========================================="
echo "修复健康检查问题"
echo "========================================="

# 步骤1: 查看Hub API的实际状态
echo ""
echo "步骤1: 检查Hub API容器内部状态"
echo "---------------------------------"
docker compose exec hub-api sh -c 'ps aux' 2>/dev/null | grep hub-api || {
    echo "✗ Hub API进程未运行！"
    echo ""
    echo "步骤2: 查看崩溃日志"
    echo "---------------------------------"
    docker compose logs --tail=50 hub-api
    
    echo ""
    echo "步骤3: 常见问题和解决方案"
    echo "---------------------------------"
    echo "1. 数据库连接失败:"
    echo "   - 检查 DB_HOST 是否设置为 'postgres'"
    echo "   - 确认数据库已启动: docker compose ps postgres"
    echo ""
    echo "2. 配置缺失:"
    echo "   - 检查 JWT_SECRET 和 ENCRYPTION_KEY 是否设置"
    echo "   - 查看 .env 文件"
    echo ""
    echo "3. 端口冲突:"
    echo "   - 检查8080端口是否被占用: lsof -i :8080"
    echo ""
    
    # 尝试手动启动Hub API查看错误
    echo "步骤4: 尝试手动启动Hub API"
    echo "---------------------------------"
    docker compose run --rm hub-api sh -c '/app/hub-api' 2>&1 | head -20
    
    exit 1
}

echo "✓ Hub API进程正在运行"

# 步骤2: 测试健康检查
echo ""
echo "步骤2: 测试健康检查端点"
echo "---------------------------------"

# 从容器内部测试
echo -n "从容器内部访问: "
docker compose exec hub-api wget -q -O - http://localhost:8080/health && echo "✓ 成功" || echo "✗ 失败"

# 从主机测试
echo -n "从主机访问: "
curl -f -s http://localhost:8080/health > /dev/null && echo "✓ 成功" || {
    echo "✗ 失败"
    
    # 获取容器IP并直接测试
    CONTAINER_IP=$(docker inspect -f '{{range .NetworkSettings.Networks}}{{.IPAddress}}{{end}}' v2-hub-api-1)
    if [ -n "$CONTAINER_IP" ]; then
        echo -n "直接访问容器IP ($CONTAINER_IP): "
        curl -f -s http://$CONTAINER_IP:8080/health > /dev/null && echo "✓ 成功" || echo "✗ 失败"
    fi
}

# 步骤3: 检查端口映射
echo ""
echo "步骤3: 检查端口映射"
echo "---------------------------------"
docker compose port hub-api 8080 || echo "端口映射未配置"

echo ""
echo "完成诊断。如果问题仍然存在，请运行："
echo "  docker compose down"
echo "  docker compose up hub-api"
echo "然后查看启动日志中的错误信息。"