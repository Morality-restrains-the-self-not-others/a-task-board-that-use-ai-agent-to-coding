# 设计文档：移除 runAll 远程同步，统一 ${INFRA_HOST} 变量

日期: 2026-06-05
状态: 已批准

## 1. 背景

runAll 当前有两套模板变量和远程同步机制：

- **`${INFRA_HOST}`** — 从 `conf/infra/docker-infra/config.yaml` 单一来源解析，用于基础设施服务健康检查和 observability URL
- **`${HOST}`** — 从第一个 `conf_app` 服务的 host 字段解析，用于无 `conf_app` 的业务服务
- **远程同步** — `scripts/runall-remote-docker.sh` 通过 SSH + rsync 将应用/配置推送到远程 CPU (`10.2.150.89`)

两个变量当前都解析为 `10.2.150.89`（远程 CPU），机制冗余。

## 2. 目标

1. **移除远程同步** — 删除 `SSH + rsync` 脚本和 Go 代码，基础设施服务改为本地 Docker Compose（通过 `run.sh`）
2. **统一模板变量** — `${HOST}` 和 `${INFRA_HOST}` 合并为 `${INFRA_HOST}`，消除双变量混淆
3. **每服务自解析** — `${INFRA_HOST}` 从每个服务自身的 `conf/<app>/config.yaml` 的 `host` 字段解析
4. **拆分基础设施配置** — `conf/infra/docker-infra/config.yaml` 拆为各服务独立配置文件

## 3. 架构变更

```
之前:
  runAll → runall-remote-docker.sh → SSH → zcpu(10.2.150.89) → run.sh start
  ${INFRA_HOST} → conf/infra/docker-infra/config.yaml (单一来源)
  ${HOST}       → 第一个 conf_app 的 host

之后:
  runAll → dockerInfra/redis/run.sh start (本地直接执行)
  ${INFRA_HOST} → 每个服务自身的 conf/<app>/config.yaml → host 字段
  ${HOST} 已移除，统一为 ${INFRA_HOST}
```

## 4. 删除清单

### 4.1 脚本
- `scripts/runall-remote-docker.sh` — 整文件删除（SSH + rsync + 远程 docker compose）

### 4.2 Go 源文件
- `runAll/src/remote_docker.go` — 整文件删除
  - `RemoteDocker` 结构体、`RemoteDockerSync` 结构体
  - `serviceSyncPathOverrides` 映射
  - `InfraHost()`, `PortainerURL()`
  - `resolveInfraHostTemplates()`, `applyInfraHostReplacer()`, `applyInfraHostToHealthCheck()`, `replaceInfraHostLiteral()`
  - `RemoteSyncEnabled()`, `SyncRemoteDocker()`, `SyncServiceToRemote()`, `ServiceSyncRelativePaths()`
  - `shellQuote()`
- `runAll/src/remote_docker_test.go` — 整文件删除

### 4.3 配置文件
- `conf/infra/docker-infra/config.yaml` — 删除（拆分为各服务独立配置）

## 5. 新增清单

### 5.1 基础设施配置（各服务独立）

| 文件 | 内容 |
|------|------|
| `conf/infra/redis/config.yaml` | `host: 127.0.0.1` + `port: 6379` |
| `conf/infra/kafka/config.yaml` | `host: 127.0.0.1` + `port: 18080` (kafkaUiPort) |
| `conf/infra/portainer/config.yaml` | `host: 127.0.0.1` + `port: 9000` |
| `conf/infra/ai-monitor/config.yaml` | `host: 127.0.0.1` + `port: 3000` (grafana) |

`conf/infra/git-service/config.yaml` 已存在，保持不变。

## 6. 修改清单

### 6.1 `conf/runAll.yaml`

