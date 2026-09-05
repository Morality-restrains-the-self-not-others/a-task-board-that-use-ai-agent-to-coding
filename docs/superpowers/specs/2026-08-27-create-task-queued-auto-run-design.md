# 创建任务可选加入自动调度队列 — 设计文档

- 日期：2026-08-27
- 入口：/goal
- 范围：taskFE 创建任务模态、taskTaskService 创建/更新 auto_run 触发
- 架构变更：否（无新服务/表/端点；不产出 ArchiMate）

## 现状

- 创建任务 `auto_run=true` → `scheduleTaskAutoRunFn` 立即 start-vm。
- 工作空间节奏：`GET/PUT /api/tenant/{tid}/workspace/{wid}/queue-schedule/`。
- 入队：body `queued_auto_run` → `applyQueuedAutoRunFromBody`（创建路径已调用）。
- 任务详情才能加入队列；创建模态只有「是否自动运行」。

## 决策

| ID | 决策 |
|----|------|
| D1 | 嵌套勾选「加入自动调度队列」，仅当 auto_run 已勾 **且** `schedule_rhythm.enabled===true` |
| D2 | 默认未勾选（opt-in），避免改变现网立即启服 |
| D3 | 勾选则 payload `queued_auto_run: true`；关闭 auto_run 则清 false |
| D4 | 服务端 queued 且非 `force_auto_run` 时跳过立即 start-vm；dispatcher 再启服（`QueuedAutoRunStarted`） |
| D5 | 模态打开时 GET 一次 queue-schedule，禁止轮询 |
| D6 | GET 失败隐藏选项，不阻断创建；有 trace 则 `data-traceId` |
| D7 | Chrome 插件不在本增量（OPT） |

## 交互

1. 打开创建任务 → GET queue-schedule。
2. 用户满足自动运行门禁并勾选「是否自动运行」。
3. 若工作空间已启用自动调度 → 显示嵌套勾选。
4. 勾选 → 入队、不立即启服；不勾 → 立即启服。

## 测试

见 `create_task_queued_auto_run.test-intent.md`。
