# 设计：租户充值页接入微信支付 Native 扫码

- **日期**：2026-07-14
- **状态**：已采纳（goal-mode 自动决策，跳过确认门）
- **页面**：`/tenant/{id}/billing/recharge/`
- **架构版本**：v22 🎯 target（基于 v21 current `application-integration`）
- **`python_api_approval`**：`scoped` — Django 仅 SMS 门禁编排 + 薄代理（与 PayPal 平行）；支付密码学、预下单、回调验签解密、入账均在 Go taskBill

## 1. 目标与成功标准

| # | 标准 | 验收 |
|---|------|------|
| S1 | 充值页可选「微信支付」，展示 Native 二维码 | 前端收到 `code_url` 并渲染 QR |
| S2 | 用户扫码支付后积分入账 | 回调验签 → 解密 → `credit-recharge` 幂等成功 |
| S3 | 缺商户私钥时可本地/mock 联调 | `mode=mock` 下可走 `mock-complete` 完成入账 |
| S4 | 与 PayPal 并行，不破坏 SMS 门禁 | 创建订单前仍检查 `sms_verified`（策略开启时） |
| S5 | 合规：密钥不进日志、回调必验签 | 代码审查 + 单测覆盖假签名拒绝 |
| S6 | Swagger / ownership 登记完整 | `openapi` + `db/api_route_ownership.yaml` 同步 |

**范围外（本迭代）**：JSAPI/H5/小程序支付、退款、分账、KYC 等级升级（沿用现有充值 SMS 门禁）。

## 2. 背景与现状

充值页当前仅支持 PayPal（Orders v2 + Webhook + client capture 回退）。架构上：

- **taskBill (Go)**：计费账户真源、`/api/internal/taskbill/credit-recharge/` 入账、充值相关租户 API 经 `handleRechargeDelegate` 转发 Django
- **billing_bridge (Django)**：SMS 验证编排、PayPal 订单创建/状态查询、Webhook 薄层；入账经 `credit_recharge()` 调 taskBill
- **幂等键**：PayPal 使用 `paypal:{order_id}`；积分来源 `user_recharge_paypal`

本需求在相同分层下新增微信支付通道，**支付密码学落 Go**，Django 保持与 PayPal 同级的薄编排。

## 3. 方案对比（自动采纳 A）

| 方案 | 描述 | 取舍 |
|------|------|------|
| **A（采纳）** | Native + 微信支付公钥验签 + Go taskBill 主路径 + mock 兜底 | 与 v3 公钥模式一致；符合 go-service-first；Django 薄代理可复用现有 delegate |
| B | Django 直连微信 API + 回调 | 违反 go-service-first；密码学分散 |
| C | JSAPI / H5 | 需 openid / 公众号配置，超出本页扫码场景 |
| D | 仅 mock，无 live 路径 | 无法上线 |

**自动采用**：Native 扫码 + `WithWechatPayPublicKeyAuthCipher` 公钥验签 + `mode=mock` 缺钥兜底。

## 4. 配置

目录：**`conf/billing/wechatPay/`**（monorepo 相对路径，不入库私钥明文）

