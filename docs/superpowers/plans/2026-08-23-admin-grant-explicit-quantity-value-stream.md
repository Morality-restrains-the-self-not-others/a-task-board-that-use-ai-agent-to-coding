# 价值流：管理端赠送显式数量

在既有「系统管理 · 赠送资源与 VIP」上增加增量：**显式数量才赠送**。

1. 管理员打开赠送页 → 数量空
2. 只改 VIP → 预览「不生成赠送订单」→ POST 空 resources
3. 填写数量后才生成零元赠送订单，租户订单列表可见

测试点见 `docs/flows/value-stream-test-integration.wsd` AGR 注释 TQ-1..5。
