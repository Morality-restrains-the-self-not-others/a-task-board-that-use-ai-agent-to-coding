# DDD 领域模型: 登录跳转循环修复 — 网关统一认证

> 输入:
> - 设计文档: `docs/design/login-redirect-loop-fix.md`
> - 价值流: `docs/superpowers/plans/2026-06-23-login-redirect-loop-gateway-auth-fix-value-stream.md`
> - NFR 澄清: `docs/superpowers/plans/2026-06-23-login-redirect-loop-gateway-auth-fix-nfr-clarification.md`

## 声明

本次修复**不引入新领域概念**。现有 `用户认证上下文` 的领域模型保持不变：

| 概念 | 类型 | 现状 |
|------|------|------|
| User | Entity | `accounts.models.User` (id, is_active, is_superuser, is_staff) |
| Token | Value Object | 40-char hex key，存储在 `accounts_customtoken` |
| CustomTokenAuthentication | Infrastructure (DRF Auth) | `accounts/authentication.py` |
| TokenRegistry | Repository (接口) | `accounts/taskauth_bridge/token_registry.py` — 内存实现 |
| IdentityClient | Infrastructure (HTTP) | `accounts/taskauth_bridge/identity_client.py` |

## 改动归属

所有改动属于 **基础设施层**，不触及领域模型：

| 文件 | 层 | 改动 |
|------|-----|------|
| `delegate.py` | Infrastructure (HTTP bridge) | 传播响应头 + 登录时调用 `upsert_local_token` |
| `token_registry.py` | Infrastructure (Repository impl) | 新增 `upsert_local_token` / `resolve_local_token` DB 实现 |
| `principal_loader.py` | Infrastructure (Principal factory) | 增加本地 DB 回退分支 |
| `conf/` | Configuration | secret 匹配 |

## 仓储接口扩展

`token_registry` 作为 Token 的仓储，原仅有内存实现。本次新增 DB 实现，不改变接口语义：

```
TokenRegistry (implicit interface)
  ├── register_test_token(key, user_id)    — 测试用内存注册
  ├── resolve_test_token(key) → user_id    — 测试用内存解析
  ├── upsert_local_token(key, user_id)     — 新增: DB 持久化
  └── resolve_local_token(key) → user_id   — 新增: DB 解析
```

## 自检

- [x] 无新增实体/值对象/聚合
- [x] 无新增仓储接口（仅在现有接口新增实现方法）
- [x] 无新增领域事件
- [x] 无新增领域服务
- [x] 改动仅限于基础设施层
- [x] 领域模型文件无需修改
