# DDD — auto_run 交付修复（轻量）

无新聚合。调整应用服务行为：

| 概念 | 变更 |
|------|------|
| AutoRunDelivery | 成功/干净跳过才标记完成；失败可重试 |
| DeliveryDoneMarker | 语义从「已尝试」改为「已成功交付」 |
| WorkBranch | 与 bootstrap 共用 `collectRepoBranchPlans` |

事件：继续使用既有 runtime-event（`AUTO_RUN_DELIVERY_*`），无新 MQ 业务事件（容器内闭环；书面例外：运维可观测事件非领域事件）。
