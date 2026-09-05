# 权限分析：评论启动日志 COS 归档

- **Date:** 2026-08-27
- **Design:** `docs/superpowers/specs/2026-08-27-startup-logs-cos-archive-design.md`

## 结论

无新对外写接口。读路径沿用既有 comment-container-bindings list（工作空间成员）。管理员 COS 配置沿用平台员工闸门。COS 对象键含 workspace/task/comment，禁止跨租户列举。

| 改动点 | 主体 | 资源层级 | 操作 | 现有检查 | 是否缺失 | 建议 |
|--------|------|----------|------|----------|----------|------|
| insertCCBLogRow → COS Put | 内部（Cloud 写路径） | Workspace/Comment | write | 绑定创建已校验租户/任务 | ✅ | 对象键只用已解析 workspace_id |
| list bindings logs（hydrate COS） | workspace 成员 | Workspace | read | handleCloudTaskRoutes + 租户/任务 | ✅ | 指针查询带 workspace_id+task_id+company_id |
| GET/PATCH `/api/system-admin/step-full-cos/` 增字段 `startupLogsPathRule` | 平台员工 | System | manage | `authz.IsPlatformStaff` | ✅ | 密钥仍不回显；pathRule 拒 `..` |
| Kafka CommentStartupLogArchived | 无自动消费者 | — | — | DLT 规则 N/A（publish-only） | ✅ | 不注册 intent 消费者 |

## 角色建模

不引入新角色。

## IDOR

- 禁止用客户端传入的任意 object_key 读取 COS。
- hydrate 仅按当前 list 的 workspace/task 解析确定性 key 或指针表行。

## 审计

结构化日志只记 object_key / bytes / comment_id，禁止 SecretId/SecretKey/PII。
