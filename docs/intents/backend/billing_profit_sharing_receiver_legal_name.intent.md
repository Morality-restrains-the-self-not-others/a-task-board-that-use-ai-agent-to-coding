# 功能意图：分账接收方与分账请求携带个人名称

## 意图

taskBill 在添加微信分账接收方、发起分账时，若已有推荐人个人名称则设置 SDK `Name`（由 wechatpay-go `encryption:"EM_APIV3"` 加密）；无名称则省略。

## 角色

- taskBill 内部接口（由 taskReferral 调用）
- 微信支付分账 API

## 行为

1. 内部 ensure 请求可读 `legal_name` 并写入 `billing_profit_sharing_receiver.legal_name`。
2. 已登记但新补名称时再次调用微信 add。
3. 创建分账订单接收方带 `Name`（有名称时）。

## 非目标

- 平台代用户改微信实名。
