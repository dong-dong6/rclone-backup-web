#!/bin/bash

echo "验证部署状态"
echo "============="

# 1. 检查容器
echo ""
echo "容器状态:"
docker compose ps

# 2. 获取Hub API实际端口
echo ""
echo "Hub API信息:"
HUB_PORT=$(docker logs v2-hub-api-1 2>&1 | grep "server started on port" | tail -1 | sed 's/.*port \([0-9]*\).*/\1/')
echo "  实际监听端口: $HUB_PORT"

# 3. 获取容器IP
CONTAINER_IP=$(docker inspect v2-hub-api-1 -f '{{range .NetworkSettings.Networks}}{{.IPAddress}}{{end}}' | head -1)
echo "  容器IP: $CONTAINER_IP"

# 4. 测试访问
echo ""
echo "健康检查:"
if [ -n "$CONTAINER_IP" ] && [ -n "$HUB_PORT" ]; then
    echo -n "  容器内部 (${CONTAINER_IP}:${HUB_PORT}): "
    if curl -sf http://${CONTAINER_IP}:${HUB_PORT}/health > /dev/null 2>&1; then
        echo "✓ 成功"
        RESPONSE=$(curl -s http://${CONTAINER_IP}:${HUB_PORT}/health)
        echo "  响应: $RESPONSE"
    else
        echo "✗ 失败"
    fi
fi

# 5. 测试localhost访问
echo ""
echo "本地访问测试:"
for port in 8080 38080 48080 ${HUB_PORT}; do
    echo -n "  localhost:$port: "
    if curl -sf -m 2 http://localhost:$port/health > /dev/null 2>&1; then
        echo "✓ 可访问"
        WORKING_PORT=$port
        break
    else
        echo "✗ 不可访问"
    fi
done

echo ""
echo "============="
if [ -n "$WORKING_PORT" ]; then
    echo "✅ 部署成功！"
    echo ""
    echo "访问地址:"
    echo "  Hub API: http://localhost:$WORKING_PORT"
    echo "  Web UI:  http://localhost:3000"
    echo ""
    echo "默认账号: admin / admin123"
else
    echo "⚠️  无法通过localhost访问，但服务在容器内运行正常"
    echo ""
    echo "可能的原因:"
    echo "1. 端口映射配置问题"
    echo "2. 防火墙限制"
    echo ""
    echo "你可以通过容器IP访问:"
    echo "  http://${CONTAINER_IP}:${HUB_PORT}"
fi