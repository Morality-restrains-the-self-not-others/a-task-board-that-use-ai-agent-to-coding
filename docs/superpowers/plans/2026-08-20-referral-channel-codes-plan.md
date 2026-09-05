# 实施计划 — 多渠道推荐码

## 切片 1 — Schema + 渠道 CRUD

- [x] `dataMigrate/taskReferral/007_referral_share_channel.sql`
- [x] `ensureUserShareCode` 写默认渠道；`create/list/disable`
- [x] 测试：多渠道、同名幂等、禁用后 lookup 空、拒绝 u{userId}

验证：`go test ./src -count=1 -run 'TestEnsureUserShareCode|TestCreateReferralChannel|TestLookupShareCodeOwner'`

## 切片 2 — 绑边快照 + 无资格不计提

- [x] `dataMigrate/taskBill/052_referral_channel_and_eligible.sql`
- [x] upsert 带 channel_code + commission_eligible
- [x] accrue / 微信分账看 eligible；findReferrerForOrder 用订单买家
- [x] 测试：eligible=false 不计提；ON DUPLICATE 不覆盖快照

验证：`go test ./src -count=1 -run 'TestReferralAccrue|TestBindReferral|TestFindReferrerForOrder'`

## 切片 3 — 分渠道分时段 stats

- [x] `getReferralStatsFiltered`
- [x] 测试：渠道过滤、from/to

验证：`go test ./src -count=1 -run 'TestGetReferralStats'`

## 切片 4 — 前端

- [x] `ReferralChannelPanel.vue` + `ReferralStatsPanel.vue`
- [x] UserReferral 组装；accessCode 回归仍绿
- [x] 创建按钮 createClickGuard

验证：vitest `UserReferral.accessCode.test.js` `ReferralChannelPanel.test.js` `ReferralStatsPanel.test.js`

## 切片 5 — 意图 / 价值流 / 架构

- [x] intents、wsd、value-stream.yaml、ADR-0025、架构 v92 四件套

## 事件任务

REFERRAL_CHANNEL_CREATED / DISABLED 结构化日志（待 Kafka）。绑边仍 HTTP 同步。

## 可观测性

日志：`referral_channel_created`、`referral_bind_edge`（含 channel_code、commission_eligible）、`referral accrue skipped ineligible`。禁止码明文以外的密钥。
