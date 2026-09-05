# 设计：推荐绩效抽屉增加「微信分账」Tab

- **日期**: 2026-08-23
- **入口**: `/system-admin/users/` → 推荐绩效抽屉 →「推荐用户支付明细」旁 Tab
- **架构变更**: 无（不新增服务/聚合/消息；不更新 `docs/architecture/`）
- **python_api_approval**: 未触发（接口落 Go `taskBill`）
- **总体设计审批**: approved（2026-08-23；查询路径：推荐人→被推荐人→打标订单→QueryOrder）

## 背景

超管在用户列表打开「推荐绩效」抽屉时，已能看到推荐人数、自身/被推荐人支付、佣金明细。运营要对账某推荐人**实际收到的微信分账**时，只能去订单页「待分账」全局队列或展开单笔订单，无法按该用户过滤。

用户期望：在支付明细旁加 Tab，**调用微信支付 SDK，用该用户微信相关 ID 查询其分账列表**。

## 🔍 可行性结论（官方 API + 仓内现状）

### 微信官方没有「按 openid/unionid 列出分账单」的接口

知识库已同步（2026-08-23）。普通商户 APIv3 分账产品接口全集：

| 接口 | 路径 | 查询键 | 能否按用户列清单 |
|------|------|--------|------------------|
| 查询分账结果 | `GET /v3/profitsharing/orders/{out_order_no}?transaction_id=` | **商户分账单号 + 微信支付单号** | 否，只能核单笔 |
| 查询剩余待分金额 | `GET /v3/profitsharing/transactions/{transaction_id}/amounts` | 微信支付单号 | 否 |
| 申请/下载分账账单 | `GET /v3/profitsharing/bills?bill_date=` | **自然日**，仅三个月内 | 否；次日 10 点后才有，且不含失败单 |
| 添加/删除分账接收方 | `POST .../receivers/add` / `delete` | 个人 **openid**（登记接收方） | 否，不是分账流水 |
| 请求分账 / 回退 / 解冻 | 写接口 | 订单级 | 本需求禁止调用 |

官方文档：

- 查询分账结果：<https://pay.weixin.qq.com/doc/v3/merchant/4012525210>
- 申请分账账单：<https://pay.weixin.qq.com/doc/v3/merchant/4012529628>
- 添加接收方（openid 用途）：<https://pay.weixin.qq.com/doc/v3/merchant/4012528995>

**因此「拿微信 ID 直接向微信要该用户全部分账列表」不可行。** OpenID 只用于登记 `PERSONAL_OPENID` 接收方；列清单必须以本仓库台账为索引，再按需用 SDK `OrdersApiService.QueryOrder` 核单笔。

### 仓内已具备的能力

| 能力 | 位置 | 本需求用法 |
|------|------|------------|
| 分账台账 + `referrer_user_id` 索引 | `billing_profit_sharing`、`idx_profit_sharing_referrer` | **列表 SSOT** |
| 接收方登记（openid 不对外） | `billing_profit_sharing_receiver`；内部 `GET /api/internal/users/id/{id}/wechat-pay-openid/` | 展示「已登记接收方」布尔/状态，**响应不含 openid** |
| SDK 查询分账结果 | `queryProfitSharingOrder` → `profitsharing.OrdersApiService.QueryOrder` | 「同步微信状态」按钮，按当前页逐笔 |
| 管理端分账队列 | `GET /api/system-admin/profit-sharing/`（ADR-0030，平台员工） | **扩展** `referrer_user_id`（推荐人→被推荐人→打标订单），抽屉复用 |
| 推荐绩效抽屉 | `SystemAdminReferralPerformanceDrawer.vue` + `GET .../referral-performance/`（taskReferral） | 增加 Tab；分账数据**不**经 taskReferral（表 owner 是 taskBill） |

网关已有 `/api/system-admin/profit-sharing/*` → taskBill，新子路径无需改 APISIX 前缀。

## 方案（待批准）

已与用户确认（含修订）：

