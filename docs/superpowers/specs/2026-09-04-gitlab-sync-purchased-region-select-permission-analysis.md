# 角色权限分析 — GitLab 同步来源选自已购买区域

- **日期**: 2026-09-04
- **设计**: `2026-09-04-gitlab-sync-purchased-region-select-design.md`
- **结论**: 无新写接口；只读复用 `GET /api/tenant/{tid}/billing/gitlab-resources/`（与 Navbar 同权限）+ 既有 `gitlab-remote-repos?gitlab_host=`

| 主体 | 资源 | 动作 | 变更 |
|------|------|------|------|
| 租户成员（已登录） | 本租户已购 GitLab 摘要 | 读 | 同步模态新增消费，权限不变 |
| 租户成员 | 选定实例远程仓 | 读 | 须带 gitlab_host；无 host 不再用平台默认作「来源」展示 |
| 匿名 | — | — | 仍须登录 |

无 PDP/角色矩阵变更；无 impersonation 特殊路径。
