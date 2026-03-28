#!/bin/bash

# NexCoreProxy Master 启动脚本

APP_NAME="NexCoreProxy Master"
APP_DIR="/root/projects/NexCoreProxy-Master"
PID_FILE="$APP_DIR/data/nexcore.pid"
LOG_FILE="$APP_DIR/logs/server.log"

cd $APP_DIR

# 创建日志目录
mkdir -p logs data

# 加载环境变量
if [ -f .env ]; then
    set -a
    source .env
    set +a
fi

# 默认值
DB_HOST=${DB_HOST:-127.0.0.1}
DB_PORT=${DB_PORT:-3307}
DB_USER=${DB_USER:-nexcore_proxy}
DB_PASS=${DB_PASS:-NexCore@2026!}
DB_NAME=${DB_NAME:-nexcore_proxy}
PORT=${PORT:-8084}

# 杀掉旧进程（强制重启）
kill_old_process() {
    # 通过 PID 文件杀
    if [ -f "$PID_FILE" ]; then
        OLD_PID=$(cat $PID_FILE 2>/dev/null)
        if [ -n "$OLD_PID" ] && ps -p $OLD_PID > /dev/null 2>&1; then
            echo "正在停止旧进程 (PID: $OLD_PID)..."
            kill $OLD_PID 2>/dev/null
            sleep 1
            # 如果还在运行，强制杀
            if ps -p $OLD_PID > /dev/null 2>&1; then
                kill -9 $OLD_PID 2>/dev/null
                sleep 1
            fi
        fi
        rm -f $PID_FILE
    fi
    
    # 通过进程名杀（兜底，确保没有残留）
    pkill -f "nexcore-server" 2>/dev/null
    sleep 1
}

# 自动关闭旧进程
kill_old_process

# 启动服务
echo "正在启动 $APP_NAME..."
./nexcore-server \
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
    tail -20 $LOG_FILE
    exit 1
fi