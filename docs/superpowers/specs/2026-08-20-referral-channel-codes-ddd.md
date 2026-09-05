# DDD — 多渠道推荐码

- **Date:** 2026-08-20

## 限界上下文

- **推荐（taskReferral）**：渠道分享码、资格申请、绑边编排。
- **计费（taskBill）**：推荐边、佣金计提、微信分账。taskFE 为展示适配器。

## 聚合

| 聚合根 | 不变式 |
|--------|--------|
| ReferralShareChannel | code 全局唯一且非 `u{userId}`；`(owner, name)` 唯一；每用户恰好一个 is_default；active≤20；禁用后不可再解析绑边 |
| ReferralQualification（既有 referral_code） | approved 且未过期才有分成资格 |
| ReferralEdge（billing_referral_edge） | PK=referred_user_id；channel_code/commission_eligible 首次写入后不变 |
| ReferralCommissionAccrual | source_txn_id 唯一；仅 eligible 边可插入 pending |

## 值对象

- `AccessCode`：10 位不透明字母数字
- `ChannelName`：trim 后 1–32 字
- `CommissionEligible`：绑边当时布尔快照

## 领域事件

| 意图 | 事件 | 发布 | 例外 |
|------|------|------|------|
| 创建渠道 | REFERRAL_CHANNEL_CREATED | handleCreateChannel 结构化日志 | 首期待 Kafka，与申请事件同档 |
| 禁用渠道 | REFERRAL_CHANNEL_DISABLED | disable handler 日志 | 同上 |
| 绑边 | — | HTTP 同步 sync-edge | 既有例外：避免统计延迟 |
| 计提 | 既有 accrual insert | taskBill | 无新 topic |

## 端口

- 命令：CreateChannel、DisableChannel、BindFromAccessCode、AccrueIfEligible
- 查询：ListChannels、ReferralStats(filter)
