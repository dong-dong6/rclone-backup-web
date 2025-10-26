#!/bin/bash
# Agent交互式部署脚本 - 支持输入远程URL和令牌

set -e

# 颜色定义
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
RED='\033[0;31m'
BLUE='\033[0;34m'
CYAN='\033[0;36m'
NC='\033[0m'

# 配置变量
AGENT_DIR="/workspace/v2/agent"
HUB_DIR="/workspace/v2/hub"
BINARY_NAME="rclone-backup-agent"
VERSION="dev-$(date +%Y%m%d-%H%M%S)"
BUILD_TIME=$(date -u '+%Y-%m-%d %H:%M:%S UTC')
GIT_COMMIT=$(git rev-parse --short HEAD 2>/dev/null || echo "unknown")

# 默认值
DEFAULT_HUB_URL="http://localhost:8080"
DEFAULT_AGENT_NAME="agent-$(hostname)"

echo -e "${BLUE}=========================================${NC}"
echo -e "${BLUE}Rclone Backup Agent 交互式部署脚本${NC}"
echo -e "${BLUE}=========================================${NC}"
echo -e "版本:     ${VERSION}"
echo -e "构建时间: ${BUILD_TIME}"
echo -e "Git提交:  ${GIT_COMMIT}"
echo ""

# 检查Go环境
if ! command -v go &> /dev/null; then
    echo -e "${RED}错误: 未找到Go环境，请先安装Go${NC}"
    exit 1
fi

echo -e "${YELLOW}Go版本: $(go version)${NC}"
echo ""

# 交互式输入配置
echo -e "${CYAN}=== 配置输入 ===${NC}"
echo ""

# 输入Hub URL
echo -e "${YELLOW}请输入Hub服务URL:${NC}"
echo -e "默认值: ${DEFAULT_HUB_URL}"
read -p "Hub URL: " HUB_URL
HUB_URL=${HUB_URL:-$DEFAULT_HUB_URL}

# 输入Agent名称
echo -e "${YELLOW}请输入Agent名称:${NC}"
echo -e "默认值: ${DEFAULT_AGENT_NAME}"
read -p "Agent名称: " AGENT_NAME
AGENT_NAME=${AGENT_NAME:-$DEFAULT_AGENT_NAME}

# 输入注册令牌
echo -e "${YELLOW}请输入注册令牌:${NC}"
echo -e "提示: 可以在Hub Web界面的Agents页面生成"
read -p "注册令牌: " REGISTRATION_TOKEN

# 验证必要参数
if [ -z "$REGISTRATION_TOKEN" ]; then
    echo -e "${RED}错误: 注册令牌不能为空${NC}"
    exit 1
fi

# 输入工作目录
echo -e "${YELLOW}请输入工作目录:${NC}"
echo -e "默认值: /opt/rclone-agent"
read -p "工作目录: " WORK_DIR
WORK_DIR=${WORK_DIR:-"/opt/rclone-agent"}

# 输入最大并发数
echo -e "${YELLOW}请输入最大并发任务数:${NC}"
echo -e "默认值: 3"
read -p "最大并发数: " MAX_CONCURRENT
MAX_CONCURRENT=${MAX_CONCURRENT:-3}

# 输入心跳间隔
echo -e "${YELLOW}请输入心跳间隔(秒):${NC}"
echo -e "默认值: 30"
read -p "心跳间隔: " HEARTBEAT_INTERVAL
HEARTBEAT_INTERVAL=${HEARTBEAT_INTERVAL:-30}

echo ""
echo -e "${CYAN}=== 配置确认 ===${NC}"
echo -e "Hub URL:           ${HUB_URL}"
echo -e "Agent名称:         ${AGENT_NAME}"
echo -e "注册令牌:          ${REGISTRATION_TOKEN:0:8}..."
echo -e "工作目录:          ${WORK_DIR}"
echo -e "最大并发数:        ${MAX_CONCURRENT}"
echo -e "心跳间隔:          ${HEARTBEAT_INTERVAL}秒"
echo ""

# 确认配置
read -p "确认以上配置? (y/N): " CONFIRM
if [[ ! $CONFIRM =~ ^[Yy]$ ]]; then
    echo -e "${YELLOW}部署已取消${NC}"
    exit 0
fi

echo ""

# 进入agent目录
cd "$AGENT_DIR"

# 检查依赖
echo -e "${YELLOW}检查Go模块依赖...${NC}"
go mod tidy
go mod download
echo -e "${GREEN}✓ 依赖检查完成${NC}"
echo ""

# 构建当前平台的二进制文件
echo -e "${YELLOW}构建agent二进制文件...${NC}"

