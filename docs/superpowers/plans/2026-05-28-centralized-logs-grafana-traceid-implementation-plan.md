# 集中日志 Grafana traceId — 实施计划

> **For agentic workers:** Increment 1 先交付；每步 TDD。Checkbox 跟踪进度。

**Goal:** 本地 runAll 运行时，AiMonitor 提供 Loki+Promtail+Grafana，runAll tee stdout 到 `RUNALL_LOG_ROOT`，Grafana 可按 traceId 查 saas-backend 日志。

**Architecture:** runAll `TeeServiceLogRepository` + `FileServiceLogSink`；AiMonitor docker-compose 增 loki/promtail；Grafana provisioning Loki datasource + Explore dashboard。

**Tech Stack:** Go (runAll), Docker Compose, Grafana Loki 3.x, Promtail, YAML

> 输入:
> - 设计: `docs/superpowers/specs/2026-05-28-centralized-logs-grafana-traceid-design.md`
> - 价值流: `docs/superpowers/plans/2026-05-28-centralized-logs-grafana-traceid-value-stream.md`
> - NFR: `docs/superpowers/plans/2026-05-28-centralized-logs-grafana-traceid-nfr-clarification.md`
> - DDD: `docs/superpowers/plans/2026-05-28-centralized-logs-grafana-traceid-ddd-domain-model.md`

---

## Increment 1: Loki MVP

### Task 1.1: runAll 领域 — TraceId VO + FileSink 端口

**Files:**
- Create: `runAll/src/domain/trace_id_value_object.go`
- Create: `runAll/src/domain/trace_id_value_object_test.go`
- Create: `runAll/src/domain/service_log_file_sink.go`

- [ ] **Step 1:** 写失败测试 `TestParseTraceId_*`
- [ ] **Step 2:** 实现 TraceId VO（对齐 http_trace 规则）
- [ ] **Step 3:** `go test ./src/domain/... -run TraceId`

### Task 1.2: FileServiceLogSink + TeeServiceLogRepository

**Files:**
- Create: `runAll/src/infrastructure/file_service_log_sink.go`
- Create: `runAll/src/infrastructure/file_service_log_sink_test.go`
- Create: `runAll/src/infrastructure/tee_service_log_repository.go`
- Create: `runAll/src/infrastructure/tee_service_log_repository_test.go`

- [ ] **Step 1:** 测试：AppendLine 写入 `{root}/{service}.log`，格式 `[ts] (stream) message`
- [ ] **Step 2:** 测试：Tee 同时写 memory + file；file 失败不 panic
- [ ] **Step 3:** `go test ./src/infrastructure/... -run 'FileService|TeeService'`

### Task 1.3: runAll 配置与 Runner 接线

**Files:**
- Modify: `runAll/src/config.go` — `Logging.FileRoot`
- Modify: `runAll.yaml` — `logging.file_root`
- Modify: `runAll/src/runner.go` — 初始化 Tee repo
- Modify: `runAll/src/main.go` 或 runner 构造（若需）

- [ ] **Step 1:** 测试：LoadConfig 解析 `logging.file_root`
- [ ] **Step 2:** env `RUNALL_LOG_ROOT` 覆盖 yaml
- [ ] **Step 3:** Runner 启动时 `MkdirAll` + Tee repo
- [ ] **Step 4:** `go test ./...`

### Task 1.4: AiMonitor Loki + Promtail + Grafana

**Files:**
- Modify: `AiMonitor/docker-compose.yaml`
- Create: `AiMonitor/loki/loki-config.yaml`
- Create: `AiMonitor/promtail/promtail.yaml`
- Modify: `AiMonitor/grafana/provisioning/datasources/prometheus.yaml` — 增 Loki
- Create: `AiMonitor/grafana/provisioning/dashboards/files/trace-log-explore.json`
- Modify: `AiMonitor/run.sh` — 打印 Loki URL、确保 log dir hint
- Create: `AiMonitor/scripts/validate_promtail_config.py`
- Create: `AiMonitor/scripts/test_promtail_config.py`

- [ ] **Step 1:** docker-compose 增 loki、promtail 服务
- [ ] **Step 2:** promtail pipeline：regex `[trace_id=([^]]+)]` + json trace_id
- [ ] **Step 3:** Grafana Loki datasource provisioning
- [ ] **Step 4:** pytest 校验 promtail yaml 含 scrape_configs
- [ ] **Step 5:** `.env.example` 增 `RUNALL_LOG_ROOT`

### Task 1.5: 文档与 value-stream

- [x] value-stream.yaml 已追加 `platform-centralized-logging`
- [ ] 设计 doc status → 已批准

---

## Increment 2（planned，本计划不执行）

- Go/Python JSON logger + X-Trace-Id 中间件
- Grafana Trace Log Journey 完整 dashboard
- runAll UI → Grafana deep link

---

## 验证命令

```bash
cd runAll && go test ./...
cd AiMonitor && python3 -m pytest scripts/test_promtail_config.py -q
# 手动：RUNALL_LOG_ROOT=/tmp/runall-logs ./bin/runAll --config ../runAll.yaml
# AiMonitor: ./run.sh start
# Grafana Explore: {trace_id="<from response header>"}
```
