# ===========================
# VoiceAssistant Dockerfile
# 前端 Vue3 + Vite
# 后端 Go + Gin
# Nginx 反向代理
# ===========================

# ----------------------------
# 第一阶段：前端构建
# ----------------------------
FROM node:20-alpine AS frontend-builder

WORKDIR /app/frontend

# 安装依赖（利用缓存）
COPY frontend/package.json frontend/package-lock.json* ./
RUN npm ci

# 复制源码
COPY frontend/ ./

# 构建生产版本
RUN npm run build

# ----------------------------
# 第二阶段：后端构建
# ----------------------------
FROM golang:1.22-alpine AS backend-builder

WORKDIR /app/backend

# 设置 Go 代理（解决国内网络问题）
ENV GOPROXY=https://goproxy.cn,direct

# 安装 Go 依赖
COPY backend/go.mod backend/go.sum ./
RUN go mod download

# 复制源码
COPY backend/ ./

# 编译 Go 程序
# 注意：确保架构与运行时 alpine 一致
RUN CGO_ENABLED=0 GOOS=linux GOARCH=arm64 go build -ldflags="-s -w" -o server .

# ----------------------------
# 第三阶段：运行阶段
# ----------------------------
FROM alpine:3.19 AS runtime

# 安装运行时依赖
RUN apk add --no-cache \
    nginx \
    supervisor \
    curl \
    ca-certificates \
    tzdata

# 设置时区
ENV TZ=Asia/Shanghai

# 创建目录结构
RUN mkdir -p /app/logs /app/frontend/dist /app/backend

# 复制前端构建产物
COPY --from=frontend-builder /app/frontend/dist /app/frontend/dist

# 复制后端可执行文件
COPY --from=backend-builder /app/backend/server /app/backend/server

# 复制配置文件
COPY docker/nginx.conf /etc/nginx/http.d/default.conf
COPY docker/supervisord.conf /etc/supervisor/conf.d/supervisord.conf
COPY docker/entrypoint.sh /entrypoint.sh

# 设置执行权限
RUN chmod +x /entrypoint.sh

# 暴露端口
EXPOSE 80 8080

# 启动脚本
ENTRYPOINT ["/entrypoint.sh"]
