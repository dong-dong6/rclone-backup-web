#!/bin/bash

# Rclone Backup Web V2.0 - 诊断脚本

# 颜色定义
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
NC='\033[0m'

echo "========================================="
echo "系统诊断"
echo "========================================="
echo ""

# 检测Docker Compose
if command -v "docker" &> /dev/null && docker compose version &> /dev/null; then
    DC="docker compose"
elif command -v "docker-compose" &> /dev/null; then
    DC="docker-compose"
else
    echo -e "${RED}✗${NC} Docker Compose 未安装"
    exit 1
fi

# 检查容器健康状态
echo "容器状态:"
for container in v2-postgres-1 v2-redis-1 v2-hub-api-1 v2-web-ui-1; do
    name=$(echo $container | sed 's/v2-\(.*\)-1/\1/' | tr '-' ' ')
    health=$(docker inspect $container --format='{{.State.Health.Status}}' 2>/dev/null || echo "not_found")
    
    case $health in
        healthy) printf "  %-12s ${GREEN}✓ healthy${NC}\n" "$name:" ;;
        unhealthy) printf "  %-12s ${RED}✗ unhealthy${NC}\n" "$name:" ;;
        starting) printf "  %-12s ${YELLOW}⟳ starting${NC}\n" "$name:" ;;
        not_found) printf "  %-12s ${RED}✗ not running${NC}\n" "$name:" ;;
        *) printf "  %-12s %s\n" "$name:" "$health" ;;
    esac
done

echo ""

# 检查是否有错误日志
if $DC logs --tail=20 2>&1 | grep -q -E "ERROR|FATAL|Failed"; then
    echo -e "${YELLOW}⚠${NC} 发现错误日志，查看详情: $DC logs --tail=50"
fi

# 显示访问信息
WEB_PORT="${WEB_PORT:-3000}"
echo "========================================="
echo "访问地址: http://localhost:${WEB_PORT}"
echo "查看日志: $DC logs -f"
echo "========================================="