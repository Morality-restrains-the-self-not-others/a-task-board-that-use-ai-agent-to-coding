# DDD 领域模型：平台集中日志（runAll + AiMonitor）

> 输入: 价值流 + NFR 澄清（2026-05-28-centralized-logs-grafana-traceid-*）

## 限界上下文

| 上下文 | 职责 |
|--------|------|
| **orchestration-lifecycle**（runAll 已有） | 子进程 stdout 采集、内存日志 API |
| **platform-observability**（本增量） | 日志持久化 tee、traceId 格式约束 |
| **request-tracing**（task2app 已有） | HTTP X-Trace-Id 传播 |

本增量仅在 **runAll/domain** 扩展；Loki/Promtail/Grafana 为基础设施，不入领域层。

## 聚合

### LogCapture（runAll）
- **聚合根:** 无新实体；扩展 `ServiceLogRepository` 组合行为
- **值对象:** `LogEntry`（已有）、`TraceId`（新增校验 VO）
- **端口:** `ServiceLogFileSink` — 按服务名追加写文件行

### TraceCorrelation（概念，Increment 2）
- Loki 查询由 Grafana 适配器实现，不在 runAll 领域层建模。

## 值对象

### TraceId (`runAll/src/domain/trace_id_value_object.go`)
- 格式: `^[A-Za-z0-9._:-]{8,256}$`
- 方法: `ParseTraceId(raw string) (TraceId, error)`、`String() string`

## 仓储 / 端口接口

### ServiceLogFileSink (`runAll/src/domain/service_log_file_sink.go`)
```go
type ServiceLogFileSink interface {
    AppendLine(serviceName, stream, message string) error
    Close() error
}
```

### TeeServiceLogRepository（基础设施组合，非领域）
- 实现 `ServiceLogRepository`，委托 memory + 可选 file sink
- file 写失败：记录 stderr，不 panic

## 领域事件
- 无（日志 tee 为基础设施关注点，不发布领域事件）

## 文件清单（Increment 1）

| 路径 | 类型 |
|------|------|
| `runAll/src/domain/trace_id_value_object.go` | VO |
| `runAll/src/domain/trace_id_value_object_test.go` | test |
| `runAll/src/domain/service_log_file_sink.go` | port |
| `runAll/src/infrastructure/file_service_log_sink.go` | adapter |
| `runAll/src/infrastructure/file_service_log_sink_test.go` | test |
| `runAll/src/infrastructure/tee_service_log_repository.go` | composite |
| `runAll/src/infrastructure/tee_service_log_repository_test.go` | test |
