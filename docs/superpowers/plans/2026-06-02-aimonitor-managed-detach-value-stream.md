# 价值流：AiMonitor managed detach 修复

**设计**: `docs/superpowers/specs/2026-06-02-aimonitor-managed-detach-design.md`  
**日期**: 2026-06-02

## 增量

单增量 **increment1-aimonitor-startup** — 恢复 runAll 远程/本地 `ai-monitor` 服务可启动。

| 步骤 | 名称 | 验证 |
|------|------|------|
| 1 | 修改 `AiMonitor/run.sh` managed → `compose up -d` | 脚本 diff 审查 |
| 2 | 远程 sync + `stack up aimonitor` | exit 0，< 30s |
| 3 | Grafana 健康探活 | `GET /api/health` → 200 |
| 4 | runAll Go 测试 | `go test ./src/...` |

## 现有价值流关系

**扩展** platform-observability 运行时能力，不新增 `value-stream.yaml` stream。

相关字段（只读运行时，无 schema 变更）：

- `ai-monitor.loki.*`
- `ai-monitor.grafana.*`
- `ai-monitor.runtime.grafana_trace_deep_link`

## YAML 变更

**无** — 纯脚本行为修复。
