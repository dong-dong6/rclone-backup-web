#!/bin/bash

# 修复部署脚本的健康检查方法

set -e

# 颜色定义
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
BLUE='\033[0;34m'
NC='\033[0m'

print_info() {
    echo -e "${BLUE}[INFO]${NC} $1"
}

print_success() {
    echo -e "${GREEN}[SUCCESS]${NC} $1"
}

print_error() {
    echo -e "${RED}[ERROR]${NC} $1"
}

# 新的健康检查函数 - 直接访问容器IP
wait_for_hub_api() {
    print_info "等待Hub API服务启动..."
    
    local max_attempts=30
    local attempt=0
    
    while [ $attempt -lt $max_attempts ]; do
        # 方法1: 获取容器IP并直接访问
        CONTAINER_IP=$(docker inspect -f '{{range .NetworkSettings.Networks}}{{.IPAddress}}{{end}}' v2-hub-api-1 2>/dev/null || echo "")
        
        if [ -n "$CONTAINER_IP" ]; then
            # 直接访问容器IP
            if curl -sf http://${CONTAINER_IP}:8080/health &> /dev/null; then
                print_success "Hub API已启动 (容器IP: ${CONTAINER_IP})"
                
                # 验证localhost映射是否工作
                if curl -sf http://localhost:8080/health &> /dev/null; then
                    print_success "端口映射正常 (localhost:8080 可访问)"
                else
                    print_error "警告: 端口映射可能有问题 (localhost:8080 不可访问)"
                    print_info "但可以通过容器IP访问: http://${CONTAINER_IP}:8080"
                fi
                
                return 0
            else
                echo -n "."
            fi
        else
            # 方法2: 如果无法获取容器IP，尝试通过docker exec
            if docker compose exec -T hub-api wget -q -O /dev/null http://localhost:8080/health 2>/dev/null; then
                print_success "Hub API已启动 (通过容器内部访问验证)"
                return 0
            else
                echo -n "."
            fi
        fi
        
        sleep 2
        ((attempt++))
    done
    
    print_error "Hub API启动超时"
    
    # 诊断信息
    echo ""
    print_info "诊断信息:"
    echo "1. 容器状态:"
    docker compose ps hub-api
    
    echo ""
    echo "2. 容器IP:"
    docker inspect v2-hub-api-1 --format='{{range .NetworkSettings.Networks}}{{.NetworkName}}: {{.IPAddress}}{{end}}' 2>/dev/null || echo "无法获取"
    
    echo ""
    echo "3. 最近的错误日志:"
    docker compose logs --tail=20 hub-api
    
    return 1
}

# 测试新的健康检查方法
print_info "测试改进的健康检查方法..."

# 检查容器是否运行
if docker ps | grep -q v2-hub-api-1; then
    wait_for_hub_api
else
    print_error "Hub API容器未运行"
    print_info "尝试启动..."
    docker compose up -d hub-api
    sleep 5
    wait_for_hub_api
fi