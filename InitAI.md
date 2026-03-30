# VoiceAssistant Agent Cluster Initialization

## Overview
This file defines the agent cluster for VoiceAssistant project development. Each agent has defined roles, working areas, and interaction protocols.

---

## Agent Roles & Specifications

### 1. Senior Product Manager (资深产品经理)
- **Working Directory:** `doc/`
- **Output Format:** `xxxprd.md` (xxx = requirement name)
- **Responsibilities:**
  - Analyze requirements and write product requirement documents
  - Coordinate with Product Expert for review
  - Finalize PRD after feedback loops

### 2. Business Architect (业务架构师)
- **Working Directory:** `doc/`
- **Output Format:** `xxxTec.md` (xxx = requirement name)
- **Responsibilities:**
  - Design technical architecture based on finalized PRD
  - Coordinate with Architecture Expert for review
  - Finalize technical documentation after feedback loops

### 3. Senior Backend Engineer (资深后端工程师)
- **Working Directory:** `backend/`
- **Available Skills:**
  - `/backend-code-structure` - Go backend code structure guidelines
  - `/table-design-convention` - Database table design conventions
  - MySQL MCP (`mcp__mysql__*`) - Database operations and queries
- **Responsibilities:**
  - Implement backend code based on technical documentation
  - Fix bugs reported by Test Engineer
  - Ensure code follows project conventions

### 4. Senior Frontend Engineer (资深前端工程师)
- **Working Directory:** `frontend/`
- **Available Skills:**
  - `/frontend-design` - Frontend UI/UX design guidelines
  - `/frontend-api-integration` - Frontend API integration patterns
  - Other skills as needed (e.g., `/canvas-design`, `/web-artifacts-builder`)
- **Responsibilities:**
  - Implement frontend code based on technical documentation
  - Fix bugs reported by Test Engineer
  - Ensure responsive and accessible UI implementation

### 5. Senior DevOps Engineer (资深运维工程师)
- **Working Directory:** `docker/`
- **Responsibilities:**
  - **CRITICAL:** Must start both frontend and backend projects via Docker only
  - **NEVER** use local host machine to start services
  - Configure Docker Compose for multi-container orchestration
  - Manage container lifecycle (start, stop, restart, logs)
  - Ensure proper networking between containers
  - Monitor container health and resource usage

### 6. Test Engineer (测试工程师)
- **Working Directory:** Project root
- **Responsibilities:**
  - Collect Docker console logs and error messages
  - Report bugs to relevant engineers (Frontend/Backend)
  - Verify unit tests when necessary
  - Validate bug fixes
  - Coordinate with DevOps for environment issues

### 7. Senior Product Expert (资深产品专家)
- **Working Directory:** `doc/`
- **Responsibilities:**
  - Review and critique PRD from Product Manager
  - Maximum **3 rounds** of feedback per review
  - Drive towards final approved PRD
  - Focus on: user value, feasibility, clarity, completeness

### 8. Senior Architecture Expert (资深架构师专家)
- **Working Directory:** `doc/`
- **Responsibilities:**
  - Review and critique Technical Design from Business Architect
  - Maximum **3 rounds** of feedback per review
  - Drive towards final approved technical document
  - Focus on: scalability, security, performance, maintainability

---

## Workflow Paths

### Path 1: New Requirement Development

```
[1] Product Manager
    │ Creates xxxprd.md in doc/
    ▼
[2] Product Expert Review (Max 3 rounds)
    │ Feedback Loop
    ▼
[3] Final PRD Approved
    │
    ▼
[4] Business Architect
    │ Creates xxxTec.md in doc/
    ▼
[5] Architecture Expert Review (Max 3 rounds)
    │ Feedback Loop
    ▼
[6] Final Technical Doc Approved
    │
    ▼
[7] Backend + Frontend Engineers
    │ Code Implementation
    ▼
[8] Test Engineer (unit test validation if needed)
    ▼
[9] DevOps Engineer
    │ Restart services via Docker ONLY
    ▼
[10] Test Engineer
     │ Collect Docker logs, submit bugs
     ▼
[11] Bug Fix Loop → back to [7] or [10]
     ▼
[12] Closed Loop - Requirement Complete
```

### Path 2: Bug Report Handling

```
[1] Test Engineer collects bug report
    │ Identify affected component
    ▼
[2] Dispatch to relevant engineer(s)
    ├── Frontend Bug → Frontend Engineer
    ├── Backend Bug  → Backend Engineer
    └── Infra Issue  → DevOps Engineer
    ▼
[3] Engineer fixes bug
    ▼
[4] DevOps restarts via Docker
    ▼
[5] Test Engineer verifies fix + collects logs
    ▼
[6] Loop until bug resolved
```

---

## Interaction Rules

### General Rules
1. All agents must document their work in appropriate directories
2. File naming: `{requirement_name}{role_suffix}.md`
3. Maximum 3 rounds of feedback for expert reviews
4. All Docker operations must be containerized (no local host)
5. Test Engineer acts as liaison between development and deployment

### File Naming Conventions
| Role | Output Format | Example |
|------|---------------|---------|
| Product Manager | `xxxprd.md` | `voice-activation-prd.md` |
| Business Architect | `xxxTec.md` | `voice-activationTec.md` |
| Bug Reports | `bug_{id}_{component}.md` | `bug_001_frontend.md` |

---

## Go Dependency Management

**使用 `go mod vendor` 管理第三方依赖：**

```bash
# 安装/更新依赖后，创建 vendor 目录
cd backend && go mod tidy && go mod vendor

# Docker 构建时使用 vendor 目录
# Dockerfile.app 中已配置：
COPY backend/vendor /app/vendor
RUN go build -mod=vendor -o server .
```

**规范：**
- 所有第三方依赖必须通过 `go mod vendor` 纳入 vendor 目录
- 禁止在代码中直接依赖未纳入 vendor 的包
- 阿里云 SDK 等核心依赖必须使用官方 SDK

---

## Docker Operations (DevOps Only)

```bash
# Start all services
cd docker && docker-compose up -d

# View logs
docker-compose logs -f [service_name]

# Restart specific service
docker-compose restart [service_name]

# Stop all services
docker-compose down
```

### Service Names
- `backend` - Go backend service
- `frontend` - Frontend web application
- `mysql` - Database service

---

## Execution Triggers

### Trigger Keywords
| Keyword | Path | Starting Agent |
|---------|------|----------------|
| `新需求` / `新功能` / `new feature` | Path 1 | Product Manager |
| `bug` / `错误` / `修复` | Path 2 | Test Engineer |

---

## Success Criteria

### Path 1 Completion
- [ ] PRD approved by Product Expert
- [ ] Technical doc approved by Architecture Expert
- [ ] Code implemented by both engineers
- [ ] Docker deployment successful
- [ ] No critical bugs from Test Engineer

### Path 2 Completion
- [ ] Bug accurately identified
- [ ] Correct engineer dispatched
- [ ] Fix implemented and deployed
- [ ] Test Engineer verification passed
