#!/bin/bash

# NexCore代理主机 停止脚本

APP_NAME="NexCore代理主机"
APP_DIR="/root/projects/NexCoreProxy-Master"
PID_FILE="$APP_DIR/data/nexcore.pid"

cd $APP_DIR

if [ ! -f "$PID_FILE" ]; then
    echo "$APP_NAME 未运行"
    exit 0
fi

PID=$(cat $PID_FILE)

if ps -p $PID > /dev/null 2>&1; then
    echo "正在停止 $APP_NAME (PID: $PID)..."
    kill $PID
    sleep 2
    
    if ps -p $PID > /dev/null 2>&1; then
        echo "强制停止..."
        kill -9 $PID
    fi
    
    rm -f $PID_FILE
    echo "$APP_NAME 已停止"
else
    echo "$APP_NAME 未运行"
    rm -f $PID_FILE
fi