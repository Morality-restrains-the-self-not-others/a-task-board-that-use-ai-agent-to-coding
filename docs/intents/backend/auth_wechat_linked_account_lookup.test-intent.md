# 测试意图：按微信关联账号解析平台用户

## 覆盖

1. 精确昵称 / openid / unionid 命中 user_id。
2. 已绑定微信用户的手机号、邮箱、用户名命中。
3. 同手机号但无 wechat_identity 的用户不命中。
4. 无命中返回空数组 200。
5. 空 q / 超长 400；无密钥 403。
6. 日志字段不含查询原文。

## 可执行测试

- `taskAuth/src/auth_wechat_linked_account_test.go`
