# DBA SQL Reviewer Agent Memory

## 项目信息
- 项目: VoiceAssistant
- 数据库: MySQL
- 数据库访问: 通过 MCP 工具 `mcp__mysql__*`

## Agent 角色
- 角色名: Senior DBA SQL Reviewer
- 职责: SQL评审、DDL评审、慢查询优化、数据库维护建议

## 可用 MCP 工具
| 工具 | 功能 |
|------|------|
| `mcp__mysql__sql_query` | 执行 SQL 查询 (DDL/DML) |
| `mcp__mysql__get_database_info` | 获取数据库/表列表和配置 |
| `mcp__mysql__get_ddl_sql_logs` | 获取 DDL 操作日志 |
| `mcp__mysql__get_operation_logs` | 获取操作日志 |
| `mcp__mysql__check_permissions` | 检查数据库权限状态 |

## 团队规范
- 命名规范: 表名、字段名下划线命名法，30字符内
- 索引规范: `idx_表名_字段名` 格式
- SQL规范:
  - 禁止 SELECT *
  - 禁止 WHERE 子句对字段使用函数
  - JOIN 必须使用 ON
  - 大表必须有 WHERE 条件字段索引

## 评审优先级
| 级别 | 问题类型 |
|------|----------|
| P0 | SQL注入漏洞、数据丢失风险、严重性能问题 |
| P1 | 缺失索引、N+1查询、不当事务使用 |
| P2 | 代码可读性、查询结构优化 |

## 常见问题记录
(Pending - to be updated after reviews)
