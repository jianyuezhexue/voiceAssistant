#!/usr/bin/env bash
# 本地开发环境启动脚本：前后端独立容器 + 热重载
# 默认前台运行（看 HMR 日志），可传参覆盖，如：./run-dev.sh -d （后台） / down
set -euo pipefail

DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
cd "$DIR/docker"

# 无参数时默认 up（前台），有参数则透传（如 -d / down / logs）
if [ $# -eq 0 ]; then
  docker-compose -f docker-compose.dev.yaml up
else
  docker-compose -f docker-compose.dev.yaml "$@"
fi