1. **查询路径（强制）**：推荐人 → 其全部被推荐人 → 被推荐人订单中**已打分账标**的订单 → 用该订单列表查分账结果（SDK `QueryOrder`）。
2. **现查策略**：打开 Tab 先走上述路径的**本地投影**；「同步微信状态」对当前页订单逐笔 `QueryOrder`。

### 「分账订单」在本仓库如何识别

微信 Native 预下单把 `SettleInfo.ProfitSharing=true` 打到微信（ADR-0033），**订单表没有独立分账标列**，微信也**没有**「列出某用户全部已打标订单」的 API。支付成功后 `markOrderForProfitSharing` 写入 `billing_profit_sharing`，这就是本地可枚举的「已打分账标订单」。

因此列表查询语义为：

```
billing_referral_edge (referrer_user_id = 抽屉用户)
  → referred_user_id
  → billing_resource_order (user_id = 被推荐人，已支付)
  → INNER JOIN billing_profit_sharing (order_id)
```

再对命中行用 `out_profit_sharing_no` + 该单 `transaction_id` 调 `QueryOrder`。  
（按 `billing_profit_sharing.referrer_user_id` 直滤在数据一致时应得到同一集合；实现须以「边 → 订单 → 台账」为验收口径，并带出被推荐人 ID，避免漏展示付款人。）

### UI

抽屉「推荐用户支付明细」改为两组 Tab：

- **支付明细**（现有表，默认）
- **微信分账**（新）

微信分账表列：被推荐人（付款人）ID、订单号（链到 `/system-admin/order-records/`）、分账金额、本地状态、最早可分账时间、失败原因、微信状态（未同步为「—」）。页头展示接收方登记摘要：`未绑定` / `已登记` / `登记失败`（来自 `billing_profit_sharing_receiver.status`，无 openid）。

「同步微信状态」：`createClickGuard` + `disabled` + `aria-busy`；只读刷新类按钮注释 Anti-Replay。失败展示 `data-traceId`。**不发起** `CreateOrder` / 回退 / 解冻。

### API（Go taskBill）

1. **扩展** `GET /api/system-admin/profit-sharing/`
   - 新增查询参数 `referrer_user_id`（抽屉用户 = 推荐人）。
   - 有该参数时：按上节 JOIN 返回该推荐人下被推荐人的分账订单；`status` 默认 `all`（抽屉传入 `status=all`）。
   - 每项增加 `referred_user_id`（订单买家）；其余字段与现网队列一致。
   - 既有「待分账」全局队列不传该参数，行为不变。
   - 仍不含 `openid` / `wechat_profit_sharing_id` / `referrer_openid`。
   - 可选同响应 `receiver_registration_status`（仅带 `referrer_user_id` 时返回一次）。

2. **新增** `POST /api/system-admin/profit-sharing/refresh-wechat/`
   - Body：`{ "referrer_user_id": "...", "ids": ["台账id", ...] }`。
   - 鉴权：网关已验证 + `IsPlatformStaff`。
   - 所有 `ids` 必须落在该推荐人 JOIN 结果内，否则 400。
   - `ids` 上限 = 列表页 `limit`（≤ 50）。
   - 对每笔：`lookupWechatTransactionIDForOrder` + 已有 `queryProfitSharingOrder(out_profit_sharing_no, transaction_id)`。
   - **不回写** `billing_profit_sharing.status`（与 timer `billing_profit_sharing_scan` 分工：本按钮只展示微信侧 `state` / 错误）。
   - 缺 `transaction_id`、微信 404/`FREQUENCY_LIMITED`：该行 `wechat_state` 为空、`wechat_error` 人类可读；其它行继续。
   - 响应不含 openid。

### 安全

- 仅平台员工；租户 403（延续 ADR-0030）。
- 日志不写查询原文 openid、不写完整微信应答里的 `account`。
- 前端不展示微信账号标识。

## 非目标

- 不按被推荐人（付款人）钻取分账（本轮明确不做）。
- 不下载微信日账单再过滤。
- 不添加/删除接收方、不立即发起分账/重试。
- 不把 openid 返回浏览器。
- 不同步回写本地 status（若运营需要与微信终态对齐，走既有 timer / 后续增量）。

