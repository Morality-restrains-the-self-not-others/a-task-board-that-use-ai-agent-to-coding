# Companion：tencent-sh-1 GitLab OAuth App

与 `conf/auth/git-oauth/providers/http-gitlab-tencent-sh-1.yaml` 必须保持
`service_provider` / `target.website` / `target.client_id` 一致。

区域实例禁止回落到 `gitlab:default`（跨 CE 令牌 → API 401）。细则见
`conf/auth/git-oauth/ai.md`「多区域 GitLab」。

`target.scope` 必须含 `write_repository`（git-over-HTTP push）。
