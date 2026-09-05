# 实施计划 — 推荐分成比例固定 5%

- **日期**: 2026-08-25
- **设计**: `docs/superpowers/specs/2026-08-25-fixed-referral-rate-5-percent-design.md`
- **DDD**: `docs/superpowers/plans/2026-08-25-fixed-referral-rate-5-percent-ddd.md`

## Tasks

- [x] **T1** 更新意图与价值流测试点
- [x] **T2** Red：`SystemAdminReferralManagement.ratios.test.js`
- [x] **T3** Green：管理页比例卡只读
- [x] **T4** Red：`referral_config_test.go` / wechat payout tests
- [x] **T5** Green：taskBill 恒 5%
- [x] **T6** Red/Green：taskReferral 展示恒 `5%`
- [x] **T7** 用户页说明改为固定 5%
- [x] **T8** 相关单测通过；无新 MQ
