# ADR-0025: 多渠道推荐码与绑边资格快照

- **Status:** accepted
- **Date:** 2026-08-20
- **Author:** Trae AI
- **Deciders:** 工程团队
- **Amended by:** [ADR-0033](0033-paytime-referral-profit-sharing.md)（微信分账标识与订单分账标记改以**支付时刻**资格为准；绑边 `commission_eligible` 仍作渠道统计快照，不再作为 Native `profit_sharing` / `markOrderForProfitSharing` 门禁）；**点数佣金计提同口径**：`accrueReferralFromConsumption` 改为支付/消费时刻现查推荐人活跃资格（OPT-20260822-059 落地）

---

## Context

推荐分享原先每用户一条 `access_code`（`UNIQUE user_id`），统计无渠道维度。跨服务约定不足：taskReferral 解析码，taskBill 持有边与计提；若不在边上快照「用了哪条码、当时有无分成资格」，后续改码/改资格会改写历史，且无资格消费可能误分账。

## Decision

We will:

1. Allow **multiple opaque share codes per user**, distinguished by `channel_name`, stored in `referral_share_code` (drop per-user uniqueness).
2. Snapshot `channel_code` and `commission_eligible` on `billing_referral_edge` at bind time; **never overwrite** on duplicate bind.
3. Accrue **点数佣金** when **支付/消费时刻**现查推荐人活跃资格（`lookupReferrerPaytimeQualification`，与微信分账同口径，见 [ADR-0033](0033-paytime-referral-profit-sharing.md)）；绑边 `commission_eligible` 仅作渠道归因快照（见 Alternative 2），不再作为计提门禁。**微信分账标识与订单分账落库**也由 [ADR-0033](0033-paytime-referral-profit-sharing.md) 在支付时刻现查 `referral_code` 活跃资格。
4. Keep table ownership: codes in taskReferral; edges/accruals in taskBill. Bind remains HTTP sync (taskAuth → taskReferral → taskBill).

## Alternatives Considered

### Alternative 1: Separate `referral_channel` table + FK on edge

- **Pros:** 更清晰的实体
- **Cons:** 跨库 FK 禁止；绑边时还要再查码表
- **Why rejected:** 边上来源码快照已足够统计；少一张跨服务表

### Alternative 2: 资格变化时重算历史计提

- **Pros:** 与「当前资格」一致
- **Cons:** 资金路径不可预测
- **Why rejected:** 分成以注册当时资格为准

### Alternative 3: 前端用 query 区分渠道、后端仍单码

- **Pros:** 无 schema
- **Cons:** 可伪造；无法服务端归因
- **Why rejected:** 统计必须服务端可信

## Consequences

### Positive

- 投放渠道可独立链接与报表
- 无资格推荐仍计人数、不产生佣金

### Negative / Trade-offs

- 旧边 `channel_code=''` 需在 UI 显示为「历史（未分渠道）」
- 存量 `UNIQUE(user_id)` 必须迁移

### Mitigations

- 007/052 幂等 DDL；fail-closed：省略 `commission_eligible` 视为 false
- 禁用渠道保留边，只停止新解析

## References

- `docs/superpowers/specs/2026-08-20-referral-channel-codes-design.md`
- 元规则 19 单表所有权；元规则 48/53 资金幂等
