#!/bin/sh
set -e

echo "Starting VoiceAssistant..."

# 初始化日志目录
mkdir -p /app/logs

# 如果存在源码挂载点，创建符号链接
if [ -d "/app/backend/source" ] && [ ! -f "/app/backend/server" ]; then
    echo "Development mode: source mounted, backend binary not available"
fi

# 启动所有服务（由 supervisord 管理）
exec /usr/bin/supervisord -c /etc/supervisor/conf.d/supervisord.conf
