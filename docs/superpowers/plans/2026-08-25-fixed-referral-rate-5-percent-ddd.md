# DDD — 推荐分成比例固定 5%

- **日期**: 2026-08-25
- **NFR**: `docs/superpowers/plans/2026-08-25-fixed-referral-rate-5-percent-nfr-clarification.md`

## 限界上下文

- **Billing (taskBill)**：佣金计提、分账金额、`billing_referral_config` 所有权（仅入账天数可写）。
- **Referral (taskReferral)**：资格/统计展示；比例展示不得读配置或微信 `max_ratio` 冒充政策。
- **Admin SPA (taskFE)**：超管只读展示常量。

## 模型

- **值对象** `ReferralCommissionRate`：唯一合法百分比分子 `5`。禁止从仓储加载。
- **既有聚合** `ReferralConfig`：可变字段仅 `settle_delay_days`（≥8）。比例列若仍落库则恒为 5，不作为政策源。
- **领域服务** `CommissionPoints(consumption) = ROUND_HALF_UP(consumption * 5 / 100)`。
- **领域服务** `PayoutPercent(wechatMax) = min(5, wechatMax)`；`wechatMax` 不可用则 0。

## 端口

不新增端口。停止把 `ReferralConfigRepository` 的比例字段用于 `getReferralRatePercent`。

## 业务意图 → 事件

| 意图 | 事件 | 例外 |
|------|------|------|
| 超管查看固定分成比例 | — | 纯只读 |
| 超管保存入账天数 | 既有 config 更新日志 | 不再发 `REFERRAL_RATIO_UPDATED` |
| 消费计提 | 既有 accrual 唯一约束 | 无新事件 |
