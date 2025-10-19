#!/bin/bash

echo "测试 curl -f 的行为"
echo "================================"

# 测试1: 404响应
echo -n "测试404响应: "
if curl -f http://httpbin.org/status/404 &> /dev/null; then
    echo "✗ 错误：不应该成功"
else
    echo "✓ 正确：返回失败 (exit code: $?)"
fi

# 测试2: 200响应
echo -n "测试200响应: "
if curl -f http://httpbin.org/status/200 &> /dev/null; then
    echo "✓ 正确：返回成功"
else
    echo "✗ 错误：不应该失败 (exit code: $?)"
fi

# 测试3: 连接拒绝
echo -n "测试连接拒绝: "
if curl -f http://localhost:99999 &> /dev/null; then
    echo "✗ 错误：不应该成功"
else
    echo "✓ 正确：返回失败 (exit code: $?)"
fi

echo ""
echo "模拟部署脚本的等待逻辑"
echo "================================"

# 模拟等待循环
max_attempts=3
attempt=0

echo "等待一个返回404的服务..."
while [ $attempt -lt $max_attempts ]; do
    if curl -f http://httpbin.org/status/404 &> /dev/null; then
        echo "✗ 不应该到这里（404应该失败）"
        break
    fi
    echo -n "."
    sleep 1
    ((attempt++))
done
echo " 超时（预期行为）"

echo ""
echo "等待一个返回200的服务..."
attempt=0
while [ $attempt -lt $max_attempts ]; do
    if curl -f http://httpbin.org/status/200 &> /dev/null; then
        echo " ✓ 成功！"
        break
    fi
    echo -n "."
    sleep 1
    ((attempt++))
done