```yaml
# 删除整个 remote_docker 块:
#   remote_docker:
#     ssh_host: zcpu
#     context: zcpu-remote
#     workspace: ~/gitClone/ramDisk/ram-mount
#     sync:
#       auto_before_start: true

# observability URL: ${INFRA_HOST} 保持，ai-monitor conf_app 提供解析值
observability:
  grafana_url: "http://${INFRA_HOST}:3000"
  loki_url: "http://${INFRA_HOST}:3100"

# 基础设施服务：加 conf_app，改 start/stop 为本地 run.sh
groups:
  - name: infrastructure
    services:
      - name: docker-redis
        conf_app: "infra/redis"
        start_command: "bash dockerInfra/redis/run.sh start"
        stop_command: "bash dockerInfra/redis/run.sh stop"
        launch_mode: detach
        health_check:
          tcp: "${INFRA_HOST}:6379"
          timeout: 120
          retries: 30
          backoff: {initial: 1.0, max: 8.0, multiplier: 2.0}
        on_failure: exit

      - name: docker-kafka
        conf_app: "infra/kafka"
        start_command: "bash dockerInfra/kafka/run.sh start"
        stop_command: "bash dockerInfra/kafka/run.sh stop"
        launch_mode: detach
        health_check:
          url: "http://${INFRA_HOST}:18080"
          timeout: 180
          retries: 40
          backoff: {initial: 2.0, max: 16.0, multiplier: 2.0}
        on_failure: exit

      - name: docker-portainer
        conf_app: "infra/portainer"
        start_command: "bash dockerInfra/portainer/run.sh start"
        stop_command: "bash dockerInfra/portainer/run.sh stop"
        launch_mode: detach
        health_check:
          url: "http://${INFRA_HOST}:9000/api/status"
          timeout: 60
          retries: 20
          backoff: {initial: 1.0, max: 8.0, multiplier: 2.0}
        on_failure: skip

      - name: ai-monitor
        conf_app: "infra/ai-monitor"
        start_command: "bash AiMonitor/run.sh start"
        stop_command: "bash AiMonitor/run.sh stop"
        launch_mode: detach
        depends_on: [docker-redis]
        health_check:
          url: "http://${INFRA_HOST}:3000/api/health"
          timeout: 180
          retries: 36
          backoff: {initial: 2.0, max: 16.0, multiplier: 2.0}
        on_failure: skip

      - name: git-service
        conf_app: "infra/git-service"
        start_command: "bash gitService/run.sh start"
        stop_command: "bash gitService/run.sh stop"
        launch_mode: detach
        depends_on: [docker-redis]
        health_check:
          url: "http://${INFRA_HOST}:8012/users/sign_in"
          timeout: 900
          retries: 60
          backoff: {initial: 3.0, max: 30.0, multiplier: 2.0}
        on_failure: skip

  # 其他服务: ${HOST} → ${INFRA_HOST}
  - name: platform
    services:
      # ... task-auth, saas-backend 等已有 conf_app 的服务不变

      # git-oauth: ${HOST} → ${INFRA_HOST}
      - name: git-oauth
        health_check:
          url: "http://${INFRA_HOST}:8002/api/health/"

      # task-events-*: ${HOST} → ${INFRA_HOST}
      - name: task-events-billing-transaction-created
        health_check:
          url: "http://${INFRA_HOST}:18020/api/health/ready"
          liveness_url: "http://${INFRA_HOST}:18020/api/health/"
      # ... 其余 15 个 task-events-* 同理

      # value-stream: ${HOST} → ${INFRA_HOST}
      - name: value-stream
        health_check:
          url: "http://${INFRA_HOST}:9998"

      # OTEL env: ${INFRA_HOST} → ${INFRA_HOST} (变量名不变，值来源改变)
      - name: task-auth
        env:
          OTEL_EXPORTER_OTLP_ENDPOINT: "http://${INFRA_HOST}:4317"
      # ... 其余服务同理
```

**关键变更总结：**
- `remote_docker` 块删除
- 基础设施服务添加 `conf_app` 字段
- 所有 `${HOST}` → `${INFRA_HOST}`
- `start_command`/`stop_command` 从 `scripts/runall-remote-docker.sh` 改为本地 `run.sh`
- 健康检查 IP 保持 `${INFRA_HOST}` 占位符（值由 conf_app 按服务解析）

