# 「改后执行」自动交付 PR — 角色权限分析

- **日期**: 2026-07-23
- **设计**: `2026-07-23-edit-run-auto-delivery-pr-comment-design.md`

## 结论

| 路径 | 鉴权 | 变更 |
|------|------|------|
| `POST …/container-job-edit-run/` | 既有用户 Token（Gateway） | 请求体增可选字段，无权限模型变化 |
| 容器 createJob / 交付 | 容器 Access-Token → Cloud/OAuth | 不变 |
| `POST …/container-agent-comments/` | **新增**与 stream/complete 一致的 `X-Access-Token` 校验 | 仅同任务 scope；不扩大跨租户 |
| `/complete` 回填 | 既有 Access-Token | 不变 |

无新角色、无匿名写评论；创建 Agent 仍受「同一 parent 仅一活跃 run」约束。
