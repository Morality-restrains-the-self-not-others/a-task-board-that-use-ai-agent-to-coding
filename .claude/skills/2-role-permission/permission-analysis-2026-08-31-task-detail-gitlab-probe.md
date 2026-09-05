# 权限分析 — 任务/项目数据 GET 与 GitLab 探活分离

- 日期：2026-08-31
- 设计：`docs/superpowers/specs/2026-08-31-task-detail-get-15s-timeout-design.md`
- 结论：绿灯 ✅ — 无新角色、无新公网写接口；沿用既有 workspace / tenant 读检查。

## 权限影响矩阵

| 改动点 | 主体 | 资源层级 | 操作 | 现有检查 | 是否缺失 | 建议 |
|--------|------|----------|------|----------|----------|------|
| GET 任务详情 | 工作空间成员 | Workspace | read | `hasWorkspaceAccess` | ✅ | 去掉同步 GET project，不放宽 ACL |
| GET 项目详情 | 租户成员 | Tenant / Project | read | `rejectIfProjectNotInTenant` + 登录 | ✅ | 去掉 GitLab enrich，权限不变 |
| POST validate-git-repos | 已登录租户用户 | Tenant | read（探活） | 现有 handler 校验 tenant + user | ✅ | 任务页复用，不新造入口 |
| loadProjects 只读 task_projects | 同上 GET 任务 | Workspace | read | 随 GET 任务 | ✅ | 不再用 internal 身份打 GET project |

无新角色。任务页探活与项目页同一 POST：不可跨租户探别人的仓库 URL（handler 已按当前用户换 token）。

## 安全审查

- [x] IDOR：任务 GET 仍校验 workspace；项目 GET 仍校验 tenant 归属
- [x] 无新 PATCH/PUT
- [x] 跨租户：探活 POST 带 tenant_id，token 为当前用户
- [x] 403 vs 404：不改
- [x] 无 user_id 注入
- [x] 非敏感写；探活失败不得泄露 token

## 权限测试

| 场景 | 角色 | 操作 | 预期 |
|------|------|------|------|
| 工作空间成员 | member | GET 任务 | 200，不等 GitLab |
| 无工作空间访问 | 外人 | GET 任务 | 403 |
| 跨租户 | other tenant | GET 项目 | 403/404（现有） |
| 已登录 | member | POST validate-git-repos | 200 或现有 4xx；不挡任务 GET |

## 风险

低：GET 项目不再带 `git_repos_status` / disk_size。项目页已独立 POST 填状态。磁盘数字延后（OPT）。
