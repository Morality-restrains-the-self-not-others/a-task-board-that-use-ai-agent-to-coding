# 设计文档：充值 SMS 验证状态丢失 Bug 修复

**日期**: 2026-06-29
**状态**: 待审批
**Trace IDs**: `web-1782712873805-9spx132pqw` (verify), `web-1782712874014-wz5uhxs6o8` (status)

---

## 1. 问题描述

充值页面 (`/tenant/{id}/billing/recharge/`) 在用户完成 SMS 验证码校验后，前端轮询验证状态时发现状态未持久化：

### 复现步骤

1. `POST /api/tenant/{id}/billing/accounts/recharge_verify_sms/` — 提交验证码 `702288`
2. 服务端返回 `{"message":"验证成功","sms_verified":true}` — **验证成功**
3. `GET /api/tenant/{id}/billing/accounts/recharge_phone_status/` — 查询验证状态
4. 服务端返回 `{"has_phone":false,"sms_verified":false,...}` — **状态丢失**

### 影响范围

- 无绑定手机号的用户在完成 SMS 验证后无法立即进行充值（PayPal 创建 / 管理员直充都需要检查 `sms_verified`）
- 用户必须等待 **60 秒**（identity 缓存 TTL 过期）后刷新页面才能正常充值

---

## 2. 根因分析

### 2.1 直接原因：两层缓存不一致

问题涉及 **两个独立的缓存层**，在 SMS 验证成功后未协同刷新：

| 缓存层 | Key | TTL | 写入时机 | 读取时机 |
|--------|-----|-----|---------|---------|
| **L1: Identity 缓存** | `taskauth:identity:user:{uid}` | 60s | `TaskAuthIdentityClient.get_user()` HTTP 调用后 | `get_user_bound_phone()` → `RechargePhoneStatusView` |
| **L2: SMS 验证缓存** | `billing_recharge_sms_ok:{uid}` | 900s (15min) | `set_recharge_sms_verified()` SMS 验证成功后 | `is_recharge_sms_verified_for_user()` — 仅在 `has_phone=True` 时调用 |

### 2.2 时序分析

```
时间线 (同一用户，user_id=858949970248626176):

T0: 用户加载充值页面
    → 前端调用 GET recharge_phone_status/
    → get_user_bound_phone(user)
    → get_identity_client().get_user("858949970248626176")
    → taskAuth HTTP 返回: {login_methods: []}  ← 无手机绑定
    → 写入 Identity 缓存: taskauth:identity:user:858949970248626176
       = ResolvedUser(login_methods=())  ← TTL=60s

T1: 用户输入手机号，点击发送验证码
    → POST recharge_send_sms/ {"phone":"..."}
    → SMSService 发送验证码
    → 写入缓存: verification_code:{phone} = "702288"

T2: 用户输入验证码 702288，点击验证
    → POST recharge_verify_sms/ {"code":"702288"}
    → VerificationCodeService.verify_code() ✅
    → upsert_phone_login_method(user_id, phone) ✅  ← taskAuth 写入手机绑定
    → set_recharge_sms_verified(user) ✅             ← 写入 SMS 缓存
    → 返回 {"sms_verified":true}

T3: 前端轮询 GET recharge_phone_status/
    → get_user_bound_phone(user)
    → get_identity_client().get_user("858949970248626176")
    → 🔴 Identity 缓存 HIT! 返回 TTL=60s 内缓存的旧数据
       ResolvedUser(login_methods=())  ← 仍然无手机!
    → has_phone = false
    → 🔴 进入 if not phone: 分支，sms_verified 硬编码为 False
    → is_recharge_sms_verified_for_user() 从未被调用!
```

### 2.3 关键代码缺陷

**缺陷 1: Identity 缓存未在 phone upsert 后刷新**

`/task2app/Saas_project/accounts/taskauth_bridge/identity_client.py:47-48`:
```python
cache_key = f'{CACHE_PREFIX}user:{uid}'
cached = cache.get(cache_key)
if cached is not None:
    return cached if cached != '__missing__' else None  # ← 60s 内返回旧数据
```

`upsert_phone_login_method()` 在 `/task2app/Saas_project/accounts/taskauth_bridge/login_methods_resolver.py:125-173` 中成功更新 taskAuth 后，**未清除** identity 缓存 `taskauth:identity:user:{uid}`。

**缺陷 2: RechargePhoneStatusView 在 has_phone=False 时跳过 SMS 缓存检查**

