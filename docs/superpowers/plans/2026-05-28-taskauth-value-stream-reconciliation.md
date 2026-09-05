# Value Stream: taskAuth 价值流 Reconciliation（Increment 1–7）

> Derived from:
> - `docs/superpowers/specs/2026-05-28-taskauth-split-design.md`
> - `docs/superpowers/specs/2026-05-28-taskauth-db-split-design.md`（Increment 6）
> - `docs/superpowers/specs/2026-05-28-taskauth-increment7-auth-table-cleanup-design.md`

## Value Summary

SaaS 用户仍通过原有 Django URL 完成注册/登录/重置；认证写路径经 **taskAuth Go (:8003)**，**auth 表真源为 `taskAuth/data/auth.db`**；Django default 仅保留 `accounts_user`（FK + sync-user）。

## Related Value Streams

| 文档 | 关系 |
|------|------|
| `2026-05-28-taskauth-split-value-stream.md` | **修改** — 原「共享 SQLite」→ auth.db 独占 + sync-user |
| `2026-05-28-taskauth-increment4-value-stream.md` | **扩展** — reset-password 已 active on task-auth |
| `2026-05-28-taskauth-increment7-value-stream.md` | **扩展** — auth 表清理、503 守卫、runAll 编排 |

## End-to-End Flow

```
[用户 POST /api/accounts/users/*]
  → [Django delegate（TASKAUTH_ENABLED）]
  → [taskAuth Go 读写 auth.db]
  → [internal：sync-user / 邮件 / Kafka / 验证码]
  → [Token 鉴权：CustomToken @ auth.db + User @ default]
  → [用户获得 token，访问业务 API]
```

**runAll 编排：** `task-auth` → `git-oauth` → `saas-backend`（`TASKAUTH_ENABLED=true`）→ `taskFE`

## Reconciliation 变更（旧 → 新）

### 字段归属规则

| 表 | 真源服务 | 说明 |
|----|----------|------|
| `accounts_login_method` | **task-auth** | 已从 default 删除 |
| `accounts_customtoken` | **task-auth** | 已从 default 删除 |
| `accounts_user` | **saas-backend**（default）+ task-auth 写 auth.db 副本 | sync-user 同步 |
| `accounts_sms_verification_code` | **saas-backend** | 验证码仍 default |

### user-auth 流 step 级变更

| step | 移除（stale） | 保留/新增 |
|------|---------------|-----------|
| email-register | `saas-backend.accounts_user.email/password` | `saas-backend.accounts_user.id` + `task-auth.*` |
| phone-register | `saas-backend.accounts_login_method.*` | 全 `task-auth.*` + sync user id |
| activate | `saas-backend.accounts_user.activation_token` | login_method 令牌在 task-auth |
| login | `saas-backend.accounts_customtoken.key` | 仅 task-auth token |
| reset-password | `saas-backend.password_reset_token` | 全 task-auth login_method 字段 |
| phone-otp-login | `saas-backend.accounts_login_method.*` | task-auth + default user id |
| profile-phone-replace | `saas-backend.accounts_login_method.*` | task-auth（router 只读） |
| **新增** verification-code-delegate | — | bridge pytest |
| **新增** taskauth-bridge-regression | — | 密码重置 bridge pytest |
| auth-table-cleanup | 测试改为 sync-user | auth.db 独占字段 |

### 其他流（planned）

`system-admin-phone-login-recharge-policy`：`accounts_login_method` / `accounts_customtoken` 字段 provider 改为 **task-auth**。

### runall_config

`runAll/config.yaml` → **`runAll.yaml`**（与 `runAll/run.sh` 一致，含 `task-auth` 服务名）。

`docker-infra` 在 `runAll.yaml` 中仍为注释组；`runall-global-start-stop-all` 暂用 `git-oauth.runtime.*` 占位，启用 infrastructure 组后可改回 `docker-infra.*`。

### user-auth 新增 step

| step | 用途 |
|------|------|
| verification-code-delegate | bridge pytest |
| taskauth-bridge-regression | 密码重置 bridge |
| runall-task-auth-orchestration | runAll 依赖链 + E2E health（`TASKAUTH_E2E=1`） |

## Value Increments（累计状态）

| Inc | 名称 | 状态 |
|-----|------|------|
| 1 | 登录薄切片 | ✅ |
| 2 | 邮箱注册闭环 | ✅ |
| 3 | 手机注册 / OTP | ✅（bridge + forward-login） |
| 4 | 密码重置 | ✅ |
| 5 | domain 包 / phone_register Go | ✅ |
| 6 | auth.db 独占 + sync-user | ✅ |
| 7 | API 收口 + drop 共享 auth 表 + runAll | ✅ |

## Self-Review

1. ✅ 每 step 字段单一真源（无 saas-backend 写 login_method/token）
2. ✅ runAll 服务名 `task-auth` 与 YAML provider 一致
3. ✅ 三段式字段名合规
4. ✅ `runall_config: runAll.yaml` 可解析
