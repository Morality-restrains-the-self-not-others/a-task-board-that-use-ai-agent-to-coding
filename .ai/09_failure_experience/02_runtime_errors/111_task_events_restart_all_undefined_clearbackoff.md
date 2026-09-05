# [运行时] 全部重启 task-events 编译失败：undefined clearBackoff / DelayForAttempt

## 基本信息

- 版本：1.0.0
- 创建日期：2026-08-22
- 编号：111
- 维护者：Trae AI 团队
- 关联：[110](./110_task_status_changed_dlt_cloud_refused_retry_backoff_reset.md)

## 失败现象

`http://10.2.150.68:9999/` 点击「全部重启」：已完成 73，失败 3。

失败服务与 stderr：

- `task-events-member-joined-1-create-default-git-identity`（`:18060`）
  `undefined: clearBackoff` / `undefined: retryState`（`consumer/runner.go:163`）
- `task-events-user-created-0-create-company`（`:18025`）
  同上
- `task-events-task-status-changed-2-fanout-work-panel-sse`（`:18048`）
  `undefined: broker.DelayForAttempt`（`consumer/retry_wait.go:16`）

健康检查 60s `connection refused`：进程因编译失败未起来，探针打空端口。

## 失败环境

- 共享包：`taskEvents/consumer`、`taskEvents/broker`
- 编排：runAll `POST /api/restart-all` 对每个 `task-events-*` 停→启；当时 `taskEvents/run.sh start` 仍隐式 `go build`（已由 ADR-0027 拆除）
- 工作树当时处于 [110](./110_task_status_changed_dlt_cloud_refused_retry_backoff_reset.md) 的未提交重构中途

## 根因

为修 Ack+republish 后进程内 `Backoff` 被重置，正把 `retryState`/`clearBackoff` 换成 `broker.DelayForAttempt` + `WaitAttempt`。全部重启按**磁盘工作树**编译，打到半成品：

1. `runner.go` 仍调用 `clearBackoff(msg, retryState)`，辅助函数已删。
2. 已新增 `retry_wait.go` 调用 `DelayForAttempt`，`broker/retry.go` 尚未加入该符号。

同包消费者并行编译，故只有落在该窗口的 2～3 个服务失败，其余 73 个编过 HEAD 或已闭环的工作树。

## 解决方案

1. 完成工作树：删除 `clearBackoff`/`retryState`/`msgKey`；`dispatchRetryWait` + `Backoff.WaitAttempt` 按 `_retry_attempt` 计算延迟。
2. `broker.DelayForAttempt` / `WaitAttempt` 与对应单测一并落地并提交，避免下次全部重启再编半成品。
3. 对仍为 `failed` 的服务做精准编译重启。

## 验证

```bash
cd taskEvents && go test ./broker ./consumer -count=1 -run 'TestDelayForAttempt|TestDispatchRetryWait|TestBackoffWaitAttempt'
gofmt -l consumer/runner.go consumer/retry_wait.go broker/retry.go
# 精准重启失败项后：
curl -sf "http://10.2.150.68:18060/api/health/"
curl -sf "http://10.2.150.68:18025/api/health/"
```

## 预防措施

1. **ADR-0027**：全部重启 / 普通 ↻ / `run.sh start` 只 exec last-good，不得编译工作树。需要新二进制时用精准编译重启（compile-then-swap）或全部重新编译。
2. 共享 `consumer`/`broker` 改动未形成可编译闭包前，不要点「精准编译重启」；先 `go build ./consumer ./broker ./cmd/...`。
3. 禁止在 Ack+republish 路径保留对 `clearBackoff`/`retryState` 的调用（见 110）。
