#!/bin/bash

# 测试构建脚本

set -e

echo "🔨 测试构建所有Docker镜像..."
echo ""

# 检测Docker Compose命令
if docker compose version &> /dev/null; then
    DOCKER_COMPOSE="docker compose"
    echo "✅ 使用 Docker Compose V2 (docker compose)"
elif docker-compose --version &> /dev/null; then
    DOCKER_COMPOSE="docker-compose"
    echo "✅ 使用 Docker Compose V1 (docker-compose)"
else
    echo "❌ 未找到 Docker Compose"
    exit 1
fi

echo ""
echo "📦 开始构建..."
echo ""

# 构建Hub API
echo "1️⃣ 构建 Hub API..."
$DOCKER_COMPOSE build hub-api

# 构建Web UI
echo ""
echo "2️⃣ 构建 Web UI..."
$DOCKER_COMPOSE build web-ui

# 构建Agent
echo ""
echo "3️⃣ 构建 Agent..."
$DOCKER_COMPOSE --profile local-agent build local-agent

echo ""
echo "✅ 所有镜像构建成功！"
echo ""
echo "📋 构建的镜像："
docker images | grep -E "rclone-backup|REPOSITORY" | head -5