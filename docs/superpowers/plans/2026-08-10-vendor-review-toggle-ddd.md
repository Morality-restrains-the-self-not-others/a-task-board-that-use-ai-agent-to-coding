# Step 6 — DDD：MarketplaceSettings

## 限界上下文

**镜像市场（taskAiProvider）** 拥有：

- Aggregate：`MarketplaceSettings`（单例 id=1）
  - `VendorApplicationReviewEnabled bool`
- Aggregate：`Vendor`（既有）— bridge 在审核关闭时允许自动 provision

## 领域规则

1. 默认审核开启（安全默认）
2. 审核关闭 ⇒ bridge 可自动创建/激活 Vendor
3. 审核开启 ⇒ 无档案拒绝；inactive 区分 pending/rejected

## 事件

配置变更不投递 MQ（书面例外：无下游订阅者）。
