# taskAuth Increment 7 — Domain

## 边界

- **Auth 写**：taskAuth Go 独占 auth.db
- **Tenant User**：Django default（FK 真源）
- **VerificationCode**：Django default（非 auth 表，internal 发送）

## 不变式

1. `TASKAUTH_USE_SEPARATE_DB=true` 时 default 不得存在 `accounts_login_method` / `accounts_customtoken`
2. 用户-facing auth 写必须经 taskAuth HTTP API
3. 跨域只读 LoginMethod 经 router 读 auth.db

## 领域事件

无新增；USER_CREATED 仍 Django internal 发出。
