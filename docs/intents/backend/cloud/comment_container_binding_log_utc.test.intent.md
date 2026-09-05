# 测试意图：binding 日志 UTC 时间

## 测试目标

UTC naive DATETIME 读回后 JSON 为 RFC3339 Z，且 `+08:00` 误标不会把墙钟平移 8 小时。

## 测试分层

- 单元：`parseCloudUTCDateTime` 表驱动（无 DB）
- API：既有 list logs 时间线测例上断言 `created_at` 后缀 `Z`

## 用例矩阵

### T1 — naive / Z / 误标 offset 同一 UTC 墙钟

- **给定** `2026-08-13 15:30:59`、`2026-08-13T15:30:59Z`、`2026-08-13T15:30:59+08:00`
- **当** `parseCloudUTCDateTime`
- **则** 均为 UTC `2026-08-13 15:30:59`

### T2 — list JSON

- **给定** create binding 产生 pending 日志
- **当** list JSON `logs[0].created_at`
- **则** 匹配 `^\d{4}-\d{2}-\d{2}T\d{2}:\d{2}:\d{2}Z$`，且与 `time.Now().UTC()` 相差不超过 2 分钟

## 通过标准

- `go test` 相关文件全绿。

## 业务意图 → 事件对照

无新事件；见功能意图。

## 自动化落点

- `taskCloudService/src/cloud_utc_datetime_test.go`
- `taskCloudService/src/comment_container_bindings_log_test.go`
