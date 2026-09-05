# 测试意图：推荐页服务号关注闸门（动态码）

| ID | 场景 | 期望 |
|----|------|------|
| UT1 | status 未绑定 | 有 `referral-mp-qr`，无 `referral-apply-submit`；QR src 来自 follow-qr 而非 jjf_qrcode.png |
| UT2 | 点「我已关注」仍 false | 仍无表单 |
| UT3 | follow-status bound | 出现名称/简介表 |
| UT4 | status 已绑定 | 直接有申请表，无强制 QR 挡板 |
| UT5 | 申请表提交 | 仍带 Idempotency-Key（回归） |
| UT6 | follow-status `has_unionid=false` | 提示先绑定微信登录，仍无申请表 |
| UT7 | 点「绑定微信登录」 | 整页跳转 `/api/auth/wechat/bind/?app=web&next=/profile/referral/` |
| UT8 | follow-status conflict | `referral-mp-follow-error` 含服务端 message，无他人 user_id |
| UT9 | 进入闸门 | POST follow-qr 带 Idempotency-Key |

可执行：`ReferralServiceAccountFollowGate.test.js`、`UserReferral.mpFollowGate.test.js`
