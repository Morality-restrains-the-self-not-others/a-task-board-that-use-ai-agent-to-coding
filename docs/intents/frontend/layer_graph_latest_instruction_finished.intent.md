# 意图：可写层摘要行展示最近指令完成时间

- **日期**: 2026-08-22
- **状态**: 已实施

## 背景与目标

任务详情「任务关联」可写层 zTree 摘要行目前只显示层数、任务数与扫描路径，例如：

`可写层 2 个（串行 · 按创建时间旧→新） · 任务 2 个 · 服务扫描 /app/onlineProject_state/layers`

操作者无法从摘要判断最近一条指令何时结束、或是否仍在跑。

目标：在同一摘要行展示**最近一条用户指令（非 clone）**的完成时间；若该指令仍在 `pending`/`running`，文案固定为 `pending`。

## 范围与边界

- 范围内：
  - 容器 `onlineServiceJS` 在 job 进入终态时写入 `finished_at`（ISO-8601），随既有 `container_layer_graph` 快照到达前端。
  - `layerGraphMetaLineFromSnapshot` 追加 `最近指令完成 <时间|pending>`。
- 范围外：不改 zTree 节点标题；不新增 HTTP API；不展示 clone 任务完成时间。

## 约束与风险

- 「最近一条指令」= 快照 `jobs` 中 `command_kind !== 'clone'`、按 `created_at` 旧→新后的最后一条。
- 进行中（`pending`/`running`，大小写不敏感）→ 字面 `pending`（用户指定英文）。
- 终态无 `finished_at`（历史落盘）→ `—`，不把 `created_at` 冒充完成时间。
- 无非 clone 任务 → 不追加该片段。
- 时间展示用 `zh-CN` 本地墙钟，与评论区 `formatDt` 一致。

## 验收标准

1. 最近非 clone job 为 `running`/`pending` → 摘要含 `最近指令完成 pending`。
2. 最近非 clone job 已完成且带 `finished_at` → 摘要含 `最近指令完成` + 本地格式化时间，不含 `pending`。
3. 仅有 clone job → 摘要不含 `最近指令完成`。
4. job close / error / interrupt 写入 `finished_at`；已有值不覆盖（中断时刻优先于进程退出时刻）。

## 业务意图 → 事件对照

| 业务意图 | 事件名（过去式） | MQ类型/契约 | 发布点 | 消费者/副作用 | 例外理由 |
|---------|----------------|-----------|--------|--------------|---------|
| 摘要行展示最近指令完成时间 | — | — | — | — | 纯展示；沿用既有 `container_layer_graph` SSE，无新领域事件 |

## 实施计划

1. `stampJobFinishedAt`：终态首次写入 `finished_at`。
2. `runJobAsync` close/error 与 `interruptJob` 调用。
3. `layerGraphMetaLineFromSnapshot` 追加片段。
4. 单测覆盖 pending / 完成时间 / 排除 clone / 不覆盖已有 `finished_at`。

## 变更记录

| 日期 | 相对旧版 | 原因 |
|------|----------|------|
| 2026-08-22 | 初版 | 可写层摘要缺少最近指令完成时间 |
