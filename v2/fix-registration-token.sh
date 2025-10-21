#!/bin/bash

# 修复 registration_tokens 表的脚本

set -e

echo "=== 修复 Registration Tokens 表 ==="

# 检测 Docker Compose 版本
if command -v docker-compose &> /dev/null; then
    DOCKER_COMPOSE="docker-compose"
elif docker compose version &> /dev/null 2>&1; then
    DOCKER_COMPOSE="docker compose"
else
    echo "错误: 未找到 docker-compose 或 docker compose"
    exit 1
fi

# 检查数据库容器是否运行
if ! $DOCKER_COMPOSE ps postgres | grep -q "Up"; then
    echo "错误: PostgreSQL 容器未运行"
    echo "请先运行: ./deploy.sh"
    exit 1
fi

echo "正在应用数据库迁移..."

# 应用迁移
$DOCKER_COMPOSE exec -T postgres psql -U postgres -d rclone_backup << 'EOF'
-- Add missing 'used' column to registration_tokens table if it doesn't exist
ALTER TABLE registration_tokens 
ADD COLUMN IF NOT EXISTS used BOOLEAN DEFAULT FALSE NOT NULL;

-- Update the 'used' field based on 'used_at'
UPDATE registration_tokens 
SET used = TRUE 
WHERE used_at IS NOT NULL;

-- Create a function to automatically set 'used_at' when 'used' is set to true
CREATE OR REPLACE FUNCTION update_used_at()
RETURNS TRIGGER AS $$
BEGIN
    IF NEW.used = TRUE AND OLD.used = FALSE AND NEW.used_at IS NULL THEN
        NEW.used_at = NOW();
    END IF;
    RETURN NEW;
END;
$$ LANGUAGE plpgsql;

-- Create trigger to update used_at when used is set to true
DROP TRIGGER IF EXISTS update_registration_token_used_at ON registration_tokens;
CREATE TRIGGER update_registration_token_used_at
    BEFORE UPDATE ON registration_tokens
    FOR EACH ROW
    EXECUTE FUNCTION update_used_at();

-- 验证修复
\d registration_tokens
EOF

echo "✓ 数据库迁移完成"

echo ""
echo "正在重启 Hub API 服务..."
$DOCKER_COMPOSE restart hub-api

echo ""
echo "✓ 修复完成！"
echo ""
echo "现在你可以："
echo "1. 刷新浏览器页面"
echo "2. 尝试生成新的注册令牌"