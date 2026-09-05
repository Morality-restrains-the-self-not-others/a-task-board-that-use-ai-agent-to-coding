# 评论级容器令牌

## 意图

容器 access/refresh 按评论隔离：同一 Task 下每个评论独立一行 `credential_container_tokens`，互不抢票。

## 验收

1. `IssueToken` 必须带 `comment_id`；两评论两行
2. 评论 A `exchange-refresh` 后评论 B 仍可用自己的 bootstrap access 换票
3. Init 路径含 `/comment/{commentId}`；by-scope / exchange / refresh 可带 `comment_id`
4. Cloud `bootstrapStartVmTokens` URL 含 comment
5. onlineServiceJS **所有**经 `postJson` 的 `server-container-token/*` 以及 container-agent-comments 出站 JSON **必须**带 `comment_id`（`COMMENT_ID` 有则带）；SaaS `resolveInboundCommentCSC` 两评论时无 comment_id 会 400「缺少评论ID」
6. DDL 在 `dataMigrate/taskCredentialService/002_comment_id.sql`；业务进程不 migrate

## 设计文档

- `docs/superpowers/specs/2026-08-14-comment-scoped-container-token-design.md`
- ADR-0005

## 变更日期

2026-08-14

## 业务意图 → 事件对照

| 业务意图 | 事件名（过去式） | MQ类型/契约 | 发布点 | 消费者/副作用 | 例外理由 |
|---------|----------------|------------|--------|--------------|---------|
| 评论级容器令牌签发/换票 | — | — | — | — | 令牌行状态机，无新增 Kafka 事件；审计表已有 token_issued / exchange_refresh |
