# Value Stream: 移除远程同步，统一 INFRA_HOST

> Derived from design: `docs/superpowers/specs/2026-06-05-remove-remote-sync-unify-infra-host-design.md`

## Value Summary

开发者通过 runAll 本地直接管理所有基础设施服务（Redis/Kafka/Portainer/AiMonitor/GitService），不再依赖 SSH+rsync 远程同步，模板变量统一为 `${INFRA_HOST}`，每个服务从自身 `conf/<app>/config.yaml` 解析 host。

## Related Value Streams

- **runall-docker-infra-split** (`2026-05-31-runall-docker-infra-split-value-stream.md`): **modification** — 基础设施服务从远程 Docker 改为本地 Docker Compose，start/stop 命令从 `runall-remote-docker.sh` 改为 `dockerInfra/*/run.sh`
- **runall-portainer-remote-docker-panel** (`2026-06-02-runall-portainer-remote-docker-panel-value-stream.md`): **supersedes** — Portainer 远程面板移除，Portainer 变为本地服务，通过 conf_app 管理
- **docker-infra-conf-sync** (`2026-06-02-docker-infra-conf-sync-value-stream.md`): **modification** — `conf/infra/docker-infra/config.yaml` 拆分为各服务独立配置，远程 conf sync 移除
- **runall-conf-migration** (`2026-06-03-runall-conf-migration-value-stream.md`): **extension** — `${HOST}` 全部迁移为 `${INFRA_HOST}`，变量解析逻辑统一到 `resolveConfApps`
- **conf-directory-restructure** (`2026-06-05-conf-directory-restructure-value-stream.md`): **extension** — 新增 `conf/infra/{redis,kafka,portainer,ai-monitor}/config.yaml`

## End-to-End Flow

[开发者打开 runAll UI] → [启动 docker-redis] → [runAll 从 conf/infra/redis/config.yaml 解析 host=127.0.0.1] → [执行 dockerInfra/redis/run.sh start] → [TCP 127.0.0.1:6379 就绪] → [级联启动 platform 服务]

## Value Increments

### Increment 1: 配置拆分 — 基础设施各服务独立 conf_app（Thin Slice）
**Value to user:** 每个基础设施服务拥有独立的 `conf/<app>/config.yaml`，可单独修改 host/port
**Scope:**
- 新增 `conf/infra/redis/config.yaml`
- 新增 `conf/infra/kafka/config.yaml`
- 新增 `conf/infra/portainer/config.yaml`
- 新增 `conf/infra/ai-monitor/config.yaml`
- 从 `conf/infra/docker-infra/config.yaml` 迁移字段（最后删除）
- `conf/runAll.yaml` 中基础设施服务添加 `conf_app` 字段
**Depends on:** nothing

### Increment 2: 统一模板变量 — ${HOST} → ${INFRA_HOST}
**Value to user:** 不再有双变量混淆，所有服务统一使用 `${INFRA_HOST}`
**Scope:**
- `conf/runAll.yaml` 中所有 `${HOST}` → `${INFRA_HOST}` (30+ 处)
- `runAll/src/config.go` 中 `resolveConfApps` 移除 `${HOST}` 替换逻辑，改为 `${INFRA_HOST}` 统一替换
- 解析规则：有 conf_app → 自身 host；无 conf_app → 首个 conf_app host（兜底）
- 顶层 Observability URL 用 ai-monitor 的 host
**Depends on:** Increment 1（基础设施服务已有 conf_app）

### Increment 3: 删除远程同步 — 脚本 + Go 代码
**Value to user:** 架构简化，无 SSH/rsync 依赖，本地直接管理 Docker
**Scope:**
- 删除 `scripts/runall-remote-docker.sh`
- 删除 `runAll/src/remote_docker.go`
- 删除 `runAll/src/remote_docker_test.go`
- 移除 `Config.RemoteDocker` 字段
- 修改基础设施服务 start/stop 为本地 `run.sh`
- 修改健康检查目标为 `127.0.0.1`（通过 conf_app host 字段）
**Depends on:** Increment 2（变量已统一，引用链已清理）

### Increment 4: UI + API 清理
**Value to user:** runAll UI 不再展示已废弃的远程同步按钮和 Portainer 面板
**Scope:**
- 移除 `/api/remote-docker/sync`、`/api/conf/sync-remote` 端点
- 移除 observability API 中 `infra_host`、`portainer_url` 字段
- 移除状态 payload 中 `syncable`、`management_url` 字段
- 移除 `conf_sync.go` 中 `SyncConfReplicaToRemote()`
- 移除 `lifecycle_exec.go` 中 `RUNALL_INFRA_HOST` 环境变量
**Depends on:** Increment 3（Go 代码无编译错误）

### Increment 5: 辅助脚本 + 监控 + 测试清理
**Value to user:** 文档和监控与实际架构一致，测试覆盖新行为
**Scope:**
- 清理 `scripts/docker-env.sh` 等 5 个脚本中的 runAll 远程引用
- 更新 `AiMonitor/prometheus/file_sd/runall-health-targets.json` 目标 IP
- 更新 `AiMonitor/scripts/generate_prometheus_from_runall.py`
- 更新 `config_test.go`、`ui_test.go` — 移除远程相关测试，新增 conf_app 解析测试
- 审查 `runAll/playwright/tests/` E2E 测试
**Depends on:** Increment 4

### Increment 6: 端到端验证
**Value to user:** 确认所有基础设施服务可本地启动、级联正常、健康检查通过
**Scope:**
- `docker-redis` 本地启动 → TCP 6379 就绪
- `docker-kafka` 本地启动 → HTTP 18080 就绪
- `ai-monitor` 本地启动 → HTTP 3000 就绪
- platform 级联启动（依赖 docker-redis）
- Web UI 展示正确的本地服务面板
- `valueStream` Go 测试全绿
**Depends on:** Increment 5
