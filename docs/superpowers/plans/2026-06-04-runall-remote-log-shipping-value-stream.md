# 价值流：runAll 本机 Promtail → 远程 Loki

> 设计：`docs/superpowers/specs/2026-06-04-runall-remote-log-shipping-design.md`

## Related Value Streams

- **platform-centralized-logging**（`value-stream.yaml`）：tee、Promtail pipeline、Grafana 仪表盘已 active；本需求修复远程拆分后的采集断点。
- **runall-remote-docker-sync**（planned）：无冲突；AiMonitor 仍在远程，仅 Promtail 迁至 Mac。

## Increments

| # | 名称 | 价值 | 验收 |
|---|------|------|------|
| 1 | local-promtail-compose | Mac 可 `up` Promtail，push 至 INFRA_HOST Loki | `test_promtail_local_push_config.py` |
| 2 | remote-compose-no-promtail | 远程栈不再空 tail | compose 无 promtail 服务 |
| 3 | runall-observability-api | UI/API 标明采集模式 | `/api/observability` 含 `loki_push_url` |
| 4 | clear-all-local-positions | 清空含本机 promtail volume | reset 脚本 + clear-all |

## YAML 更新

见根目录 `value-stream.yaml` → `platform-centralized-logging.steps` 新增 `increment-remote-promtail-push`。