`/task2app/Saas_project/billing_bridge/recharge_views.py:60-66`:
```python
def get(self, request, tenant_id=None, **kwargs):
    phone = get_user_bound_phone(request.user)
    if not phone:
        return Response({
            'has_phone': False,
            'phone_masked': '',
            'sms_verified': False,  # ← 硬编码 False，未检查 SMS 缓存
            ...
        })
```

即使用户已完成 SMS 验证且 `billing_recharge_sms_ok:{uid}=1` 已写入缓存，只要 Identity 缓存返回的 `has_phone` 为 false，`is_recharge_sms_verified_for_user()` 就永远不会被调用。

---

## 3. Kafka 事件分析

### 3.1 当前事件流

本问题的 SMS 验证流程 **不涉及任何 Kafka 事件**：

- `VerificationCodeService.send_code()` → 仅写 DB + 缓存，**不发事件**
- `VerificationCodeService.verify_code()` → 仅标记 `is_used=True`，**不发事件**
- `upsert_phone_login_method()` → HTTP PATCH 到 taskAuth，**不发事件**
- `set_recharge_sms_verified()` → 仅写缓存，**不发事件**

相关的 Kafka 事件仅有 `BILLING_TRANSACTION_CREATED`（充值入账后发出），在本 Bug 的时间点（SMS 验证阶段）尚未触发。

### 3.2 结论

本 Bug **与 Kafka 无关**，不需要新增 Kafka 事件。SMS 验证是会话级短时状态（15 分钟 TTL），不适合用领域事件传递。

---

## 4. Grafana / Loki 日志分析

### 4.1 可通过 Loki 查询验证的关键日志

以下日志点可用于确认根因（按 `X-Trace-Id` 关联）：

```
# 验证 SMS 成功的日志 (trace: web-1782712873805-9spx132pqw)
{service="django"} |= "RechargeVerifySmsView"
  → 预期看到 "验证成功" 但无 Identity 缓存清理日志

# 状态查询返回 false 的日志 (trace: web-1782712874014-wz5uhxs6o8)
{service="django"} |= "RechargePhoneStatusView"
  → 预期看到 taskAuth HTTP 调用被跳过（因为 Identity 缓存命中）

# taskAuth HTTP 日志
{service="django"} |= "taskAuth HTTP" |= "GET" |= "858949970248626176"
  → 预期：T2 之后 60s 内无 GET /api/accounts/users/{uid}/ 请求
  → 说明 Identity 缓存命中，跳过了 HTTP 调用
```

### 4.2 关键日志缺失

当前代码中，`TaskAuthIdentityClient.get_user()` 的缓存命中/未命中 **无日志记录**，这是可观测性盲区。建议补日志。

---

## 5. 解决方案设计

### 5.1 修复策略：双保险

采用 **两个互补修复**，任一修复单独生效即可解决问题，双重保障：

#### 修复 A: Identity 缓存刷新（治本）

在 `upsert_phone_login_method()` 成功后清除 identity 缓存，确保下一次 `get_user()` 走 HTTP 获取最新数据。

**文件**: `accounts/taskauth_bridge/login_methods_resolver.py`

```python
def upsert_phone_login_method(user_id: str, phone: str) -> None:
    # ... existing code ...
    # 成功后清除 Identity 缓存
    from django.core.cache import cache
    cache.delete(f'taskauth:identity:user:{uid}')
```

同样需要在 `upsert_username_login_method()` 中做相同处理（保持一致性）。

#### 修复 B: RechargePhoneStatusView 防御性检查（治标）

当 `has_phone=False` 时，仍然检查 SMS 验证缓存，防止 Identity 缓存不一致导致的状态丢失。

**文件**: `billing_bridge/recharge_views.py`

```python
class RechargePhoneStatusView(APIView):
    def get(self, request, tenant_id=None, **kwargs):
        phone = get_user_bound_phone(request.user)
        sms_verified = is_recharge_sms_verified_for_user(request.user)
        return Response({
            'has_phone': bool(phone),
            'phone_masked': mask_phone(phone) if phone else '',
            'sms_verified': sms_verified,  # ← 无论 has_phone 如何都检查
            'paypal_enabled': paypal_configured(),
            'paypal_currency': ...,
        })
```

### 5.2 推荐方案：A + B 同时实施

两个修复不冲突，组合后形成纵深防御：
- 修复 A 消除了缓存不一致的根源
- 修复 B 提供了兜底保护，防止未来任何原因导致的 `has_phone` / `sms_verified` 不一致

### 5.3 不推荐的方案

