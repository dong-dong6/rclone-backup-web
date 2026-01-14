#!/bin/bash
# 简单的Agent启动脚本

set -e

# 颜色定义
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
RED='\033[0;31m'
BLUE='\033[0;34m'
NC='\033[0m'

# 默认配置
DEFAULT_HUB_URL="http://localhost:8080"
DEFAULT_AGENT_NAME="agent-$(hostname)"

echo -e "${BLUE}=========================================${NC}"
echo -e "${BLUE}Rclone Backup Agent 启动脚本${NC}"
echo -e "${BLUE}=========================================${NC}"
echo ""

# 显示帮助信息
show_help() {
    echo "Agent启动脚本 - 从Hub下载并运行Agent"
    echo ""
    echo "用法: $0 [选项]"
    echo ""
    echo "选项:"
    echo "  -u, --hub-url URL         Hub服务URL (默认: $DEFAULT_HUB_URL)"
    echo "  -n, --agent-name NAME     Agent名称 (默认: $DEFAULT_AGENT_NAME)"
    echo "  -t, --token TOKEN         注册令牌 (必需)"
    echo "  -w, --work-dir DIR        工作目录 (默认: /opt/rclone-agent)"
    echo "  -c, --max-concurrent NUM  最大并发数 (默认: 3)"
    echo "  -i, --heartbeat-interval NUM  心跳间隔秒数 (默认: 30)"
    echo "  -p, --platform PLATFORM  目标平台 (linux/amd64, linux/arm64, darwin/amd64, darwin/arm64, windows/amd64)"
    echo "  -a, --arch ARCH           目标架构 (amd64, arm64, arm)"
    echo "  -d, --daemon              以守护进程模式运行"
    echo "  -h, --help               显示此帮助信息"
    echo ""
    echo "示例:"
    echo "  $0 --token abc123 --hub-url http://hub.example.com"
    echo "  $0 -t abc123 -u http://hub.example.com -n my-agent -d"
    echo "  $0 -t abc123 -p linux -a arm64"
    echo ""
}

# 解析命令行参数
HUB_URL="$DEFAULT_HUB_URL"
AGENT_NAME="$DEFAULT_AGENT_NAME"
REGISTRATION_TOKEN=""
WORK_DIR="/opt/rclone-agent"
MAX_CONCURRENT=3
HEARTBEAT_INTERVAL=30
PLATFORM=""
ARCH=""
DAEMON=false

while [[ $# -gt 0 ]]; do
    case $1 in
        -u|--hub-url)
            HUB_URL="$2"
            shift 2
            ;;
        -n|--agent-name)
            AGENT_NAME="$2"
            shift 2
            ;;
        -t|--token)
            REGISTRATION_TOKEN="$2"
            shift 2
            ;;
        -w|--work-dir)
            WORK_DIR="$2"
            shift 2
            ;;
        -c|--max-concurrent)
            MAX_CONCURRENT="$2"
            shift 2
            ;;
        -i|--heartbeat-interval)
            HEARTBEAT_INTERVAL="$2"
            shift 2
            ;;
        -p|--platform)
            PLATFORM="$2"
            shift 2
            ;;
        -a|--arch)
            ARCH="$2"
            shift 2
            ;;
        -d|--daemon)
            DAEMON=true
            shift
            ;;
        -h|--help)
            show_help
            exit 0
            ;;
        *)
            echo -e "${RED}错误: 未知参数 $1${NC}"
            show_help
            exit 1
            ;;
    esac
done

# 验证必需参数
if [ -z "$REGISTRATION_TOKEN" ]; then
    echo -e "${RED}错误: 注册令牌是必需的${NC}"
    echo "使用 -t 或 --token 参数提供注册令牌"
    echo "使用 -h 或 --help 查看完整帮助信息"
    exit 1
fi

# 检测平台和架构
if [ -z "$PLATFORM" ] || [ -z "$ARCH" ]; then
    echo -e "${YELLOW}检测平台和架构...${NC}"
    
    case "$(uname -s)" in
        Linux*)
            PLATFORM="linux"
            case "$(uname -m)" in
                x86_64) ARCH="amd64" ;;
                aarch64) ARCH="arm64" ;;
                armv7l) ARCH="arm" ;;
                *) ARCH="amd64" ;;
            esac
            ;;
        Darwin*)
            PLATFORM="darwin"
            case "$(uname -m)" in
                arm64) ARCH="arm64" ;;
                x86_64) ARCH="amd64" ;;
                *) ARCH="amd64" ;;
            esac
            ;;
        CYGWIN*|MINGW32*|MSYS*|MINGW*)
            PLATFORM="windows"
            ARCH="amd64"
            ;;
        *)
            echo -e "${YELLOW}警告: 无法检测平台，使用默认值 linux/amd64${NC}"
            PLATFORM="linux"
            ARCH="amd64"
            ;;
    esac
