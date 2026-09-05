# GitHub「已取消授权」但凭据仍在（DELETE 静默跳过）

- **日期**：2026-07-17
- **现象**：页面绿字「已取消 xxx 的 GitHub 授权」，刷新后仍显示已绑定；DB `api_gitoauthappusercredential` 行仍 `active`。
- **根因**：`GithubAppConnectionView.delete` 调用 `delete_gitoauth_user_credential` 时未传 `provider_key`，函数把 key 落成裸 `github` → `resolve_gitoauth_service_base` 失败 → **直接 return，不调 gitOauth**；同时 summary 用 `default_github_provider_key()` 仍能读到已绑定，故 `was_connected=true` 误报成功。
- **修复**：DELETE 按 `service_provider` 解析并传入完整 `provider_key`；裸 key 回退 `default_github_provider_key()`；删除失败返回 502。
- **相关**：GitLab DELETE 原本已传 `provider_key`；provider 对齐见 `41_github_oauth_ok_but_unbound_provider_key_mismatch.md`。
