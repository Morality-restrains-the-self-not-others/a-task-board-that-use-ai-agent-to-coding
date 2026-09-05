# 权限分析：PayPal 充值失败修复

> 日期：2026-06-29 | 基于：docs/specs/paypal-recharge-failure-fix/design.md

---

## 1. 权限影响矩阵

逐改动点分析：

| # | 改动点 | 主体 | 资源层级 | 操作 | 现有检查 | 是否缺失 | 建议 |
|---|--------|------|----------|------|----------|:---:|------|
| 1 | `RechargePaypalStatusView.get()` — 新增 client capture 回退 | 已认证用户 | Tenant | **读→写** (副作用) | `IsAuthenticated` + `resolve_tenant_id()` + `pending.user_id == request.user.id` | ⚠️ 风险可控 | GET 不应有写副作用；但幂等保证安全。建议加频率限制 |
| 2 | `RechargePaypalCreateView.post()` — 新增事件发布 | 已认证用户 | Tenant | write (已有) | `IsAuthenticated` + SMS验证 + tenant resolve | ✅ 充分 | — |
| 3 | `paypal_webhook()` — 新增事件发布 | PayPal 服务器 | System | write (已有) | `@csrf_exempt` + 签名校验 `verify_paypal_webhook_request()` | ✅ 充分 | 必须保持 auth-free，签名校验已足够 |
| 4 | `paypal_recharge.py` — 内部函数新增事件/日志 | 内部调用 | N/A | N/A | 内部服务函数，调用方已有权限检查 | ✅ 充分 | — |
| 5 | `taskBill credit.go` — outbox 重试 | 内部服务 | N/A | N/A | `TASKBILL_INTERNAL_SECRET` | ✅ 充分 | — |
| 6 | `BillingRecharge.vue` — 前端文案优化 | 浏览器 | N/A | read | 前端展示，API 层已有权限 | ✅ 充分 | — |

---

## 2. 关键风险分析：GET 端点新增写副作用

### 问题

修复方案 4.2 在 `RechargePaypalStatusView.get()` 中增加了服务端主动 capture 调用。
这使得一个 **GET 请求**具有**金融写操作**副作用（触发 PayPal capture + 账户 credit）。

### 风险评估

| 风险 | 缓解措施 | 评级 |
|------|---------|:---:|
| 重复 capture 导致重复扣款 | `_apply_paypal_credited_inner()` 幂等：同 order_id 只入账一次；`paypal_capture_order()` 幂等 | ✅ 安全 |
| 用户 A 触发用户 B 的 capture | `pending.user_id == request.user.id` 校验 | ✅ 安全 |
| 恶意高频调用冲击 PayPal API | 每次轮询都尝试 capture（正常 2s 间隔），PayPal rate limit 无风险 | 🟡 低风险 |
| GET 不符合 HTTP 语义 | REST 约定 GET 应无副作用，但这是 Webhook 降级回退 | 🟡 设计债 |

### 建议

在 `RechargePaypalStatusView.get()` 中，capture 触发前加以下保护：

```python
# 1. 仅当确认为 pending 且 webhook 未到达时触发
# 2. 对同一 order_id 的 capture 添加 Redis 锁（TTL 30s）防止并发重复调用
# 3. 单用户每分钟最多触发 3 次 capture fallback
```

---

## 3. 现有端点权限复查

对充值流程涉及的全部端点做一次快速审计：

