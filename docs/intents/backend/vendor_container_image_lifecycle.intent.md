# 功能意图：厂商撤回/下架/删除容器镜像版本

- **日期**: 2026-08-29
- **状态**: 实施中
- **接口**: `POST /api/vendor/container-images/{id}/withdraw/`；`DELETE /api/vendor/container-images/{id}/`

## 背景与目标

withdraw 仅支持待审核；DELETE 未做状态守卫。补齐已上架下架（含取消激活）与非激活删除。

## 范围与边界

- 范围内：taskAiProvider domain + vendor handlers + UpdateContainerImageStatus 写 is_active。
- 范围外：新 path、Kafka 消费者。

## 验收标准

1. pending_review withdraw → draft + ReviewWithdrawn。
2. approved withdraw → draft、is_active=0 + Unpublished。
3. 已是 draft 的 withdraw → 200 空操作、不重复事件亦可。
4. 激活 DELETE → 400；非激活 approved/draft/rejected DELETE → 204 + Deleted。
5. pending_review DELETE → 400。

## 业务意图 → 事件对照

| 业务意图 | 事件名 | MQ类型/契约 | 发布点 | 例外理由 |
|---------|--------|-----------|--------|---------|
| 撤回待审核 | ContainerImageReviewWithdrawn | LogEventBus | handleVendorContainerAction withdraw | 无 Kafka |
| 下架已上架 | ContainerImageUnpublished | LogEventBus | 同上 | 同上 |
| 删除版本 | ContainerImageDeleted | LogEventBus | handleVendorContainerImages DELETE | 同上 |
