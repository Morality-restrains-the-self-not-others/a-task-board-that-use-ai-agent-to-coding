# GitHub `?github=ok` 但页面仍显示未绑定（provider_key 不一致）

- **日期**：2026-07-17
- **现象**：回调成功（`github=ok`），红字「GitHub 回调已成功，但读取绑定状态仍为未绑定…」。
- **根因**：taskGitOauth 落库 `provider=github:github-official-daydaymoney`；Django 目录仍为 `service_provider=github-official`，前端 connection 用后者查 summary → 查不到行 → `connected=false`（重试 8 次后误报环境不一致）。
- **修复**：对齐 `conf/core/django/git-oauth-providers/http-github-com.yaml`（及 task-credential）与 `conf/auth/git-oauth/providers/*daydaymoney*`；`ExpandProviderKeys` 增加 `github-official` ↔ `github-official-daydaymoney` 别名。
- **验收**：`ListCredentialsForUser("github:github-official", uid)` 能命中 daydaymoney 行；页面刷新后显示已绑定。
