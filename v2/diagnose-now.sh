#!/bin/bash

echo "立即诊断Hub API状态"
echo "==================="

# 1. 检查容器状态
echo ""
echo "1. 容器运行状态:"
docker compose ps

# 2. 获取Hub API容器ID
echo ""
echo "2. Hub API容器信息:"
CONTAINER_ID=$(docker compose ps -q hub-api)
if [ -n "$CONTAINER_ID" ]; then
    echo "   容器ID: $CONTAINER_ID"
    
    # 获取容器状态
    STATUS=$(docker inspect $CONTAINER_ID --format='{{.State.Status}}')
    echo "   状态: $STATUS"
    
    # 获取容器IP
    echo "   网络信息:"
    docker inspect $CONTAINER_ID --format='{{range $net, $conf := .NetworkSettings.Networks}}   {{$net}}: {{$conf.IPAddress}}{{"\n"}}{{end}}'
else
    echo "   ✗ 未找到Hub API容器"
fi

# 3. 测试各种访问方式
echo ""
echo "3. 健康检查测试:"

# localhost
echo -n "   localhost:8080: "
timeout 2 curl -sf http://localhost:8080/health > /dev/null 2>&1 && echo "✓ 成功" || echo "✗ 失败"

# 127.0.0.1
echo -n "   127.0.0.1:8080: "
timeout 2 curl -sf http://127.0.0.1:8080/health > /dev/null 2>&1 && echo "✓ 成功" || echo "✗ 失败"

# 容器内部
echo -n "   容器内部: "
docker compose exec -T hub-api sh -c 'curl -sf http://localhost:8080/health' > /dev/null 2>&1 && echo "✓ 成功" || echo "✗ 失败"

# 容器IP
if [ -n "$CONTAINER_ID" ]; then
    CONTAINER_IP=$(docker inspect $CONTAINER_ID --format='{{range .NetworkSettings.Networks}}{{.IPAddress}}{{end}}' | head -1)
    if [ -n "$CONTAINER_IP" ]; then
        echo -n "   容器IP ($CONTAINER_IP): "
        timeout 2 curl -sf http://${CONTAINER_IP}:8080/health > /dev/null 2>&1 && echo "✓ 成功" || echo "✗ 失败"
    fi
fi

# 4. 检查进程
echo ""
echo "4. Hub API进程:"
docker compose exec -T hub-api ps aux 2>/dev/null | grep hub-api || echo "   ✗ 进程未运行或容器已退出"

# 5. 查看错误
echo ""
echo "5. 最近的错误日志:"
docker compose logs --tail=10 hub-api 2>&1 | grep -E "error|Error|ERROR|fatal|Fatal|FATAL|panic" || echo "   (无错误日志)"

# 6. 端口监听
echo ""
echo "6. 端口监听情况:"
netstat -tlnp 2>/dev/null | grep :8080 || ss -tlnp 2>/dev/null | grep :8080 || echo "   主机上8080端口未监听"

echo ""
echo "==================="
echo "诊断完成"
echo ""

# 提供解决建议
echo "如果健康检查都失败，请执行:"
echo "  1. docker compose logs hub-api  # 查看完整日志"
echo "  2. docker compose down          # 停止所有服务"  
echo "  3. docker compose up hub-api    # 前台启动查看实时输出"