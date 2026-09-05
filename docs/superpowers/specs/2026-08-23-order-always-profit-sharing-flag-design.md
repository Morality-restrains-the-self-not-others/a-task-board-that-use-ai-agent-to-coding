# 设计：租户下单对所有微信订单打分账标识

- **日期**: 2026-08-23
- **入口**: 租户 `POST /api/tenant/{tid}/billing/orders/{orderId}/pay/`（`payment_method=wechat`）
- **架构变更**: 无（不新增服务/聚合/消息；不更新 `docs/architecture/`）
- **python_api_approval**: 未触发（接口落 Go `taskBill`，无新 HTTP 路径）
- **ADR**: [ADR-0040](../../adr/0040-always-flag-wechat-profit-sharing.md)（取代 ADR-0033 的预下单打标条件）

## 背景

ADR-0033 仅在付款人有推荐边且推荐人支付时刻资格 active 时，才给 Native 预下单设置 `settle_info.profit_sharing=true`。微信规定该字段只能在下单时设置，漏标则该笔已付款交易永远不能分账。产品要求：**无论是否有推荐人，租户下单都自动打上分账标识。**

## 官方依据

- Native 下单 `settle_info.profit_sharing`：<https://pay.weixin.qq.com/doc/v3/merchant/4012791877.md>
  - `true`：支付成功后资金冻结；可请求分账，或 [解冻剩余资金](https://pay.weixin.qq.com/doc/v3/merchant/4012526374.md)，或 30 天后自动解冻。
  - `false`/不传：资金直接可用，不可再分账。
- 解冻剩余资金专用于「不需要进行分账的订单」把金额全部解冻给本商户；同一 `out_order_no` 多次请求等同一次。

## 方案（goal-mode 自动采用）

### 1. 预下单：一律打标

`wechatPrepay` live 路径在调用 `native.NativeApiService.Prepay` 前**无条件**设置：

```go
prepayReq.SettleInfo = &native.SettleInfo{ProfitSharing: core.Bool(true)}
```

不再调用 `shouldFlagWechatProfitSharing`。mock 模式不打微信，行为不变。PayPal 无对等字段。

### 2. 台账：仍要有资格推荐人

`markOrderForProfitSharing` 保持 ADR-0033 支付时刻资格：无边 / 资格 false / 查询失败 → 不插入 `billing_profit_sharing`（`referrer_user_id` NOT NULL；无接收方不能 `CreateOrder`）。

本地「已打标可分账给推荐人」仍等于存在佣金行。微信侧「已打分账标识」对**本增量之后的全部新 Native 单**为真，订单表不新增列。

### 3. 无佣金对象：解冻剩余资金

微信支付成功（非 mock）且本单无 `billing_profit_sharing` 行时，调用已有 `unfreezeProfitSharing`：

- `transaction_id` = `lookupWechatTransactionIDForOrder`
- `out_order_no` = `UF{order_id}`（稳定、可重放）
- `description` = 固定中文原因，如「无分账接收方，解冻剩余资金」
- 失败只打 warn，不回滚支付
- 有佣金行：不在支付回调解冻，仍走 8 天 delay + `CreateOrder(UnfreezeUnsplit=true)`

### 4. 租户不可见

分账标识对租户 API/UI 仍不可见（ADR-0030）。本增量不改前端。

## 非目标

- 不为历史已支付未打标微信单补标
- 不把无推荐人订单插入佣金台账
- 不改 PayPal
- 不改点数计提资格口径
- 不新增 HTTP API

## 权限

无新端点。预下单仍走租户已鉴权 pay；解冻为支付成功内部副作用，不对外暴露。

## 路径分片键（NFR 预览）

| 路径 | 分片 ID | 等级 | 说明 |
|------|---------|------|------|
| `POST .../orders/{id}/pay/` | tenant_id + order_id | 适配 | 已有租户路径 |
| Native Prepay / UnfreezeOrder | 微信商户单号 | L0 出站 | 无本库分片 |

## 幂等性

| 路径 | 副作用 | 级别 |
|------|--------|------|
| Native Prepay `profit_sharing=true` | 微信创单 | 已有 out_trade_no |
| 插入 `billing_profit_sharing` | 按 order_id 已存在则跳过 | L3 |
| Unfreeze `UF{order_id}` | 微信解冻 | L3，同号重放等价 |

## 业务意图 → 事件对照

| 业务意图 | 事件名 | 发布点 | 消费者 | 例外理由 |
|---------|--------|--------|--------|----------|
| 预下单一律打分账标识 | — | `wechatPrepay` | 微信冻结可分账资金 | 出站字段；支付成功仍走既有 `PAYMENT_SUCCEEDED` |
| 无接收方解冻剩余 | — | `markOrderPaid` 异步 | 微信解冻 | 支付成功路径覆盖；不新发 MQ |
| 有资格落佣金行 | — | `markOrderForProfitSharing` | `billing_profit_sharing` | 同上 |

## 领域概念（供 /6-ddd）

- **Bounded Context**: Billing（taskBill）
- **Policy**: `WechatOrderProfitSharingFlag` = 恒 true（预下单）；`ReferralCommissionEligibility` = 支付时刻资格（台账）
- **Domain Events**: 无新增

## 🕸️ Code Review Graph 分析

- CRG 根图以 JS/Python 为主，Go taskBill 符号未入图。
- `CRG unavailable for Go blast radius`；人工爆炸半径：`wechat_pay_prepay.go`、`referral_paytime_qualification.go`、`wechat_profit_sharing.go`、`order_payment.go`、`wechat_pay_profit_sharing_flag_test.go`。

## 🏛️ 架构变更影响

- **迭代版本**: 沿用 v104 current，**不新建 target 文件**。
- **理由**: 仅改变已有 Native Prepay 请求字段与支付成功后已有 Unfreeze 调用条件；无新组件、无新数据对象、无新服务间关系。

## 实现落点

| 层 | 改动 |
|----|------|
| taskBill | 预下单无条件 SettleInfo；支付成功无佣金行则 Unfreeze |
| 意图/价值流 | 更新 VS-PS-4/5；新增无推荐人仍打标 + 解冻测点 |
| ADR | 0040；0033 superseded（打标条款） |
| taskFE | 无 |