| 文件 / 键 | 用途 | 备注 |
|-----------|------|------|
| `merchant_private_key.pem` | 商户 API 私钥（预下单签名） | live 必填；缺失则 `mode=mock` |
| `pub_key.pem` | 微信支付平台公钥（验签） | 见 [公钥验签模式](https://pay.weixin.qq.com/doc/v3/partner/4012925323) |
| `pub_key_id` | 平台公钥 ID | 与 `pub_key.pem` 配对 |
| `config.yaml`（或等价） | `mch_id`、`app_id`、`notify_url`、`mode` | `mode`: `mock` \| `live` |

加载优先级：环境变量 > `conf/billing/wechatPay/` > 默认 `mode=mock`。

**notify_url**（live）：`{API_ORIGIN}/api/billing/wechat/notify/`（经网关直达 taskBill 或 delegate 至 Go handler）。

## 5. SDK 与密码学

- **Go SDK**：`github.com/wechatpay-apiv3/wechatpay-go`
- **验签/解密**：`option.WithWechatPayPublicKeyAuthCipher(merchantId, merchantSerialNumber, privateKey, wechatPayPublicKeyId, wechatPayPublicKey)`
- **预下单**：Native 下单 API，返回 `code_url`
- **回调**：读取 `Wechatpay-Signature` / `Wechatpay-Timestamp` / `Wechatpay-Nonce` / `Wechatpay-Serial`，验签后解密 `resource` 得到支付结果

合规（`.ai/01_project_constraints/16_payment_kyc_compliance.md`）：

- 私钥、APIv3 密钥、完整回调体 **禁止** 写入 INFO 及以上日志
- 回调必须验签；金额、商户号、`out_trade_no` 与本地订单一致才入账
- 测试仅用 mock / 微信沙箱；E2E 使用 `mock-complete`，禁止真实证件/卡号 fixture

## 6. 服务落点

### 6.1 taskBill (Go) — 主责

| 职责 | 说明 |
|------|------|
| 预下单 | `POST .../recharge_wechat_create/`（经 delegate 或 Go 直挂）→ 调微信 Native API → 返回 `{out_trade_no, code_url, expires_at}` |
| 订单暂存 | SQLite/Redis：`wechat_recharge_pending:{out_trade_no}` → tenant_id, user_id, amount_fen, description |
| 回调 | `POST /api/billing/wechat/notify/` → 验签 → 解密 → 幂等入账 |
| 状态查询 | `GET .../recharge_wechat_status/?out_trade_no=` → pending / success / failed |
| mock-complete | `POST /api/internal/taskbill/wechat/mock-complete/`（internal secret + `mode=mock`） |
| 入账 | `POST /api/internal/taskbill/credit-recharge/` |

**入账契约**：

- `points_source_type`: **`user_recharge_wechat`**
- `transaction_id`: **`wechat:{out_trade_no}`**（最长 100 字符，与 PayPal 模式对齐）
- 重复回调 / 重复 credit → `duplicate: true`，无副作用

### 6.2 billing_bridge (Django) — 薄编排

与 PayPal 平行，**不做**微信 API 签名/验签：

| 端点 | 职责 |
|------|------|
| `recharge_wechat_create` | SMS 门禁 + 租户成员校验 → delegate Go 预下单 |
| `recharge_wechat_status` | 鉴权 + 订单归属校验 → delegate Go 查状态 |
| `recharge_phone_status` | 扩展响应：`wechat_enabled`, `wechat_mode` |
| internal delegate | 接收 taskBill `handleRechargeDelegate` 转发（与 `recharge_paypal_*` 同模式） |

Webhook **优先**挂 taskBill 路由；若网关暂经 Django，Django view 仅 raw body 转发 Go，**禁止**在 Python 内验签后改写字段。

## 7. API 摘要

### 7.1 租户 API（需登录）

**POST** `/api/tenant/{tenant_id}/billing/accounts/recharge_wechat_create/`

请求体：

```json
{
  "amount_yuan": "100.00",
  "description": "账户充值"
}
```

响应（200）：

```json
{
  "out_trade_no": "wx20260714143000123456",
  "code_url": "weixin://wxpay/bizpayurl?...",
  "expires_at": "2026-07-14T14:45:00+08:00",
  "mode": "live"
}
```

**GET** `/api/tenant/{tenant_id}/billing/accounts/recharge_wechat_status/?out_trade_no=...`

响应：`status`: `pending` | `success` | `failed` | `expired`；成功时含 `recharge_points`, `recharge_yuan`。

### 7.2 微信回调（公开）

**POST** `/api/billing/wechat/notify/`

- 无 session；必须验签
- 成功：200 + `{"code":"SUCCESS","message":"成功"}`
- 验签失败：401/403，不入账

### 7.3 内部 mock（仅测试）

**POST** `/api/internal/taskbill/wechat/mock-complete/`

- Header：`X-Internal-Secret`（与现有 internal API 一致）
- 前置：`mode=mock`
- Body：`{ "out_trade_no": "..." }` → 模拟支付成功并入账

## 8. 前端（BillingRecharge.vue）

- 支付方式切换：PayPal | 微信支付（`wechat_enabled` 时展示）
- 微信分支：创建订单 → 展示 QR（`code_url`）→ 轮询 `recharge_wechat_status`（2s 间隔，120s 超时）
- 文案：「请使用微信扫一扫完成支付」；超时提示手动刷新/查看交易记录
- mock 模式：开发环境可显示「模拟支付完成」按钮（仅 `wechat_mode=mock` 且非生产）

## 9. Domain Inventory（计费限界上下文）

| 概念 | 类型 | 说明 |
|------|------|------|
| **WechatRechargeOrder** | 聚合 / 实体 | `out_trade_no`、tenant_id、user_id、amount_fen、status、expires_at |
| **PaymentNotify** | 值对象 | 微信回调头 + 解密后 resource 摘要（脱敏持久化可选） |
| **RechargeCredit** | 领域服务 | 调 taskBill credit-recharge；幂等键 `wechat:{out_trade_no}` |
| **SmsVerificationGate** | 领域服务（已有） | 创建订单前 SMS 校验，与 PayPal 共用 |
| **PointsSourceType** | 枚举值 | 新增 `user_recharge_wechat` |

候选领域事件（可选 P2）：

- `WECHAT_RECHARGE_ORDER_CREATED`
- `WECHAT_RECHARGE_PAYMENT_SUCCEEDED`

入账后仍发既有 `BILLING_TRANSACTION_CREATED`（taskBill outbox）。

## 10. Value Stream Impact

- **影响流**：租户充值（`/tenant/{id}/billing/recharge/`）
- **新增步骤**：选择微信支付 → 创建微信订单 → 展示二维码 → 用户扫码 → 微信回调 → 轮询成功
- **与现有流关系**：
  - **recharge-sms-verification**：前置依赖（SMS 门禁不变）
  - **paypal-recharge-webhook-fix**：sibling — 并行支付通道
  - **increment4-billing-sse-go**：入账后触发 SSE/事件消费

建议在 `conf/value-stream.yaml` 登记新流 `wechat-pay-recharge`（实施阶段同步）。

## 11. 架构变更（v22 target）

| 制品 | 路径 |
|------|------|
| PlantUML | `docs/architecture/v22-application-integration-20260714-wechat-pay-claude.puml` |
| ArchiMate | 同名 `.archimate`（含 Plateau v21 → Gap → WP → Plateau v22） |
| Mermaid | 同名 `.mermaid.md` |
| VERSION_HISTORY | v22 条目：taskBill ↔ 微信支付外部系统 |

**拓扑要点**：

- 新增外部系统：**WeChat Pay**（Native + 支付通知）
- **taskBill** 🟡 MODIFIED：预下单、notify handler、credit
- **saas-backend billing_bridge** 🟡 MODIFIED：薄 delegate + SMS 门禁（无密码学）
- **Vue BillingRecharge** 🟡 MODIFIED：双通道 UI

## 12. python_api_approval（scoped）

**结论**：允许 **scoped** 新增 Django 路由，理由与 PayPal 模式一致 — Python 仅负责已存在的 SMS/租户编排与 delegate，**不**新增支付密码学。

| 拟新增 Django 路由 | 方法 | 理由 |
|-------------------|------|------|
| `.../recharge_wechat_create/` | POST | SMS 门禁 + 调 internal/Go 预下单；与 `recharge_paypal_create` 对称 |
| `.../recharge_wechat_status/` | GET | 鉴权 + 订单归属 + 查状态；与 `recharge_paypal_status` 对称 |
| `api/internal/taskbill/delegate/recharge_wechat_create/` | POST | taskBill delegate 回环（若沿用现有 delegate 模式） |
| `api/internal/taskbill/delegate/recharge_wechat_status/` | POST | 同上 |
| `api/billing/wechat/notify/`（可选薄转发） | POST | 仅当网关未直挂 Go 时 raw 转发；**不得**在 Python 验签 |

**禁止**：在 Django 新增微信 SDK 调用、私钥加载、验签解密逻辑。

## 13. 测试策略

| 层级 | 内容 |
|------|------|
| Go 单测 | 验签失败拒绝、幂等入账、mock-complete、金额不一致拒绝 |
| Django 单测 | SMS 未验证拒绝创建、跨用户查状态 403 |
| Playwright | mock 模式：选微信 → 创建 → mock-complete → 轮询成功 |
| 沙箱 | live 配置下用微信沙箱 `out_trade_no` 联调（CI 可选 nightly） |

## 14. Swagger / Ownership

- taskBill `openapi.yaml` 登记 notify、internal mock-complete、wechat create/status（若 Go 直暴露）
- `db/api_route_ownership.yaml` 登记 Django 薄路由 owner=`billing_bridge`
- `docs/architecture/api-route-to-owner.md` 补充一行

## 15. 关键决策摘要

| 决策 | 理由 |
|------|------|
| Native 扫码 | 充值页 PC/Web 场景，无需 openid |
| 公钥验签模式 | 微信 v3 推荐；配置 `pub_key.pem` + `pub_key_id` |
| Go 主路径 | go-service-first + 单点幂等入账 |
| mock 兜底 | 无商户私钥时仍可 E2E/开发 |
| txn `wechat:{out_trade_no}` | 与 `paypal:{order_id}` 对齐，全局唯一 |

## 16. 相关文档

- 权限：`docs/superpowers/specs/2026-07-14-wechat-pay-recharge-permission-analysis.md`
- 价值流：`docs/superpowers/plans/2026-07-14-wechat-pay-recharge-value-stream.md`
- NFR：`docs/superpowers/plans/2026-07-14-wechat-pay-recharge-nfr-clarification.md`
- 计划：`docs/superpowers/plans/2026-07-14-wechat-pay-recharge-plan.md`
- 意图：`docs/intents/wechat-pay-recharge.intent.md`
