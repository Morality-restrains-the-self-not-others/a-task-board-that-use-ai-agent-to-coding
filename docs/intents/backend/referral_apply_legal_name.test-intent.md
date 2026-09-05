# 测试意图：推荐资格申请个人名称

## 覆盖

- `normalizeLegalName`：空、过短、含数字、合法中文/间隔号。
- `applyReferralCode` 缺名称失败；合法名称落库并出现在 status。
- HTTP 申请缺 `legal_name` 返回 `invalid_legal_name`。

## 文件

- `taskReferral/src/referral_legal_name_test.go`
- `taskReferral/src/referral_identity_bind_consent_test.go`
- `taskReferral/src/handlers_test.go`
