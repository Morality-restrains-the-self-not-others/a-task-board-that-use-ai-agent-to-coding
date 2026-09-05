# 角色权限分析：PR 回复与一键合并

- **日期:** 2026-08-22
- **设计:** `docs/superpowers/specs/2026-08-22-pr-reply-merge-status-design.md`
- **RBAC 模型:** 工作空间成员（非新租户控制台页）；不新增 page/region。任务详情既有入口。元规则 45 不触发新租户页面。

## 改动点权限表

| 改动点 | 主体 | 资源层级 | 操作 | 现有检查 | 是否缺失 | 建议 |
|--------|------|----------|------|----------|----------|------|
| POST comments + git_pr | 已认证工作空间成员 | Workspace / Task | write | hasWorkspaceAccess + requirePostActive | ✅ | 沿用；git_pr 不绕过 @mention 校验（无 mentions） |
| GET comments 含 git_pr 字段 | 工作空间成员 | Task | read | hasWorkspaceAccess | ✅ | 只增字段 |
| POST merge-request-status | 已认证租户成员 | Tenant + 用户 OAuth | read 外部 Git | ensureTenantMember + user token | ✅ 充分 | 无 token → 404/403 不扫他人仓 |
| POST merge-request-merge | 同上 | Tenant + 远端 PR | write 外部 Git | ensureTenantMember + Git ACL | ✅ | 审计必须带 user_id；Idempotency-Key 可选记录 |
| 审计表写入 | 系统 | 用户/任务 | append-only | 服务端写入 | ✅ | 禁止前端直写审计表 |

## 角色

不新增角色。能推送的人即可尝试一键合并；Git 平台最终拒绝无权限者。

## IDOR

- merge body 的 `html_url` 必须解析为 http(s) GitHub/GitLab URL；禁止 file/内网任意 SSRF（host 须匹配已配置 provider website 或 github.com）。
- `task_id` 仅用于审计与评论刷新，不作为跨租户越权通道；tenant_id 来自路径。

## 前端

- 一键合并：`createClickGuard` + Idempotency-Key
- 无 region 新种子（非租户控制台）
