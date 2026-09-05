# taskAuth Increment 4 设计：密码重置迁移与认证闭环补全

> 延续：`docs/superpowers/specs/2026-05-28-taskauth-split-design.md`  
> 审查输入：`docs/superpowers/reviews/2026-05-28-taskauth-split-review.md`（Important: 密码重置未迁入、E2E 缺口）

## 目标

完成 taskAuth 拆分剩余认证路径，使用户**忘记密码 / 链接重置 / 验证码重置**在 `TASKAUTH_ENABLED=true` 时经 Go 服务处理，URL 与响应格式不变；补齐桥接集成测试，关闭 Increment 1–3 与 Increment 4 之间的能力缺口。

## 成功标准

| 标准 | 验收 |
|------|------|
| 链接重置 | `send_password_reset_link` + `reset-password-with-link/{token}` 经 taskAuth 读写共享 SQLite |
| 验证码重置 | `send_password_reset_code` + `reset_password_with_code` 经 taskAuth（验证码副作用仍 Django） |
| API 兼容 | 路径、JSON 字段、HTTP 状态码与现有 pytest 一致 |
| 测试 | `UserViewSet_reset_password_test.py` 全绿（settings_test 仍禁用桥接）；新增 bridge 集成测试覆盖 TASKAUTH_ENABLED 路径 |
| 降级 | taskAuth 不可达时 Django 本地 PasswordResetService fallback |
| 价值流 | `user-auth` → `reset-password` step 增补 `task-auth.*` 字段 |

## 方案对比

### 方案 A：链接重置优先（推荐）

**做法：** 分两薄切片——先迁链接重置（token 生成/校验/清 token + 密码写入均在 Go），邮件经 Django internal 回调；再迁验证码重置（Go 调 Django internal 发码/验码，Go 写 password_hash）。

**优点：** 与 Increment 2（激活/resend）模式一致；现有 pytest 主要覆盖链接流；风险可控。  
**缺点：** 验证码流需新增 internal 端点，第二切片略复杂。

### 方案 B：全部 forward 至 Django

**做法：** taskAuth 仅 HTTP 代理到 Django internal，不在 Go 写 DB。

**优点：** 实现快。  
**缺点：** 不达成「认证拆分」目标；review Important 项未关闭。

### 方案 C：Increment 4 同时含 phone_register 原生 Go + domain 包提取

**做法：** 一次性迁密码重置、手机注册、领域层 refactor。

**优点：** 架构更干净。  
**缺点：** phone_register 含号码规范化、VerificationCode、隐私条款、Kafka 事件，范围过大；违反薄切片原则。

**推荐：方案 A。** phone_register 原生 Go（原 Task 4.2）与 `taskAuth/domain/` 提取（原 Task 4.4）**顺延至 Increment 5**，本增量聚焦密码重置 + E2E。

---

## 架构

```
前端 ──▶ Django UserViewSet ──▶ taskauth_bridge.delegate_* ──▶ taskAuth Go
                                      │                              │
                                      │ fallback                     ├─ SQLite: login_method.password_reset_token / password_hash
                                      ▼                              └─▶ Django internal:
                                   Django PasswordResetService              post-password-reset-link/
                                                                            send-password-reset-code/
                                                                            verify-password-reset-code/
```

### 端点（taskAuth 新增）

| 路径 | 方法 | Go 职责 | Django 副作用 |
|------|------|---------|---------------|
| `/api/accounts/users/send_password_reset_link/` | POST | 查 login_method、生成 token、UPDATE | `post-password-reset-link/` → Kafka 邮件 |
| `/api/accounts/users/reset-password-with-link/{token}/` | POST | 校验 token 过期、写 password_hash、清 token | 无 |
| `/api/accounts/users/get-reset-user-info/{token}/` | GET | 校验 token、返回 identifier | 无 |
| `/api/accounts/users/send_password_reset_code/` | POST | 校验用户存在 | `send-password-reset-code/` → 写 VerificationCode + Kafka/SMS |
| `/api/accounts/users/reset_password_with_code/` | POST | 调 internal 验码、写 password_hash | `verify-password-reset-code/` |

### Django internal 回调（新增）

