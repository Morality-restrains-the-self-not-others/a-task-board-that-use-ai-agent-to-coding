# 订单评论 — NFR

- **Date:** 2026-08-11
- **Level:** L2 Standard；鉴权路径 L3

| 类别 | 级别 | 要求 |
|------|------|------|
| 安全 | L3 | 鉴权+归属校验；content 长度限制；日志无 PII 正文 |
| 性能 | L2 | 单订单评论列表 limit≤100；索引 (order_id, created_at) |
| 可用性 | L2 | Kafka 发布失败不阻断写库成功（与既有 billing 事件一致 warn） |
| 可观测 | L2 | tracelog + event publish 埋点 |
| 兼容 | L2 | 仅新增 API/表，无破坏性变更 |
