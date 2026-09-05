# 价值流：25 天分账兜底

触发：资源订单已支付且存在（或应存在）推荐分账义务。  
结果：在微信约 30 天冻结窗口内完成分账申请，避免资金解冻给商户后无法再分。

```
支付成功 ──8 天主路径──► pending ∩ settle_after<=now ──► CreateOrder
                │
                └──失败/延迟过大/漏行──► 满 25 天且 <30 天 ──► 同一 process-pending 兜底 CreateOrder
```

测试点 T59（`docs/flows/value-stream-test-integration.wsd`）：主路径 8 天不变；25–30 天窗口补申请；重放幂等。
