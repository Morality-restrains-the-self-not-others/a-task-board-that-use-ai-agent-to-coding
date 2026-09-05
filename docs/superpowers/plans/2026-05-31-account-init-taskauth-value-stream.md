# Value Stream: 账号 init 双库对齐（taskAuth bootstrap + saas 旧表清理）

> Derived from: `docs/superpowers/specs/2026-05-31-account-init-taskauth-architecture-design.md`

## Value Summary

开发者在 runAll「初始化全部数据库」后，能立即用 `author@example.com` 登录；saas 库不再保留 auth 凭证旧表。

## Related Value Streams

- **platform-dev-database-reset**（extension）：init 验收增加 ruandao 可登录 + verify 脚本
- **user-auth**（modification）：admin 凭证真源从 saas init 改为 task-auth bootstrap
- **2026-05-28-taskauth-increment7-value-stream**（dependency）：物理删表策略延续

## End-to-End Flow

[开发者点击 init-databases] → [migrate 双库] → [task-auth bootstrap 凭证] → [saas init 角色/业务种子] → [drop saas 旧 auth 表 + verify] → [启动 platform] → [ruandao 登录成功]

## Value Increments

### Increment 1: 旧表清理 + migrate 门禁（Thin Slice）
**Value to user:** saas 库 schema 符合双库目标（无 login_method/customtoken 表）
**Scope:** `TASKAUTH_USE_SEPARATE_DB=1` on migrate + drop/verify 脚本接入
**Depends on:** nothing

### Increment 2: taskAuth admin bootstrap
**Value to user:** auth.db 有 ruandao 凭证（离线 init）
**Scope:** Go `bootstrap-admin` + `db/task-auth/init.sh`
**Depends on:** Increment 1

### Increment 3: saas SuperAdmin 对齐 + 全链路验收
**Value to user:** 登录页 + Playwright 全绿
**Scope:** 重构 `create_admin_ruandao.py`；registry order 调整；测试
**Depends on:** Increment 2
