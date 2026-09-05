# NFR 澄清: PayPal 充值 Webhook 修复

> 输入:
> - 设计文档: `docs/specs/paypal-recharge-failure-fix/design.md`
> - 价值流文档: `docs/superpowers/plans/2026-06-29-paypal-recharge-webhook-fix-value-stream.md`
>
> 输出使用者: DDD 建模, 实施计划, 构建

## NFR 概览表

| 类别 | 等级 | 一句话量化 |
|------|------|-----------|
| 安全性 | L3 | Webhook 签名校验 + 内部 API secret + user_id 校验 |
| 数据一致性 | L3 | 幂等 capture/credit，outbox 事件至少一次投递 |
| 可用性 | L2 | Webhook 不可达时 client capture 回退，RTO < 2min |
| 容错机制 | L2 | Outbox 重试 (1min 间隔)，capture 回退有幂等保护 |
| 可观测性 | L2 | 全链路 INFO 日志 + 3 个新领域事件 |
| 性能 | L0 | 跳过 — Bug 修复，无新性能要求 |
| 可伸缩性 | L0 | 跳过 — 当前用户量级，单实例满足 |
| 合规与隐私 | L0 | 跳过 — 无新增合规要求 |
| 可维护性 | L0 | 跳过 — 无新增部署/版本策略要求 |

## 逐增量 NFR 分析

### Increment 1: Webhook URL 修复 (P0)

#### 安全性 — L3
- **等级**: L3 增强
- **量化目标**: Webhook 签名校验成功率 100%（真实 PayPal 回调）
- **已有保障**: `verify_paypal_webhook_request()` 校验 transmission_id + cert_url + signature

#### 可用性 — L2
- **等级**: L2 标准
- **量化目标**: Webhook URL 更新后 PayPal 投递成功率 > 99%
- **降级**: 依赖 Increment 2 的 client capture 回退

### Increment 2: Client Capture 回退 (P1)

#### 数据一致性 — L3
- **等级**: L3 增强
- **量化目标**: 同一 order_id 仅入账一次（幂等），重复 capture 调用零副作用
- **保障**: `_apply_paypal_credited_inner()` 幂等检查 + taskBill `transaction_id` 唯一约束

#### 安全性 — L3
- **等级**: L3 增强
- **量化目标**: 仅订单所属用户可触发 capture
- **已有保障**: `pending.user_id == request.user.id` 校验 + `resolve_tenant_id()`

#### 容错机制 — L2
- **等级**: L2 标准
- **量化目标**: Webhook 未到达时，120s 内通过轮询完成 capture
- **重试策略**: 每 2s 轮询一次，每次 pending 时尝试 capture（幂等安全）

### Increment 3: 日志 + 事件补充 (P1)

#### 可观测性 — L2
- **等级**: L2 标准
- **量化目标**: 全链路每个阶段有 INFO 日志 + 领域事件，排查时无需推断
- **指标**: 新增 5 条 INFO 日志 + 3 个 Kafka 事件类型

### Increment 4: Outbox 重试 + 前端 (P2)

#### 容错机制 — L2
- **等级**: L2 标准
- **量化目标**: pending outbox 消息 1min 内重试，最多重试 10 次
- **策略**: 指数退避 (1min → 2min → 4min ...)，超过 1h 标记 failed

## 质量场景

### QS-01: Webhook 签名校验
| 要素 | 内容 |
|------|------|
| 类别 | 安全性 |
| 等级 | L3 |
| 刺激源 | PayPal 服务器 |
| 刺激 | POST webhook 事件（带正确签名头） |
| 制品 | `paypal_webhook()` |
| 环境 | 正常 |
| 响应 | 200 ok，触发 capture + credit |
| 响应度量 | 签名校验通过率 100%（假签名拒绝率 100%） |

### QS-02: 重复 Capture 幂等
| 要素 | 内容 |
|------|------|
| 类别 | 数据一致性 |
| 等级 | L3 |
| 刺激源 | Webhook + 客户端轮询并发 |
| 刺激 | 同一 order_id 被 capture 两次 |
| 制品 | `process_checkout_order_approved_event()` |
| 环境 | 并发 |
| 响应 | 仅入账一次，第二次返回 duplicate=true |
| 响应度量 | taskBill billing_transaction 中 transaction_id 唯一 |

### QS-03: Client Capture 回退
| 要素 | 内容 |
|------|------|
| 类别 | 可用性 |
| 等级 | L2 |
| 刺激源 | 用户从 PayPal 返回 |
| 刺激 | Webhook 未到达，前端轮询 recharge_paypal_status |
| 制品 | `RechargePaypalStatusView.get()` |
| 环境 | Webhook 故障 |
| 响应 | 轮询 10s 内触发 capture，入账完成 |
| 响应度量 | 从 paypal_return=1 到 status=completed < 30s |

### QS-04: 跨用户 Capture 隔离
| 要素 | 内容 |
|------|------|
| 类别 | 安全性 |
| 等级 | L3 |
| 刺激源 | 恶意用户 |
| 刺激 | 查询/触发他人 order_id 的 capture |
| 制品 | `RechargePaypalStatusView.get()` |
| 环境 | 正常 |
| 响应 | 403 Forbidden |
| 响应度量 | `pending.user_id != request.user.id` 校验 100% 拦截 |

## 领域模型影响

| NFR 决策 | 模型影响 | 对应 DDD 动作 |
|----------|---------|-------------|
| L3 幂等 capture/credit | BillingTransaction 的 transaction_id 是天然幂等键 | 聚合根不拆分，capture + credit 在同一事务边界 |
| L2 最终一致事件 | Outbox 消息可异步投递，允许短暂不一致 | BillingOutboxMessage 作为独立实体，与 BillingTransaction 同事务写入 |
| L2 Webhook 回退 | Capture 路径有两条（Webhook + 轮询），需统一入口 | `process_checkout_order_approved_event()` 作为唯一领域服务入口 |
| L2 可观测性 | 新领域事件需要 topic 注册和消费者 | 新增 3 个 Event 类型在 Kafka config 中注册 |

## 权衡与边界

### 取舍
- **GET 写副作用 vs REST 纯净性**: 选择在 GET status 端点中触发 capture，以最小化前端改动，接受 REST 语义不纯
- **实时事件 vs 最终一致**: 选择 outbox + goroutine 异步投递事件，接受短暂延迟

### 明确不做什么
- 不引入新的 capture 专用 POST 端点（前端改动成本 > 收益）
- 不做 PayPal API 调用的分布式事务（PayPal 是外部服务，用幂等兜底）
- 不改造 taskBill 数据库 schema（outbox 表已存在）

### 升级触发条件
- 当日充值订单 > 1000/天 → 性能从 L0 升级到 L2，需要 capture API 的 rate limit
- 当客户要求 PCI-DSS 合规 → 安全性从 L3 升级到 L4，需要全链路加密审计
- 当 outbox pending 积压 > 100 → 容错从 L2 升级到 L3，需要独立 outbox processor 服务

## 自检

- [x] 每个相关 NFR 类别都有明确的支撑等级
- [x] 每个 L1-L4 的 NFR 类别至少有一个量化目标
- [x] 每个 L2-L4 的 NFR 类别至少有一个质量场景（4 个 QS）
- [x] 每个质量场景的响应度量可验证
- [x] 影响领域模型的 NFR 决策已标注
- [x] 权衡和边界已明确
- [x] 跳过的 NFR 类别有理由说明
- [x] 文档位置正确
