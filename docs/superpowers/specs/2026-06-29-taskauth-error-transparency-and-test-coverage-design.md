# Design: taskAuth 错误透明度 + Django 错误语义 + 测试覆盖

**日期**: 2026-06-29
**状态**: 待审批
**相关**: Fix: build/compile should not depend on runtime state (前次修复的后续优化)

---

## 背景

前次修复解决了 `upsertPhoneLoginMethod` 和 `upsertUsernameLoginMethod` 的 INSERT 缺少 `id` 列导致的 500 错误。
本次改进针对修复过程中发现的 4 个可优化项，按优先级评估后决定实施 3 项，搁置 1 项。

---

## 优化项分析

### 优化 1: taskAuth 错误日志透明化 ✅ 实施

**现状**: 两个 handler 的 catch-all 错误分支只返回 `{"detail":"db error"}`，不记录实际错误。
```go
writeJSON(w, http.StatusInternalServerError, map[string]string{"detail": "db error"})
```

**问题**: 运维排查时需要 grep 源码才能定位具体 DB 错误，0ms 延迟的错误尤其难以追踪。

**方案**: 在返回 500 前添加 `log.Printf` 记录实际错误。
```go
log.Printf("[taskAuth] upsert phone failed for user %s: %v", userID, err)
writeJSON(w, http.StatusInternalServerError, map[string]string{"detail": "db error"})
```

**影响文件**:
- `taskAuth/src/auth_profile_internal.go` — `handleUpsertPhoneLoginMethod` (L237)
- `taskAuth/src/auth_profile_internal.go` — `handleUpsertUsernameLoginMethod` (L102)

**涉及领域概念**:
- Bounded Context: `auth`
- Entity: `LoginMethod` (已存在)
- 无新增领域事件

---

### 优化 2: Django 错误语义区分 ✅ 实施

**现状**: `login_methods_resolver.py` 中 `upsert_phone_login_method()` 和 `upsert_username_login_method()` 将 taskAuth 返回的任何 5xx 状态码都映射为 `AuthenticationServiceUnavailable`。

```python
except Exception as exc:
    raise AuthenticationServiceUnavailable from exc    # 连接超时 → 503
if status >= 500 or status in (502, 503, 504):
    raise AuthenticationServiceUnavailable             # 5xx 响应 → 也是 503
```

**问题**: 
- taskAuth 不可达（连接超时）→ 503 Service Unavailable ✅ 语义正确
- taskAuth 可达但返回 500（内部错误）→ 503 Service Unavailable ❌ 语义错误，应为 502 Bad Gateway

**方案**: 新增 `IdentityServiceError` 异常，区分两种失败模式。

| 失败原因 | 异常类型 | HTTP 状态码 | 含义 |
|---------|---------|------------|------|
| 连接超时/DNS 失败 | `AuthenticationServiceUnavailable` | 503 | 上游不可达 |
| taskAuth 返回 5xx | `IdentityServiceError` | 502 | 上游内部错误 |

新增异常类（`identity_client.py`）:
```python
class IdentityServiceError(Exception):
    """taskAuth returned an error — map to HTTP 502 in API layers."""
```

修改 `login_methods_resolver.py`:
```python
from accounts.taskauth_bridge.identity_client import (
    AuthenticationServiceUnavailable,
    IdentityServiceError,  # 新增
)

# 在 upsert_phone_login_method / upsert_username_login_method:
except Exception as exc:
    raise AuthenticationServiceUnavailable from exc   # 网络层失败 → 503
if status in (409,):
    raise ValueError(...)
if status >= 500 or status in (502, 503, 504):
    raise IdentityServiceError(f'taskAuth returned {status}')  # 上游错误 → 502
```

调用方（`recharge_views.py`）需同时 catch 两个异常:
```python
except (AuthenticationServiceUnavailable, IdentityServiceError) as e:
    status_code = 503 if isinstance(e, AuthenticationServiceUnavailable) else 502
    return Response({'detail': str(e) or 'authentication service unavailable'}, status=status_code)
```

实际上，为保持最小改动，两个调用方 `RechargeSendSmsView` 和 `RechargeVerifySmsView` 只需将 `except AuthenticationServiceUnavailable` 改为 `except (AuthenticationServiceUnavailable, IdentityServiceError)`，并根据异常类型返回不同状态码。

