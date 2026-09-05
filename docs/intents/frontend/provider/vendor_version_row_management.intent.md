# 功能意图：厂商门户版本行提供下架/删除等管理手段

- **日期**: 2026-08-29
- **状态**: 实施中
- **页面**: https://provider.daydaymoney.com/ 厂商门户「我的镜像组」展开后的 `div.version-row`
- **接口**: 既有 `DELETE /api/vendor/container-images/{id}/`；`POST /api/vendor/container-images/{id}/withdraw/`

## 背景与目标

已上架版本行缺少可见的删除等管理按钮（5 列 grid 被 URL 占满操作轨），且「撤回」对已上架会失败。目标：每一版本行第一行可见操作列；非激活已上架可下架或删除；激活版本须先下架。

## 范围与边界

- 范围内：厂商门户版本行布局与操作矩阵；vendor withdraw/DELETE 领域守卫与事件。
- 范围外：新 API path、管理员审核流 UI、镜像组删除确认弹窗（仍为存量 confirm，记 OPT）。

## 约束与风险

- 写操作禁止 `window.confirm`；须自定义弹窗 + `createClickGuard` + `Idempotency-Key`。
- 失败展示 `data-traceId`。
- 激活版本不可 DELETE。

## 验收标准

1. 已上架非激活行可见「展开区域明细」「设为激活」「下架」「删除」。
2. 已上架激活行可见「下架」，无「删除」。
3. 版本 URL 不挤掉操作列（操作列与 ID 同行）。
4. 下架已上架 → 草稿且非激活；删除非激活已上架 → 204 后列表消失。

## 业务意图 → 事件对照

| 业务意图 | 事件名（过去式） | MQ类型/契约 | 发布点 | 消费者/副作用 | 例外理由 |
|---------|----------------|-----------|--------|--------------|---------|
| 展开区域明细 | — | — | — | — | 纯前端 |
| 厂商下架已上架 | ContainerImageUnpublished | LogEventBus | vendor withdraw | 公开目录不再展示 | 服务无 Kafka |
| 厂商撤回待审核 | ContainerImageReviewWithdrawn | LogEventBus | vendor withdraw | 回到草稿 | 同上 |
| 厂商删除版本 | ContainerImageDeleted | LogEventBus | vendor DELETE | 组内版本减少 | 同上 |

## 实施计划

见 `docs/superpowers/plans/2026-08-29-vendor-version-row-management-plan.md`。
