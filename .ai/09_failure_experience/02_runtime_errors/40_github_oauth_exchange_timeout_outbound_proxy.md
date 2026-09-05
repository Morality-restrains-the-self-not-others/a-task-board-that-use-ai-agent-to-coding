# GitHub OAuth 换票 dial timeout → 配置 outbound_proxy 回退

- **日期**：2026-07-17
- **现象**：`/profile/git-site-oauth/` 红字「授权失败：无法与 GitHub 交换令牌」；`data-traceId` 对应日志 `dial tcp …:443: i/o timeout` / `Client.Timeout exceeded while awaiting headers`。
- **根因**：本机到 `github.com`（如 `20.205.243.166:443`）直连间歇/持续超时；业务进程禁止继承 shell `HTTP(S)_PROXY`（死代理 `127.0.0.1:1234`），故仅直连会失败。
- **缓解**：本机 `ssh -D 1080` 等动态转发可达 GitHub；在 `conf/auth/git-oauth/providers/*.yaml` 配置 `service.outbound_proxy: socks5://127.0.0.1:1080`（或 `GITOAUTH_GITHUB_OUTBOUND_PROXY`）。`taskGitOauth` 换票/profile：**直连优先 → 网络失败后回退该代理**（非读环境代理）。
- **相关**：`.ai/01_project_constraints/23_app_startup_no_env_proxy.md`；同目录 `39_github_oauth_exchange_failed_bind_field_mismatch.md`（bind 字段另一类根因）。
