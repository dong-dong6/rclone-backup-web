#!/bin/bash

# 更新包管理器索引
echo "更新包管理器索引..."
sudo apt-get update

# 安装必要的软件包列表
packages=("p7zip-full" "p7zip-rar" "rclone" "tar" "curl" "cpulimit")

# 循环安装各个软件包
for package in "${packages[@]}"; do
  echo "检查并安装 $package ..."
  if ! dpkg -s "$package" >/dev/null 2>&1; then
    sudo apt-get install -y "$package"
    if [ $? -eq 0 ]; then
      echo "$package 安装成功"
    else
      echo "$package 安装失败，请检查网络或源配置"
      exit 1
    fi
  else
    echo "$package 已安装，跳过..."
  fi
done

# 验证rclone配置
if ! rclone config file >/dev/null 2>&1; then
  echo "rclone 未配置，请使用 'rclone config' 命令进行配置"
else
  echo "rclone 已配置"
fi

echo "所有必要的软件包已安装完毕！"
