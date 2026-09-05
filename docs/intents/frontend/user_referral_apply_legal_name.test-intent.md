# 测试意图：申请推荐资格个人名称

## 覆盖

- 未填个人名称或不足 2 字：提交禁用。
- 警示文案含「填错」与「分账失败」。
- 勾选同意 + 合法介绍 + 合法名称后提交，body 含 `legal_name`。
- 后端缺 `legal_name` 返回 `invalid_legal_name`。
- 有名称时添加分账接收方 payload 含 `Name`；无名称时仍省略。

## 文件

- `taskFE/app/src/components/ReferralQualificationApplyForm.test.js`
- `taskFE/app/src/views/UserReferral.applyIntro.test.js`
- `taskReferral/src/referral_legal_name_test.go`
- `taskBill/src/wechat_profit_sharing_receiver_test.go`
