#!/bin/bash

echo "快速健康检查测试"
echo "=================="

# 测试健康检查端点
echo ""
echo "1. 测试健康检查端点:"
for i in {1..3}; do
    echo -n "  尝试 $i: "
    RESPONSE=$(curl -s -w "\nHTTP_CODE:%{http_code}" http://localhost:8080/health 2>/dev/null)
    HTTP_CODE=$(echo "$RESPONSE" | grep "HTTP_CODE:" | cut -d: -f2)
    
    if [ "$HTTP_CODE" = "200" ]; then
        echo "✓ 成功 (HTTP 200)"
        echo "    响应: $(echo "$RESPONSE" | grep -v "HTTP_CODE:")"
        exit 0
    elif [ "$HTTP_CODE" = "000" ]; then
        echo "✗ 连接失败（服务未启动或端口未开放）"
    else
        echo "✗ HTTP $HTTP_CODE"
        echo "    响应: $(echo "$RESPONSE" | grep -v "HTTP_CODE:")"
    fi
    
    sleep 2
done

echo ""
echo "2. 检查Hub API容器状态:"
docker compose exec hub-api sh -c 'ps aux | grep hub-api' || echo "容器可能已崩溃"

echo ""
echo "3. 查看Hub API错误日志:"
docker compose logs hub-api 2>&1 | grep -E "ERROR|FATAL|Failed|panic" | tail -10

echo ""
echo "建议："
echo "  1. 运行: docker compose logs hub-api"
echo "  2. 查看完整的错误信息"
echo "  3. 可能是数据库连接失败或配置问题"