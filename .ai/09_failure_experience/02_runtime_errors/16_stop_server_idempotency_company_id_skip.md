# [运行时] 停止服务器点击无效果（CLOUD_SERVER_STOPPED 被 company_id 幂等跳过）

## 现象

任务详情页「服务器运行状态」点击「停止服务器」后：

- HTTP `stop-vm` 可能返回成功（「虚拟机停止请求已提交」）
- 实例仍显示「运行中」（含 Mock 实例 `mock-*`）
- 无「停止虚拟机成功」SSE，UI 几乎无反馈

典型路径：同租户内曾成功停止过任意一台机器后，再停另一任务（含 Mock）。

## 环境与上下文

- 前端：`POST …/cloud/compute/stop-vm/` → 依赖事件消费者清实例 + SSE
- 发布：`taskCloudService` `handleStopVmNative` → `CLOUD_SERVER_STOPPED`
- 消费：`taskEvents` `cloud_server_stopped/1_process_server_stop`
- 幂等：`IdempotencyKeyFromEnvelope` + 进程内 `MemoryStore`

## 根因

1. **幂等键误用 `company_id`**：`IdempotencyKeyFromEnvelope` 字段优先序为 `event_id → user_id → company_id → … → task_id`。`CLOUD_SERVER_STOPPED` 载荷始终带 `company_id`，导致**同租户首次成功停止后，后续所有停止被 `Seen(key)` 静默跳过**（日志仅有 `received → dispatch_ok`，无 `stop_vm_begin`）。
2. **Mock 实例无短路**：即便消费执行，也会对 `mock-*` 调阿里云 `DeleteInstance`，易失败；且 `server-runtime-status` 对 `mock-*` 曾固定 `Running`，清实例前刷新仍像「没效果」。
3. **前端成功路径无即时反馈**：`stopServer` 在 HTTP 200 时不刷新运行态、不更新启动状态文案，完全依赖 SSE。

## 修复

- `consumer/key.go`：`CLOUD_SERVER_STOPPED` 专用键（优先 `stop_request_id`，否则 `task_id:instance_id`）；通用字段序将 `task_id` 置于 `company_id` 之前
- 各发布点写入唯一 `stop_request_id`（snowflake）
- `cloudserverstopped`：`mock-*` 跳过云 API，直接 `clear-after-stop` + SSE 成功
- 前端 `stopServer`：提交中/已受理即时 `updateServerStatus`，并刷新运行态

## 预防

- 事件幂等键必须与「业务重复边界」同粒度；载荷含租户级字段时，不得默认用其作键
- 为每次「用户意图触发」的域事件生成唯一 request/event id
- 异步停机路径：HTTP 受理后须有可见 UI 态，不能只等 SSE

## 后续加固（2026-07-16）

- 启动类 `CLOUD_SERVER_STARTED` / `CLOUD_SERVER_START_AUTO`：专用幂等键，优先 `event_id`，缺省时 `task_id:trace_id`，**禁止** `company_id`
- `START_AUTO` 链式发布 `STARTED` 前若缺 `event_id`，从 pending start event 回填
- Mock 停止 SSE：处理中「正在停止 Mock 实例…」、成功「Mock 实例已停止」（前端同步识别）
