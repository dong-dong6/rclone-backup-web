#!/bin/bash

# 快速访问脚本 - 绕过健康检查直接获取访问地址

echo "========================================="
echo "🎉 服务部署成功！"
echo "========================================="

# 获取实际运行的端口
HUB_PORT=$(docker logs v2-hub-api-1 2>&1 | grep "server started on port" | tail -1 | sed 's/.*port \([0-9]*\).*/\1/')
WEB_PORT=3000

echo ""
echo "📍 访问地址："
echo "  Hub API: http://localhost:${HUB_PORT:-48080}"
echo "  Web UI:  http://localhost:${WEB_PORT}"
echo ""

# 测试访问
echo "🔍 测试连接："
echo -n "  Hub API健康检查: "
if curl -sf http://localhost:${HUB_PORT:-48080}/health > /dev/null 2>&1; then
    echo "✅ 正常"
    RESPONSE=$(curl -s http://localhost:${HUB_PORT:-48080}/health)
    echo "    响应: $RESPONSE"
else
    echo "⚠️  无法访问"
fi

echo -n "  Web UI: "
if curl -sf http://localhost:${WEB_PORT}/ > /dev/null 2>&1; then
    echo "✅ 正常"
else
    echo "⚠️  无法访问"
fi

echo ""
echo "🔑 默认管理员账号："
echo "  用户名: admin"
echo "  密码: admin123"
echo ""

echo "📊 服务状态："
docker compose ps

echo ""
echo "💡 提示："
echo "  查看日志: docker compose logs -f hub-api"
echo "  重启服务: docker compose restart hub-api"
echo "========================================="