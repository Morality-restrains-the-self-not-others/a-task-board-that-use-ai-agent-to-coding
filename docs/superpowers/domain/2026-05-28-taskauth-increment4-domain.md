# 领域模型: taskAuth Increment 4 — 密码重置

> 输入: 价值流 `2026-05-28-taskauth-increment4-value-stream.md`

**Bounded Context:** 用户认证（taskAuth）+ accounts-side-effects（Django）

## 聚合扩展

### PasswordResetAggregate（聚合根: LoginMethodResetSession）

- **实体**: LoginMethodRef（id + identifier + method_type）
- **值对象**:
  - `ResetToken` — token + expires_at（24h）
  - `PasswordCredential` — 前端哈希 password_hash
- **不变量**:
  - 仅已注册且 method_type=email 可发链接
  - token 使用后必须清空 password_reset_token 字段
  - 过期 token 不可重置

## 领域服务

| 服务 | 职责 |
|------|------|
| `IssueResetLinkService` | 生成 token、持久化、触发 SideEffectPort 发邮件 |
| `CompleteResetWithLinkService` | 校验 token、写 password_hash、清 token |
| `CompleteResetWithCodeService` | 经 VerificationCodePort 验码后写 password_hash |

## 防腐层端口

```
SideEffectPort
  - PostPasswordResetLink(email, token) → 发 Kafka 邮件

VerificationCodePort (Django)
  - SendPasswordResetCode(phone?, email?)
  - VerifyPasswordResetCode(phone?, email?, code) → bool
```

## 仓储（Go 基础设施）

```
LoginMethodRepository
  - FindByEmail(email)
  - FindByPasswordResetToken(token)
  - SetPasswordResetToken(id, token, expires)
  - UpdatePasswordHashClearToken(id, passwordHash)
```

实现位于 `taskAuth/src/db.go`（Increment 4 不强制提取 `taskAuth/domain/` 包，与 Increment 1–3 一致）。
