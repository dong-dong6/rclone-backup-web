#!/bin/bash

# API 测试脚本

API_URL="http://localhost:43000/api/v1"
USERNAME="admin"
PASSWORD="admin123"

echo "=== 测试 API 连接 ==="
echo ""

# 1. 测试登录
echo "1. 测试登录接口..."
LOGIN_RESPONSE=$(curl -s -X POST "${API_URL}/admin/login" \
  -H "Content-Type: application/json" \
  -d "{\"username\":\"${USERNAME}\",\"password\":\"${PASSWORD}\"}")

echo "登录响应: ${LOGIN_RESPONSE}"
echo ""

# 提取 token
TOKEN=$(echo "${LOGIN_RESPONSE}" | grep -o '"token":"[^"]*' | sed 's/"token":"//')

if [ -z "$TOKEN" ]; then
  echo "❌ 登录失败，无法获取 token"
  exit 1
fi

echo "✓ 登录成功，获取到 token"
echo ""

# 2. 测试获取 agents 列表
echo "2. 测试获取 agents 列表..."
AGENTS_RESPONSE=$(curl -s -X GET "${API_URL}/admin/agents" \
  -H "Authorization: Bearer ${TOKEN}")

echo "Agents 响应: ${AGENTS_RESPONSE}"
echo ""

# 3. 测试创建注册令牌
echo "3. 测试创建注册令牌..."
REG_TOKEN_RESPONSE=$(curl -s -X POST "${API_URL}/admin/agents/registration-token" \
  -H "Authorization: Bearer ${TOKEN}" \
  -H "Content-Type: application/json")

echo "注册令牌响应: ${REG_TOKEN_RESPONSE}"
echo ""

# 4. 测试获取任务列表
echo "4. 测试获取任务列表..."
TASKS_RESPONSE=$(curl -s -X GET "${API_URL}/admin/tasks" \
  -H "Authorization: Bearer ${TOKEN}")

echo "Tasks 响应: ${TASKS_RESPONSE}"
echo ""

# 5. 测试获取远程存储列表
echo "5. 测试获取远程存储列表..."
REMOTES_RESPONSE=$(curl -s -X GET "${API_URL}/admin/remotes" \
  -H "Authorization: Bearer ${TOKEN}")

echo "Remotes 响应: ${REMOTES_RESPONSE}"
echo ""

# 6. 测试获取执行历史
echo "6. 测试获取执行历史..."
EXECUTIONS_RESPONSE=$(curl -s -X GET "${API_URL}/admin/executions" \
  -H "Authorization: Bearer ${TOKEN}")

echo "Executions 响应: ${EXECUTIONS_RESPONSE}"
echo ""

# 7. 测试获取仪表板统计
echo "7. 测试获取仪表板统计..."
STATS_RESPONSE=$(curl -s -X GET "${API_URL}/admin/dashboard/stats" \
  -H "Authorization: Bearer ${TOKEN}")

echo "Dashboard Stats 响应: ${STATS_RESPONSE}"
echo ""

echo "=== 测试完成 ==="