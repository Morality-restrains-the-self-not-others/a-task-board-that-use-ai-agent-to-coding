# 设计文档: recharge_verify_sms API 500 AuthenticationServiceUnavailable 修复

**日期**: 2026-06-28
**请求**: `POST /api/tenant/{tenant_id}/billing/accounts/recharge_verify_sms/`
**现象**: HTTP 500 `AuthenticationServiceUnavailable` — taskAuth 不可达时未处理异常导致 500

---

## 1. 根因分析

### 1.1 错误定位

```
File: billing_bridge/recharge_views.py, line 123
    upsert_phone_login_method(str(request.user.id), phone)
    ↑ 未捕获 AuthenticationServiceUnavailable → 500

File: accounts/taskauth_bridge/login_methods_resolver.py, line 157
    raise AuthenticationServiceUnavailable
```

### 1.2 完整调用链

```
RechargeVerifySmsView.post()                          # recharge_views.py:109
  ├─ VerificationCodeService.verify_code()            # line 118 — 消费短信验证码 ✅
  ├─ phone_active_bound_to_other_user()               # line 121 — 可抛 AuthenticationServiceUnavailable ❌
  │   └─ phone_taken_by_other()                       # login_methods_resolver.py:86
  │       └─ forward_to_taskauth() → HTTP             # 不可达时 raise AuthenticationServiceUnavailable
  └─ upsert_phone_login_method()                      # line 123 — 可抛 AuthenticationServiceUnavailable ❌
      └─ forward_to_taskauth() → HTTP PATCH           # login_methods_resolver.py:143
          └─ 不可达时 raise AuthenticationServiceUnavailable # line 153
```

### 1.3 两个缺陷

**缺陷 A: 缺少异常处理**
- `phone_active_bound_to_other_user` (line 121) 调用 `phone_taken_by_other` → taskAuth HTTP，异常未捕获
- `upsert_phone_login_method` (line 123) → taskAuth HTTP PATCH，异常未捕获
- 两者均可能因 taskAuth 不可达而抛出 `AuthenticationServiceUnavailable` → HTTP 500

**缺陷 B: 操作顺序不当**
- 短信验证码在 line 118 被消费（`verify_code`）
- taskAuth 检查在 line 121 和 line 123 才执行
- 若 taskAuth 不可达，验证码已被消费但流程失败，用户需重新获取验证码

### 1.4 为什么 bound 用户不受影响

```python
# line 113-126
bound = get_user_bound_phone(request.user)
...
if not bound:                                          # line 120 — 仅有未绑定手机的用户进入此分支
    if phone_active_bound_to_other_user(...):          # ← 仅未绑定用户触发 taskAuth 调用
    upsert_phone_login_method(...)                     # ← 仅未绑定用户触发 taskAuth 调用
```

**触发条件**: 用户首次充值、未绑定手机号。已绑定手机的用户走 `bound` 分支直接跳过 taskAuth 调用。

### 1.5 对比：项目中已有的正确处理模式

项目中多处已正确处理 `AuthenticationServiceUnavailable`，例如：

| 文件 | 处理方式 |
|------|----------|
| `accounts/authentication.py:48,59` | try-except → 503 |
| `accounts/github_app_views.py:332,508` | try-except → 503 |
| `accounts/taskauth_internal_views.py:159` | try-except → 503 |
| `accounts/views/user_views.py:251` | try-except → re-raise（让上层处理） |
| `accounts/views/utils.py:21` | try-except → 降级处理 |

`billing_bridge/recharge_views.py` 是少数**完全没有处理**此异常的视图之一。

---

## 2. 影响范围

| 维度 | 分析 |
|------|------|
| **受影响端点** | `POST /api/tenant/{id}/billing/accounts/recharge_verify_sms/` |
| **触发条件** | 用户未绑定手机号 + taskAuth 服务不可达（网络故障、宕机、超时） |
| **数据完整性** | 短信验证码被消费但绑定未完成，用户状态不一致（需重新发码） |
| **频率** | taskAuth 正常运行时不触发；仅在 taskAuth 故障时影响未绑定手机用户 |
| **同类问题** | `phone_active_bound_to_other_user` 在 `billing_bridge/utils.py:48-51` 也只是 re-raise，调用方若不处理同样会 500。已在本次修复中一并覆盖。 |

