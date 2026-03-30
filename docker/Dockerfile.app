# ===========================
# VoiceAssistant Backend
# Go + Gin
# ===========================
FROM golang:1.22-alpine AS builder

WORKDIR /app

# 设置 Go 代理（解决国内网络问题）
ENV GOPROXY=https://goproxy.cn,direct

# 安装依赖
COPY backend/go.mod backend/go.sum ./
RUN go mod download

# 复制源码和 vendor 目录
COPY backend/ ./

# 编译 Go 程序（使用 vendor）
RUN CGO_ENABLED=0 GOOS=linux GOARCH=arm64 go build -mod=vendor -ldflags="-s -w" -o server .

# ===========================
# 运行阶段
# ===========================
FROM alpine:3.19

# 安装运行时依赖
RUN apk add --no-cache ca-certificates tzdata curl

# 设置时区
ENV TZ=Asia/Shanghai

WORKDIR /app

# 创建配置目录
RUN mkdir -p /app/config

# 复制可执行文件和配置
COPY --from=builder /app/server .
COPY backend/config/config.yaml /app/config/config.yaml

# 暴露端口
EXPOSE 8080

# 启动命令
CMD ["./server"]
