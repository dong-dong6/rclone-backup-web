#!/bin/bash

# 测试Web UI构建脚本

set -e

echo "🔍 测试Web UI构建..."
echo ""

# 颜色定义
GREEN='\033[0;32m'
RED='\033[0;31m'
YELLOW='\033[1;33m'
BLUE='\033[0;34m'
NC='\033[0m'

# 进入web目录
cd hub/web

# 检查必要文件
echo -e "${BLUE}1️⃣ 检查必要文件...${NC}"
files_to_check=(
    "package.json"
    "package-lock.json"
    "tsconfig.json"
    "tsconfig.node.json"
    "vite.config.ts"
    "index.html"
)

all_files_exist=true
for file in "${files_to_check[@]}"; do
    if [ -f "$file" ]; then
        echo -e "${GREEN}✅ $file 存在${NC}"
    else
        echo -e "${RED}❌ $file 缺失${NC}"
        all_files_exist=false
    fi
done

if [ "$all_files_exist" = false ]; then
    echo -e "${RED}❌ 缺少必要文件，无法构建${NC}"
    exit 1
fi

# 检查node_modules
echo ""
echo -e "${BLUE}2️⃣ 检查依赖...${NC}"
if [ ! -d "node_modules" ]; then
    echo -e "${YELLOW}⚠️  node_modules不存在，安装依赖...${NC}"
    npm ci
fi

# 测试构建
echo ""
echo -e "${BLUE}3️⃣ 执行构建...${NC}"
if npm run build; then
    echo -e "${GREEN}✅ 构建成功！${NC}"
    
    # 检查dist目录
    if [ -d "dist" ]; then
        echo ""
        echo -e "${BLUE}📦 构建产物：${NC}"
        du -sh dist
        ls -la dist | head -10
    fi
else
    echo -e "${RED}❌ 构建失败${NC}"
    exit 1
fi

echo ""
echo -e "${GREEN}🎉 Web UI构建测试通过！${NC}"
echo ""
echo "现在可以安全地运行Docker构建："
echo "  cd ../.."
echo "  ./deploy.sh hub-with-agent"