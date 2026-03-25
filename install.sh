#!/bin/bash

# NexCore代理主机 一键安装脚本
# 使用方法: 
#   bash <(curl -Ls https://raw.githubusercontent.com/DoBestone/NexCoreProxy-Master/main/install.sh) -p 8082 -u admin -pass your_password
#   bash <(curl -Ls https://raw.githubusercontent.com/DoBestone/NexCoreProxy-Master/main/install.sh) --port 8082 --user admin --password your_password

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
ADMIN_USER="admin"
ADMIN_PASS=""

# 解析参数
while [[ $# -gt 0 ]]; do
    case $1 in
        -p|--port)
            WEB_PORT="$2"
            shift 2
            ;;
        -u|--user)
            ADMIN_USER="$2"
            shift 2
            ;;
        -pass|--password)
            ADMIN_PASS="$2"
            shift 2
            ;;
        --db-host)
            DB_HOST="$2"
            shift 2
            ;;
        --db-port)
            DB_PORT="$2"
            shift 2
            ;;
        --db-user)
            DB_USER="$2"
            shift 2
            ;;
        --db-pass)
            DB_PASS="$2"
            shift 2
            ;;
        --db-name)
            DB_NAME="$2"
            shift 2
            ;;
        -h|--help)
            echo "使用方法: $0 [选项]"
            echo ""
            echo "选项:"
            echo "  -p, --port PORT       Web服务端口 (默认: 8082)"
            echo "  -u, --user USER       管理员用户名 (默认: admin)"
            echo "  -pass, --password     管理员密码 (必填)"
            echo "  --db-host HOST        数据库主机 (默认: localhost)"
            echo "  --db-port PORT        数据库端口 (默认: 3306)"
            echo "  --db-user USER        数据库用户 (默认: root)"
            echo "  --db-pass PASS        数据库密码"
            echo "  --db-name NAME        数据库名 (默认: nexcore)"
            echo ""
            echo "示例:"
            echo "  $0 -p 8082 -u admin -pass MyPassword123"
            echo "  $0 --port 9000 --user admin --password SecretPass --db-pass dbpass"
            exit 0
            ;;
        *)
            shift
            ;;
    esac
done

echo -e "${green}========================================${plain}"
echo -e "${green}  $APP_NAME 安装脚本${plain}"
echo -e "${green}========================================${plain}"

# 检查必填参数
if [[ -z "$ADMIN_PASS" ]]; then
    echo -e "${red}错误: 请通过 -pass 或 --password 参数设置管理员密码${plain}"
    echo ""
    echo "示例: $0 -pass YourPassword123"
    exit 1
fi

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
echo "端口: $WEB_PORT"
echo "管理员: $ADMIN_USER"
echo "数据库: $DB_HOST:$DB_PORT/$DB_NAME"

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
EnvironmentFile=$APP_DIR/.env
ExecStart=$APP_DIR/nexcore \
    -port $WEB_PORT \
    -db-host \$DB_HOST \
    -db-port \$DB_PORT \
    -db-user \$DB_USER \
    -db-pass "\$DB_PASS" \
    -db-name \$DB_NAME
Restart=on-failure
RestartSec=5s

[Install]
WantedBy=multi-user.target
EOF

# 初始化管理员密码（修改 user.go 的 InitAdmin）
# 这里通过环境变量传递
export NCP_ADMIN_USER="$ADMIN_USER"
export NCP_ADMIN_PASS="$ADMIN_PASS"

# 启动服务
systemctl daemon-reload
systemctl enable nexcore
systemctl start nexcore

# 检查状态
sleep 3
if systemctl is-active --quiet nexcore; then
    SERVER_IP=$(curl -s ifconfig.me || curl -s ip.sb || echo "YOUR_IP")
    
    echo ""
    echo -e "${green}========================================${plain}"
    echo -e "${green}  安装成功!${plain}"
    echo -e "${green}========================================${plain}"
    echo ""
    echo -e "访问地址: ${green}http://${SERVER_IP}:${WEB_PORT}${plain}"
    echo -e "管理员账号: ${green}${ADMIN_USER}${plain}"
    echo -e "管理员密码: ${green}${ADMIN_PASS}${plain}"
    echo ""
    echo -e "管理命令:"
    echo "  启动: systemctl start nexcore"
    echo "  停止: systemctl stop nexcore"
    echo "  状态: systemctl status nexcore"
    echo "  日志: journalctl -u nexcore -f"
    echo ""
else
    echo -e "${red}安装失败，请查看日志: journalctl -u nexcore${plain}"
    journalctl -u nexcore --no-pager -n 50
    exit 1
fi