# Value Stream: taskAuth Increment 4 — 密码重置迁移

> Derived from design: `docs/superpowers/specs/2026-05-28-taskauth-increment4-design.md`

## Value Summary

SaaS 用户在忘记密码时，仍通过原有 URL 完成链接/验证码重置，底层由 taskAuth 读写共享 SQLite，邮件与验证码副作用经 Django internal 回调。

## End-to-End Flow

[用户请求重置密码] → [Django delegate] → [taskAuth 生成/校验 token 或协调验证码] → [Django internal 发邮件/验证码] → [taskAuth 更新 password_hash] → [用户用新密码登录]

## Value Increments

### Increment 4.1: 链接重置薄切片（Thin Slice）
**Value to user:** 邮箱用户可通过重置链接设置新密码。  
**Scope:** send_password_reset_link、reset-password-with-link、get-reset-user-info；post-password-reset-link internal。  
**Depends on:** Increment 1–2（login/register 已迁）。  
**验证:** `accounts/view_test/UserViewSet_reset_password_test.py`

### Increment 4.2: 验证码重置
**Value to user:** 手机/邮箱验证码重置密码。  
**Scope:** send_password_reset_code、reset_password_with_code；send/verify internal。  
**Depends on:** 4.1。

### Increment 4.3: 桥接 E2E 保障
**Value to user:** TASKAUTH_ENABLED 路径长期可回归。  
**Scope:** `tests/test_taskauth_password_reset_bridge.py`、Go handler 单测。  
**Depends on:** 4.1–4.2。

## YAML 变更

`user-auth` → `reset-password` 增补 `task-auth.accounts_login_method.password_reset_token`、`task-auth.accounts_login_method.password_hash`。
