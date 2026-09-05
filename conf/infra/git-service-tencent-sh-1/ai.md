# git-service-tencent-sh-1 配置 Companion

本目录是区域实例 `tencent-sh-1` 的 SSOT（ADR-0014）。

登录/注册与默认实例相同（元规则 50 / ADR-0016）：`signupEnabled` / `passwordAuthWeb` / `passwordAuthGit` **必须为 false**，仅 taskAuth OIDC SSO。禁止为排障打开注册。

细则：`.ai/01_project_constraints/55_gitlab_sso_only_no_self_signup.md`

镜像与现网锁同步：`imageTag`（当前 `19.2.4-ce.0`）。升级只改此键，再 `scp` 更新 `/opt/daydaymoney/gitservice-tencent-sh-1/docker-compose.yml` 与 `.env` 的 `GITLAB_IMAGE`。

其它 runAll 服务不得 `depends_on: git-service-tencent-sh-1`（ADR-0048）。实例由人工在 9999 对齐。
