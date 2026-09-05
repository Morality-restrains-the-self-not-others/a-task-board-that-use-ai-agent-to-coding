# 实施计划：runAll 本机 Promtail → 远程 Loki

## Task 1: promtail-local + compose overlay
- [ ] `AiMonitor/promtail/promtail-local.yaml`（`${LOKI_PUSH_URL}` + expand-env）
- [ ] `AiMonitor/docker-compose.promtail-local.yaml`
- [ ] `AiMonitor/scripts/test_promtail_local_push_config.py`

## Task 2: 远程 compose 移除 promtail
- [ ] `AiMonitor/docker-compose.yaml` 删除 promtail
- [ ] `reset_observability_storage.sh` 不再启停远程 promtail
- [ ] 更新 `test_reset_observability_storage.py`

## Task 3: 本机启停脚本
- [ ] `scripts/runall-local-promtail.sh`（up/down/status/reset）
- [ ] `AiMonitor/scripts/reset_local_promtail.sh`

## Task 4: runAll API
- [ ] `Observability` 配置 + `LokiPushURL` 域函数
- [ ] `/api/observability` 字段
- [ ] `clear-all` 调用 local promtail reset

## Task 5: 文档与 value-stream.yaml
- [ ] `conf/runAll.yaml` observability 注释
- [ ] `value-stream.yaml` increment
- [ ] `AiMonitor/run.sh` 提示 + conf 路径

## Verify
```bash
cd runAll && go test ./...
python3 AiMonitor/scripts/test_promtail_local_push_config.py
python3 AiMonitor/scripts/test_reset_observability_storage.py
```
