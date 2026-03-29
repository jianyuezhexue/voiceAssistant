#!/bin/sh
set -e

echo "Starting VoiceAssistant..."

# 初始化日志目录
mkdir -p /app/logs

# 检查是否启用开发模式（热更新）
if [ "${DEV_MODE}" = "true" ]; then
    echo "Running in DEV mode with hot reload enabled..."

    # 安装前端依赖（如果 node_modules 不存在）
    if [ ! -d "/app/frontend/node_modules" ]; then
        echo "Installing frontend dependencies..."
        cd /app/frontend && npm install
    fi

    # 安装后端热重载工具（air）
    if ! command -v air > /dev/null 2>&1; then
        echo "Installing air (Go hot reload)..."
        go install github.com/air-verse/air@latest
    fi

    # 启动所有服务
    exec /usr/bin/supervisord -c /etc/supervisor/conf.d/supervisord.conf
else
    echo "Running in PRODUCTION mode..."
    # 生产模式只启动后端和 nginx
    /app/backend/server &
    /usr/sbin/nginx -g "daemon off;" -c /etc/nginx/httpd.conf
fi
