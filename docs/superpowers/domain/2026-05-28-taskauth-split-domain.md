# 领域模型: taskAuth 认证拆分

> 输入: 价值流 `2026-05-28-taskauth-split-value-stream.md`、NFR `2026-05-28-taskauth-split-nfr-clarification.md`

**Bounded Context:** 用户认证（taskAuth 限界上下文，Django accounts 为桥接与副作用上下文）

## 限界上下文

| 上下文 | 职责 | 部署 |
|--------|------|------|
| **taskAuth** | 凭证校验、Token 签发、注册/激活 DB 写入 | Go :8003 |
| **accounts-bridge** | URL 不变、delegate、fallback | Django |
| **accounts-side-effects** | 邮件、Kafka、UserSerializer、隐私条款 | Django internal API |

## 聚合

### CredentialAggregate（聚合根: SessionCredential）

- **实体**: 逻辑 UserRef（object_id + content_type_id）
- **值对象**:
  - `LoginIdentifier` — email/phone/username
  - `PasswordCredential` — 前端哈希密码（明文比较，与 Django 一致）
  - `AuthToken` — 40 字符 hex key
- **不变量**:
  - 邮箱登录必须 `is_verified=true`
  - 用户必须 `is_active=true`
  - Token 每 user 至多一条（get_or_create）

### RegistrationAggregate（聚合根: PendingEmailRegistration）

- **实体**: PendingEmailRegistration
- **值对象**: `EmailAddress`, `ActivationToken`, `PasswordCredential`
- **行为**: `register()` → 创建 user + login_method；`activate(token)` → is_verified

## 领域服务

| 服务 | 职责 |
|------|------|
| `AuthenticateWithPasswordService` | identifier + password → UserRef |
| `IssueTokenService` | UserRef → AuthToken |
| `AuthDelegationService` | 决定 taskAuth vs Django fallback（桥接策略） |

## 仓储接口（Go port / Django ABC）

```
CredentialRepository
  - FindLoginMethod(identifier) → LoginMethod
  - GetOrCreateToken(userRef) → AuthToken
  - DeleteToken(key)

RegistrationRepository
  - CreateUserWithEmailLogin(email, password) → (userID, activationToken)
  - ActivateByToken(token) → error

SideEffectPort (防腐层)
  - EnrichLogin(userID, body) → user JSON + redirect
  - PostRegister(userID, email, activationURL, body)
  - PostActivate(userID, methodType, identifier)
```

## 领域事件（由 Django 侧发出，taskAuth 触发）

| 事件 | 触发点 |
|------|--------|
| `UserCreated` | post-register |
| `UserActivated` | post-activate |

## 与 NFR 对齐

- **可用性 L3**: `AuthDelegationService` — remote 失败 → local fallback
- **一致性 L2**: CredentialAggregate 内 SQLite 事务
- **安全 L3**: SideEffectPort 调用带 internal secret

## 文件布局（已实现）

```
taskAuth/src/          # 应用层 + 基础设施（SQLite、HTTP）
task2app/.../taskauth_bridge/   # Django 桥接
task2app/.../taskauth_internal_views.py  # SideEffectPort 实现
```

领域逻辑当前内联于 Go handlers；后续可提取至 `taskAuth/domain/` 包。
