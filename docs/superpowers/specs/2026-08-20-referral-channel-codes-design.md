# 设计：多渠道推荐码、推荐链接与分渠道统计/分账

- **Date:** 2026-08-20
- **Status:** accepted（goal-mode 自动采用）
- **Architecture:** v92
- **ADR:** ADR-0025

## 问题

推荐人只有一条不透明 `access_code`（`referral_share_code` UNIQUE `user_id`），无法为不同投放渠道生成独立链接。统计接口 `GET /api/referral/stats/user_id/{userId}/` 只有全量 `referral_count` 与按月收益，无法按渠道、按时间段查看推荐人数与分账。

绑边时必须快照推荐资格：无资格仍计人数，消费**不计提、不分账**。

## 决策

1. **一用户多渠道码**：同一 `referral_share_code` 表去掉 `UNIQUE(user_id)`，增加 `channel_name` / `is_default` / `status`。每用户默认渠道「默认」；最多 20 条。码仍全局 UNIQUE，禁止 `u{userId}`。
2. **链接**：`{origin}/auth/register/?accessCode={code}`，与现网一致。
3. **绑边快照**（taskBill 拥有 `billing_referral_edge`）：
   - `channel_code`：注册所用码
   - `commission_eligible`：绑边当时 `getActiveReferralCode` 是否存在
   - ON DUPLICATE **不覆盖**上述两字段
   - 禁用渠道不再解析为新绑边；历史边保留
4. **计提/微信分账**仅当 `commission_eligible=1`；订单买家用 `billing_resource_order.user_id` 关联边（禁止 `referred_user_id=''`）。
5. **统计**：`GET /api/referral/stats/?channel_code=&from=&to=` 返回总人数、按渠道人数与分账（佣金点数）、按月收益。人数过滤 `bound_at`；分账过滤 `consumed_at`。
6. **渠道 API**（taskReferral Go）：
   - `GET /api/referral/channels/`
   - `POST /api/referral/channels/` `{name}`，幂等键 `(user_id, channel_name)` + `Idempotency-Key`
   - `POST /api/referral/channels/code/{code}/disable/` 禁用非默认渠道
7. **前端**：`/profile/referral/` 渠道列表+创建、复制链接、渠道+日期筛选、分渠道人数与分账表。创建按钮 `createClickGuard`。行数门禁：从 `UserReferral.vue` 抽出组件。

## 非目标

- 不解析 `u{userId}`
- 不把无证据的历史用户回填到某推荐人
- 不新建服务；扩展 taskReferral / taskBill / taskFE

## 验收

- 可为「微信」「抖音」等渠道各建码并生成不同链接
- 按渠道+时间段看到推荐人数与佣金/分账
- 无资格绑边：人数 +1，消费 0 计提
- 默认码仍由 status.`access_code` 返回
