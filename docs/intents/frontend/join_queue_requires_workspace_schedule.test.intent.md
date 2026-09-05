# 测试意图：加入自动执行队列前须确认工作空间已启用自动调度

## 测试目标

验证任务详情「加入自动执行队列」在工作空间自动调度未启用时不入队，已启用时仍走既有 PATCH。

## 测试分层

| 层 | 文件 | 覆盖 |
|----|------|------|
| 单元 | `useQueuedAutoRunPanel.test.js` | `enabled` 判定与 GET 解析 |
| 组件 | `TaskDetailQueuedScheduleToggle.test.js` | 未启用弹窗/不 PATCH；已启用 PATCH；GET 失败 data-traceId |
| 组件 | `TaskDetailQueuedScheduleToggle.click-guard.test.js` | 已启用路径 PATCH 仍带 Idempotency-Key |
| E2E | 既有 panels-regression 仅断言按钮可见，不点加入 | 无需改点击；join 行为由组件测覆盖 |

## 用例矩阵

| ID | 给定 | 当 | 则 |
|----|------|----|----|
| T1 | `schedule_rhythm.enabled=true` | 点加入 | GET queue-schedule 后 PATCH `queued_auto_run=true` |
| T2 | `enabled=false` 或 `schedule_rhythm=null` | 点加入且取消确认 | 不 PATCH；弹出「尚未启用自动调度」 |
| T3 | 未启用且确认「前往设置」 | 点加入并确认 | `location.assign` 指向 queue-schedule 且带 workspace_id |
| T4 | GET 500 且响应带 trace | 点加入 | 错误节点 `data-traceId`；不 PATCH |
| T5 | 已启用 | 点加入 | PATCH 头含 `Idempotency-Key`（既有 click-guard） |

## 数据与环境

- Vitest + jsdom；`window.apiFetch` mock 按 URL 分流 GET `/queue-schedule/` 与 PATCH todos。
- `modalService.confirm` mock：resolve=确认，reject=取消。

## 通过标准

上述 T1–T5 全绿；既有未入队渲染、离开队列、入口链接用例仍通过。
