# DDD 轻量记录：released-server-comments-retain-layer-loading

- **日期**: 2026-07-15
- **结论**: **无新服务端领域模型 / 无新领域事件**（纯前端）。

## 概念

| 概念 | 类型 | 说明 |
|------|------|------|
| ServerNotServingUiState | UI 状态 | pause 心跳、清 endpoint、层图空态 |
| RetainedTaskComments | 既有读模型 | taskAIComment + task comments；生命周期独立于容器 |

## 业务意图 → 事件

| 业务意图 | 事件 | 例外理由 |
|---------|------|---------|
| 展示已释放任务的历史评论 | — | 纯查询 |
| 进入非服务 UI 态 | — | 纯前端 |

既有 v27 `CLOUD_SERVER_STOPPED` 等事件本期不修改。
