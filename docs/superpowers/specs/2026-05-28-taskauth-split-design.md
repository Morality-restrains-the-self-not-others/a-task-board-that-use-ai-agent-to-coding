# taskAuth 拆分设计

## 目标

将 task2app（Django Saas_project）中的**注册、登录、认证**能力拆分到独立 Go 服务 `taskAuth`，对外 URL、请求/响应格式、Token 机制保持不变，用户与前端无感知。

## 成功标准（SMART）

| 标准 | 验收 |
|------|------|
| 独立服务 | `taskAuth` 在 port 8003 运行，runAll 可编排 |
| API 兼容 | `/api/auth/`、`/api/accounts/users/login/` 等路径不变 |
| Token 兼容 | 仍写入 `accounts_customtoken`，Django `CustomTokenAuthentication` 无需改动 |
| 数据共享 | 读写同一 SQLite（`task2app/Saas_project/db.sqlite3`） |
| 测试通过 | user-auth 价值流 pytest 全绿 |
| 降级 | taskAuth 不可用时 Django 本地逻辑 fallback |

## 架构：Strangler Fig + 透明桥接

```
前端 ──▶ Django (8001) ──▶ taskAuth Go (8003)
              │                    │
              │                    ├─ SQLite 读写 (user/login_method/token)
              │                    └─▶ Django internal API (邮件/Kafka/序列化/隐私条款)
              └─ Token 校验仍走 Django ORM（读同一 DB）
```

参照已有 `gitOauth` 拆分模式，但 auth 与用户主数据强耦合，采用**共享数据库 + Django 内部回调**处理副作用（邮件、Kafka、UserSerializer、隐私条款记录）。

## 价值流影响

影响 `value-stream.yaml` → `user-auth` 全部 active 步骤：

- email-register, phone-register, password-policy, activate, login
- reset-password, resend-activation, phone-otp-login

`frontend-auth-guard-redirect` 不变（前端域模型）。

## 端点清单（taskAuth 实现）

| 路径 | 方法 | 说明 |
|------|------|------|
| `/api/health/` | GET | 健康检查 |
| `/api/auth/` | POST | login 别名 |
| `/api/accounts/users/login/` | POST | 密码/验证码登录 |
| `/api/accounts/users/logout/` | POST | 注销 Token |
| `/api/accounts/users/email_register/` | POST | 邮箱注册 |
| `/api/accounts/users/phone_register/` | POST | 手机注册 |
| `/api/accounts/users/send_verification_code/` | POST | 发送验证码 |
| `/api/accounts/users/confirm_activation/{token}/` | POST | 激活 |
| `/api/accounts/users/resend_activation_email/` | POST | 重发激活邮件 |
| `/api/accounts/users/send_password_reset_code/` | POST | 密码重置验证码 |
| `/api/accounts/users/reset_password_with_code/` | POST | 验证码重置密码 |
| `/api/accounts/users/send_password_reset_link/` | POST | 发送重置链接 |
| `/api/accounts/users/reset-password-with-link/{token}/` | POST | 链接重置密码 |

## Django 内部回调（taskAuth → Django）

| 路径 | 用途 |
|------|------|
| `POST /api/internal/taskauth/enrich-login/` | 序列化 user、redirect_url、记录隐私/协议、django session |
| `POST /api/internal/taskauth/post-register/` | 发激活邮件、Kafka USER_CREATED、记录隐私/协议 |
| `POST /api/internal/taskauth/validate-policies/` | 校验 privacy/license id |

鉴权：`X-TaskAuth-Internal-Secret` 与 `port_config.json` → `taskAuth.internalSecret` 一致。

## 配置

`port_config.json` 新增：

```json
"taskAuth": {
  "allowedHost": "http://auth.api.daydaymoney.com",
  "host": "localhost",
  "port": 8003,
  "internalSecret": "taskauth-local-dev-secret",
  "djangoInternalApiBase": "http://127.0.0.1:8001"
}
```

`django.taskAuthBaseUrl` 供 Django 桥接调用 taskAuth。

## 不在本次范围

- Git OAuth（已在 gitOauth）
- 公司/成员/组管理（保留 Django accounts）
- SuperAdmin 认证（taskAuth 支持读 SuperAdmin content_type，逐步迁移）
