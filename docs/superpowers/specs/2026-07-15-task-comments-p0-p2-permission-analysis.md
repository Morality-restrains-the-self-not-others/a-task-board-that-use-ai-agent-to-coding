# 角色权限分析：任务评论 P0–P2 硬化

- **日期**: 2026-07-15
- **关联设计**: `docs/superpowers/specs/2026-07-15-task-comments-sql-vs-nosql-scale-design.md`
- **python_api_approval**: n/a（零新增 Python 接口）

## 变更的接口与权限

| 接口 | 变更 | 鉴权 | 结论 |
|------|------|------|------|
| `GET .../todos/{id}/` | 响应增加 comments / ai_comments / container_agent_comments / comments_feed_errors | 既有 workspace 成员 token | 无提权；仅同任务可见数据 |
| `GET .../tasks/{id}/comments/?limit=&cursor=` | 可选分页包装 | 既有 | 无提权 |
| `GET .../ai-comments/?limit=&cursor=&preview=` | 分页 + preview | 既有 | 无提权；preview 默认减少正文暴露面（仍同权限） |
| `GET .../container-agent-comments` | 同上 | 既有 | 无提权 |
| `GET /api/internal/task-ai-comment/tasks/{id}/ai-comments` | **新增** internal | Internal secret | 仅服务间；禁止公网 |
| `GET /api/internal/.../container-agent-comments` | **新增** internal | Internal secret | 仅服务间 |
| Agent `/stream` | 批写实现变更 | 既有 container agent 鉴权 | 无权限语义变更 |

## 风险与缓解

- Internal 列表若无 secret：与现有 internal 路由一致拒绝。
- `comments_feed_errors` 不包含 token/正文，仅源标签与 HTTP 状态。

## 结论

批准按既有租户/工作空间 ACL 落地；无新公网 Python 接口。
