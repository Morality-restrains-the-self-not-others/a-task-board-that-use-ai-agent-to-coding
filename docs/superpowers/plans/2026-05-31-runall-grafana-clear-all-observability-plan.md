# Implementation Plan: runAll 一键清空 Grafana 可观测数据

> Design: `docs/superpowers/specs/2026-05-31-runall-grafana-clear-all-logs-design.md`
> Value stream: `docs/superpowers/plans/2026-05-31-runall-grafana-clear-all-observability-value-stream.md`

## Task 1: FileServiceLogSink Truncate
- [ ] Add `TruncateService` / `TruncateAll` to `file_service_log_sink.go`
- [ ] Tests in `file_service_log_sink_test.go`

## Task 2: TeeServiceLogRepository Clear syncs file
- [ ] `Clear()` calls `TruncateService` when file sink supports it
- [ ] Test in `tee_service_log_repository_test.go`

## Task 3: Domain ObservabilityStackReset
- [ ] `domain/observability_stack_reset.go` — service + result types
- [ ] `domain/observability_storage_resetter.go` — interface
- [ ] Unit tests

## Task 4: Infrastructure storage reset
- [ ] `infrastructure/observability_storage_reset.go` — script executor
- [ ] `AiMonitor/scripts/reset_observability_storage.sh`
- [ ] `AiMonitor/scripts/test_reset_observability_storage.py`

## Task 5: API + UI
- [ ] `POST /api/observability/clear-all` in `ui.go`
- [ ] Button in `status.html`
- [ ] `ui_test.go` coverage

## Task 6: value-stream.yaml
- [ ] Append `increment4-runall-clear-all-observability` step

## Verification
```bash
cd runAll && ./test.sh
cd AiMonitor && python3 scripts/test_reset_observability_storage.py
```
