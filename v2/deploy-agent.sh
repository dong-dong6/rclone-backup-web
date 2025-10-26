#!/bin/bash
# Agent部署脚本 - 用于开发阶段构建和部署agent二进制文件

set -e

# 颜色定义
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
RED='\033[0;31m'
BLUE='\033[0;34m'
NC='\033[0m'

# 配置变量
AGENT_DIR="/workspace/v2/agent"
HUB_DIR="/workspace/v2/hub"
BINARY_NAME="rclone-backup-agent"
VERSION="dev-$(date +%Y%m%d-%H%M%S)"
BUILD_TIME=$(date -u '+%Y-%m-%d %H:%M:%S UTC')
GIT_COMMIT=$(git rev-parse --short HEAD 2>/dev/null || echo "unknown")

echo -e "${BLUE}=========================================${NC}"
echo -e "${BLUE}Rclone Backup Agent 部署脚本${NC}"
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

# 复制到hub目录的静态文件目录
echo -e "${YELLOW}部署二进制文件到Hub...${NC}"

# 创建静态文件目录
mkdir -p "${HUB_DIR}/static/binaries"

# 复制二进制文件
cp "${BINARY_NAME}" "${HUB_DIR}/static/binaries/${BINARY_NAME}"
echo -e "${GREEN}✓ 二进制文件已复制到 ${HUB_DIR}/static/binaries/${NC}"

# 创建符号链接（用于下载）
ln -sf "${BINARY_NAME}" "${HUB_DIR}/static/binaries/rclone-backup-agent-latest"
echo -e "${GREEN}✓ 创建最新版本符号链接${NC}"

echo ""

# 更新DownloadAgent API以提供实际文件
echo -e "${YELLOW}更新DownloadAgent API...${NC}"

# 备份原始文件
cp "${HUB_DIR}/api/admin_handlers.go" "${HUB_DIR}/api/admin_handlers.go.backup"

# 更新DownloadAgent函数
cat > "${HUB_DIR}/api/admin_handlers.go.tmp" << 'EOF'
// DownloadAgent provides agent binary download
func (h *Handler) DownloadAgent(c *gin.Context) {
	// 设置二进制文件下载的HTTP头
	c.Header("Content-Type", "application/octet-stream")
	c.Header("Content-Disposition", "attachment; filename=rclone-backup-agent")
	c.Header("Content-Transfer-Encoding", "binary")
	
	// 读取实际的二进制文件
	binaryPath := "./static/binaries/rclone-backup-agent"
	fileData, err := os.ReadFile(binaryPath)
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{
			"error": "Agent binary not found",
			"message": "Binary file not available. Please run deploy-agent.sh first.",
		})
		return
	}
	
	// 返回二进制文件
	c.Data(http.StatusOK, "application/octet-stream", fileData)
}
EOF

# 替换DownloadAgent函数
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
			"message": "Binary file not available. Please run deploy-agent.sh first.",\
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
	"database/sql"\
	"encoding/json"\
	"fmt"\
	"net/http"\
	"os"\
	"strconv"\
	"strings"\
	"time"\
\
	"github.com/gin-gonic/gin"\
	"github.com/google/uuid"\
	"github.com/jackc/pgx/v5/pgxpool"\
	"github.com/rclone-backup-web/hub/models"\
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

# 显示部署信息
echo -e "${GREEN}=========================================${NC}"
echo -e "${GREEN}Agent部署完成！${NC}"
echo -e "${GREEN}=========================================${NC}"
echo ""
echo -e "${BLUE}部署信息:${NC}"
echo -e "  二进制文件: ${HUB_DIR}/static/binaries/${BINARY_NAME}"
echo -e "  下载链接:   http://localhost:8080/api/v1/agent/download"
echo -e "  文件大小:   $(du -h "${HUB_DIR}/static/binaries/${BINARY_NAME}" | cut -f1)"
echo ""
echo -e "${BLUE}使用方法:${NC}"
echo -e "  1. 启动Hub服务: cd ${HUB_DIR} && go run main.go"
echo -e "  2. 访问Web界面: http://localhost:8080"
echo -e "  3. 在Agents页面生成注册令牌"
echo -e "  4. 使用提供的命令下载并运行agent"
echo ""
echo -e "${BLUE}测试命令:${NC}"
echo -e "  curl -L http://localhost:8080/api/v1/agent/download -o test-agent"
echo -e "  chmod +x test-agent"
echo -e "  ./test-agent --help"
echo ""

# 清理agent目录中的临时文件
cd "${AGENT_DIR}"
rm -f "${BINARY_NAME}"

echo -e "${GREEN}部署完成！现在可以启动Hub服务进行测试。${NC}"