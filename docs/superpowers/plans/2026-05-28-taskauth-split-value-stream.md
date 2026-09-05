# Value Stream: taskAuth 认证拆分

> 来源设计：`docs/superpowers/specs/2026-05-28-taskauth-split-design.md`

## Value Summary

**SaaS 用户**继续通过原有 URL 完成注册、登录、激活与登出，**无感知**地由 Go 服务 `taskAuth` 承担认证写入，Django 仍负责 Token 校验与业务 API。

## End-to-End Flow

```
[用户提交登录/注册] 
  → [Django API 网关，URL 不变]
  → [taskauth_bridge 委托 taskAuth]
  → [taskAuth 读写共享 SQLite：user / login_method / customtoken]
  → [Django internal 回调：序列化 user、邮件、Kafka、隐私条款]
  → [返回 token + user JSON，与拆分前一致]
  → [用户获得会话，访问业务功能]
```

**触发**：前端 POST `/api/accounts/users/login/` 等既有路径。  
**交付点**：响应含 `token` 与 `user`，后续 API 仍用 `Authorization: Token …` 通过 Django 鉴权。

## 能力分类

| 能力 | 分类 | 说明 |
|------|------|------|
| login / logout | 核心价值 | 用户最频繁路径，M1 已迁 |
| email_register + activate + resend | 核心价值 | 邮箱注册闭环，M2 已迁 |
| health + runAll 编排 | essential support | 服务可观测、可启动 |
| Django bridge + internal API | essential support | 无感拆分与副作用 |
| Django fallback | essential support | taskAuth 宕机时降级 |
| phone_register / OTP login | enhancement | M3 部分经 forward-login 委托 Django |
| reset-password 系列 | future | M4 仍留 Django，后续迁移 |
| frontend-auth-guard-redirect | 不变 | 前端域模型，不经过 taskAuth |

## Value Increments

### Increment 1: 登录薄切片（Thin Slice）✅ 已交付

**用户价值：** 用邮箱+密码登录，拿到与以前相同的 token 与 user。  
**范围：** taskAuth health、login、logout；Django delegate；共享 DB token 写入；runAll `task-auth` 服务。  
**Depends on：** 无（复用现有 accounts 表结构）。  
**验证：** `accounts/view_test/UserViewSet_login_test.py`  
**字段归属：**

- `task-auth.accounts_customtoken.key` — Go 写入 token
- `task-auth.accounts_login_method.password_hash` — 凭证校验
- `saas-backend.accounts_user.is_active` — 桥接 enrich-login 读 user

---

### Increment 2: 邮箱注册闭环 ✅ 已交付

**用户价值：** 注册 → 收激活邮件 → 激活 → 登录。  
**范围：** email_register、confirm_activation、resend_activation；post-register / post-activate internal 回调。  
**Depends on：** Increment 1（登录验证闭环）。  
**验证：**

- `accounts/view_test/UserViewSet_email_register_test.py`
- `accounts/view_test/UserViewSet_activate_test.py`
- `accounts/view_test/UserViewSet_resend_activation_email_test.py`

**字段归属：**

- `task-auth.accounts_user.id` — 雪花 ID 创建
- `task-auth.accounts_login_method.activation_token` — 激活令牌
- `saas-backend.accounts_user.is_active` — 激活后状态（共享表）

---

### Increment 3: 手机号注册与 OTP 登录 🔄 部分交付

**用户价值：** 手机号+验证码注册/登录。  
**范围：** taskAuth 将 phone+code 请求 forward 至 Django `forward-login`；验证码发送仍走 Django。  
**Depends on：** Increment 1。  
**验证：**

- `accounts/view_test/UserViewSet_phone_register_test.py`
- `tests/test_login_phone_code.py`
- `tests/test_verification_code_service.py`

**字段归属：** 仍以 `saas-backend` 为主；`task-auth` 仅作 HTTP 转发层（无独立表写入）。

---

### Increment 4: 密码策略与重置 📋 待迁移

**用户价值：** 忘记密码、验证码/链接重置。  
**范围：** 当前实现仍在 Django；taskAuth 未接管。  
**Depends on：** Increment 1–2。  
**验证：**

- `accounts/tests/test_password_policy.py`
- `accounts/view_test/UserViewSet_reset_password_test.py`

**状态：** `planned` 对 task-auth 字段；`active` 对 saas-backend 字段。

---

### Increment 5: 无感保障与编排 ✅ 已交付

**用户价值：** 用户不感知拆分；开发/运维可一键启动。  
**范围：** port_config `taskAuth` 块、runAll 依赖链、`TASKAUTH_ENABLED`、pytest 内存库禁用桥接、Django fallback。  
**Depends on：** Increment 1。  
**验证：** runAll health `http://127.0.0.1:8003/api/health/`；pytest user-auth 套件（bridge 关闭时 fallback）。

---

## 依赖与阻塞点

| 阻塞点 | 说明 |
|--------|------|
| 共享 SQLite 路径 | taskAuth 与 Django 必须指向同一 `db.sqlite3` |
| internal secret 一致 | `port_config.taskAuth.internalSecret` ↔ Django `TASKAUTH_INTERNAL_SECRET` |
| enrich-login 可用 | runAll 下 saas-backend 需先于 enrich 回调就绪 |
| pytest 隔离 | `settings_test` 内存库 → `TASKAUTH_ENABLED=False`，避免跨进程 DB 不一致 |

## YAML 变更（已写入 `value-stream.yaml`）

已在 `user-auth` 流各相关 step **增补** `task-auth` 服务字段（保留原有 `saas-backend` 字段）：

| step | 新增 fields |
|------|-------------|
| email-register | `task-auth.accounts_user.id`, `task-auth.accounts_login_method.identifier` |
| phone-register | `task-auth.accounts_login_method.identifier` |
| activate | `task-auth.accounts_login_method.is_verified`, `task-auth.accounts_login_method.activation_token` |
| login | `task-auth.accounts_customtoken.key`, `task-auth.accounts_login_method.identifier` |
| resend-activation | `task-auth.accounts_login_method.activation_token` |
| phone-otp-login | `task-auth.accounts_customtoken.key` |

独立 `taskauth-service` 流暂未添加（health 需 Go test，valueStream 仅跑 pytest；可后续补 Django bridge 集成测试再建流）。

## Self-Review

1. ✅ 每个 increment 均有用户可感知价值或 essential support 闭环  
2. ✅ Increment 1 为端到端薄切片（登录拿 token）  
3. ✅ 依赖顺序：1 → 2 → 3/4，5 横切  
4. ✅ YAML 已写入 `value-stream.yaml`，语法校验通过，`cd valueStream && go test ./...` 通过  
5. ✅ 字段名遵循 `<runAll-service>.<table>.<field>`