### 6.2 Go 源代码

#### `runAll/src/config.go`
- 从 `Config` 结构体移除 `RemoteDocker RemoteDocker` 字段
- 从 `fillDefaults()` 移除 `resolveInfraHostTemplates()` 调用
- 从 `LoadConfig` 末尾移除 `resolveInfraHostTemplates()` 调用
- 重写变量解析逻辑（在 `resolveConfApps` 中统一处理）：
  - 移除 `${HOST}` 替换代码块
  - 对所有服务（含 `Env`、`HealthCheck`）和顶层 `Observability` 统一进行 `${INFRA_HOST}` 替换
  - 解析规则：
    - 有 `conf_app` 的服务 → 用自身的 `host` 替换该服务的 `${INFRA_HOST}`
    - 无 `conf_app` 的服务 → 用首个 conf_app 的 `host` 替换（兜底）
    - 顶层 `Observability.grafana_url` / `Observability.loki_url` → 用 ai-monitor 的 `host` 或首个 conf_app host 替换

#### `runAll/src/conf_sync.go`
- 移除 `SyncConfReplicaToRemote()` 函数
- 移除 `confReplicaRemoteSyncLabel` 常量
- 保留 `SyncMonorepoConf()` 和 `confSyncLifecycleLabel`

#### `runAll/src/ui.go`
- 移除 API 端点注册：`/api/remote-docker/sync`, `/api/conf/sync-remote`
- 移除 `handleRemoteDockerSyncAction()`, `handleConfReplicaRemoteSyncAction()`
- 从 observability API 移除 `infra_host` 和 `portainer_url` 字段
- 从状态 payload 移除 `syncable` 字段和 `ManagementURL`
- 保留 `/api/conf/sync`（本地配置生成）

#### `runAll/src/runner.go`
- 移除 `RemoteSyncEnabled()` 条件分支调用
- 移除远程同步相关的调用链

#### `runAll/src/lifecycle_exec.go`
- 从 `buildServiceEnv()` 移除 `RUNALL_INFRA_HOST` 环境变量设置（由 `resolveConfApps` 统一处理）

### 6.3 辅助脚本

| 文件 | 变更 |
|------|------|
| `scripts/docker-env.sh` | 移除 runAll 远程编排提示（第 106 行附近），保留 SSH 隧道/端口定义 |
| `scripts/docker-tunnel-remote.sh` | 移除 "runAll health checks use 127.0.0.1:6379" 注释 |
| `scripts/docker-status.sh` | 移除 "local runAll access to remote container ports" 注释 |
| `scripts/docker-setup.sh` | 移除 runAll build/run 示例中远程相关说明 |
| `scripts/docker-desktop-helper.sh` | 移除 runAll 远程堆栈引用 |

### 6.4 监控配置

| 文件 | 变更 |
|------|------|
| `AiMonitor/prometheus/file_sd/runall-health-targets.json` | 目标 IP 更新（远程 IP → 本地 IP），移除不再需要的远程目标 |
| `AiMonitor/scripts/generate_prometheus_from_runall.py` | 移除远程 IP 生成逻辑 |

### 6.5 测试

| 文件 | 变更 |
|------|------|
| `runAll/src/config_test.go` | 移除 `TestResolveConfApps_InfraHostTemplateResolution`（依赖 `remote_docker.host`）；更新 `${HOST}` → `${INFRA_HOST}` 相关断言 |
| `runAll/src/ui_test.go` | 移除 `TestAPIObservability_IncludesPortainerURL`、`TestAPIRemoteDockerSyncService_*` 测试；更新变量名断言 |
| `runAll/src/remote_docker_test.go` | 整文件删除 |
| `runAll/playwright/tests/` | 审查 runall-* E2E 测试，更新远程堆栈 UI 选择器和断言 |

## 7. 保留不变