| 端点 | 权限 | 租户校验 | 资源归属 | IDOR 风险 | 判定 |
|------|------|:---:|:---:|:---:|:---:|
| `GET recharge_phone_status/` | `IsAuthenticated` | ✅ URL tenant_id | ✅ user-scoped | — | ✅ |
| `POST recharge_send_sms/` | `IsAuthenticated` | ✅ URL tenant_id | ✅ user phone | 有：`phone_active_bound_to_other_user()` 检查 | ✅ |
| `POST recharge_verify_sms/` | `IsAuthenticated` | ✅ URL tenant_id | ✅ user phone | `phone_active_bound_to_other_user()` 检查 | ✅ |
| `POST recharge_paypal_create/` | `IsAuthenticated` | ✅ `resolve_tenant_id()` | ✅ `request.user.id` in custom_id | — | ✅ |
| `GET recharge_paypal_status/` | `IsAuthenticated` | ✅ `resolve_tenant_id()` | ✅ `pending.user_id == request.user.id` | — | ✅ |
| `POST recharge/` (admin) | `IsAuthenticated` + `is_staff` | ✅ `resolve_tenant_id()` | ✅ admin only | — | ✅ |
| `POST paypal/webhook/sandbox/` | `@csrf_exempt` | N/A (系统回调) | ✅ PayPal 签名校验 | — | ✅ |
| `POST emit-billing-event/` | `TASKBILL_INTERNAL_SECRET` | N/A (内部) | — | — | ✅ |

---

## 4. 安全审计清单

- [x] **IDOR 风险**: `RechargePaypalStatusView` 已有 `pending.user_id != request.user.id` 校验 → 安全
- [x] **权限提升**: 无新增 PATCH/PUT 端点 → 不适用
- [x] **跨租户泄露**: 所有查询限定 `resolve_tenant_id(request, tenant_id)` → 安全
- [x] **403 vs 404**: 充值端点均已认证；webhook 端点返回通用错误避免信息泄露 → 安全
- [x] **user_id 注入**: `RechargePaypalCreateView` 使用 `request.user.id`（非调用方传入） → 安全
- [x] **敏感操作**: 新增 capture 操作有幂等保护 + pending.user_id 校验 → 可控
- [x] **金融操作**: `credit_recharge` 通过 taskBill 幂等 txn_id 保护 → 安全

---

## 5. 测试用例清单

| # | 测试场景 | 角色 | 操作 | 预期 |
|---|---------|------|------|------|
| 1 | 正常用户轮询自己的 pending 订单 | authenticated user | `GET recharge_paypal_status/?order_id=own` | 200 + status=pending + 触发 capture fallback |
| 2 | 用户查询他人的订单 | authenticated user | `GET recharge_paypal_status/?order_id=other` | 403 |
| 3 | 未登录用户创建订单 | anonymous | `POST recharge_paypal_create/` | 401 |
| 4 | 未登录用户查询状态 | anonymous | `GET recharge_paypal_status/` | 401 |
| 5 | Webhook 无签名 POST | (PayPal) | `POST paypal/webhook/sandbox/` (无签名头) | 400 invalid signature |
| 6 | Webhook 有正确签名 POST | (PayPal) | `POST paypal/webhook/sandbox/` (正确签名) | 200 ok |
| 7 | taskBill emit-billing-event 无 secret | (taskBill) | `POST emit-billing-event/` (无 secret) | 403 |
| 8 | 同一 order_id 重复 capture | authenticated user + PayPal webhook | 两次 capture | 幂等，仅入账一次 |
| 9 | 跨租户查询 order status | other_tenant user | `GET recharge_paypal_status/?order_id=...` | 403 或 404 |

---

## 6. 风险评级

| 评级 | 项 | 说明 |
|:---:|------|------|
| 🟢 低 | 所有现有端点 | 权限模型充分，IDOR/跨租户/权限提升均有检查 |
| 🟡 中 | GET 写副作用 | 幂等保护有效，但违反 REST 约定。建议加频率限制 + Redis 锁 |
| 🟢 低 | 事件发布 | `send_event()` 是内部调用，不影响权限边界 |
| 🟢 低 | 日志补充 | 纯可观测性，无权限影响 |

**结论**: ✅ 绿灯 — 所有改动点权限边界清晰，现有检查充分。GET 写副作用是已知的设计权衡，安全可控。

---

## 总结清单

- RechargePaypalStatusView capture fallback: 方案1（保持现有 IsAuthenticated + user_id 校验，添加频率限制）、方案2（新增独立 POST 端点触发 capture，更 RESTful）
- GET 副作用: 方案1（接受 — 简单，幂等安全）、方案2（改为 POST — 更符合 REST，但前端改动大）