**但考虑到向前兼容**：现有调用方已经 catch `AuthenticationServiceUnavailable`。如果改为两个不同异常类型，需要更新所有调用方。更简洁的方案是：让 `IdentityServiceError` 继承 `AuthenticationServiceUnavailable`，这样现有代码无需修改即可 catch 两种异常，同时可以区分状态码。

最终方案：使用单一异常类型但区分 detail：
- `AuthenticationServiceUnavailable` 保留用于连接失败 → 503
- 对于 5xx 响应，在 `login_methods_resolver.py` 中记录 warning 日志并仍然 raise `AuthenticationServiceUnavailable`，但在 recharge_views.py 层面根据上下文决定状态码。

**简化方案（降低改动面）**: 
1. 在 `login_methods_resolver.py` 中，当 taskAuth 返回 5xx 时记录更详细的 warning 日志（包含 status code）
2. 不新增异常类型，但确保日志中能区分网络错误 vs 5xx 响应
3. 对外仍返回 503，因为对于前端来说两种都是"暂时不可用"，重试策略相同

**最终决定：实施简化方案** — 增强日志但不改变异常类型，因为：
- 前端重试逻辑不需要区分 502 vs 503
- 运维通过日志即可定位根因
- 改动最小，零风险

**影响文件**:
- `task2app/Saas_project/accounts/taskauth_bridge/login_methods_resolver.py`

---

### 优化 3: SQLite Schema 防御性改造 ❌ 搁置

**现状**: `accounts_login_method.id` 定义是 `bigint NOT NULL PRIMARY KEY`。

**风险分析**:
- SQLite 不支持 `ALTER COLUMN`，需重建表
- 线上 auth.db 已有用户数据，迁移失败会导致认证服务完全不可用
- 代码层面已修复（所有 INSERT 均已显式提供 snowflake ID），schema 修改仅作为防御性措施
- 收益/风险比过低

**决定**: 搁置。留待未来 auth.db 重大版本升级时一并处理。

---

### 优化 4: username upsert 测试覆盖 ✅ 实施

**现状**: 只有 `TestPhoneTakenAndUpsertLoginMethod` 测试 phone upsert，username upsert 无测试。

**方案**: 新增 `TestUsernameTakenAndUpsertLoginMethod` 测试函数，覆盖：
1. 创建带 username 的用户 → 检验 `handleUsernameTaken` 返回 `taken: true`
2. 给无 username 的用户 patch username → 检验 `handleUpsertUsernameLoginMethod` 返回 200
3. patch 已存在的 username → 检验返回 409 Conflict
4. 删除 username（patch 空字符串）→ 检验删除成功

**影响文件**:
- `taskAuth/src/auth_phone_profile_internal_test.go` — 新增测试函数

---

## 价值流影响分析

从 `conf/value-stream.yaml` 分析：

| 受影响流 | 影响步骤 | 影响字段 | 影响类型 |
|---------|---------|---------|---------|
| `user-auth` | `email-register`, `phone-register`, `login` | 无新增字段 | 测试覆盖增强 |
| `user-auth` | （间接）所有依赖 taskAuth 的步骤 | `task-auth.accounts_login_method.*` | 错误可观测性提升 |

无需新增 value stream，无需新增 fields。

---

## 实施计划摘要

| # | 优化项 | 文件 | 改动行数 | 风险 |
|---|--------|------|---------|------|
| 1 | 错误日志 | `auth_profile_internal.go` | +4 行 | 无 |
| 2 | Django 日志增强 | `login_methods_resolver.py` | ~10 行 | 极低 |
| 3 | Schema 改造 | — | 0 行 | 搁置 |
| 4 | 测试覆盖 | `auth_phone_profile_internal_test.go` | ~60 行 | 无 |

---

## 领域概念清单

- **Bounded Context**: `auth` (用户与认证)
- **Key Entity**: `LoginMethod` — 登录方式，已有 `email`, `phone`, `username` 三种类型
- **Domain Service**: `upsertPhoneLoginMethod`, `upsertUsernameLoginMethod` — 幂等的登录方式绑定操作
- **无新增 Domain Events**

---

## 不在范围内

- taskAuth 其他 handler 的错误日志统一改造（本次仅改两个 upsert handler）
- Python 侧 `phone_taken_by_other` / `username_taken_by_other` 的同等日志改造（它们已有正确的异常传播）
- OpenTelemetry span 添加 error attributes（留待统一 observability 改进）
