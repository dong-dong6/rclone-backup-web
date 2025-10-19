#!/bin/bash

echo "快速修复端口问题"
echo "================"

# 1. 检查当前端口配置
echo ""
echo "1. 当前配置:"
echo "   PORT环境变量: $(grep "^PORT=" .env 2>/dev/null || echo "未设置")"
echo "   HUB_PORT环境变量: $(grep "^HUB_PORT=" .env 2>/dev/null || echo "未设置")"

# 2. 修复.env文件
echo ""
echo "2. 修复端口配置..."

# 检查是否有错误的PORT设置
if grep -q "^PORT=38080" .env 2>/dev/null; then
    echo "   发现PORT=38080，正在修正..."
    sed -i 's/^PORT=38080/PORT=8080/' .env
    echo "   ✓ 已修改为PORT=8080"
elif grep -q "^HUB_PORT=38080" .env 2>/dev/null; then
    echo "   发现HUB_PORT=38080，正在修正..."
    sed -i 's/^HUB_PORT=38080/HUB_PORT=8080/' .env
    echo "   ✓ 已修改为HUB_PORT=8080"
else
    # 检查是否缺少PORT设置
    if ! grep -q "^PORT=" .env 2>/dev/null && ! grep -q "^HUB_PORT=" .env 2>/dev/null; then
        echo "   未找到端口设置，添加HUB_PORT=8080..."
        echo "HUB_PORT=8080" >> .env
        echo "   ✓ 已添加HUB_PORT=8080"
    fi
fi

# 3. 重启服务
echo ""
echo "3. 重启Hub API服务..."
docker compose restart hub-api

# 4. 等待服务启动
echo ""
echo "4. 等待服务启动..."
sleep 5

# 5. 测试健康检查
echo ""
echo "5. 测试健康检查:"

# 测试8080端口
echo -n "   http://localhost:8080/health: "
if curl -sf http://localhost:8080/health > /dev/null 2>&1; then
    echo "✓ 成功"
    RESPONSE=$(curl -s http://localhost:8080/health)
    echo "   响应: $RESPONSE"
else
    echo "✗ 失败"
    
    # 如果8080失败，测试38080
    echo -n "   http://localhost:38080/health: "
    if curl -sf http://localhost:38080/health > /dev/null 2>&1; then
        echo "✓ 成功（仍在38080端口）"
        echo ""
        echo "   ⚠️  端口仍然是38080，可能需要重新部署"
        echo "   建议执行："
        echo "     docker compose down"
        echo "     docker compose up -d"
    else
        echo "✗ 失败"
    fi
fi

echo ""
echo "================"
echo "完成！"
echo ""
echo "访问地址："
echo "  Hub API: http://localhost:8080"
echo "  Web UI:  http://localhost:3000"
echo ""
echo "如果仍有问题，请检查："
echo "  docker compose logs hub-api"