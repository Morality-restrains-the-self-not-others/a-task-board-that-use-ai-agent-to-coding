# NFR 澄清: 微信支付 Native 充值

> 输入:
> - 设计文档: `docs/superpowers/specs/2026-07-14-wechat-pay-recharge-design.md`
> - 价值流文档: `docs/superpowers/plans/2026-07-14-wechat-pay-recharge-value-stream.md`
>
> 输出使用者: DDD 建模, 实施计划, 构建

**域分级**：金融/支付域 → **L3 增强**（与 PayPal 充值同级）

## NFR 概览表

| 类别 | 等级 | 一句话量化 |
|------|------|-----------|
| 安全性 | L3 | 回调公钥验签 100%；密钥零日志；mock-complete 双门禁 |
| 数据一致性 | L3 | `wechat:{out_trade_no}` 全局幂等；金额/商户号一致才入账 |
| 可用性 | L2 | 回调失败可依赖前端轮询 + 微信重试；RTO 入账可见 < 2min |
| 容错机制 | L2 | 微信重复通知安全；pending 订单 TTL 过期标记 |
| 可观测性 | L2 | 全链路 audit：create / notify / credit 结构化日志 + trace |
| 合规与隐私 | L3 | 支付合规约束；测试仅 mock/沙箱 |
| 性能 | L1 | 预下单 P95 < 3s；轮询不放大写库 |
| 可伸缩性 | L0 | 跳过 — 当前量级 |
| 可维护性 | L1 | mode mock/live 显式配置，与 PayPal 配置模式对齐 |

## 逐增量 NFR 分析

### Increment 1: mock 通路 (P0)

#### 数据一致性 — L3
- **量化目标**：同一 `out_trade_no` mock-complete 多次仅入账一次
- **保障**：taskBill `transaction_id` UNIQUE + `duplicate: true`

#### 安全性 — L3
- **量化目标**：无 internal secret → 401；live 模式 mock-complete → 403
- **保障**：常量时间 secret 比较 + 配置 `mode` 硬校验

### Increment 2: 创建订单 + QR (P1)

#### 安全性 — L3
- **量化目标**：未 SMS 验证（策略开）创建成功率 0%
- **保障**：Django 与 PayPal create 相同门禁

#### 性能 — L1
- **量化目标**：create API P95 < 3s（含微信 RTT）
- **降级**：超时时前端可重试 create（新 out_trade_no）

### Increment 3: 回调验签 (P1)

#### 安全性 — L3
- **量化目标**：假签名拒绝率 100%；真签名通过率 100%（沙箱/live）
- **保障**：`WithWechatPayPublicKeyAuthCipher` + serial 匹配

#### 数据一致性 — L3
- **量化目标**：回调金额（分）与 pending 订单一致，否则拒绝入账
- **保障**：fen 整数比较；币种 CNY

#### 合规 — L3
- **量化目标**：日志中 0 条完整私钥/解密 PII
- **保障**：仅记录 out_trade_no、tenant_id、result code

### Increment 4: 轮询 (P1)

#### 可用性 — L2
- **量化目标**：支付成功后 120s 内前端轮询到 success ≥ 99%（含 notify 延迟）
- **策略**：2s 间隔；120s 超时提示

#### 可观测性 — L2
- **量化目标**：每次 notify / credit 有 INFO + trace_id
- **指标**：`wechat_recharge_create_total`, `wechat_notify_verified_total`, `wechat_credit_duplicate_total`

## 质量场景

### QS-01: 回调验签失败

| 要素 | 内容 |
|------|------|
| 类别 | 安全性 |
| 等级 | L3 |
| 刺激源 | 攻击者 / 错误配置 |
| 刺激 | POST notify 带伪造签名 |
| 制品 | `handleWechatNotify` |
| 响应 | 401/403，无入账 |
| 响应度量 | billing_transaction 无新增行 |

### QS-02: 重复通知幂等

| 要素 | 内容 |
|------|------|
| 类别 | 数据一致性 |
| 等级 | L3 |
| 刺激源 | 微信服务器 |
| 刺激 | 同一 out_trade_no 通知 3 次 |
| 制品 | credit-recharge |
| 响应 | 仅一条 recharge 流水；后续 duplicate |
| 响应度量 | transaction_id 唯一 |

### QS-03: mock-complete 误暴露

| 要素 | 内容 |
|------|------|
| 类别 | 安全性 |
| 等级 | L3 |
| 刺激源 | 外部 HTTP 客户端 |
| 刺激 | live 环境调用 mock-complete |
| 制品 | internal mock handler |
| 响应 | 403 |
| 响应度量 | 生产无 mock 路由或硬拒绝 |

### QS-04: 支付成功入账延迟

| 要素 | 内容 |
|------|------|
| 类别 | 可用性 |
| 等级 | L2 |
| 刺激源 | 用户 |
| 刺激 | 扫码完成，notify 延迟 30s |
| 制品 | 前端轮询 + notify |
| 响应 | 30s 内 status 变 success |
| 响应度量 | P95 入账可见 < 60s |

## 审计日志最低字段

| 事件 | 必记字段（脱敏） |
|------|------------------|
| order_created | trace_id, tenant_id, user_id, out_trade_no, amount_fen, mode |
| notify_received | trace_id, out_trade_no, verify_ok, trade_state |
| credit_applied | trace_id, out_trade_no, points_delta, duplicate |

**禁止**：merchant_private_key、完整 notify body、用户手机号明文。

## 结论

金融域 L3 要求驱动：**验签、幂等、审计、mock 隔离** 为 P1 阻断项；可用性 L2 由轮询 + 微信重试满足。实施计划须先覆盖 QS-01/02/03 单测再合并。
