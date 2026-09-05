# DDD：ContainerImage 生命周期（下架/删除）

限界上下文：AI Provider Marketplace。聚合根：`ContainerImage`（已有）。

## 不变量

1. 仅 `approved` 可 `IsActive=true`；离开 approved 必须 `IsActive=false`。
2. 组内至多一个激活版本（既有 Activate 事务）。
3. 删除不得针对激活版本或 pending_review。

## 领域方法

- `VendorWithdraw(actorID, note, now)`：pending→draft；approved→Unpublish；draft→no-op。
- `Unpublish`：approved→draft 且 `IsActive=false`。
- `DeleteGuard() error` / `CanDelete() bool`。

## 事件

`ContainerImageReviewWithdrawn`、`ContainerImageUnpublished`、`ContainerImageDeleted`。载荷：`image_id`、`vendor_id`、`image_group_id`。

## 端口

既有 `EventBus`；无新仓储接口（`DeleteContainerImage` / `UpdateContainerImageStatus` 扩展写 `is_active`）。
