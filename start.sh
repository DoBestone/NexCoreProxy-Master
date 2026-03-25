#!/bin/bash

# NexCore代理主机 启动脚本

APP_NAME="NexCore代理主机"
APP_DIR="/root/projects/NexCoreProxy-Master"
PID_FILE="$APP_DIR/data/nexcore.pid"
LOG_FILE="$APP_DIR/data/nexcore.log"

cd $APP_DIR

# 创建数据目录
mkdir -p data

# 加载环境变量
if [ -f .env ]; then
    export $(cat .env | grep -v '^#' | xargs)
fi

# 默认值
DB_HOST=${DB_HOST:-127.0.0.1}
DB_PORT=${DB_PORT:-3307}
DB_USER=${DB_USER:-root}
DB_PASS=${DB_PASS:-}
DB_NAME=${DB_NAME:-nexcore}
PORT=${PORT:-8082}

# 检查是否已经运行
if [ -f "$PID_FILE" ]; then
    PID=$(cat $PID_FILE)
    if ps -p $PID > /dev/null 2>&1; then
        echo "$APP_NAME 已经在运行 (PID: $PID)"
        exit 0
    fi
fi

# 启动服务
echo "正在启动 $APP_NAME..."
./nexcore \
    -port $PORT \
    -db-host $DB_HOST \
    -db-port $DB_PORT \
    -db-user $DB_USER \
    -db-pass "$DB_PASS" \
    -db-name $DB_NAME \
    > $LOG_FILE 2>&1 &

echo $! > $PID_FILE

sleep 2

if ps -p $(cat $PID_FILE) > /dev/null 2>&1; then
    echo "$APP_NAME 启动成功 (PID: $(cat $PID_FILE))"
    echo "访问地址: http://localhost:$PORT"
    echo "默认账号: admin"
    echo "默认密码: admin123"
else
    echo "启动失败，请查看日志: $LOG_FILE"
    cat $LOG_FILE
    exit 1
fi