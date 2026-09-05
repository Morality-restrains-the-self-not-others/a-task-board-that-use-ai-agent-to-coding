# [运行时] TASK_STATUS_CHANGED 释放机器因 8018 connection refused 20s 内重试耗尽进 DLT

## 基本信息

- 版本：1.0.0
- 创建日期：2026-08-22
- 编号：110
- 维护者：Trae AI 团队
- trace_id：`6a5afce3-26bd-41de-a559-1c6a819275e2`

## 失败现象

- Kafka DLT 邮件：`event_type=TASK_STATUS_CHANGED`，`original_topic=task-status-changed`，`failure_reason=retry_exhausted`，`retry_count=11`
- key：`task_878617443474370560`（进度列改为「已取消」）
- error：`Get "http://10.2.150.68:8018/api/internal/cloud-server-config/list-by-task/…": dial tcp 10.2.150.68:8018: connect: connection refused`
- SSE fanout 消费者 `dispatch_ok` / `acked`；看板进度已变，释放路径进 DLT

## 失败环境

- 消费者：`task-events-task-status-changed-1-release-servers-on-terminal`
- 上游：`taskCloudService` `:8018`
- 编排：runAll `POST /api/precise-restart`（10:31:48）随后 `POST /api/restart-all`（10:32:19 与 10:37:10）

## 排查过程（日志时间线，CST）

1. Loki `{job=~".+"} | json | trace_id="6a5afce3-…"`：release 消费者 10:35:49–10:36:09 共 11 次 `config_load_failed` / `dispatch_retryable`，间隔约 2s，然后 `dispatch_retry_exhausted`。
2. 同 trace 的 fanout 在 10:36:59 `dispatch_ok`（部分 `idempotency skip`）。
3. `task-cloud-service` 在窗口内无日志；进程于 **10:37:00** 才再次 `listening on 0.0.0.0:8018`。
4. Kafka UI `task-status-changed-dlt` offset 2 确认 `progress_column_name=已取消`，`_retry_attempt=10`。
5. `list-by-task`：CSC 在 09:14 已 `Released`、`running_machine_count=0`（无孤儿机）。10:47 重放原事件后 `dispatch_ok`，`terminal_hard_release_done released=1`。

## 根因

1. **直接原因**：释放消费者调用 `:8018` 时 taskCloudService 正处于 runAll 全部重启空窗（connection refused）。
2. **放大原因**：`consumer.Run` 对 retryable 会 Ack 后 **re-publish** 带 `_retry_attempt` 的新消息，并 `clearBackoff`。下一次消费构造 **全新** `Backoff`，`Wait()` 永远从 Initial（1s）开始。注释写的 1s→2s→4s→…→30s（合计约 181s）并未生效，11 次重试在约 **20s** 内耗尽，远短于一次服务重启。
3. 同类先例：`.learnings/LEARNINGS.md` LRN-20260821-001（loopback `:8018`）；本案 URL 已是 `INFRA_HOST`，问题在重试窗口而非地址。

## 解决方案

1. 新增 `broker.DelayForAttempt` / `WaitAttempt`：延迟只由耐久 `_retry_attempt` 计算。
2. `dispatchRetryWait` 在共享 `consumer/runner.go` 使用该延迟；`dispatch_retryable` 日志带 `retry_wait_ms`。
3. 8018 恢复后把 DLT 原 payload（去掉 `_retry_attempt`）重投 `task-status-changed`，确认 `dispatch_ok`。

## 验证

```bash
cd taskEvents && go test ./broker ./consumer -count=1 -run 'TestDelayForAttempt|TestDispatchRetryWait|TestBackoffWaitAttempt'
curl -sS 'http://127.0.0.1:8018/api/internal/cloud-server-config/list-by-task/?tenant_id=877397588196749312&workspace_id=ws_-2309487803472456748&task_id=task_878617443474370560'
# running_machine_count=0, last_runtime_status=Released
```

## 预防措施

1. 禁止在 Ack+republish 后依赖进程内 `Backoff.current`。
2. 全部重启仍可能超过 ~3min 重试窗；补偿路径（入站 410 / reconcile）继续保留。见 OPT-20260822-002。
3. 排障有 traceId 时先查 Loki 全 job，再看 DLT 与 `:8018` 监听。
