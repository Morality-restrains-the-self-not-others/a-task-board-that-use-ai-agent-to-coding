# 功能意图：推荐资格申请校验并持久化个人名称

## 意图

`POST /api/accounts/users/referral-codes/apply/` 必须接收并校验 `legal_name`，通过后写入 `referral_code.legal_name`，供后续向微信登记分账接收方使用。

## 角色

- 已登录申请用户
- taskReferral 服务

## 行为

1. 缺少或非法 `legal_name`（空、不足 2 字、超过 32 字、含数字等非法字符）返回 `invalid_legal_name`。
2. 合法名称写入申请行；状态接口回传 `legal_name`。
3. 向 taskBill 登记分账接收方时带上 `legal_name`。
4. 日志只记录 `legal_name_len`，不记录名称原文。

## 非目标

- 代用户修改微信实名；存量空名称仍省略微信 `name` 字段。

## 业务意图 → 事件对照

无新事件。
