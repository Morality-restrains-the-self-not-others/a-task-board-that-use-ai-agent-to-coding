# [运行时] Fork/创建任务后 work-panel SSE 偶发不刷新：TASK_CREATED 幂等键误用 user_id

## 现象

在已打开的 `https://www.daydaymoney.com/tenant/{tenant}/work-panel/` 上，通过任务详情 **Fork**（或同用户连续创建）成功后，看板不出现新卡，需手动刷新。用户感知为「SSE 推送失败」。

同页「创建任务」仍可能正常（本机 `tasks-updated` → `fetchTodos`），故表现为「本页创建正常、Fork/跨端创建偶发需刷」。

## 根因

`2_fanout_work_panel_sse` 的进程内幂等键 `IdempotencyKeyFromEnvelope` 对未特判事件按字段顺序取键：

`event_id` → **`user_id`** → `transaction_id` → `task_id` → `company_id`

`TASK_CREATED` / `TASK_DELETED` 载荷始终带同一操作者的 `user_id`，且排在 `task_id` 之前。于是：

1. 同一用户首次 `TASK_CREATED` → Redis 发布成功（看板可刷新）。
2. 同用户后续每次 Fork/创建 → 幂等 `Seen` 命中 → **跳过 `Dispatch`**，日志仅有 `dispatch_ok` / `acked`，**无** `work_panel_fanout_published`。
3. work-panel SSE 收不到 `task_created`，前端不 `fetchTodos`。

实测（租户 `850256677331562496` / workspace `857903329669984256`）：3 次 `TASK_CREATED` received，仅 1 次 `work_panel_fanout_published`；被吞事件的 `user_id` 相同、`task_id` 不同。

## 修复

1. `taskEvents/consumer/key.go`：`TASK_CREATED` / `TASK_DELETED` 专用 `idempotencyKeyForTaskLifecycle`（按 `task_id`，可选 `event_id`）；通用字段顺序改为 `task_id` 优先于 `user_id`。
2. 单测：`TestIdempotencyKeyFromEnvelopeTaskCreatedIgnoresUserID` / `TaskDeletedIgnoresUserID`。
3. 双保险：Fork 成功后 `emit('task-updated')`，work-panel 弹窗内立即 `fetchTodos`（不依赖 SSE）。
4. 部署：重建并重启 `task_status_changed/2_fanout_work_panel_sse`；SPA 需 `runall-lifecycle.sh build` 后静态资源生效。

## 验收

- 同 `user_id` 连续两条 Kafka `TASK_CREATED`（不同 `task_id`）均出现 `work_panel_fanout_published`
- `go test ./consumer/ -run TestIdempotencyKeyFromEnvelopeTask`
- 手工：已打开 work-panel 时 Fork 两次，无需 F5 即出现两张新卡

## 关联

- 相近：`63_chrome_plugin_create_task_work_panel_no_refresh.md`（缺事件/前端丢弃；本条为幂等吞事件）
- 代码：`taskEvents/consumer/key.go`、`workpanelfanout`、`taskDetailEditing.js` / `useTaskDetail.js`
