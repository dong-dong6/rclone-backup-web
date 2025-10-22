#!/bin/bash

echo "=== API 问题诊断脚本 ==="
echo ""

# 检测 Docker Compose
if command -v docker-compose &> /dev/null; then
    DOCKER_COMPOSE="docker-compose"
elif docker compose version &> /dev/null 2>&1; then
    DOCKER_COMPOSE="docker compose"
else
    echo "错误: 未找到 docker-compose 或 docker compose"
    exit 1
fi

# 1. 检查容器状态
echo "1. 检查容器状态..."
$DOCKER_COMPOSE ps
echo ""

# 2. 检查数据库表结构
echo "2. 检查数据库表结构..."
$DOCKER_COMPOSE exec -T postgres psql -U postgres -d rclone_backup -c "\d registration_tokens"
echo ""

# 3. 应用数据库修复（如果需要）
echo "3. 应用数据库修复..."
$DOCKER_COMPOSE exec -T postgres psql -U postgres -d rclone_backup << 'EOF'
-- 添加缺失的 'used' 列
ALTER TABLE registration_tokens 
ADD COLUMN IF NOT EXISTS used BOOLEAN DEFAULT FALSE NOT NULL;

-- 更新现有数据
UPDATE registration_tokens 
SET used = TRUE 
WHERE used_at IS NOT NULL;

-- 显示表结构
\d registration_tokens

-- 显示现有数据
SELECT token, used, used_at, expires_at FROM registration_tokens;
EOF
echo ""

# 4. 检查 Hub API 日志
echo "4. Hub API 最近的日志..."
$DOCKER_COMPOSE logs --tail=20 hub-api
echo ""

# 5. 测试 API 连接
echo "5. 测试 API 健康检查..."
curl -s http://localhost:43000/api/v1/health || echo "API 健康检查失败"
echo ""

# 6. 重启 Hub API
echo "6. 重启 Hub API 服务..."
$DOCKER_COMPOSE restart hub-api
echo "等待服务启动..."
sleep 5

# 7. 再次测试
echo "7. 测试登录..."
LOGIN_RESPONSE=$(curl -s -X POST "http://localhost:43000/api/v1/admin/login" \
  -H "Content-Type: application/json" \
  -d '{"username":"admin","password":"admin123"}')

if echo "$LOGIN_RESPONSE" | grep -q "token"; then
    echo "✓ 登录成功"
    TOKEN=$(echo "$LOGIN_RESPONSE" | grep -o '"token":"[^"]*' | sed 's/"token":"//')
    
    echo ""
    echo "8. 测试创建注册令牌..."
    REG_TOKEN_RESPONSE=$(curl -s -X POST "http://localhost:43000/api/v1/admin/agents/registration-token" \
      -H "Authorization: Bearer $TOKEN" \
      -H "Content-Type: application/json")
    
    if echo "$REG_TOKEN_RESPONSE" | grep -q "error"; then
        echo "❌ 创建令牌失败: $REG_TOKEN_RESPONSE"
        
        # 查看更多日志
        echo ""
        echo "查看错误日志..."
        $DOCKER_COMPOSE logs --tail=10 hub-api | grep -E "ERROR|Failed|error"
    else
        echo "✓ 创建令牌成功: $REG_TOKEN_RESPONSE"
    fi
else
    echo "❌ 登录失败: $LOGIN_RESPONSE"
fi

echo ""
echo "=== 诊断完成 ==="