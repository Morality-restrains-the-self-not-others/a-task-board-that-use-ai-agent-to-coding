# domain-events 配置 Companion

## 核心约束

**每个 intent 的监听端口只在本目录 `<event>/config.yaml` 的 `intents.<intent>.port` 定义。**

新增/改端口时必须同步：

1. `taskEvents/config/intent_registry.go` 的 `Port`
2. `conf/runAll.yaml` 对应 `task-events-*` 的 `health_check.url` / `liveness_url`
3. `AiMonitor/prometheus/file_sd/runall-health-targets.json`

**禁止**只改 YAML 或只改 runAll。禁止从兄弟 intent 复制 runAll `health_check` 块后漏改端口。

细则：`.ai/01_project_constraints/52_runall_health_port_ssot.md`（约束索引第 47 条）。  
验收：`python3 db/scripts/ci/check_task_events_runall_health_ports.py`
