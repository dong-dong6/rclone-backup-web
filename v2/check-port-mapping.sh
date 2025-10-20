#!/bin/bash

echo "端口映射诊断"
echo "============="

# 1. 检查端口映射配置
echo ""
echo "1. Docker端口映射:"
docker port v2-hub-api-1

# 2. 检查容器内部监听
echo ""
echo "2. 容器内部监听端口:"
docker exec v2-hub-api-1 netstat -tlnp 2>/dev/null || docker exec v2-hub-api-1 ss -tlnp 2>/dev/null || echo "无法获取"

# 3. 检查主机端口监听
echo ""
echo "3. 主机端口监听:"
netstat -tlnp 2>/dev/null | grep -E "48080|8080" || ss -tlnp 2>/dev/null | grep -E "48080|8080" || echo "未找到监听"

# 4. 测试容器内部访问
echo ""
echo "4. 容器内部健康检查:"
docker exec v2-hub-api-1 curl -s http://localhost:48080/health || docker exec v2-hub-api-1 wget -q -O - http://localhost:48080/health || echo "失败"

# 5. 测试主机访问
echo ""
echo "5. 主机访问测试:"
for port in 8080 48080; do
    echo -n "  localhost:$port: "
    timeout 2 curl -s http://localhost:$port/health && echo " ✓" || echo " ✗"
done

# 6. 检查iptables
echo ""
echo "6. 防火墙规则（Docker相关）:"
sudo iptables -L DOCKER -n 2>/dev/null | grep -E "48080|8080" || echo "无法获取或无相关规则"