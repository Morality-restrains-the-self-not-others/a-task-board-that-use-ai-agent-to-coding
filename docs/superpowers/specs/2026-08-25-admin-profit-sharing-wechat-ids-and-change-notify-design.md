# 超管微信分账：单号展示、手工分账、动账通知

日期：2026-08-25  
状态：accepted（/goal 跳过 Step 1 用户闸门）  
迭代：admin-profit-sharing-wechat-ids-change-notify

## 问题

系统管理员在 `/system-admin/users/` 推荐绩效抽屉「微信分账」Tab 只能看到本地台账与同步后的微信状态文案。失败行（如可见文本「失败 / 最早可分账 / 推荐…」）无法对照 **微信订单号**、**微信分账单号**，也不能在填写审计缘由后直接向微信发起分账。存量 `POST /api/billing/profitsharing/notify/` 未走 `core/notify` AEAD 解密，不能作为商户后台「分账动账通知」URL。

## 当前架构理解

- 应用层：taskFE SPA → taskGateway（APISIX）→ taskBill；微信 APIv3 唯一客户端为仓内 `sdk/wechatpay-go`（ADR-0032）。
- 分账写路径：`executeProfitSharing` → `profitsharing.OrdersApiService.CreateOrder`；推荐人 15–30 天窗口手动分账；25 天兜底扫描。
- 微信订单号已落在 `billing_resource_order.wechat_transaction_id` / `billing_payment_ledger.provider_capture_id`；微信分账单号在 `billing_profit_sharing.wechat_profit_sharing_id`。
- 管理端列表 **故意** 不回传分账单号（旧权限分析 ADR-0030 口径）。本需求 **仅对平台员工** 放开两列；推荐人本人 API 仍禁止回传。
- 支付回调已正确使用 `wechatNotifyH.ParseNotifyRequest`；分账通知 handler 是未完成桩。

## 🕸️ Code Review Graph 分析

- `code-review-graph update --brief`：增量 2 文件，风险 0。
- 影响面：`listProfitSharingQueue`、`handleSystemAdminListProfitSharing`、`executeProfitSharing`、`handleProfitSharingNotify`、`ReferralWechatProfitSharingTab.vue`、`SystemAdminProfitSharingPanel.vue`、APISIX `billing-profitsharing-notify`。

## 决策

1. **列表回传（仅平台员工）**  
   `GET /api/system-admin/profit-sharing/` 每行增加：
   - `wechat_transaction_id`（微信支付订单号，来自订单列，空则 `""`）
   - `wechat_profit_sharing_id`（微信分账单号，空则 `""`）  
   仍禁止 `openid` / `referrer_openid` / `receiver.account`。

2. **超管手工分账**  
   `POST /api/system-admin/profit-sharing/{id}/share/`  
   - 主体：`X-Gateway-Auth-Verified=1` + `authz.IsPlatformStaff`  
   - Body：`{ "reason": "<审计缘由>" }`，trim 后 8–500 字，缺省 400  
   - Header：`Idempotency-Key` 必填；同键重放 200 + `idempotent: true`，不重复出站  
   - 可执行：`pending` / `failed`（**可绕过推荐人 15 天冻结**，因资金仍在微信分账冻结池；审计缘由强制）  
   - `processing` / `finished`：200 空操作  
   - `voided` / `returned`：409  
   - 出站复用 `executeProfitSharing`；失败 502，本地标 `failed`  
   - 审计表 `billing_profit_sharing_admin_action`：actor、impersonator 字段、reason、idempotency_key UNIQUE、profit_sharing_id、order_id  
   - 日志：`impersonating` / `impersonator_user_id` / `impersonated_user_id` / `impersonation_session_id`（有则写；禁止打 token / 完整 openid）

3. **分账动账通知**（商户后台填写的 HTTPS URL）  
   `POST https://www.daydaymoney.com/api/billing/profitsharing/change-notify/`  
   - 无用户会话；网关 `auth_mode: machine`（与现网支付/分账 notify 一致）  
   - 验签+AEAD：`wechatNotifyH.ParseNotifyRequest` → 解密 `resource` 为 `profitsharing` 对象（官方文档 2026-01-28）  
   - Mock 模式：接受明文 JSON（与支付 notify 一致），便于单测  
   - 幂等：`notify.id` UNIQUE 插入；已处理直接 `{ "code": "SUCCESS" }`  
   - 行锁：按 `out_order_no` `SELECT … FOR UPDATE` 后更新 `wechat_profit_sharing_id`、必要时回写订单 `wechat_transaction_id`、`status=finished`（动账成功即视为分账到账）  
   - 应答 HTTP 200 + `{ "code": "SUCCESS" }`；验签失败 400 `{ "code": "FAIL" }`  
   - 存量 `POST /api/billing/profitsharing/notify/` **改为同一处理函数**（Expand：双 URL；不删旧路径）

4. **前端**  
   微信分账 Tab 与待分账队列增加两列 + 「分账」按钮（pending/failed）。点击后就地展开缘由表单（不 Teleport），`createClickGuard` + 同一次意图同一 `Idempotency-Key`。失败节点 `data-traceId`。

## 非目标

- 不向推荐人/买家泄漏微信单号。
- 不改 25 天兜底扫描；不新增 Kafka 领域事件（与现网分账写路径一致：更新台账 + 出站 CreateOrder）。
- 接收方类型仍为 `PERSONAL_OPENID`；动账通知官方仅保证向 `MERCHANT_ID` 接收方发送——接口仍按文档实现并登记商户后台，以便本商户作为接收方或微信按 V3 规则投递时可用。

## 接口契约

### GET `/api/system-admin/profit-sharing/`（扩展字段，向后兼容可选）

新增可选字符串：`wechat_transaction_id`、`wechat_profit_sharing_id`。

### POST `/api/system-admin/profit-sharing/{id}/share/`

请求：`{ "reason": "客服复核后补分账" }` + `Idempotency-Key`  
成功：`{ "status": "ok", "state": "shared"|"processing", "idempotent": false }`  
错误：`{ "error": "...", "trace_id": "..." }`（现网 `writeErrorJSON` 格式）。

### POST `/api/billing/profitsharing/change-notify/`

解密后资源（官方）：`mchid`、`transaction_id`、`order_id`、`out_order_no`、`receiver.{type,account,amount,description,success_time}`。  
成功：`{ "code": "SUCCESS" }`。

## 架构交付物

| 格式 | 路径 |
|------|------|
| PlantUML | `docs/architecture/v108-application-integration-20260825-1735-cursor.puml`、`v108-enterprise-landscape-20260825-1735-cursor.puml` |
| 增量 ArchiMate | 同名 `.diff.archimate` |
| 全量 ArchiMate | 同名 `.full.archimate` |
| Mermaid | 同名 `.mermaid.md` |

## 数据（冷热）

`billing_profit_sharing_admin_action`、`billing_profit_sharing_change_notify`：时间累积审计/通知收件箱，预估年增量 ≪ 10 万行，**单表 + 索引**，不做分区；保留合规期限不 TTL。主键 Snowflake（通知表以微信 `notify.id` 为 PK）。

## 例外

- 无新 MQ 事件：`No-event: same aggregate write path as executeProfitSharing / inbound WeChat notify`  
- Python API：不适用（Go taskBill）
