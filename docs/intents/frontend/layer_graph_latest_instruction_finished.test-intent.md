# 测试意图：可写层摘要行最近指令完成时间

## 测试目标

验证任务关联 zTree 摘要行根据最近非 clone 指令状态显示 `pending` 或完成时间；容器侧 job 终态写入 `finished_at`。

## 测试分层

| 层 | 位置 | 覆盖 |
|----|------|------|
| 单元 | `taskFE/app/src/composables/taskDetail/commentLayerPanelBind.test.js` | 摘要行文案 |
| 单元 | `trae-agent/onlineServiceJS/src/jobsRuntime.finishedAt.test.mjs` | `stampJobFinishedAt` 幂等 |
| 单元 | `trae-agent/onlineServiceJS/src/jobsRuntimeRunJob.closeDelivery.test.mjs` | close/error 写入 `finished_at` |

## 用例矩阵

### T1 — 执行中显示 pending

- **给定** snapshot.jobs 最近一条非 clone 为 `running` 或 `pending`
- **当** `layerGraphMetaLineFromSnapshot`
- **则** 文案含 `最近指令完成 pending`

### T2 — 已完成显示 finished_at

- **给定** 最近非 clone job `status=completed` 且 `finished_at=2026-08-22T14:01:03.000Z`
- **当** 生成摘要行
- **则** 含 `最近指令完成` 与本地格式化后的 2026 日期，不含 `pending`

### T3 — 排除 clone

- **给定** 仅 clone job，或最新是 clone、更早的 trae 已完成
- **当** 生成摘要行
- **则** 「最近」取非 clone 最后一条；仅 clone 时不追加片段

### T4 — 无指令任务

- **给定** jobs 空
- **当** 生成摘要行
- **则** 不含 `最近指令完成`

### T5 — finished_at 首次写入且不覆盖

- **给定** job close/error/interrupt 进入终态
- **当** `stampJobFinishedAt`
- **则** 空则写入 ISO；已有值保持不变

### T6 — close 路径落盘

- **给定** `runJobAsync` 进程 close 0
- **当** 进入 completed
- **则** `rec.finished_at` 为非空 ISO 字符串

## 数据与环境

- 前端 vitest node；时间断言用年份/前缀，避免依赖具体 locale 分隔符。
- onlineServiceJS 测例须设置 `ONLINE_PROJECT_STATE_ROOT` 临时目录。

## 通过标准

上述 T1–T6 全绿。意图发生时无新 MQ 事件（纯展示例外）。

## 业务意图 → 事件对照（测试侧）

| 用例 | 对应意图 | 事件断言 | 例外 |
|------|----------|----------|------|
| T1–T6 | 摘要展示完成时间 | 无 | 纯前端/快照字段，复用既有 SSE |
