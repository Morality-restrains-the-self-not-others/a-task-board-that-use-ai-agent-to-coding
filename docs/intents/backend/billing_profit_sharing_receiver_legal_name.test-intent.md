# 测试意图：分账接收方个人名称 payload

## 覆盖

- 有 `LegalName` 时 add payload 含 Name。
- 无名称时仍省略 Name（兼容存量）。

## 文件

- `taskBill/src/wechat_profit_sharing_receiver_test.go`
