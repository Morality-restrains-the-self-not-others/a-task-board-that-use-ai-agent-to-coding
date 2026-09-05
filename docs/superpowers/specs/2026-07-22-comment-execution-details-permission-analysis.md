# 角色权限分析：评论级执行细节

- **关联设计**: `2026-07-22-comment-execution-details-design.md`
- **日期**: 2026-07-22
- **状态**: approved（goal-mode）

## 结论

**无新敏感 API、无新权限点。** 本期为纯前端 UI 重组与展示契约；所有数据仍经既有任务详情读路径与容器 token 写路径。

## 权限矩阵

| 能力 | 角色 | 资源 | 操作 | 既有鉴权 | 状态 |
|------|------|------|------|----------|------|
| 读评论 Feed | 任务成员 | comments / container_agent | read | 任务详情访问权（tenant/workspace/task） | ✅ 不变 |
| 读容器连接/SSE | 任务成员 | server-startup-status-sse | read | 同上 + 任务 CSC 上下文 | ✅ 不变 |
| 读/操作 layer 图 | 任务成员 | layer graph / jobs | read/write | 既有 layer 命令与 job API | ✅ 不变；仅 UI 挂载点迁移 |
| 展示依赖模式 | 任务成员 | — | read（本地 state） | 无后端字段 | ✅ 无新闸 |
| 写 container_agent | 容器 access_token | complete / stream | write | 既有 token 归属校验 | ✅ 不变 |

## 风险与处置

| 风险 | 级别 | 处置 |
|------|------|------|
| 非成员窥视执行细节 | 低 | 仍受任务详情路由与 API 403 保护；无新泄露面 |
| active 评论误判导致 layer 面板错位 | 低 | 启发式可观测；仅影响 UI 挂载，不改授权 |
| 未来持久化 dependencyMode | 中（后续） | 须随 Go 编排一并做 task 成员 write 闸 |

## CRG 触点

- graph_status: n/a（纯前端重组）
- 敏感路径：无新增；沿用 `X-Access-Token`、任务 JWT
