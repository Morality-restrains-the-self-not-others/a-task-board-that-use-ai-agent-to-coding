# 功能意图：任务帖配额拆分赠送与购买剩余

## 背景与目标

租户账单页「任务帖配额」只展示 `task_post_quota` 合计。后台赠送走 `billing_resource_grant`（消耗时优先扣赠送），购买只加账户计数。用户无法分辨剩余里多少是赠送、多少是购买。目标是在既有 `GET /api/tenant/{id}/billing/quotas/` 上返回拆分字段，合计字段保持兼容。

## 范围与边界

- 范围内：`handleResourceQuotas` 响应增加 `task_post_quota_gifted`、`task_post_quota_purchased`；`task_post_quota` 仍为总剩余。购买支付写入 `source_kind=purchase` 批次，消耗优先扣赠送。
- 范围外：GitLab 磁盘/流量拆分、新独立 endpoint

## 约束与风险

- 向后兼容：旧客户端可忽略新字段
- 赠送剩余 = 未过期且 `remaining>0` 的 `task_post` grant 之和
- 购买剩余 = `max(0, 总剩余 - 赠送剩余)`；赠送剩余不超过总剩余

## 验收标准

1. 总剩余 83、未过期赠送 remaining 50 → `task_post_quota=83`、`task_post_quota_gifted=50`、`task_post_quota_purchased=33`
2. 仅购买、无 grant → gifted=0，purchased=总剩余
3. 已过期 grant 的 remaining 不计入 gifted
4. OpenAPI 描述上述字段

## 实施计划

1. `getTaskPostQuotaBreakdown` 查询总剩余与未过期赠送 remaining
2. `handleResourceQuotas` 写入三个字段
3. 单测 + OpenAPI

## 业务意图 → 事件对照

| 业务意图 | 事件名（过去式） | MQ类型/契约 | 发布点 | 消费者/副作用 | 例外理由 |
|---------|----------------|------------|--------|--------------|---------|
| 查询任务帖配额拆分 | — | — | GET quotas | — | 纯查询，无对应事件 |
