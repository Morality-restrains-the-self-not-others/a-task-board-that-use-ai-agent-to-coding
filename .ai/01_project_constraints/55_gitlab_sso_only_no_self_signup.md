# GitLab 禁止自行注册，仅允许 SSO 登录（元规则）

## 基本信息

- 版本：1.0.0
- 创建日期：2026-08-18
- 维护者：Trae AI 团队
- 约束索引：`00_project_constraints.md` 第 50 条
- Cursor：`.cursor/rules/gitlab-sso-only-no-self-signup.mdc`（alwaysApply）
- ADR：`docs/adr/0016-gitlab-sso-only-no-self-signup.md`
- 门禁：`db/scripts/ci/check_gitlab_sso_only_no_self_signup.py`
- 自测：`db/scripts/ci/test_check_gitlab_sso_only_no_self_signup.py`

## 背景（为何是元规则）

平台 GitLab（`gitService`，含 ADR-0014 多区域实例）面向租户与开发者。若开放 GitLab 自带的 `/users/sign_up` 或账密登录：

- 任何人可绕过 taskAuth 身份体系自建账号；
- 与平台 SSO（taskAuth OIDC）双轨并存，权限与审计无法对齐；
- Agent / 运维常以「登录坏了」为由把 `signupEnabled` 改回 `true`，导致公网实例被注册刷号。

本仓库已用 conf 开关 + Omnibus + DB 级 `apply_auth_policy.sh` 关闭注册与账密。本条把该策略升为**禁止忽略的元规则**，并加 CI 防止回退。

## 核心原则

**平台 GitLab 默认禁止用户自行注册新用户。Web 登录仅允许通过平台 SSO（taskAuth OIDC / OmniAuth `openid_connect`）。禁止把公开注册或账密登录当作排障手段打开。**

## 强制要求

### 1. 关闭自行注册

- `conf/infra/git-service*/config.yaml` 的 **`signupEnabled` 必须显式为 `false`**（含默认实例与每个区域实例）。
- 对应 GitLab `application_setting.signup_enabled = false`（`/users/sign_up` 不可用）。
- **禁止**在已提交的 `config.yaml`、`docker-compose.yml` Omnibus 片段中把 signup 设为 `true`。
- `config.local.yaml` 不得作为「长期打开注册」的第二 SSOT；禁止提交该覆盖。

### 2. Web / Git 账密登录默认关闭，仅 SSO

- **`passwordAuthWeb: false`** — 登录页无用户名密码表单。
- **`passwordAuthGit: false`** — HTTPS Git 账密认证关闭（SSH key 不受影响）。
- **`omniauth_enabled = true`**，且 `omniauth_allow_single_sign_on` 包含 **`openid_connect`**（label：taskAuth SSO）。
- **`omniauth_block_auto_created_users = false`** — 首次 SSO 成功可自动开通 GitLab 用户。这是 SSO 供给，**不是**自行注册。

### 3. SSOT 与落地路径（须同时生效）

GitLab 的 signup / password 开关是 **DB 级** `application_setting`：仅写 `gitlab.rb` 不会覆盖已有 DB 值。

| 层 | 路径 | 职责 |
|----|------|------|
| SSOT | `conf/infra/git-service*/config.yaml` | 人类/Agent 只改这里 |
| 消费 | `gitService/scripts/load_gitservice_config.py` | 默认值必须为 `False`；导出 `GITLAB_SIGNUP_ENABLED` 等 |
| Omnibus | `gitService/docker-compose.yml` | `${GITLAB_SIGNUP_ENABLED:-false}` 等；`omniauth_enabled = true` |
| DB 强制 | `gitService/scripts/apply_auth_policy.sh` | 启动/bootstrap 按 conf 写回 ApplicationSetting |
| 调用 | `gitService/run.sh` | 必须调用 `apply_auth_policy.sh` |

新增区域实例（`conf/infra/git-service-<slug>/`）**必须**带齐上述三键且均为 `false`，并走同一 `run.sh` / compose。

### 4. 允许与禁止

| 场景 | 正确 | 错误 |
|------|------|------|
| 新用户进入 GitLab | 先有平台账号，再点 taskAuth SSO；首次 SSO 自动建 GitLab 用户 | 打开 `/users/sign_up` 让路人注册 |
| 排障「无法登录」 | 查 OIDC issuer / redirect_uri / OmniAuth；修 taskAuth | 把 `signupEnabled` 或 `passwordAuthWeb` 改为 `true` |
| 运维建用户 | GitLab Admin / `gitlab-rails runner`（管理员） | 恢复公开注册当运维入口 |
| Git 访问 | SSO 后 SSH key 或后续平台发放的凭证 | 打开 Git 账密当默认 |

## 触发

- 新增/修改 `conf/infra/git-service*/config.yaml`、`gitService/docker-compose.yml`、`run.sh`、`apply_auth_policy.sh`、`load_gitservice_config.py`
- 新增 GitLab 区域实例（ADR-0014）
- 评审「GitLab 登录 / 注册 / OmniAuth / OIDC」相关变更
- Agent 拟以开启注册或账密「临时绕过」SSO 故障

## 与既有规则的关系

| 规则 | 管什么 | 与本条关系 |
|------|--------|------------|
| 人工可改配置落在 conf（第 42 条） | 开关编辑落点 | 本条约束这些开关的**合法默认值** |
| 可插拔多区域 GitLab（ADR-0014） | 每区域独立 CE 实例 | **每个**区域 SSOT 都须 SSO-only |
| 异域前端与 API（第 18 条） | OIDC redirect / 域名 | SSO 回调仍须跨域正确，不得改回账密凑合 |

## 验收

```bash
python3 db/scripts/ci/test_check_gitlab_sso_only_no_self_signup.py
python3 db/scripts/ci/check_gitlab_sso_only_no_self_signup.py
python3 gitService/scripts/test_load_gitservice_config.py
python3 db/scripts/ci/test_git_service_config_resource_keys.py
```

运行时（GitLab 已启动时）补充：

- `GET /users/sign_up` → 302 `/users/sign_in`（注册不可用）
- 登录页无 password 字段；可见 `openid_connect` / taskAuth SSO

## 变更日志

- 2026-09-01：验收命令改为 `db/scripts/ci/test_git_service_config_resource_keys.py`；区域 compose 迁至 `gitService/docker-compose.tencent-sh-1.yml` 后 SSO 门禁亦扫描该文件
- 2026-08-18：版本 1.0.0 - 初版；与 ADR-0016 同步；落地沿用 OPT-20260807-034
