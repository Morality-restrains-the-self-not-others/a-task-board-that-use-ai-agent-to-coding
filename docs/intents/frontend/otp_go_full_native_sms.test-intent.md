# 测试意图：otp_go_full_native_sms

对应：`otp_go_full_native_sms.intent.md`

| ID | 场景 | 方式 | 期望 |
|----|------|------|------|
| T1 | mock 发码 | Go test | 200 + DB 有码 |
| T2 | aliyun 缺配置 | Go test | 失败 |
| T3 | password_reset 发+验 | Go test | 改密成功 |
| T4 | phone+code 登录 | Go test | enrich-login 被调；forward-login 不调 |
| T5 | 充值 send_code 委托 | 既有 Django→Go | 不回归 |