| 组件 | 说明 |
|------|------|
| `scripts/conf-sync-all.sh` + `conf-sync.py` | 本地配置生成，继续使用 |
| `runAll/src/conf_sync.go` 的 `SyncMonorepoConf()` | 本地配置生成入口 |
| `scripts/runall-local-promtail.sh` | 日志推送（本就是本地 → 远程 Loki） |
| 16 个本地业务服务 | task-auth, saas-backend, taskFE, task-events-*, value-stream, git-oauth 等 |
| Web UI 本地服务面板 | 启停/日志/状态面板（远程面板移除） |
| `scripts/docker-env.sh` 核心功能 | SSH 隧道、Docker context、端口映射 |

## 8. 影响文件总览

```
删除:  4 文件
  scripts/runall-remote-docker.sh
  runAll/src/remote_docker.go
  runAll/src/remote_docker_test.go
  conf/infra/docker-infra/config.yaml

新增:  4 文件
  conf/infra/redis/config.yaml
  conf/infra/kafka/config.yaml
  conf/infra/portainer/config.yaml
  conf/infra/ai-monitor/config.yaml

修改: ~15 文件
  YAML:     conf/runAll.yaml
  Go:       config.go, conf_sync.go, runner.go, ui.go, lifecycle_exec.go
  测试:     config_test.go, ui_test.go, compose_lifecycle_test.go
  脚本:     docker-env.sh, docker-tunnel-remote.sh, docker-status.sh, docker-setup.sh, docker-desktop-helper.sh
  监控:     runall-health-targets.json, generate_prometheus_from_runall.py
  E2E:      runAll/playwright/tests/*.test.js
```

## 9. Value Stream 影响

查阅 `value-stream.yaml`，以下价值流受影响：

| 价值流 | 状态 | 影响 |
|--------|------|------|
| `runall-global-start-stop-all` | planned | docker-redis/kafka 字段从远程变为本地，测试文件需更新 |
| `runall-docker-infra-split` | planned | 测试文件 `compose_lifecycle_test.go` 可能受影响 |
| `runall-portainer-management-ui` | planned | Portainer URL 逻辑移除，`ui_test.go` 相关测试需更新 |
| `runall-explicit-lifecycle-commands` | planned | docker-kafka/redis 命令从远程脚本变为本地 run.sh |
| `runall-cascade-lifecycle` | planned | 级联启停逻辑不涉及远程同步，影响较小 |

这些 planned 状态的流将在 `/3-value-stream` 步骤中重新映射。

## 10. 领域概念清单（供 DDD 步骤使用）

- **Bounded Context**: 基础设施编排（Infrastructure Orchestration）— 从 runAll 中移除远程概念，基础设施服务与业务服务统一管理
- **Key Entities**: `ManagedService` — 无论本地还是远程，统一为"受管服务"
- **Candidate Aggregates**: `ServiceLifecycle` — 服务生命周期聚合根，不再区分本地/远程
- **Domain Events**: 无新增事件，移除 `RemoteSyncCompleted` 相关事件

## 11. 风险与缓解

| 风险 | 缓解 |
|------|------|
| conf_app 解析顺序：observability URL (顶层) 需从 ai-monitor 的 host 解析，但 observability 不属任何服务 | `resolveConfApps` 在遍历完所有服务后，用 ai-monitor 的 host 或首个 conf_app host 统一替换顶层 `Observability.grafana_url` / `Observability.loki_url` 中的 `${INFRA_HOST}` |
| `${INFRA_HOST}` 按服务解析：不同服务可能解析出不同 host | 当前所有 conf_app 的 host 均为同一值（本地/同主机），短期内无差异；若未来拆分到多主机，按服务解析是正确行为 |
| 无 conf_app 服务的兜底 host | 保留首个 conf_app host 作为兜底（现有逻辑迁移） |
| gitService 的 remote-compose-helper.sh 引用 | `gitService/run.sh` 当前 source `scripts/remote-compose-helper.sh`，需改为本地 compose 模式 |
| Playwright E2E 测试断裂 | 审查所有 runall-* 测试，更新选择器和断言 |
| docker-env.sh 被其他脚本 source | 只移除 runAll 引用，保留通用配置 |
