# Feature-Params 迁 Go — 权限分析

| 端点 | 认证 | 授权 | Access gate |
|------|------|------|-------------|
| 公司 feature-params | Gateway token → X-User-Id | 活跃租户成员；POST 需 is_admin | company_settings / summary |
| 工作空间 feature-params | 同上 | 成员 + workspace access；POST 需 ws/tenant admin | workspace_settings / summary |
| 个人 configs | 同上 | 仅本人 user_id（IDOR） | 无（个人密钥本属用户） |

审计：每次公司/工作空间访问记 user/auth/context/view/status。
