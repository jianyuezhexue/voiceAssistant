#!/usr/bin/env bash
# 生产环境启动脚本：前后端一体化单容器（构建并后台启动）
# 默认 up -d --build，可传参覆盖，如：./run-prod.sh down / logs -f / restart
set -euo pipefail

DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"

# 预先同步 Go 依赖：确保 Dockerfile.backend 的 vendor 是最新版本，
# 避免生产构建因 go.mod / vendor 陈旧而失败。失败也不中断（best-effort）。
cd "$DIR/app" && go mod tidy && go mod vendor
cd "$DIR/docker"

# 无参数时默认构建并后台启动，有参数则透传
if [ $# -eq 0 ]; then
  docker compose up -d --build
else
  docker compose "$@"
fi