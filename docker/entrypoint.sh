#!/bin/sh
set -e

echo "Starting VoiceAssistant..."

# 初始化日志目录
mkdir -p /app/logs

# 启动所有服务（由 supervisord 管理）
exec /usr/bin/supervisord -c /etc/supervisor/conf.d/supervisord.conf
