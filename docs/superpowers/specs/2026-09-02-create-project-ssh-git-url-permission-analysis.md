# 角色权限分析 — 创建项目 ssh:// Git URL

- **Date:** 2026-09-02
- **Design:** `docs/superpowers/specs/2026-09-02-create-project-ssh-git-url-design.md`

## 结论

无新端点、无新角色。沿用创建/编辑项目与 `validate-git-repos` 既有租户成员权限。

| 改动点 | 主体 | 资源层级 | 操作 | 现有检查 | 是否缺失 | 建议 |
|--------|------|----------|------|----------|----------|------|
| CreateProject / ProjectEdit 格式校验 | 已登录租户成员 | Tenant / Project | write 前校验 | 页面需登录 + tenant 路由 | ✅ | 不放宽鉴权 |
| POST validate-git-repos（存量） | 同上 | Tenant | read/probe | tenant_id 路径 + session | ✅ | 仅扩展 URL 解析 |
| 创建/PATCH 项目 git_repos | 同上 | Project | write | 现网项目写权限 | ✅ | 原样存 URL，不因 scheme 提权 |

## IDOR / SSRF

- 仍禁止任意 scheme（`javascript:`、`ftp:`、`git://`）。
- `ssh://` 规范化后走既有 Git provider HTTP 探测，不新增出站 SSH 探测。
- 不把 URL 中的 userinfo 当密钥写入日志。