| 方案 | 理由 |
|------|------|
| 缩短 Identity 缓存 TTL | 治标不治本，窗口期缩小但未消除 |
| 新增 Kafka 事件通知缓存刷新 | SMS 验证是短时会话状态，Kafka 异步传递延迟 + 复杂度远超收益 |
| 前端轮询增加延迟重试 | 把后端 bug 推给前端，用户体验差 |

---

## 6. 价值流影响

### 6.1 现有流影响

| 价值流 | 影响 |
|--------|------|
| `system-admin-phone-login-recharge-policy` (L817) | 修复后 `recharge-policy-core` 步骤的 SMS 验证状态读取正确性得到保障 |
| `increment4-billing-sse-go` (L1519) | 无直接影响（充值入账后的事件流不变） |
| `billing-account-event-driven-init` (L2279) | 无影响 |

### 6.2 字段影响

| 服务.表.字段 | 变更类型 | 说明 |
|-------------|---------|------|
| `saas-backend.accounts_sms_verification_code.is_used` | 不变 | 验证码消费逻辑不变 |
| `task-auth.accounts_login_method.identifier` | 不变 | phone upsert 逻辑不变，仅增加缓存刷新 |

### 6.3 测试影响

| 测试文件 | 操作 | 说明 |
|---------|------|------|
| `tests/test_billing_recharge_validation.py` | 新增用例 | 新增 `test_recharge_phone_status_sms_verified_without_bound_phone` — 模拟 Identity 缓存有旧数据但 SMS 缓存已设置的场景 |
| `tests/test_billing_recharge_validation.py` | 新增用例 | 新增 `test_upsert_phone_login_method_invalidates_identity_cache` — 验证 phone upsert 后 identity 缓存被清除 |

---

## 7. 领域概念清单

| 概念 | 类型 | 所属限界上下文 |
|------|------|-------------|
| `RechargeSmsVerification` | 领域服务 | Billing (充值) |
| `VerificationCode` | 实体 | Accounts (账号) |
| `PhoneLoginMethod` | 实体 (taskAuth 侧) | Auth (认证) |
| `IdentityCache` | 基础设施关注点 | Auth (认证) |
| `SmsVerifiedFlag` | 值对象 (缓存键) | Billing (充值) |
| `BillingAccount` | 聚合根 | Billing (充值) |

**候选领域事件**（本次不引入，留待后续 DDD 建模时评估）：
- `PHONE_BOUND` — 手机号成功绑定到用户
- `RECHARGE_SMS_VERIFIED` — 充值 SMS 验证完成

---

## 8. 实现计划摘要

| 步骤 | 文件 | 改动 |
|------|------|------|
| 1 | `accounts/taskauth_bridge/login_methods_resolver.py` | `upsert_phone_login_method()` 成功后 `cache.delete(f'taskauth:identity:user:{uid}')` |
| 2 | `accounts/taskauth_bridge/login_methods_resolver.py` | `upsert_username_login_method()` 成功后同样清除 identity 缓存（一致性） |
| 3 | `billing_bridge/recharge_views.py` | `RechargePhoneStatusView.get()` 无论 `has_phone` 都调用 `is_recharge_sms_verified_for_user()` |
| 4 | `tests/test_billing_recharge_validation.py` | 新增 2 个测试用例 |
| 5 | `accounts/taskauth_bridge/identity_client.py` | 缓存命中/未命中补 info 日志（可观测性提升） |

---

## 9. 风险评估

| 风险 | 概率 | 影响 | 缓解 |
|------|------|------|------|
| Identity 缓存删除影响其他并发请求 | 低 | 低 | `cache.delete` 是原子操作，最坏情况是一次额外的 taskAuth HTTP 调用 |
| 修复后 phone upsert 失败导致缓存被误删 | 极低 | 低 | 代码仅在 `upsert` 成功后（无异常抛出）才执行 `cache.delete` |
| 多 worker 环境下 locmem 缓存不一致 | 中 | 已存在 | 此问题在修复前已存在（两个缓存键在不同 worker 中设置），修复 B 可作为缓解。生产环境应使用 Redis 集中缓存 |

---

## 总结清单

- **根因**: Identity 缓存 (60s TTL) 在 phone upsert 后未刷新 + RechargePhoneStatusView 在 `has_phone=false` 时跳过 SMS 缓存检查
- **修复 A**: `upsert_phone_login_method` 成功后清除 identity 缓存（治本）
- **修复 B**: `RechargePhoneStatusView` 无论 `has_phone` 如何都检查 SMS 缓存（治标 + 兜底）
- **Kafka**: 本 Bug 不涉及 Kafka 事件，无需新增
- **Grafana/Loki**: Trace ID 已提供，可通过 Loki 查询关联日志验证分析；建议补 identity 缓存命中日志
