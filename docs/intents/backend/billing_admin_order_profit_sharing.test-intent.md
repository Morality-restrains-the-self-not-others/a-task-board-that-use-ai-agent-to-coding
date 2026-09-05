# 测试意图：管理员订单分账详情

| ID | 场景 | 期望 |
|----|------|------|
| T1 | staff GET 有分账订单 | 200，`profit_sharing[0].receiver_user_id` 与 `amount_yuan` 正确 |
| T2 | staff GET 无分账订单 | 200，`profit_sharing` 为 `[]` |
| T3 | 无网关头 | 401 |
| T4 | roles=member | 403 |
| T5 | 未知 order_id | 404 |
| T6 | 租户 GET 同单 | body 无 `profit_sharing` 键 |
| T7 | 响应不含 openid / referrer_openid | JSON 序列化无该字段 |
| T8 | 失败行已落 fail_trace_id | `profit_sharing[0].fail_trace_id` 等于该值 |

映射：`taskBill/src/handlers_admin_get_order_test.go`