# 设置构建标志
LDFLAGS="-X main.Version=${VERSION}"
LDFLAGS="${LDFLAGS} -X 'main.BuildTime=${BUILD_TIME}'"
LDFLAGS="${LDFLAGS} -X main.GitCommit=${GIT_COMMIT}"
LDFLAGS="${LDFLAGS} -s -w"

# 构建
go build \
    -ldflags "${LDFLAGS}" \
    -o "${BINARY_NAME}" \
    main_standalone.go

# 设置执行权限
chmod +x "${BINARY_NAME}"

# 检查构建结果
if [ -f "${BINARY_NAME}" ]; then
    SIZE=$(du -h "${BINARY_NAME}" | cut -f1)
    echo -e "${GREEN}✓ 构建成功: ${BINARY_NAME} (${SIZE})${NC}"
else
    echo -e "${RED}错误: 构建失败${NC}"
    exit 1
fi

echo ""

# 创建配置文件
echo -e "${YELLOW}创建配置文件...${NC}"
cat > "${BINARY_NAME}.json" << EOF
{
  "hub_url": "${HUB_URL}",
  "registration_token": "${REGISTRATION_TOKEN}",
  "agent_name": "${AGENT_NAME}",
  "work_dir": "${WORK_DIR}",
  "max_concurrent": ${MAX_CONCURRENT},
  "heartbeat_interval": ${HEARTBEAT_INTERVAL},
  "enable_local_fallback": true,
  "enable_auto_update": false,
  "enable_metrics": true,
  "metrics_port": 9091,
  "run_as_service": false,
  "log_file": "",
  "pid_file": ""
}
EOF
echo -e "${GREEN}✓ 配置文件已创建: ${BINARY_NAME}.json${NC}"

echo ""

# 部署二进制文件到Hub
echo -e "${YELLOW}部署二进制文件到Hub...${NC}"

# 创建静态文件目录
mkdir -p "${HUB_DIR}/static/binaries"

# 复制二进制文件
cp "${BINARY_NAME}" "${HUB_DIR}/static/binaries/${BINARY_NAME}"
echo -e "${GREEN}✓ 二进制文件已复制到 ${HUB_DIR}/static/binaries/${NC}"

# 创建符号链接（用于下载）
ln -sf "${BINARY_NAME}" "${HUB_DIR}/static/binaries/rclone-backup-agent-latest"
echo -e "${GREEN}✓ 创建最新版本符号链接${NC}"

# 复制配置文件
cp "${BINARY_NAME}.json" "${HUB_DIR}/static/binaries/agent-config.json"
echo -e "${GREEN}✓ 配置文件已复制${NC}"

echo ""

# 更新DownloadAgent API以提供实际文件
echo -e "${YELLOW}更新DownloadAgent API...${NC}"

# 备份原始文件
cp "${HUB_DIR}/api/admin_handlers.go" "${HUB_DIR}/api/admin_handlers.go.backup"

# 更新DownloadAgent函数
sed -i '/^\/\/ DownloadAgent provides agent binary download/,/^}$/c\
// DownloadAgent provides agent binary download\
func (h *Handler) DownloadAgent(c *gin.Context) {\
	// 设置二进制文件下载的HTTP头\
	c.Header("Content-Type", "application/octet-stream")\
	c.Header("Content-Disposition", "attachment; filename=rclone-backup-agent")\
	c.Header("Content-Transfer-Encoding", "binary")\
	\
	// 读取实际的二进制文件\
	binaryPath := "./static/binaries/rclone-backup-agent"\
	fileData, err := os.ReadFile(binaryPath)\
	if err != nil {\
		c.JSON(http.StatusNotFound, gin.H{\
			"error": "Agent binary not found",\
			"message": "Binary file not available. Please run deploy-agent-interactive.sh first.",\
		})\
		return\
	}\
	\
	// 返回二进制文件\
	c.Data(http.StatusOK, "application/octet-stream", fileData)\
}' "${HUB_DIR}/api/admin_handlers.go"

# 添加必要的import
if ! grep -q "os" "${HUB_DIR}/api/admin_handlers.go"; then
    sed -i '/import (/,/)/c\
import (\
	"log"\
	"net/http"\
	"os"\
	"strconv"\
	"time"\
\
	"github.com/gin-gonic/gin"\
	"github.com/google/uuid"\
	"github.com/rclone-backup-web/hub/models"\
	"github.com/rclone-backup-web/hub/services"\
)' "${HUB_DIR}/api/admin_handlers.go"
fi

echo -e "${GREEN}✓ DownloadAgent API已更新${NC}"

# 清理临时文件
rm -f "${HUB_DIR}/api/admin_handlers.go.tmp"

