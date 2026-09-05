# 意图：ECS 二次启动孤儿防护

- **日期**: 2026-07-18
- **设计**: `docs/superpowers/specs/2026-07-18-ecs-orphan-double-start-guard-design.md`
- **OPT**: OPT-20260718-018

## 行为

1. `CLOUD_SERVER_STARTED` 在 RunInstances 前若 CSC 已有非 mock `instance_id`，同步 DeleteInstance 并 `ClearAfterStop(instance_id=旧)`。
2. `ClearAfterStop` / history close 仅关闭匹配 `instance_id` 的 open history；不清空指向其他实例的 CSC。
3. workspace runtime reconcile 按本评论 CSC 的 InstanceName（`task_{taskId}_{commentId}`）对账，删除非当前绑定的云实例。任务级模板行（空 comment_id）不参与按名 Describe。

## 业务意图 → 事件对照

**无对应事件**：本意图不新增领域事件类型；孤儿防护挂在既有 `CLOUD_SERVER_STARTED` / `CLOUD_SERVER_STOPPED` 编排路径内（start 前同步 StopVM / DeleteInstance），不单独投递新 MQ 契约。

| 业务意图 | 事件名（过去式） | MQ类型/契约 | 发布点 | 消费者/副作用 | 例外理由 |
|---------|----------------|------------|--------|--------------|---------|
| ECS 二次启动孤儿防护 | — | — | — | — | 无对应事件：复用既有 CLOUD_SERVER_STARTED/STOPPED 路径，不新增 publish |

## 变更记录

- 2026-08-16：对账 InstanceName 仅评论级规范名；去掉任务级/legacy 回退。