## 权限

| 路径 | 角色 | 范围 |
|------|------|------|
| `GET /api/system-admin/profit-sharing/?referrer_user_id=` | 网关已验证 + `IsPlatformStaff` | 跨租户；推荐人→被推荐人→打标订单 |
| `POST /api/system-admin/profit-sharing/refresh-wechat/` | 同上 | 仅核对该推荐人 JOIN 结果内的 id |

## 路径分片键（NFR 预览）

| 路径 | 分片 ID | 等级 | 说明 |
|------|---------|------|------|
| GET 列表 + referrer_user_id | 无租户 ID | L0 | 超管按推荐人图查询；边表 + 订单 + 台账均有推荐人/买家键；升级触发：单推荐人下打标订单 >1 万或 QPS>20 |
| POST refresh-wechat | 无 | L0 | 出站只读 QueryOrder，本机无写；升级触发：单页 >50 或微信 429 常态 |

## 幂等性

| 路径 | 副作用 | 级别 |
|------|--------|------|
| GET 列表 | 无 | L0，重复 GET 同结果 |
| POST refresh-wechat | 无本地写；微信 QueryOrder 只读 | L0；前端同步门闩防连点；不需要 Idempotency-Key |

## 业务意图 → 事件对照

| 业务意图 | 事件名 | 发布点 | 消费者 | 例外理由 |
|---------|--------|--------|--------|----------|
| 超管按推荐人查看被推荐人打标订单的分账结果 | 无对应事件 | — | — | 只读图查询 |
| 超管同步当前页微信分账状态 | 无对应事件 | — | — | 出站只读 QueryOrder，不改变本库事实 |

## 领域概念（供 /6-ddd）

- **Bounded Context**: Billing（taskBill）— 分账台账；Identity（taskAuth）— 仅内部 openid，本增量不新增公开查询。
- **Key Entities**: `billing_profit_sharing`（已有）；不新建聚合。
- **Domain Events**: 无。

## 🕸️ Code Review Graph 分析

- CRG `status` 正常（108 nodes），但索引以 JS/TS/Python/bash 为主，**未覆盖 Go taskBill**。
- `code-review-graph search profitsharing` → 0 nodes。
- `CRG unavailable for Go blast radius: graph has no taskBill symbols`；爆炸半径改为人工：`handlers_admin_list_profit_sharing.go`、`profit_sharing_admin.go`、`wechat_profit_sharing.go`、`SystemAdminReferralPerformanceDrawer.vue`、`openapi.yaml`。

## 价值流影响

- 触及 `conf/value-stream.yaml` 域「推荐」既有流（`referral-channel-codes` 等）的**只读运营核对**，不改绑边/计提。
- 不新增独立价值流名；`/4-value-stream` 可将本增量挂在系统管理用户/推荐绩效下，测试点覆盖抽屉 Tab + 推荐人图查询 + 同步按钮。
- 字段：`task-bill.billing_referral_edge.referrer_user_id` / `referred_user_id`（读）、`task-bill.billing_resource_order.user_id`（读）、`task-bill.billing_profit_sharing.order_id` / `status`（读）。不新增列。

## 🏛️ 架构变更影响

- **迭代版本**: 沿用 v104 current，**不新建 target 文件**。
- **理由**: 扩展已有管理端 Application_Interface 查询参数 + 同前缀只读刷新；taskBill → 微信 QueryOrder 关系已存在（分账执行路径）。无新组件、无新数据对象、无新事件。

## Swagger

- 更新 `taskBill/src/openapi.yaml`：`referrer_user_id` query；新 POST `refresh-wechat` 的 request/response/4xx。

## 实现落点

| 层 | 改动 |
|----|------|
| taskBill | 列表按推荐人 JOIN 边→订单→台账；POST 按订单 QueryOrder |
| taskFE | 抽屉 Tab + 调已有 profit-sharing GET；同步按钮 |
| OpenAPI | 如上 |
| 网关 | 前缀已覆盖，预期零改 |

抽屉文件已接近行数上限，Tab 逻辑抽到同目录小组件，避免超过 500 行。
