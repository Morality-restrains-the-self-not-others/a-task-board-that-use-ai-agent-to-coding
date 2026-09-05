# Grafana 分布式追踪运行手册

日期：2026-05-29  
关联设计：`docs/superpowers/specs/2026-05-28-centralized-logs-grafana-traceid-design.md`

---

## 1. 栈组成

| 组件 | 端口 | 职责 |
|------|------|------|
| **OTel Collector** | 4317（gRPC）、4318（HTTP） | 统一接收各服务 OTLP → 转发 Tempo |
| **Loki** | 3100 | 日志聚合 |
| **Promtail** | — | tail `RUNALL_LOG_ROOT/*.log` → Loki |
| **Tempo** | 3200（查询） | 分布式 trace span + Service Map 指标生成 |
| **Grafana** | 3000 | 统一 UI（瀑布图 / Service Map / 日志） |
| **Prometheus** | 9090 | 指标 + Tempo remote write 接收 |

---

## 2. 启动顺序

```bash
# 1. 可观测栈
cd AiMonitor && ./run.sh start

# 2. 全栈（tee 日志 + OTel 环境变量已写入 runAll.yaml）
cd runAll && ./bin/runAll --config ../runAll.yaml
```

`runAll.yaml` 已为以下服务注入 OTLP：

| 服务 | `OTEL_EXPORTER_OTLP_ENDPOINT` | 协议 |
|------|-------------------------------|------|
| saas-backend | `http://127.0.0.1:4318` | HTTP |
| task-auth、go-relay、task-events-* | `127.0.0.1:4317` | gRPC |
| onlineServiceJS（容器内） | `127.0.0.1:4317` | gRPC（`TRACE_ID` + HTTP span） |

所有 OTLP 流量经 **OTel Collector** 统一转发至 Tempo。

未设置 `OTEL_EXPORTER_OTLP_ENDPOINT` 时，应用照常运行，仅不上报 span。

---

## 3. 一键验收

```bash
cd AiMonitor && python3 scripts/verify_trace_stack.py
```

通过即表示 Tempo / Loki / Grafana 就绪，且 OTLP  ingest + trace 查询正常。

---

## 4. 日常排查

### 4.1 从 API 响应头取 trace

1. 浏览器 DevTools → Network → 任意 API → Response Headers → `X-Trace-Id`
2. 或 runAll UI → 服务日志 → **Grafana**（自动深链）

### 4.2 Grafana 仪表盘

| 仪表盘 | UID | 用途 |
|--------|-----|------|
| **Distributed Trace View** | `distributed-trace-view` | Tempo 瀑布图 + Loki 关联日志 |
| **Trace Log Journey** | `trace-log-journey` | 纯日志时间线 |

**变量说明：**

- `trace_id`：原始 `X-Trace-Id`（含 UUID 连字符），用于 Loki `|=`
- `tempo_trace_id`：32 位 hex，用于 Tempo 瀑布图；runAll 深链会自动填充

UUID 示例：`1cd1a1cc-e64d-4325-8b31-caabdd8aa74d` → Tempo 用 `1cd1a1cce64d43258b31caabdd8aa74d`（或日志 JSON 中的 `otel_trace_id`）

### 4.3 日志字段

```json
{
  "trace_id": "1cd1a1cc-e64d-4325-8b31-caabdd8aa74d",
  "otel_trace_id": "1cd1a1cce64d43258b31caabdd8aa74d",
  "service": "saas-backend",
  "msg": "..."
}
```

---

## 5. 与 Datadog 能力对照

| 能力 | 状态 |
|------|------|
| 按 traceId 查跨服务日志（Loki） | ✅ |
| Trace 瀑布图（Tempo） | ✅（需服务上报 OTLP） |
| Trace ↔ Log 关联 | ✅（`otel_trace_id` + Grafana 配置） |
| runAll 一键跳转 Grafana | ✅ |
| **Service Map** | ✅（Tempo metrics-generator → Prometheus） |
| taskEvents / onlineServiceJS OTel | ✅ |
| OTel Collector 统一路由 | ✅ |

---

## 6. 故障排查

| 现象 | 检查 |
|------|------|
| 瀑布图为空 | `echo $OTEL_EXPORTER_OTLP_ENDPOINT`；Tempo `curl -sf http://127.0.0.1:3200/ready` |
| 日志为空 | `RUNALL_LOG_ROOT` 是否有 `*.log`；Promtail 容器是否挂载该目录 |
| onlineServiceJS 不在 Loki | 日志经 go-relay 子进程转发；查 `go-relay.log` 是否含 `"service":"onlineServiceJS"`；重建 go-relay |
| 错误堆栈被切成多行 | go-relay `subprocessLogGrouper` 合并；确认已重建 go-relay |
| Tempo not ready | 启动后等待 ~15s（ingester warmup） |
| UUID 搜不到 Tempo | 使用 `tempo_trace_id` 变量或去掉连字符 |
