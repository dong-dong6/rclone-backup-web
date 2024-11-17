#!/bin/bash

set -e

# 定义变量
APP_DIR="/opt/rclone-backup-web"
GITHUB_REPO="dong-dong6/rclone-backup-web"

# 获取最新发布版本
echo "获取最新发布版本..."
LATEST_TAG=$(curl -s "https://api.github.com/repos/$GITHUB_REPO/releases/latest" | jq -r .tag_name)

if [ -z "$LATEST_TAG" ]; then
  echo "无法获取最新版本信息！"
  exit 1
fi

echo "最新版本：$LATEST_TAG"
# 克隆项目代码并更新
if [ ! -d "$APP_DIR" ]; then
    echo "项目目录不存在，创建并解压..."
    mkdir -p "$APP_DIR"
else
    echo "项目目录已存在，清空并更新..."
    find "$APP_DIR" -mindepth 1 ! -name 'Bash' ! -name 'users.json' -exec rm -rf {} +
fi
# 下载最新版本的 tar.gz 包
DOWNLOAD_URL="https://github.com/$GITHUB_REPO/releases/download/$LATEST_TAG/rclone-backup-web-$LATEST_TAG.tar"
TEMP_DIR=$(mktemp -d)

echo "下载并解压最新版本..."
curl -L "$DOWNLOAD_URL" -o "$TEMP_DIR/rclone-backup-web.tar"
tar -xvf "$TEMP_DIR/rclone-backup-web.tar" -C "$APP_DIR"

rm -rf $TEMP_DIR
# 配置和启动后端应用
echo "启动后端应用..."
pkill rclone-backup-app || true  # 停止已有的后端应用
cd $APP_DIR && nohup ./rclone-backup-app > app.log 2>&1 &

# 检查是否安装了 Nginx
if ! command -v nginx &> /dev/null; then
    echo "未检测到 Nginx。"
    PS3="请选择是否安装 Nginx："
    select install_nginx in "是" "否"; do
        case $install_nginx in
            是)
                echo "正在安装 Nginx..."
                sudo apt-get install -y nginx
                break
                ;;
            否)
                echo "由于未安装 Nginx，前端将无法通过80端口访问。请自行配置前端服务。"
                # 不退出脚本，继续执行后续操作
                break
                ;;
            *)
                echo "无效选择，请选择 '是' 或 '否'。"
                ;;
        esac
    done
fi

# 让用户选择域名
read -p "请输入您的域名（例如: www.example.com）: " DOMAIN

# 验证域名是否为空
if [ -z "$DOMAIN" ]; then
    echo "域名不能为空！"
    exit 1
fi

# 配置 Nginx
echo "配置 Nginx..."
sudo tee /etc/nginx/sites-available/rclone-backup-web <<EOF > /dev/null
server {
    listen 80;
    server_name $DOMAIN;

    location /api/ {
        proxy_pass http://127.0.0.1:628/;
        proxy_set_header Host \$host;
        proxy_set_header X-Real-IP \$remote_addr;
    }

    location / {
        root $APP_DIR/web;
        index index.html index.htm;
        try_files \$uri \$uri/ /index.html;
    }
}
EOF

sudo ln -sf /etc/nginx/sites-available/rclone-backup-web /etc/nginx/sites-enabled/
sudo nginx -t

echo "重新启动 Nginx..."
sudo systemctl restart nginx

echo "部署完成！请在浏览器中访问您的域名 ($DOMAIN) 或服务器 IP 地址查看应用。"