echo ""

# 测试构建
echo -e "${YELLOW}测试Hub构建...${NC}"
cd "${HUB_DIR}"
if go build -o hub-api ./main.go; then
    echo -e "${GREEN}✓ Hub构建成功${NC}"
    rm -f hub-api
else
    echo -e "${RED}错误: Hub构建失败${NC}"
    # 恢复备份文件
    mv "${HUB_DIR}/api/admin_handlers.go.backup" "${HUB_DIR}/api/admin_handlers.go"
    exit 1
fi

echo ""

# 生成运行命令
echo -e "${YELLOW}生成运行命令...${NC}"

# 创建环境变量文件
cat > "${AGENT_DIR}/.env" << EOF
HUB_URL=${HUB_URL}
REGISTRATION_TOKEN=${REGISTRATION_TOKEN}
AGENT_NAME=${AGENT_NAME}
WORK_DIR=${WORK_DIR}
MAX_CONCURRENT=${MAX_CONCURRENT}
HEARTBEAT_INTERVAL=${HEARTBEAT_INTERVAL}
EOF

# 生成Docker运行命令
DOCKER_CMD="docker run -d --name rclone-backup-agent \\
  -e HUB_URL=\"${HUB_URL}\" \\
  -e REGISTRATION_TOKEN=\"${REGISTRATION_TOKEN}\" \\
  -e AGENT_NAME=\"${AGENT_NAME}\" \\
  -e WORK_DIR=\"${WORK_DIR}\" \\
  -e MAX_CONCURRENT=\"${MAX_CONCURRENT}\" \\
  -e HEARTBEAT_INTERVAL=\"${HEARTBEAT_INTERVAL}\" \\
  rclone-backup-web/agent:latest"

# 生成二进制运行命令
BINARY_CMD="# 下载并运行二进制文件
curl -L \"${HUB_URL}/api/v1/agent/download\" -o rclone-backup-agent
chmod +x rclone-backup-agent

# 使用配置文件运行
./rclone-backup-agent -config agent-config.json

# 或者使用环境变量运行
export HUB_URL=\"${HUB_URL}\"
export REGISTRATION_TOKEN=\"${REGISTRATION_TOKEN}\"
export AGENT_NAME=\"${AGENT_NAME}\"
export WORK_DIR=\"${WORK_DIR}\"
export MAX_CONCURRENT=\"${MAX_CONCURRENT}\"
export HEARTBEAT_INTERVAL=\"${HEARTBEAT_INTERVAL}\"
./rclone-backup-agent"

echo -e "${GREEN}✓ 运行命令已生成${NC}"

echo ""

# 显示部署信息
echo -e "${GREEN}=========================================${NC}"
echo -e "${GREEN}Agent交互式部署完成！${NC}"
echo -e "${GREEN}=========================================${NC}"
echo ""
echo -e "${BLUE}部署信息:${NC}"
echo -e "  二进制文件: ${HUB_DIR}/static/binaries/${BINARY_NAME}"
echo -e "  配置文件:   ${HUB_DIR}/static/binaries/agent-config.json"
echo -e "  环境文件:   ${AGENT_DIR}/.env"
echo -e "  下载链接:   ${HUB_URL}/api/v1/agent/download"
echo -e "  文件大小:   $(du -h "${HUB_DIR}/static/binaries/${BINARY_NAME}" | cut -f1)"
echo ""
echo -e "${BLUE}配置信息:${NC}"
echo -e "  Hub URL:           ${HUB_URL}"
echo -e "  Agent名称:         ${AGENT_NAME}"
echo -e "  工作目录:          ${WORK_DIR}"
echo -e "  最大并发数:        ${MAX_CONCURRENT}"
echo -e "  心跳间隔:          ${HEARTBEAT_INTERVAL}秒"
echo ""
echo -e "${BLUE}使用方法:${NC}"
echo -e "  1. 启动Hub服务: cd ${HUB_DIR} && go run main.go"
echo -e "  2. 访问Web界面: ${HUB_URL}"
echo -e "  3. 在Agents页面查看注册的agent"
echo ""
echo -e "${BLUE}运行命令:${NC}"
echo -e "${CYAN}二进制方式:${NC}"
echo -e "${YELLOW}${BINARY_CMD}${NC}"
echo ""
echo -e "${CYAN}Docker方式:${NC}"
echo -e "${YELLOW}${DOCKER_CMD}${NC}"
echo ""

# 清理agent目录中的临时文件
cd "${AGENT_DIR}"
rm -f "${BINARY_NAME}"

echo -e "${GREEN}交互式部署完成！现在可以启动Hub服务进行测试。${NC}"