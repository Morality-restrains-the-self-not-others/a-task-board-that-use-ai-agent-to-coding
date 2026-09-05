# 测试意图：管理员待分账订单列表

## 覆盖

| ID | 场景 | 期望 |
|----|------|------|
| UT-PSQ-01 | staff GET 默认 open | 仅 pending/processing/failed，不含 finished |
| UT-PSQ-02 | staff GET status=pending | 仅 pending |
| UT-PSQ-03 | 无网关头 | 401 |
| UT-PSQ-04 | 非平台员工 | 403 |
| UT-PSQ-05 | 非法 status | 400 |
| UT-PSQ-06 | 响应体 | 含 `app_id` / `openid` / 微信单号；不含 `referrer_openid` 键 |
| UT-PSQ-09 | 台账 openid 空、接收方有值 | `openid` 回退 receiver.openid |
| UT-PSQ-10 | 台账快照为网站应用 openid、接收方为支付/mp 对 | 返回接收方成对 `app_id`/`openid`，不得混用快照 |
| UT-PSQ-07 | 分页 | limit/offset/total 正确；settle_after 升序 |
| UT-PSQ-08 | fail_trace_id 已落库 | 列表项返回该字段，与 fail_reason 同属该失败次 |

## 落点

`taskBill/src/handlers_admin_list_profit_sharing_test.go`