---

## 3. 修复方案

### 方案 A: 异常捕获 + 操作重排（推荐）⭐

在 `RechargeVerifySmsView.post()` 中：
1. **重排操作顺序**：将 `phone_active_bound_to_other_user` 检查移到 `verify_code` 之前
2. **添加异常处理**：对所有 taskAuth 调用添加 `AuthenticationServiceUnavailable` 捕获 → 503
3. **添加 ValueError 处理**：`upsert_phone_login_method` 的 `ValueError` 捕获 → 400

```python
# 改造后的 RechargeVerifySmsView.post()

def post(self, request, tenant_id=None, **kwargs):
    serializer = RechargeVerifySmsSerializer(data=request.data)
    if not serializer.is_valid():
        return Response(serializer.errors, status=status.HTTP_400_BAD_REQUEST)

    bound = get_user_bound_phone(request.user)
    pending_key = recharge_pending_phone_cache_key(request.user.id)
    phone = bound if bound else cache.get(pending_key)
    if not phone:
        return Response({'error': '请先获取短信验证码'}, status=status.HTTP_400_BAD_REQUEST)

    # --- 新增：taskAuth 检查前置（在消费验证码之前）---
    if not bound:
        try:
            if phone_active_bound_to_other_user(phone, request.user.id):
                return Response({'error': '该手机号已被其他账号绑定'}, status=status.HTTP_400_BAD_REQUEST)
        except AuthenticationServiceUnavailable:
            return Response(
                {'detail': 'authentication service unavailable'},
                status=status.HTTP_503_SERVICE_UNAVAILABLE,
            )

    # 原有：验证码校验
    if not VerificationCodeService.verify_code(phone=phone, code=serializer.validated_data['code']):
        return Response({'error': '验证码无效或已过期'}, status=status.HTTP_400_BAD_REQUEST)

    # --- 修改：phone 绑定添加异常处理 ---
    if not bound:
        try:
            upsert_phone_login_method(str(request.user.id), phone)
        except AuthenticationServiceUnavailable:
            return Response(
                {'detail': 'authentication service unavailable'},
                status=status.HTTP_503_SERVICE_UNAVAILABLE,
            )
        except ValueError as e:
            return Response({'error': str(e)}, status=status.HTTP_400_BAD_REQUEST)
        cache.delete(pending_key)

    set_recharge_sms_verified(request.user)
    return Response({'message': '验证成功', 'sms_verified': True})
```

**优点：**
- taskAuth 不可达时返回 503（而非 500），符合项目已有错误处理惯例
- 前置检查确保验证码不被浪费：taskAuth 故障时验证码尚未消费
- `ValueError` 捕获提供友好的 `phone_taken` / `username_taken` 错误消息
- 改动最小，仅修改一个视图方法
- 与项目中 `accounts/authentication.py`、`github_app_views.py` 等已有的处理模式一致

**缺点：**
- `upsert_phone_login_method` 仍在验证码消费之后执行，若此步骤失败（小概率，因为 `phone_taken_by_other` 已先成功），验证码仍被浪费

### 方案 B: 异步绑定（大重构）

将 phone login method 绑定从充值流程中解耦，改为发领域事件异步处理：
- `RechargeVerifySmsView` 只负责短信验证 + 标记 `sms_verified`
- 绑定手机到 taskAuth 由事件消费者异步执行
- 失败可重试，不阻塞充值流程

**优点：**
- 彻底解耦，充值流程不依赖 taskAuth 可用性
- 失败可自动重试

**缺点：**
- 改动范围大（涉及领域事件定义、消费者实现、taskAuth 回调）
- 违反"版本 ≤ 1.0.0 未上线阶段禁止降级/兼容层"约束（异步绑定实质上是降级：绑定失败不影响充值）
- 需引入异步基础设施，超出 bug fix 范围

### 推荐