fi

echo -e "${YELLOW}配置信息:${NC}"
echo -e "  Hub URL:           ${HUB_URL}"
echo -e "  Agent名称:         ${AGENT_NAME}"
echo -e "  注册令牌:          ${REGISTRATION_TOKEN:0:8}..."
echo -e "  工作目录:          ${WORK_DIR}"
echo -e "  最大并发数:        ${MAX_CONCURRENT}"
echo -e "  心跳间隔:          ${HEARTBEAT_INTERVAL}秒"
echo -e "  目标平台:          ${PLATFORM}/${ARCH}"
echo -e "  守护进程模式:      ${DAEMON}"
echo ""

# 创建工作目录
echo -e "${YELLOW}创建工作目录...${NC}"
mkdir -p "$WORK_DIR"
cd "$WORK_DIR"

# 构建下载URL
DOWNLOAD_URL="${HUB_URL}/api/v1/agent/download?platform=${PLATFORM}&arch=${ARCH}"
echo -e "${YELLOW}下载Agent二进制文件...${NC}"
echo -e "  下载URL: ${DOWNLOAD_URL}"

# 下载二进制文件
if command -v curl &> /dev/null; then
    curl -L -o rclone-backup-agent "$DOWNLOAD_URL"
elif command -v wget &> /dev/null; then
    wget -O rclone-backup-agent "$DOWNLOAD_URL"
else
    echo -e "${RED}错误: 需要curl或wget来下载文件${NC}"
    exit 1
fi

# 设置执行权限
chmod +x rclone-backup-agent

# 检查下载是否成功
if [ ! -f "rclone-backup-agent" ] || [ ! -x "rclone-backup-agent" ]; then
    echo -e "${RED}错误: 下载失败或文件不可执行${NC}"
    exit 1
fi

echo -e "${GREEN}✓ Agent二进制文件下载成功${NC}"

# 创建配置文件
echo -e "${YELLOW}创建配置文件...${NC}"
cat > agent-config.json << EOF
{
  "hub_url": "${HUB_URL}",
  "registration_token": "${REGISTRATION_TOKEN}",
  "agent_name": "${AGENT_NAME}",
  "work_dir": "${WORK_DIR}",
  "max_concurrent": ${MAX_CONCURRENT},
  "heartbeat_interval": ${HEARTBEAT_INTERVAL},
  "enable_local_fallback": true,
  "run_as_service": false,
  "log_level": "info",
  "log_file": "",
  "pid_file": ""
}
EOF

echo -e "${GREEN}✓ 配置文件已创建: agent-config.json${NC}"

# 创建环境变量文件
cat > .env << EOF
HUB_URL=${HUB_URL}
REGISTRATION_TOKEN=${REGISTRATION_TOKEN}
AGENT_NAME=${AGENT_NAME}
WORK_DIR=${WORK_DIR}
MAX_CONCURRENT=${MAX_CONCURRENT}
HEARTBEAT_INTERVAL=${HEARTBEAT_INTERVAL}
EOF

echo -e "${GREEN}✓ 环境变量文件已创建: .env${NC}"

# 显示运行信息
echo ""
echo -e "${GREEN}=========================================${NC}"
echo -e "${GREEN}Agent准备完成！${NC}"
echo -e "${GREEN}=========================================${NC}"
echo ""
echo -e "${BLUE}文件位置:${NC}"
echo -e "  二进制文件: ${WORK_DIR}/rclone-backup-agent"
echo -e "  配置文件:   ${WORK_DIR}/agent-config.json"
echo -e "  环境文件:   ${WORK_DIR}/.env"
echo ""

# 运行Agent
echo -e "${YELLOW}启动Agent...${NC}"

if [ "$DAEMON" = true ]; then
    echo -e "${BLUE}以守护进程模式启动...${NC}"
    nohup ./rclone-backup-agent -config agent-config.json > agent.log 2>&1 &
    PID=$!
    echo -e "${GREEN}✓ Agent已启动，PID: ${PID}${NC}"
    echo -e "  日志文件: ${WORK_DIR}/agent.log"
    echo -e "  停止命令: kill ${PID}"
else
    echo -e "${BLUE}以前台模式启动...${NC}"
    echo -e "${YELLOW}按 Ctrl+C 停止Agent${NC}"
    echo ""
    ./rclone-backup-agent -config agent-config.json
fi
