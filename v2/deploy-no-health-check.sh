#!/bin/bash

# 临时部署脚本 - 跳过健康检查

echo "启动服务（跳过健康检查）..."

docker compose up -d

echo "等待10秒..."
sleep 10

echo "服务状态："
docker compose ps

echo ""
echo "访问地址："
echo "  Web UI: http://localhost:3000"
echo "  Hub API: http://localhost:8080"
echo ""
echo "查看日志："
echo "  docker compose logs -f hub-api"