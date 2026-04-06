# VoiceAssistant Project

## 项目概述

VoiceAssistant 是一个基于 Go + Vue 3 的语音助手项目，采用 Agent Cluster 工作流进行开发。

---

## Agent 角色与职责

| 角色 | 工作目录 | 输出格式 | 核心职责 |
|------|---------|---------|---------|
| Product Manager (产品经理) | `doc/` | `xxxprd.md` | 分析需求，编写 PRD，协调产品专家评审 |
| Business Architect (业务架构师) | `doc/` | `xxxTec.md` | 基于 PRD 设计技术架构，协调架构专家评审 |
| Senior Backend Engineer | `backend/` | Go 代码 | 基于技术文档实现后端代码，**完成后必须触发 DevOps 重启 Docker** |
| Senior Frontend Engineer | `frontend/` | Vue 3 代码 | 基于技术文档实现前端 UI，**完成后必须触发 DevOps 重启 Docker** |
| Senior DevOps Engineer | `docker/` | Docker 配置 | **仅通过 Docker 启动服务**，配置容器编排 |
| Test Engineer | 项目根目录 | Bug 报告 | 收集日志、报告 bug、验证修复 |
| Senior Product Expert | `doc/` | 评审意见 | 评审 PRD，最多 3 轮反馈 |
| Senior Architecture Expert | `doc/` | 评审意见 | 评审技术文档，最多 3 轮反馈 |

---

## 关键规范

### 任务路由原则（强制执行）
**接到任何任务后，必须先分析是否应交给特定 Agent 执行：**
- PRD/需求分析 → Product Manager
- 技术架构设计 → Business Architect
- Go 后端代码/修复 → Senior Backend Engineer
- Vue 前端代码/修复 → Senior Frontend Engineer
- Docker/服务部署 → Senior DevOps Engineer
- Bug 报告/测试验证 → Test Engineer
- PRD 评审 → Senior Product Expert
- 技术文档评审 → Senior Architecture Expert
- SQL 评审/数据库操作 → DBA SQL Reviewer

**何时主 agent 直接执行**：简单文件编辑/阅读/搜索、分析性任务、多 Agent 协调调度。

### Docker 操作（DevOps）
- **禁止**在本地主机启动服务，必须通过 Docker
- 服务名：`backend`、`frontend`、`mysql`
- 常用命令：
  ```bash
  cd docker && docker-compose up -d        # 启动所有服务
  docker-compose logs -f [service_name]    # 查看日志
  docker-compose restart [service_name]    # 重启服务
  docker-compose down                      # 停止服务
  ```

### Go 依赖管理
- 使用 `go mod vendor` 管理第三方依赖
- 安装/更新依赖后执行：
  ```bash
  cd backend && go mod tidy && go mod vendor
  ```
- Dockerfile 使用 vendor 目录构建：`go build -mod=vendor -o server .`

### 文件命名规范
| 角色 | 输出格式 | 示例 |
|------|---------|------|
| Product Manager | `xxxprd.md` | `voice-activation-prd.md` |
| Business Architect | `xxxTec.md` | `voice-activationTec.md` |
| Bug Reports | `bug_{id}_{component}.md` | `bug_001_frontend.md` |

---

## 工作流程

### Path 1: 新需求开发
```
Product Manager → PRD 评审(最多3轮) → Business Architect → 技术评审(最多3轮)
    → Backend/Frontend 实现 → [自动] DevOps 重启 Docker → Test Engineer 验证 → 验收
```

### Path 2: Bug 处理
```
Test Engineer 收集 bug → 分发到相关工程师 → 修复 → [自动] DevOps 重启 Docker → Test Engineer 验证
```

---

## 触发关键词

| 关键词 | 触发路径 | 起始角色 |
|--------|---------|---------|
| `新需求` / `新功能` / `new feature` | Path 1 | Product Manager |
| `bug` / `错误` / `修复` | Path 2 | Test Engineer |

---

## 可用技能

### Backend
- `/backend-code-structure` - Go 后端代码结构规范
- `/table-design-convention` - 数据库表设计规范
- `/go-backend-testing` - Go 单元测试编写指南

### Frontend
- `/frontend-design` - 前端 UI/UX 设计指南
- `/frontend-api-integration` - 前端 API 集成规范

### 其他
- `/canvas-design` / `/web-artifacts-builder` - 可选前端技能
- MySQL MCP (`mcp__mysql__*`) - 数据库操作

---

## 完成标准

### Path 1 完成条件
- [ ] PRD 经产品专家评审通过
- [ ] 技术文档经架构专家评审通过
- [ ] 前后端代码实现完成
- [ ] Docker 部署成功
- [ ] 无关键 bug

### Path 2 完成条件
- [ ] Bug 准确定位
- [ ] 分发到正确的工程师
- [ ] 修复并部署
- [ ] 测试工程师验证通过

---

## 设计规范

### Design Context

**Users**: 需要即时记录灵感/日程的人群（创意工作者、职场人士），碎片化输入场景

**Brand Personality**: 温暖 · 高效 · 可靠

**Aesthetic Direction**: 暖色奶油风 (Warm Cream)，保持现有VoiceAssistant的UI风格一致性

**Design Principles**:
1. 即时反馈 - 每个操作都有明确的视觉/听觉反馈
2. 状态清晰 - 5种语音状态一眼可辨
3. 打断友好 - 用户可随时打断AI
4. 流畅动画 - 波形动画平滑，不干扰阅读
5. 温暖色调 - 橙色暖色系，降低认知负担

### 技术栈
- Vue 3 + TypeScript + Vite
- Tailwind CSS v4
- 主题色：#f97316 (orange)
- 背景色：#fef7ed, #fff7ed
- 字体：Inter, DM Sans
