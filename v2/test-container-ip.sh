#!/bin/bash

echo "测试容器IP直接访问"
echo "=================="

# 1. 获取所有容器的IP
echo ""
echo "1. 容器IP地址:"
echo "---------------"

# Hub API容器
HUB_IP=$(docker inspect -f '{{range .NetworkSettings.Networks}}{{.IPAddress}}{{end}}' v2-hub-api-1 2>/dev/null)
if [ -n "$HUB_IP" ]; then
    echo "Hub API: $HUB_IP"
    
    echo ""
    echo "2. 测试健康检查端点:"
    echo "--------------------"
    
    # 测试容器IP
    echo -n "  直接访问容器IP ($HUB_IP:8080): "
    RESPONSE=$(curl -s -w "\nHTTP:%{http_code}" http://${HUB_IP}:8080/health 2>/dev/null)
    HTTP_CODE=$(echo "$RESPONSE" | grep "HTTP:" | cut -d: -f2)
    
    if [ "$HTTP_CODE" = "200" ]; then
        echo "✓ 成功"
        echo "  响应: $(echo "$RESPONSE" | grep -v "HTTP:")"
    else
        echo "✗ 失败 (HTTP: $HTTP_CODE)"
    fi
    
    # 测试其他端点
    echo ""
    echo "3. 测试其他端点:"
    echo "----------------"
    echo -n "  /api/v1/admin/login: "
    curl -s -o /dev/null -w "%{http_code}\n" http://${HUB_IP}:8080/api/v1/admin/login -X POST
    
    echo -n "  /: "
    curl -s -o /dev/null -w "%{http_code}\n" http://${HUB_IP}:8080/
    
else
    echo "✗ 无法获取Hub API容器IP"
    echo ""
    echo "容器可能未运行或网络配置有问题"
fi

# 其他容器
echo ""
echo "4. 所有容器网络信息:"
echo "-------------------"
docker compose ps --format json 2>/dev/null | jq -r '.[] | "\(.Service): \(.State)"' || docker compose ps

echo ""
echo "5. Docker网络列表:"
echo "------------------"
docker network ls | grep v2

echo ""
echo "6. 网络详情:"
echo "------------"
docker network inspect v2_backend 2>/dev/null | jq -r '.[] | .Containers | to_entries[] | "\(.value.Name): \(.value.IPv4Address)"' || echo "无法获取网络详情"

echo ""
echo "7. 端口映射:"
echo "------------"
docker compose port hub-api 8080 2>/dev/null || echo "Hub API: 无端口映射"

echo ""
echo "建议："
echo "-----"
if [ -z "$HUB_IP" ]; then
    echo "1. 检查容器是否运行: docker compose ps"
    echo "2. 查看容器日志: docker compose logs hub-api"
    echo "3. 重启服务: docker compose restart hub-api"
else
    echo "1. 容器IP可访问，使用: http://${HUB_IP}:8080"
    echo "2. 如果localhost不工作，可能是端口映射问题"
    echo "3. 可以修改部署脚本使用容器IP进行健康检查"
fi