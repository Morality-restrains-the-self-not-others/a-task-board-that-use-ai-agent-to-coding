# 权限分析：工作面板过滤选项持久化

- 日期：2026-07-13
- 设计：`docs/superpowers/specs/2026-07-13-work-panel-filter-persistence-design.md`

## 端点权限矩阵

| 改动点 | 主体 | 资源层级 | 操作 | 现有检查 | 是否缺失 | 建议 |
|--------|------|----------|------|----------|----------|------|
| GET work-panel-filters | 已认证用户（本人） | User×Workspace | read | 网关 token + 工作空间属租户 | ✅ | user_id 强制来自 auth header |
| PUT work-panel-filters | 已认证用户（本人） | User×Workspace | write | 同上 | ✅ | 禁止 body 指定 user_id；upsert 仅本人行 |

## IDOR / 越权

| 风险 | 缓解 |
|------|------|
| 读他人偏好 | PK 含 auth user_id；不接受 query/body 覆盖 |
| 写他人偏好 | 同上 |
| 跨租户 workspace | `verifyWorkspaceInTenant` → 404 |
| 未登录 | 无 `X-Auth-User-Id` → 401 |

## 角色

不引入新角色。任意可打开该工作空间面板的认证用户均可读写**自己的**偏好。

## 结论

无阻断项；实现时强制 auth user_id 与租户归属校验即可。
