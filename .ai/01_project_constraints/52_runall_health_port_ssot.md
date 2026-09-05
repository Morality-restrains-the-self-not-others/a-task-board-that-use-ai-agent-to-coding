# runAll 健康检查端口必须等于进程监听 SSOT（元规则）

## 基本信息

- 版本：1.0.0
- 创建日期：2026-08-16
- 维护者：Trae AI 团队
- 优先级：高（禁止忽略）
- 约束索引：`00_project_constraints.md` 第 47 条
- Cursor：`.cursor/rules/runall-health-port-ssot.mdc`（alwaysApply）
- ADR：`docs/adr/0012-runall-health-port-ssot.md`
- 门禁：`db/scripts/ci/check_task_events_runall_health_ports.py`
- 自测：`db/scripts/ci/test_check_task_events_runall_health_ports.py`
- 案例：`.ai/09_failure_experience/02_runtime_errors/94_task_events_fanout_health_port_mismatch.md`

## 背景

2026-08-16 精准编译重启失败：`task-events-task-status-changed-2-fanout-work-panel-sse` 监听 **18048**，`conf/runAll.yaml` 的 `health_check` 被从相邻服务拷成 **18056**。runAll 对空端口探活 60s → `READINESS_TIMEOUT` / `connection refused`。进程是否起来与探针端口无关。

根因是 **探活地址手写、且从兄弟条目复制**，与监听端口 SSOT 脱节。

## 核心原则

**runAll 健康检查探活的 host:port 必须等于该进程实际 listen 的 SSOT。禁止从相邻服务复制 `health_check.url` 后只改 `name`。**

## 端口 SSOT（按服务类型）

| 服务类型 | 监听端口 SSOT | runAll 探活 |
|----------|---------------|-------------|
| 平台服务（task-auth 等） | `conf/<area>/<app>/config.yaml` 的 `port` / `httpPort` | `conf_app` + `health_path`（禁止再写一份 `url` 端口） |
| **task-events intent 消费者** | `conf/events/domain-events/<event>/config.yaml` → `intents.<intent>.port` | `health_check.url` / `liveness_url` 的端口 **必须等于** 该 `port`；`intent_registry.go` 的 `Port` 必须相同 |
| 基础设施（显式 url） | 对应 `conf/infra/**/config.yaml` 或 compose 发布端口 | `${INFRA_HOST}:<同一端口>` |

Prometheus `AiMonitor/prometheus/file_sd/runall-health-targets.json` 的 task-events target **必须**与同一 SSOT 一致。

## 强制要求

1. **新增/修改 task-events intent** 时，先写 domain-events YAML 的 `port` + `groupId`，再填 `intent_registry.go`，再写 `conf/runAll.yaml` 的 health URL；三处端口相同。
2. **禁止**复制相邻 `task-events-*` 的 `health_check` 块后只改 `name` / `start_command`。
3. **同一 health 端口不得被两个 runAll 服务共用**（否则后启动者会把先启动者探活成假 healthy）。
4. **新的 domain-events `groupId` 必须有对应 runAll 条目**。存量未编入的 9 个 intent 仅允许留在门禁 `LEGACY_UNMANAGED_GROUP_IDS`（只减不增）。
5. `health_check.url` 与 `liveness_url` 端口必须相同。

## 新增 task-events intent 检查清单

- [ ] `conf/events/domain-events/<event>/config.yaml`：`intents.<intent>.port` + `groupId`
- [ ] `taskEvents/config/intent_registry.go`：`Port` 与 YAML 相同
- [ ] `conf/runAll.yaml`：`name` = `groupId`；health/liveness 使用该 `port`
- [ ] `AiMonitor/prometheus/file_sd/runall-health-targets.json`：同一端口
- [ ] `python3 db/scripts/ci/check_task_events_runall_health_ports.py` 通过

## 验收

```bash
python3 db/scripts/ci/test_check_task_events_runall_health_ports.py
python3 db/scripts/ci/check_task_events_runall_health_ports.py
```

## 与其他规则的关系

| 规则 | 关系 |
|------|------|
| `33_new_service_runall_registration.md` | 管「必须注册到 runAll」；本条管「探活端口 = 监听 SSOT」 |
| 第 42 条 conf SSOT | 监听端口仍只改 conf；runAll 只消费 |
| 第 24 条 listen `0.0.0.0` | 探活打 `${INFRA_HOST}`，进程仍绑全网卡 |

## 变更日志

- 2026-08-16：1.0.0 — 首版；与 ADR-0012、FE-20260816-EVENTS-FANOUT-HEALTH-PORT 同步。
