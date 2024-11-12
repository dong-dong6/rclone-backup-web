#!/bin/bash

set -e

# 定义变量
APP_DIR="$HOME/rclone-backup-web"
FRONTEND_DIR="$APP_DIR/rclone-web"
BACKEND_DIR="$APP_DIR/rclone-go"
GO_VERSION="1.23.2"  # 根据需要调整Go版本
NODE_VERSION="20.15.0"     # 根据需要调整Node.js版本
NGINX_HTML_DIR="/usr/share/nginx/html"

echo "开始部署 rclone-backup-web 应用..."

# 更新系统并安装必要的软件
echo "更新系统并安装必要的软件..."
sudo apt-get update
sudo apt-get install -y git curl wget build-essential

# 检查并安装 rclone
if ! command -v rclone &> /dev/null; then
    echo "未检测到 rclone，正在安装 rclone..."
    sudo -v  # 确保使用 sudo 权限
    curl https://rclone.org/install.sh | sudo bash
else
    echo "rclone 已安装，跳过安装步骤。"
fi

# 安装 Go
if ! command -v go &> /dev/null; then
    echo "正在安装 Go $GO_VERSION..."
    wget https://golang.org/dl/go$GO_VERSION.linux-amd64.tar.gz -O /tmp/go$GO_VERSION.linux-amd64.tar.gz
    sudo tar -C /usr/local -xzf /tmp/go$GO_VERSION.linux-amd64.tar.gz
    echo "export PATH=\$PATH:/usr/local/go/bin" >> ~/.bashrc
    source ~/.bashrc
fi

# 安装 Node.js 和 npm
if ! command -v node &> /dev/null; then
    echo "正在安装 Node.js $NODE_VERSION..."
    curl -sL https://deb.nodesource.com/setup_$NODE_VERSION.x | sudo -E bash -
    sudo apt-get install -y nodejs
fi

# 克隆仓库
if [ ! -d "$APP_DIR" ]; then
    echo "克隆项目代码到 $APP_DIR..."
    git clone https://github.com/dong-dong6/rclone-backup-web.git $APP_DIR
else
    echo "项目已存在，拉取最新代码..."
    cd $APP_DIR
    git pull
fi

# 编译前端
echo "编译前端应用..."
cd $FRONTEND_DIR
npm install
npm run build

# 将打包文件移动到 Nginx 根目录
echo "将前端打包文件移动到 $NGINX_HTML_DIR..."
sudo mkdir -p $NGINX_HTML_DIR  # 确保目录存在
sudo rm -rf $NGINX_HTML_DIR/*  # 清空现有文件
sudo cp -r $FRONTEND_DIR/dist/* $NGINX_HTML_DIR/  # 复制打包的文件

# 编译后端
echo "编译后端应用..."
cd $BACKEND_DIR
go build -o app

# 配置和启动后端应用
echo "启动后端应用..."
pkill app || true  # 停止已有的后端应用
nohup $BACKEND_DIR/app > $BACKEND_DIR/app.log 2>&1 &

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
        root $NGINX_HTML_DIR;  # 修改根目录为 Nginx HTML 目录
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