**方案 A**：bug fix 应做最小精确修复。方案 A 解决了核心问题（500 → 503、验证码浪费），改动局限在一个视图方法内，且与项目既有错误处理模式一致。方案 B 的异步绑定可作为后续架构演进的方向，但不适合作为 bug fix。

---

## 4. 修复文件清单

| 文件 | 改动 | 类型 |
|------|------|------|
| `Saas_project/billing_bridge/recharge_views.py:106-126` | `RechargeVerifySmsView.post()` 重排操作顺序 + 添加 try-except | ~25行修改 |
| `Saas_project/tests/test_billing_recharge_validation.py` | 新增 `test_recharge_verify_sms_503_when_taskauth_unavailable` 测试 | 新增测试函数 |
| `Saas_project/tests/test_billing_recharge_validation.py` | 恢复 `test_recharge_verify_binds_phone_when_user_had_no_binding`（取消 skip，加 taskAuth HTTP mock） | 修改已有测试 |

---

## 5. 领域概念清单

（本次为 bug fix，不引入新领域概念）

| 概念 | 类型 | 上下文 |
|------|------|--------|
| BillingAccount | Aggregate Root (已有) | 计费 — 充值账户 |
| PhoneLoginMethod | Entity (已有) | 用户与认证 — 手机号登录方式，由 taskAuth 管理 |
| RechargeVerification | Domain Service (已有) | 计费 — 充值前的短信验证流程 |
| AuthenticationServiceUnavailable | Domain Exception (已有) | 用户与认证 — taskAuth 不可达信号 |

---

## 6. 价值流影响

无 `value-stream.yaml` 配置文件。此修复涉及的价值场景：
- **充值流程** — 修复后未绑定手机用户在 taskAuth 不可达时收到 503（而非 500），已绑定用户不受影响
- **用户认证** — 不改变认证逻辑，仅修复异常传播路径

---

## 7. 测试策略

### 7.1 新增测试：taskAuth 不可达时返回 503

```python
@pytest.mark.django_db
def test_recharge_verify_sms_503_when_taskauth_unavailable():
    """未绑定手机用户充值验证时 taskAuth 不可达 → 503，且验证码未消费。"""
    # 1. 创建未绑定手机的用户
    # 2. mock forward_to_taskauth 抛出 ConnectionError
    # 3. POST recharge_verify_sms
    # 4. 断言 status_code == 503
    # 5. 断言 VerificationCode.is_used == False（验证码未被消费）
```

### 7.2 恢复已有测试

当前两个相关测试均被 skip (`LoginMethod ORM deleted — needs taskAuth HTTP mock`)：
- `test_recharge_verify_sms_sets_short_lived_flag` (line 226)
- `test_recharge_verify_binds_phone_when_user_had_no_binding` (line 277)

在完成 taskAuth HTTP mock 基础设施后应恢复这些测试。本次修复可先新增 503 测试，skip 的测试留待后续统一处理。

---

## 8. 附加：`billing_bridge/utils.py` 中的同类问题

`phone_active_bound_to_other_user()` (utils.py:38-51) 对 `AuthenticationServiceUnavailable` 仅 re-raise，未提供有意义的处理。调用方必须自行捕获。当前调用方：

| 调用方 | 是否处理 | 风险 |
|--------|----------|------|
| `RechargeVerifySmsView.post:121` | ❌ (本次修复后 ✅) | 已修复 |
| `RechargeSendSmsView.post:96` | ❌ 无处理 | ⚠️ 同样可能 500 |

`RechargeSendSmsView.post:96` 调用 `phone_active_bound_to_other_user` 也未处理异常，但该调用在**验证码发送之前**，失败时验证码尚未发送，不会浪费。但同样应返回 503 而非 500。可作为本次修复的附加改动一并处理。

---

总结清单：
- 缺陷A（缺少异常处理）: 方案A（try-except + 503）、方案B（异步绑定解耦）
- 缺陷B（操作顺序）: 重排（taskAuth检查前置）、不重排（仅加异常处理）
- 附加修复范围: 仅修 RechargeVerifySmsView、同时修复 RechargeSendSmsView 的同类问题
