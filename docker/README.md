# VoiceAssistant Docker 部署指南

## 目录结构

```
docker/
├── Dockerfile        # 生产环境构建
├── Dockerfile.dev   # 开发环境构建（包含热更新工具）
├── docker-compose.yml       # 基础编排（支持 dev/prod profiles）
├── docker-compose.dev.yml   # 开发专用编排
├── docker-compose.prod.yml  # 生产专用编排
├── nginx.conf       # Nginx 配置
├── supervisord.conf # 进程管理配置
├── entrypoint.sh    # 容器启动脚本
└── .dockerignore    # Docker 构建忽略
```

## 快速开始

### 开发模式（热更新）

```bash
# 启动开发环境（前后端热更新）
docker compose --profile dev up

# 或使用专用开发配置
docker compose -f docker/docker-compose.dev.yml up -d
```

访问：
- 前端: http://localhost
- 后端 API: http://localhost:8080
- Vite HMR: http://localhost:5173

### 生产模式

```bash
docker compose --profile prod up -d
```

## 热更新实现原理

### 前端热更新 (Vite HMR)

1. **开发模式**: 源码通过 volume 挂载到容器
2. **Vite HMR**: 检测文件变化，WebSocket 推送更新
3. **浏览器**: 无刷新更新模块

```yaml
volumes:
  - ../frontend:/app/frontend   # 源码挂载
  - ../frontend/node_modules:/app/frontend/node_modules:rw  # 依赖不挂载
```

### 后端热更新 (Air)

1. **开发模式**: 源码通过 volume 挂载到容器
2. **Air**: 监听文件变化，自动重新编译并重启
3. **无需重启容器**: 进程自动重载

```bash
# Air 配置文件（.air.toml）可选
# 容器内已预装 air，监听 .go 文件变化
```

## 手动构建

```bash
# 构建生产镜像
docker build -f docker/Dockerfile -t voice-assistant:latest .

# 构建开发镜像
docker build -f docker/Dockerfile.dev -t voice-assistant:dev .
```

## 环境变量

| 变量 | 说明 | 默认值 |
|------|------|--------|
| DEV_MODE | 开发/生产模式 | false |
| TZ | 时区 | Asia/Shanghai |

## 进程管理

使用 `supervisord` 管理多进程：

- **backend**: Go 服务（端口 8080）
- **nginx**: 反向代理（端口 80）
- **vite**: 前端开发服务器（开发模式，端口 5173）

## 日志

日志挂载在 `./logs` 目录：

```
logs/
├── supervisord.log
├── backend.out.log
├── backend.err.log
├── nginx.out.log
├── nginx.err.log
├── vite.out.log
└── vite.err.log
```

## 注意事项

1. **node_modules**: 不挂载到源码，保持容器内独立
2. **后端依赖**: 通过 `go mod download` 在构建时缓存
3. **端口占用**: 确保 80/8080/5173 端口未被占用
