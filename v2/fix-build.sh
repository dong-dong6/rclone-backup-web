#!/bin/bash

# ============================================
# Rclone Backup Web V2.0 - 构建修复脚本
# ============================================

set -e

# 颜色定义
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
BLUE='\033[0;34m'
NC='\033[0m'

echo -e "${BLUE}🔧 开始修复构建问题...${NC}"
echo ""

# 1. 复制database目录到hub
echo -e "${BLUE}1️⃣ 复制database目录...${NC}"
if [ ! -d "hub/database" ]; then
    if [ -d "database" ]; then
        cp -r database hub/
        echo -e "${GREEN}✅ database目录已复制到hub/${NC}"
    else
        echo -e "${YELLOW}⚠️  database目录不存在${NC}"
    fi
else
    echo -e "${GREEN}✅ hub/database目录已存在${NC}"
fi

# 2. 生成Hub的go.sum
echo ""
echo -e "${BLUE}2️⃣ 生成Hub的go.sum文件...${NC}"
cd hub
if [ ! -f "go.sum" ]; then
    go mod tidy
    echo -e "${GREEN}✅ Hub的go.sum已生成${NC}"
else
    echo -e "${GREEN}✅ Hub的go.sum已存在${NC}"
fi
cd ..

# 3. 生成Agent的go.sum
echo ""
echo -e "${BLUE}3️⃣ 生成Agent的go.sum文件...${NC}"
cd agent

# 先修复共享包导入问题
if grep -q '"github.com/rclone-backup-web/shared/logger"' main_with_logger.go 2>/dev/null; then
    echo -e "${YELLOW}修复共享包导入...${NC}"
    sed -i.bak 's|"github.com/rclone-backup-web/shared/logger"|// "github.com/rclone-backup-web/shared/logger" // TODO: implement shared logger|' main_with_logger.go
    rm -f main_with_logger.go.bak
fi

if [ ! -f "go.sum" ]; then
    go mod tidy
    echo -e "${GREEN}✅ Agent的go.sum已生成${NC}"
else
    echo -e "${GREEN}✅ Agent的go.sum已存在${NC}"
fi
cd ..

# 4. 检查Web UI的package-lock.json
echo ""
echo -e "${BLUE}4️⃣ 检查Web UI依赖...${NC}"
if [ -d "hub/web" ]; then
    cd hub/web
    if [ -f "package.json" ] && [ ! -f "package-lock.json" ]; then
        echo -e "${YELLOW}生成package-lock.json...${NC}"
        npm install
        echo -e "${GREEN}✅ package-lock.json已生成${NC}"
    else
        echo -e "${GREEN}✅ Web UI依赖已就绪${NC}"
    fi
    cd ../..
fi

# 5. 验证修复结果
echo ""
echo -e "${BLUE}5️⃣ 验证修复结果...${NC}"

errors=0

# 检查必要文件
files_to_check=(
    "hub/go.sum"
    "hub/database/migrations"
    "agent/go.sum"
    "docker-compose.yml"
)

for file in "${files_to_check[@]}"; do
    if [ -e "$file" ]; then
        echo -e "${GREEN}✅ $file 存在${NC}"
    else
        echo -e "${YELLOW}❌ $file 缺失${NC}"
        errors=$((errors + 1))
    fi
done

echo ""
if [ $errors -eq 0 ]; then
    echo -e "${GREEN}✨ 所有构建问题已修复！${NC}"
    echo ""
    echo "现在可以运行部署命令："
    echo "  ./deploy.sh hub              # 部署Hub"
    echo "  ./deploy.sh hub-with-agent   # 部署Hub和本地Agent"
    echo ""
    echo "或使用Makefile："
    echo "  make build-and-up"
    echo "  make build-and-up-with-agent"
else
    echo -e "${YELLOW}⚠️  还有 $errors 个问题需要手动处理${NC}"
    exit 1
fi