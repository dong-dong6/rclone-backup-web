#!/bin/bash

# 测试编译脚本 - 快速验证Go代码是否能编译通过

set -e

echo "🔍 测试Go代码编译..."
echo ""

# 颜色定义
GREEN='\033[0;32m'
RED='\033[0;31m'
YELLOW='\033[1;33m'
NC='\033[0m'

# 测试Hub编译
echo "1️⃣ 测试Hub代码编译..."
cd hub
if go build -o /tmp/test-hub-build ./main.go; then
    echo -e "${GREEN}✅ Hub代码编译成功${NC}"
    rm -f /tmp/test-hub-build
else
    echo -e "${RED}❌ Hub代码编译失败${NC}"
    exit 1
fi
cd ..

# 测试Agent编译
echo ""
echo "2️⃣ 测试Agent代码编译..."
cd agent
if go build -o /tmp/test-agent-build ./main.go; then
    echo -e "${GREEN}✅ Agent代码编译成功${NC}"
    rm -f /tmp/test-agent-build
else
    echo -e "${RED}❌ Agent代码编译失败${NC}"
    exit 1
fi
cd ..

echo ""
echo -e "${GREEN}🎉 所有代码编译测试通过！${NC}"
echo ""
echo "现在可以安全地运行Docker构建："
echo "  ./deploy.sh hub-with-agent"