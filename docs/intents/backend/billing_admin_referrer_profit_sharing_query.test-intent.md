# 测试意图：超管按推荐人查分账台账与微信核单

## 测试目标

过滤正确、权限正确、staff 列表含 `app_id`/`openid`（不含 `referrer_openid` 键）、QueryOrder 不写库且越权 id 被拒。

## 测试分层

- taskBill Go 单测（httptest + 注入 `profitSharingQueryOrderCall`）

## 用例矩阵

| ID | 场景 | 期望 |
|----|------|------|
| T1 | 推荐人 A 有被推荐人 U1/U2；仅 U1 订单有 `billing_profit_sharing` | 只返回 U1 的打标订单，且 `referred_user_id=U1` |
| T2 | U2 有已支付订单但无分账台账行（未打标） | 不出现在列表 |
| T3 | 不传 `referrer_user_id` | 与现网全局队列一致 |
| T4 | `status=all` | 含 finished |
| T5 | 响应 JSON | 含 `app_id`/`openid`/微信单号；不含 `referrer_openid` 键；有接收方时成对取自登记行 |
| T6 | 非 staff | 403 |
| T7 | refresh：ids 含非该推荐人图内记录 | 400，不调微信 |
| T8 | refresh：微信返回 FINISHED | 响应 `wechat_state=FINISHED`，DB status 不变 |
| T9 | refresh：缺 transaction_id | 该行 wechat_error，其它行成功 |
| T10 | QueryOrder 被 FREQUENCY_LIMITED | 该行错误人类可读，不暴露密钥 |

## 通过标准

T1–T10 全绿。无新领域事件。
