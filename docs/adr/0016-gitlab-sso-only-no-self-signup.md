# ADR-0016: 平台 GitLab 禁止自行注册，仅允许 SSO 登录

- **Status:** accepted
- **Date:** 2026-08-18
- **Author:** Trae AI
- **Deciders:** 工程团队

---

## Context

平台 GitLab CE（`gitService`，公网如 `gitlab.${baseDomain}`，以及 ADR-0014 的多区域独立实例）是代码托管面。身份权威在 **taskAuth**（OIDC）。GitLab 自带：

1. `/users/sign_up` 公开注册；
2. 用户名/密码 Web 登录与 Git HTTPS 账密。

若两者默认开启，外部用户可绕过平台账号体系自建 GitLab 用户，与租户配额、审计、OIDC `sub` 对齐全部失效。运维与 Agent 在 SSO 故障时也倾向「先打开注册/账密」作为绕过，造成配置漂移。

已有落地（OPT-20260807-034）：conf `signupEnabled` / `passwordAuthWeb` / `passwordAuthGit` 默认 `false`，compose Omnibus + `apply_auth_policy.sh` 写 DB。需要把该安全策略升为架构决策，约束所有现有与未来区域实例。

## Decision

We will **disable GitLab self-registration and password authentication by default**. The only supported interactive login is **platform SSO** (taskAuth OIDC via OmniAuth `openid_connect`).

We will:

1. Keep SSOT in `conf/infra/git-service*/config.yaml` with `signupEnabled`, `passwordAuthWeb`, `passwordAuthGit` all explicit `false`.
2. Keep OmniAuth enabled; allow SSO auto-provision (`omniauth_block_auto_created_users = false`) so the first successful SSO may create the GitLab user. This is IdP-driven provisioning, not public signup.
3. Re-apply DB-level `ApplicationSetting` on start via `apply_auth_policy.sh`, because Omnibus `gitlab.rb` does not override existing DB values.
4. Treat enabling signup or password auth in committed config as a policy violation, not a debugging tactic.

SSH deploy keys / user SSH keys remain allowed. GitLab Admin / `gitlab-rails runner` may create users for break-glass operations.

## Alternatives Considered

### Alternative 1: Keep GitLab signup + require admin approval

- **Pros:** GitLab 原生流程
- **Cons:** 仍暴露注册面；审批与平台租户开通脱节；双身份源
- **Why rejected:** 身份必须以 taskAuth 为唯一入口

### Alternative 2: Password login for admins only, signup off

- **Pros:** 排障时可用 root 密码
- **Cons:** 公网密码面仍在；与「仅 SSO」产品规则冲突；密钥泄漏面更大
- **Why rejected:** 管理员走 SSO 或 rails console；不保留 Web 账密默认路径

### Alternative 3: Per-environment exception (dev signup on, prod off)

- **Pros:** 本地少配 OIDC
- **Cons:** 配置分叉；Agent 易把 dev 默认提交进 SSOT；与未上线阶段禁止 mock/旁路精神冲突
- **Why rejected:** 所有已提交区域 SSOT 同一默认；本地 `config.local.yaml` 不得提交打开注册

## Consequences

### Positive

- GitLab 用户与平台账号经 OIDC `sub` / email 对齐
- 公网无开放注册面
- 新区域实例有可检查的强制三键

### Negative / Trade-offs

- SSO 故障时无人能用账密登录 GitLab Web（须修 OIDC，或 rails console 急救）
- 首次使用必须先有平台账号
- DB 级设置与 `gitlab.rb` 双写，漏跑 `apply_auth_policy.sh` 会漂移

### Mitigations

- 启动路径强制 `apply_auth_policy.sh`；CI 检查 conf / compose / loader 默认值
- OIDC 排障文档与既有 Playwright SSO 测例
- 急救：`gitlab-rails runner` / Admin，禁止改 SSOT 打开注册

## References

- `.ai/01_project_constraints/55_gitlab_sso_only_no_self_signup.md`
- `.cursor/rules/gitlab-sso-only-no-self-signup.mdc`
- `.ai/01_project_constraints/47_conf_app_human_editable_config_ssot.md`
- [ADR-0014: 可插拔多区域 GitLab](0014-pluggable-multi-region-gitlab.md)
- `gitService/scripts/apply_auth_policy.sh`
- `conf/infra/git-service/config.yaml`
