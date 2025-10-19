#!/bin/bash

# 紧急健康检查调试脚本

echo "========================================="
echo "健康检查调试"
echo "========================================="

# 1. 检查容器状态
echo ""
echo "1. 容器状态:"
docker compose ps

# 2. 检查hub-api是否在运行
echo ""
echo "2. Hub API容器详情:"
docker ps | grep hub-api

# 3. 测试不同的健康检查方式
echo ""
echo "3. 健康检查测试:"

echo -n "  a) curl localhost:8080/health: "
curl -s -o /dev/null -w "%{http_code}\n" http://localhost:8080/health || echo "失败"

echo -n "  b) curl 127.0.0.1:8080/health: "
curl -s -o /dev/null -w "%{http_code}\n" http://127.0.0.1:8080/health || echo "失败"

echo -n "  c) wget localhost:8080/health: "
wget -q -O /dev/null http://localhost:8080/health 2>&1 && echo "200" || echo "失败"

# 4. 检查端口监听
echo ""
echo "4. 端口监听状态:"
netstat -tlnp 2>/dev/null | grep 8080 || ss -tlnp | grep 8080 || echo "端口8080未监听"

# 5. 检查Hub API日志
echo ""
echo "5. Hub API最近日志:"
docker compose logs --tail=20 hub-api

# 6. 尝试从容器内部访问
echo ""
echo "6. 从Hub API容器内部测试:"
docker compose exec hub-api wget -q -O - http://localhost:8080/health || echo "容器内部访问失败"

# 7. 检查环境变量
echo ""
echo "7. Hub API环境变量:"
docker compose exec hub-api sh -c 'echo "PORT=$PORT"'

# 8. 检查网络
echo ""
echo "8. Docker网络:"
docker network ls | grep v2

# 9. 容器IP地址
echo ""
echo "9. Hub API容器IP:"
docker inspect v2-hub-api-1 2>/dev/null | grep -A 2 '"Networks"' | grep '"IPAddress"' || echo "无法获取IP"

# 10. 直接测试容器IP
CONTAINER_IP=$(docker inspect -f '{{range .NetworkSettings.Networks}}{{.IPAddress}}{{end}}' v2-hub-api-1 2>/dev/null)
if [ -n "$CONTAINER_IP" ]; then
    echo ""
    echo "10. 直接访问容器IP ($CONTAINER_IP):"
    curl -s -o /dev/null -w "%{http_code}\n" http://$CONTAINER_IP:8080/health || echo "失败"
fi