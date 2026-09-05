# 测试意图：停服日志注明触发源头

## 覆盖

1. `stopReasonTriggerLabel` / `annotateStopServerMessage` 码表与幂等。
2. `buildStopVmEventData` 写入 `stop_reason` + `stop_reason_label`。
3. `releaseMachineForTerminal` 将带触发说明的停服行写入 binding logs。
4. `CLOUD_SERVER_STOPPED` SSE processing 文案含 `触发：`。

## 对应源码测例

- `taskCloudService/src/stop_reason_display_test.go`
- `taskCloudService/src/compute_stop_vm_event_test.go`
- `taskCloudService/src/request_machine_release_test.go`
- `taskEvents/internal/handlers/cloudserverstopped/handler_test.go`