| 路径 | 用途 |
|------|------|
| `POST /api/internal/taskauth/post-password-reset-link/` | `{email, token}` → 构建 reset_url、Kafka EMAIL_SENT |
| `POST /api/internal/taskauth/send-password-reset-code/` | `{phone?, email?}` → VerificationCodeService + 发送 |
| `POST /api/internal/taskauth/verify-password-reset-code/` | `{phone?, email?, code}` → bool |

鉴权：沿用 `X-TaskAuth-Internal-Secret`。

### Django 桥接（delegate.py 增补）

```python
delegate_send_password_reset_link(request)
delegate_reset_password_with_link(request, token)
delegate_get_reset_user_info(request, token)  # GET forward
delegate_send_password_reset_code(request)
delegate_reset_password_with_code(request)
```

UserViewSet 各 action 开头：`delegated = delegate_*()` → 有响应则 return，否则 fallback 现有 Django 逻辑。

---

## 领域概念清单（供 DDD 步骤消费）

| 概念 | 上下文 | 说明 |
|------|--------|------|
| **PasswordResetAggregate** | taskAuth | 聚合根：按 login_method 管理 reset token 生命周期 |
| **ResetToken** | taskAuth | 值对象：token + expires_at（24h，与 Django 一致） |
| **VerificationCodeChallenge** | accounts-side-effects | 验证码生成/校验仍 Django 真源（Increment 4 不迁表逻辑） |
| **PasswordCredential** | taskAuth | 复用 Increment 1：前端哈希 password_hash 直写 |

**领域事件（Django 发出）：** `PASSWORD_RESET_REQUESTED`（Kafka 邮件，已有行为保持不变）。

---

## 价值流影响

影响 `value-stream.yaml` → **`user-auth`** 流：

| step | 变更 |
|------|------|
| `reset-password` | 增补 `task-auth.accounts_login_method.password_reset_token`、`task-auth.accounts_login_method.password_hash` |
| `verification-code` | 备注：验证码表仍 `saas-backend` 真源；taskAuth 仅调 internal |

**不新建独立流**；`phone-register` / `phone-otp-login` 本增量不变。

**测试：**

- 保留：`accounts/view_test/UserViewSet_reset_password_test.py`（bridge 关闭）
- 新增：`tests/test_taskauth_password_reset_bridge.py`（mock taskAuth 或 `@pytest.mark.integration` + 本地 :8003）
- 可选：`taskAuth/src/auth_password_reset_test.go`

---

## 实施切片（Increment 4 内部）

### Slice 4.1：链接重置薄切片（Core）

1. Go：`auth_password_reset.go` — send_link / reset_with_link / get_reset_user_info
2. Django internal：`post-password-reset-link`
3. bridge delegate + UserViewSet hook
4. Go unit test + 现有 pytest 回归

### Slice 4.2：验证码重置

1. Django internal：send + verify code
2. Go handlers + delegate
3. 扩展 bridge 集成测试

### Slice 4.3：桥接 E2E 保障

1. `tests/test_taskauth_bridge_integration.py`：login + reset link 在 `TASKAUTH_ENABLED=true` + 共享 fixture DB 或 testcontainers 式 subprocess taskAuth
2. 文档：runAll 手动验证清单（`TASKAUTH_ENABLED=true` 启动 saas-backend）

### 顺延 Increment 5（不在本增量实现）

- phone_register 原生 Go（去 Django ORM）
- `taskAuth/domain/` 包提取
- SuperAdmin 认证迁移

---

## NFR（Increment 4 增量）

| 类别 | 等级 | 说明 |
|------|------|------|
| 可用性 | L3 | 与 Increment 1 相同 fallback |
| 安全 | L3 | reset token 64 字符 urlsafe；日志不打印 token/密码 |
| 一致性 | L2 | token 生成与清理由 Go 单事务 UPDATE |
| 容错 | L2 | post-password-reset-link 失败 → 502（与 resend 激活一致，不 silent success） |

---

## 风险与缓解

| 风险 | 缓解 |
|------|------|
| pytest 内存库无法跨进程 | settings_test 保持 `TASKAUTH_ENABLED=False`；集成测试单独 settings 或 mock forward |
| 验证码 cache 与 DB 双写 | verify 走 Django internal 单入口 |
| GET reset-user-info 未 delegate | 一并纳入 bridge GET forward |

---

## 不在范围

- phone_register / phone OTP 登录逻辑重写
- 密码策略规则变更（`test_password_policy.py` 行为不变）
- 前端 auth 守卫改动
- SuperAdmin 密码重置特殊路径
