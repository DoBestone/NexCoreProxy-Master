#!/bin/bash

# NexCore代理主机 一键安装脚本
# 使用方法: bash <(curl -Ls https://raw.githubusercontent.com/DoBestone/NexCoreProxy-Master/main/install.sh)

set -e

red='\033[0;31m'
green='\033[0;32m'
yellow='\033[0;33m'
plain='\033[0m'

APP_NAME="NexCore代理主机"
APP_DIR="/opt/nexcore"
REPO_URL="https://github.com/DoBestone/NexCoreProxy-Master"

# 默认配置
DB_HOST="localhost"
DB_PORT="3306"
DB_USER="root"
DB_PASS=""
DB_NAME="nexcore"
WEB_PORT="8082"

echo -e "${green}========================================${plain}"
echo -e "${green}  $APP_NAME 安装脚本${plain}"
echo -e "${green}========================================${plain}"

# 检查 root
if [[ $EUID -ne 0 ]]; then
    echo -e "${red}请使用 root 用户运行此脚本${plain}"
    exit 1
fi

# 检查系统
if [[ -f /etc/os-release ]]; then
    . /etc/os-release
    OS=$ID
else
    echo -e "${red}无法检测系统版本${plain}"
    exit 1
fi

echo "系统: $OS"

# 安装依赖
echo -e "${yellow}安装依赖...${plain}"
if [[ "$OS" == "ubuntu" || "$OS" == "debian" ]]; then
    apt update && apt install -y wget curl git mysql-client
elif [[ "$OS" == "centos" ]]; then
    yum install -y wget curl git mysql
else
    echo -e "${red}不支持的系统: $OS${plain}"
    exit 1
fi

# 检查 Go
if ! command -v go &> /dev/null; then
    echo -e "${yellow}安装 Go...${plain}"
    wget -q https://go.dev/dl/go1.22.0.linux-amd64.tar.gz
    rm -rf /usr/local/go
    tar -C /usr/local -xzf go1.22.0.linux-amd64.tar.gz
    export PATH=$PATH:/usr/local/go/bin
    echo 'export PATH=$PATH:/usr/local/go/bin' >> /etc/profile
    rm go1.22.0.linux-amd64.tar.gz
fi

echo "Go 版本: $(go version)"

# 创建目录
mkdir -p $APP_DIR
cd $APP_DIR

# 克隆代码
if [[ -d ".git" ]]; then
    echo -e "${yellow}更新代码...${plain}"
    git pull
else
    echo -e "${yellow}克隆代码...${plain}"
    git clone $REPO_URL .
fi

# 配置数据库
echo ""
echo -e "${yellow}数据库配置${plain}"
read -p "数据库主机 [localhost]: " input_db_host
DB_HOST=${input_db_host:-$DB_HOST}

read -p "数据库端口 [3306]: " input_db_port
DB_PORT=${input_db_port:-$DB_PORT}

read -p "数据库用户 [root]: " input_db_user
DB_USER=${input_db_user:-$DB_USER}

read -p "数据库密码: " input_db_pass
DB_PASS=${input_db_pass:-$DB_PASS}

read -p "数据库名 [nexcore]: " input_db_name
DB_NAME=${input_db_name:-$DB_NAME}

read -p "Web端口 [8082]: " input_web_port
WEB_PORT=${input_web_port:-$WEB_PORT}

# 创建数据库
echo -e "${yellow}创建数据库...${plain}"
mysql -h$DB_HOST -P$DB_PORT -u$DB_USER -p$DB_PASS -e "CREATE DATABASE IF NOT EXISTS $DB_NAME CHARACTER SET utf8mb4 COLLATE utf8mb4_unicode_ci;" 2>/dev/null || {
    echo -e "${red}数据库连接失败，请检查配置${plain}"
    exit 1
}

# 写入配置
cat > .env << EOF
DB_HOST=$DB_HOST
DB_PORT=$DB_PORT
DB_USER=$DB_USER
DB_PASS=$DB_PASS
DB_NAME=$DB_NAME
PORT=$WEB_PORT
EOF

# 编译
echo -e "${yellow}编译程序...${plain}"
go mod tidy
go build -o nexcore .

# 设置权限
chmod +x nexcore start.sh stop.sh

# 创建 systemd 服务
cat > /etc/systemd/system/nexcore.service << EOF
[Unit]
Description=NexCore Proxy Master
After=network.target mysql.service

[Service]
Type=simple
WorkingDirectory=$APP_DIR
ExecStart=$APP_DIR/nexcore -port $WEB_PORT -db-host $DB_HOST -db-port $DB_PORT -db-user $DB_USER -db-pass "$DB_PASS" -db-name $DB_NAME
Restart=on-failure
RestartSec=5s

[Install]
WantedBy=multi-user.target
EOF

# 启动服务
systemctl daemon-reload
systemctl enable nexcore
systemctl start nexcore

# 检查状态
sleep 2
if systemctl is-active --quiet nexcore; then
    SERVER_IP=$(curl -s ifconfig.me || curl -s ip.sb || echo "YOUR_IP")
    
    echo ""
    echo -e "${green}========================================${plain}"
    echo -e "${green}  安装成功!${plain}"
    echo -e "${green}========================================${plain}"
    echo ""
    echo -e "访问地址: ${green}http://${SERVER_IP}:${WEB_PORT}${plain}"
    echo -e "默认账号: ${green}admin${plain}"
    echo -e "默认密码: ${green}admin123${plain}"
    echo ""
    echo -e "管理命令:"
    echo "  启动: systemctl start nexcore"
    echo "  停止: systemctl stop nexcore"
    echo "  状态: systemctl status nexcore"
    echo "  日志: journalctl -u nexcore -f"
    echo ""
else
    echo -e "${red}安装失败，请查看日志: journalctl -u nexcore${plain}"
    exit 1
fi