#!/bin/bash

# 前端配置检查脚本
set -e

# 颜色输出
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
BLUE='\033[0;34m'
NC='\033[0m'

print_info() { echo -e "${BLUE}[INFO]${NC} $1"; }
print_success() { echo -e "${GREEN}[SUCCESS]${NC} $1"; }
print_warning() { echo -e "${YELLOW}[WARNING]${NC} $1"; }
print_error() { echo -e "${RED}[ERROR]${NC} $1"; }

echo "================================================"
echo "   前端配置检查工具"
echo "================================================"
echo ""

# 检查前端目录
if [ ! -d "hub/web" ]; then
    print_error "未找到前端目录 hub/web"
    exit 1
fi

cd hub/web

# 检查 package.json
print_info "检查 package.json..."
if [ -f "package.json" ]; then
    print_success "找到 package.json"
    
    # 检查关键依赖
    if grep -q "react-i18next" package.json; then
        print_success "✓ react-i18next 已安装"
    else
        print_warning "✗ react-i18next 未安装"
        print_info "  运行: npm install react-i18next i18next"
    fi
    
    if grep -q "axios" package.json; then
        print_success "✓ axios 已安装"
    else
        print_warning "✗ axios 未安装"
        print_info "  运行: npm install axios"
    fi
else
    print_error "未找到 package.json"
    exit 1
fi

# 检查 i18n 配置
print_info ""
print_info "检查国际化配置..."
if [ -f "src/i18n/index.ts" ] || [ -f "src/i18n/index.js" ]; then
    print_success "✓ i18n 配置文件存在"
else
    print_error "✗ i18n 配置文件不存在"
fi

if [ -f "src/i18n/locales/zh-CN.json" ]; then
    print_success "✓ 中文语言文件存在"
    KEYS=$(grep -c '"' src/i18n/locales/zh-CN.json || true)
    print_info "  包含 $KEYS 个翻译键"
else
    print_error "✗ 中文语言文件不存在"
fi

# 检查 API 服务
print_info ""
print_info "检查 API 服务配置..."
if [ -f "src/services/api.ts" ] || [ -f "src/services/api.js" ]; then
    print_success "✓ API 服务文件存在"
    
    if grep -q "axios" src/services/api.ts 2>/dev/null || grep -q "axios" src/services/api.js 2>/dev/null; then
        print_success "✓ 使用 axios 进行 API 调用"
    fi
else
    print_error "✗ API 服务文件不存在"
fi

# 检查环境变量
print_info ""
print_info "检查环境变量配置..."
if [ -f ".env" ]; then
    print_success "✓ .env 文件存在"
    source .env
    print_info "  API_URL: ${VITE_API_URL:-未设置}"
else
    print_warning "✗ .env 文件不存在"
    if [ -f ".env.example" ]; then
        print_info "  找到 .env.example，可以复制它："
        print_info "  cp .env.example .env"
    fi
fi

# 检查 node_modules
print_info ""
print_info "检查依赖安装状态..."
if [ -d "node_modules" ]; then
    print_success "✓ node_modules 目录存在"
    COUNT=$(ls node_modules | wc -l)
    print_info "  已安装 $COUNT 个包"
else
    print_warning "✗ node_modules 不存在"
    print_info "  需要运行: npm install"
fi

# 提供修复建议
echo ""
print_info "修复建议："
echo ""

if [ ! -f ".env" ] && [ -f ".env.example" ]; then
    echo "1. 创建 .env 文件："
    echo "   cp .env.example .env"
    echo ""
fi

if [ ! -d "node_modules" ]; then
    echo "2. 安装依赖："
    echo "   npm install"
    echo ""
fi

echo "3. 启动开发服务器："
echo "   npm run dev"
echo ""
echo "4. 在浏览器中打开："
echo "   http://localhost:5173"
echo ""
echo "5. 检查浏览器控制台（F12）："
echo "   - 查看是否有错误"
echo "   - 检查网络请求是否正常"

cd ../..
print_success "检查完成！"