# 后端：意见与建议链接（租户累计消耗可见）

- **状态**: accepted
- **日期**: 2026-08-30
- **服务 Owner**: taskBill（`billing_feedback_link_*`）
- **设计:** `docs/superpowers/specs/2026-08-30-tenant-feedback-links-by-consumption-design.md`

## 业务意图

平台超管配置多组外链；每组可绑多种资源的 **≥ 阈值（AND）**。租户成员拉取链接时，只返回当前租户**历史累计消耗**已全部达标的启用组。资源种类可扩展（任务帖、GitLab 磁盘/流量、已消费金额及未来种类）。同一租户全员结果相同。

## 业务意图 → 事件对照

| 业务意图 | 事件名 | MQ类型/契约 | 发布点 | 消费者/副作用 | 例外理由 |
|---------|--------|------------|--------|--------------|---------|
| 新建链接组 | FEEDBACK_LINK_GROUP_CREATED | Kafka feedback-link-group-created | 超管 POST | 审计 | — |
| 更新链接组 | FEEDBACK_LINK_GROUP_UPDATED | Kafka feedback-link-group-updated | 超管 PUT | 审计 | — |
| 删除链接组 | FEEDBACK_LINK_GROUP_DELETED | Kafka feedback-link-group-deleted | 超管 DELETE | 审计 | — |
| 租户 GET 可见链接 | — | — | — | — | 纯查询 |

## 验收要点

- 阈值 AND + `>=`；空阈值组对所有租户可见
- 响应不含阈值字段
- URL 非 https → 400
- 非超管写 → 403；无 `nav.feedback.main` region 的租户 GET → 403
