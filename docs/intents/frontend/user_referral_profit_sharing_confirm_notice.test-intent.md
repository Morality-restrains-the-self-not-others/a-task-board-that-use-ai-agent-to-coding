# 测试意图：推荐资格卡注明微信分账须有效期内手动确认

## 覆盖

1. 「推荐资格」卡片存在 `data-testid="referral-profit-sharing-confirm-notice"`。
2. 未获资格（申请表可见）时文案含：有效期内、点击确认、逾期无法分账（或无法补分）。
3. 已获资格时同一说明仍可见。
4. 组件单独挂载时文案完整、无空节点。

## 可执行测试

- `taskFE/app/src/components/ReferralProfitSharingConfirmNotice.test.js`
- `taskFE/app/src/views/UserReferral.profitSharingConfirmNotice.test.js`
