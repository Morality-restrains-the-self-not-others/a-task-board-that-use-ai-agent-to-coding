# 无推荐资格也展示 accessCode — 测试意图

对应功能意图：`user_referral_access_code.intent.md`

## 单元测试

| ID | 场景 | 期望 | 测试文件 |
|----|------|------|----------|
| T1 | 无资格 + 状态接口返回不透明 access_code | 分享链接含该码，不含 userId | `UserReferral.accessCode.test.js` |
| T2 | 无资格 + 接口无 access_code | 不回退 `u{userId}`，链接无 `accessCode=` | 同上 |
| T3 | 接口只有 referral_code（形如 u+userId） | 不把它当分享码 | 同上 |
| T4 | 无申请记录 | status.access_code 非空且非 userId 派生 | `referral_code_test.go` / `handlers_test.go` / `referral_share_code_test.go` |
| T5 | pending / expired | access_code 仍为同一不透明码 | `referral_code_test.go` |
| T6 | 未填满 20 字介绍 | 申请按钮 disabled | `ReferralQualificationApplyForm.test.js` |
| T6b | 已填介绍未勾选身份绑定 | 申请按钮仍 disabled；勾选后可提交且 payload 含 identityBindConsent | 同上 |
| T7 | 填写合法介绍后申请 | POST body 含 personal_intro + identity_bind_consent + Idempotency-Key | `UserReferral.applyIntro.test.js` |
| T8 | 注册页 `?accessCode=` | 提交体含 `access_code`；空 query 不覆盖已存码 | `registrationInviteUtils.test.js` |
| T9 | 创建渠道 | POST `/api/referral/channels/` 带 Idempotency-Key | `ReferralChannelPanel.test.js` |
| T10 | 分渠道统计筛选 | stats 请求带 channel_code/from/to | `ReferralStatsPanel.test.js` |
| T11 | 收益分成比例来自 stats.referral_rate_display | 接口返回 12% 则卡片展示 12%，不得写死 5% | `ReferralStatsPanel.test.js` |
| T12 | 资格文案来自 status.referral_rate_display | 状态接口返回 12% 则申请区展示 12% | `UserReferral.accessCode.test.js` |
| T13 | 有资格且 wechat_receiver_status=registered | 展示「已在微信分账后台登记」 | `UserReferral.accessCode.test.js` |
| T14 | 有资格且 pending_openid | 展示「尚未绑定微信登录」 | `UserReferral.accessCode.test.js` |
| T15 | 收益规则文案 | 统计面板写明下单时无资格才不分账，且先推荐后获资可从后续下单分成；不得写「绑边时」 | `ReferralStatsPanel.test.js` |
| T16 | 无资格提示 | 「暂无推荐资格」说明此时下单不分账、获资后后续下单仍可分成；不得写「无法获得收益分成」绝对句 | `UserReferral.accessCode.test.js` |
| T17 | 渠道介绍 | 渠道卡说明无资格仍可发码，分账以下单时资格为准 | `ReferralChannelPanel.test.js` |
| T18 | 即将过期提示 | 过期后「被推荐用户下单」不再分成，不得写仅「新注册用户」 | `UserReferral.accessCode.test.js` |

## 手工验收

1. 无推荐资格账号打开 `https://www.daydaymoney.com/profile/referral/`
2. 推荐链接为 `.../auth/register/?accessCode=<不透明码>`，不是空参数，也不是 `u<userId>`
