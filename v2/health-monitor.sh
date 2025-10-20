#!/bin/bash

# Docker 容器健康状态监控脚本

# 颜色定义
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
BLUE='\033[0;34m'
NC='\033[0m' # No Color

# 容器列表
CONTAINERS=("v2-postgres-1" "v2-redis-1" "v2-hub-api-1" "v2-web-ui-1")
CONTAINER_NAMES=("PostgreSQL" "Redis" "Hub API" "Web UI")

# 清屏函数
clear_screen() {
    printf "\033c"
}

# 获取健康状态的颜色
get_health_color() {
    case $1 in
        healthy)
            echo "${GREEN}"
            ;;
        unhealthy)
            echo "${RED}"
            ;;
        starting)
            echo "${YELLOW}"
            ;;
        *)
            echo "${NC}"
            ;;
    esac
}

# 获取健康状态的符号
get_health_symbol() {
    case $1 in
        healthy)
            echo "✓"
            ;;
        unhealthy)
            echo "✗"
            ;;
        starting)
            echo "⟳"
            ;;
        *)
            echo "?"
            ;;
    esac
}

# 监控循环
monitor_health() {
    while true; do
        clear_screen
        echo "========================================="
        echo "   Docker 容器健康状态监控"
        echo "   $(date '+%Y-%m-%d %H:%M:%S')"
        echo "========================================="
        echo ""
        
        for i in "${!CONTAINERS[@]}"; do
            container="${CONTAINERS[$i]}"
            name="${CONTAINER_NAMES[$i]}"
            
            # 获取容器状态
            if docker ps -q -f name=$container &> /dev/null; then
                health=$(docker inspect $container --format='{{.State.Health.Status}}' 2>/dev/null || echo "no_health_check")
                state=$(docker inspect $container --format='{{.State.Status}}' 2>/dev/null)
                
                # 获取健康检查详情
                if [ "$health" != "no_health_check" ] && [ "$health" != "" ]; then
                    failing_streak=$(docker inspect $container --format='{{.State.Health.FailingStreak}}' 2>/dev/null || echo "0")
                    
                    # 获取最后一次健康检查的输出
                    last_output=$(docker inspect $container --format='{{if .State.Health.Log}}{{(index .State.Health.Log 0).Output}}{{end}}' 2>/dev/null | head -c 50)
                    
                    color=$(get_health_color $health)
                    symbol=$(get_health_symbol $health)
                    
                    printf "%-15s ${color}%s %-10s${NC}" "$name:" "$symbol" "$health"
                    
                    if [ "$health" = "unhealthy" ] && [ "$failing_streak" != "0" ]; then
                        printf " (失败次数: %s)" "$failing_streak"
                    fi
                    
                    if [ "$health" = "starting" ]; then
                        # 显示启动进度
                        retries=$(docker inspect $container --format='{{len .State.Health.Log}}' 2>/dev/null || echo "0")
                        printf " (检查次数: %s/3)" "$retries"
                    fi
                    
                    echo ""
                    
                    if [ "$health" = "unhealthy" ] && [ -n "$last_output" ]; then
                        echo "      └─ 错误: $last_output..."
                    fi
                else
                    printf "%-15s %s %-10s\n" "$name:" "○" "$state"
                fi
            else
                printf "%-15s ${RED}✗ 未运行${NC}\n" "$name:"
            fi
        done
        
        echo ""
        echo "-----------------------------------------"
        echo "固定IP地址:"
        echo "  PostgreSQL: 172.30.0.10:5432"
        echo "  Redis:      172.30.0.11:6379"
        echo "  Hub API:    172.30.0.20:8080"
        echo "  Web UI:     172.30.0.30:80"
        echo "-----------------------------------------"
        echo ""
        echo "按 Ctrl+C 退出监控"
        
        sleep 2
    done
}

# 单次检查模式
single_check() {
    echo "Docker 容器健康状态"
    echo "==================="
    echo ""
    
    all_healthy=true
    
    for i in "${!CONTAINERS[@]}"; do
        container="${CONTAINERS[$i]}"
        name="${CONTAINER_NAMES[$i]}"
        
        health=$(docker inspect $container --format='{{.State.Health.Status}}' 2>/dev/null || echo "not_found")
        
        if [ "$health" = "healthy" ]; then
            echo -e "${GREEN}✓${NC} $name: healthy"
        elif [ "$health" = "unhealthy" ]; then
            echo -e "${RED}✗${NC} $name: unhealthy"
            all_healthy=false
        elif [ "$health" = "starting" ]; then
            echo -e "${YELLOW}⟳${NC} $name: starting"
            all_healthy=false
        elif [ "$health" = "not_found" ]; then
            echo -e "${RED}✗${NC} $name: 容器不存在"
            all_healthy=false
        else
            echo "? $name: $health"
            all_healthy=false
        fi
    done
    
    echo ""
    
    if $all_healthy; then
        echo -e "${GREEN}所有服务健康！${NC}"
        exit 0
    else
        echo -e "${YELLOW}部分服务未就绪${NC}"
        exit 1
    fi
}

# 主程序
case "${1:-}" in
    --once|-o)
        single_check
        ;;
    --watch|-w)
        monitor_health
        ;;
    *)
        echo "Docker 容器健康监控工具"
        echo ""
        echo "用法:"
        echo "  $0 --once   单次检查"
        echo "  $0 --watch  持续监控"
        echo ""
        echo "或直接运行 $0 显示此帮助"
        echo ""
        single_check
        ;;
esac