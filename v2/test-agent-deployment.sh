#!/bin/bash
# 测试Agent部署和下载功能

set -e

# 颜色定义
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
RED='\033[0;31m'
BLUE='\033[0;34m'
NC='\033[0m'

echo -e "${BLUE}=========================================${NC}"
echo -e "${BLUE}Agent部署测试脚本${NC}"
echo -e "${BLUE}=========================================${NC}"
echo ""

# 检查二进制文件是否存在
echo -e "${YELLOW}检查二进制文件...${NC}"
if [ -f "/workspace/v2/hub/static/binaries/rclone-backup-agent" ]; then
    SIZE=$(du -h "/workspace/v2/hub/static/binaries/rclone-backup-agent" | cut -f1)
    echo -e "${GREEN}✓ 二进制文件存在: ${SIZE}${NC}"
else
    echo -e "${RED}✗ 二进制文件不存在${NC}"
    exit 1
fi

# 检查符号链接
if [ -L "/workspace/v2/hub/static/binaries/rclone-backup-agent-latest" ]; then
    echo -e "${GREEN}✓ 符号链接存在${NC}"
else
    echo -e "${RED}✗ 符号链接不存在${NC}"
    exit 1
fi

echo ""

# 测试二进制文件基本功能
echo -e "${YELLOW}测试二进制文件基本功能...${NC}"
cd /workspace/v2/hub/static/binaries/

# 测试版本信息
if ./rclone-backup-agent --help > /dev/null 2>&1; then
    echo -e "${GREEN}✓ 二进制文件可执行${NC}"
else
    echo -e "${RED}✗ 二进制文件无法执行${NC}"
    exit 1
fi

echo ""

# 测试Hub API构建
echo -e "${YELLOW}测试Hub API构建...${NC}"
cd /workspace/v2/hub/

if go build -o test-hub-api ./main.go; then
    echo -e "${GREEN}✓ Hub API构建成功${NC}"
    rm -f test-hub-api
else
    echo -e "${RED}✗ Hub API构建失败${NC}"
    exit 1
fi

echo ""

# 显示使用说明
echo -e "${GREEN}=========================================${NC}"
echo -e "${GREEN}测试完成！${NC}"
echo -e "${GREEN}=========================================${NC}"
echo ""
echo -e "${BLUE}下一步操作:${NC}"
echo -e "1. 启动Hub服务:"
echo -e "   ${YELLOW}cd /workspace/v2/hub && go run main.go${NC}"
echo ""
echo -e "2. 在另一个终端测试下载:"
echo -e "   ${YELLOW}curl -L http://localhost:8080/api/v1/agent/download -o test-agent${NC}"
echo -e "   ${YELLOW}chmod +x test-agent${NC}"
echo -e "   ${YELLOW}./test-agent --help${NC}"
echo ""
echo -e "3. 访问Web界面:"
echo -e "   ${YELLOW}http://localhost:8080${NC}"
echo ""
echo -e "4. 在Agents页面生成注册令牌并测试完整流程"
echo ""

echo -e "${GREEN}Agent部署测试完成！${NC}"