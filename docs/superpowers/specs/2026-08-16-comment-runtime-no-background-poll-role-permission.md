# 角色权限分析 — 评论运行态推送同步

- **日期**: 2026-08-16
- **设计**: `docs/superpowers/specs/2026-08-16-comment-runtime-no-background-poll-design.md`
- **结论**: 无新角色、无新 endpoint；沿用既有租户/工作空间/任务读权限。

## 改动点权限表

| 改动点 | 主体 | 资源层级 | 操作 | 现有检查 | 是否缺失 | 建议 |
|--------|------|----------|------|----------|----------|------|
| GET `server-runtime-status`（刷新按钮） | 工作空间成员 | Workspace + Task + comment_id | read | 既有 cloud compute 鉴权 + comment CSC 范围 | ✅ 充分 | 禁止无 comment_id 回落任务级 |
| SSE `server_status_update` / heartbeat | 任务详情订阅者 | Task（payload 含 comment_id） | read | SSE 任务订阅鉴权 | ✅ 充分 | 前端只 apply 匹配 comment_id |
| POST start-vm / stop-vm | 工作空间成员 | comment CSC | write | 既有 | ✅ 充分 | 命令回包可 apply，不启动 timer |
| 删除 FE/BE UI 轮询 | — | — | — | n/a | ✅ | 降低未授权 Describe 放大面 |

## 建模

不引入新角色。Describe 仍须带该面板 `comment_id`，避免 IDOR 刷到其它评论实例。
