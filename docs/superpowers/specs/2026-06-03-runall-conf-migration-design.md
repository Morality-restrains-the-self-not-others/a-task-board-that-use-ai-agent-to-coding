# runAll + conf/ 配置统一设计

> 日期: 2026-06-03 | 状态: approved

## 背景

`runAll.yaml`（仓库根目录）和 `conf/<app>/config.yaml` 存在配置冗余：
- `runAll.yaml` 硬编码 IP 地址（`172.20.10.3` / `172.20.10.7`）
- `conf/<app>/config.yaml` 也有相同 IP
- IP 变更需两处同步，已出现 drift（conf/ 已更新为 10.2.150.89/10.2.150.119，runAll.yaml 未更新）
- `.ai.md` 规则文件已规定 conf 为权威来源，但 runAll 程序未实际读取 conf/

## 目标

1. runAll 程序读取 `conf/runAll.yaml`，并从 `conf/<app>/` 自动派生健康检查 URL
2. 消除 `runAll.yaml` 和 `conf/<app>/` 之间的冗余
3. 删除根目录 `runAll.yaml`，以 `conf/runAll.yaml` 为唯一配置入口

## 设计决策

| 决策 | 选择 |
|------|------|
| 服务与 conf/ 的关联方式 | **显式映射**：每个服务声明 `conf_app` 字段 |
| 健康检查 URL 构建 | **`health_path` + 自动拼接**：runAll 从 `conf/<app>/config.yaml` 读取 `host`:`port`，拼接 `http://{host}:{port}{health_path}` |
| 基础设施服务（Redis/Kafka等） | **保留显式 `url`/`tcp`**，不强制 conf_app 映射 |
| `remote_docker.host` | **自动读取** `conf/docker-infra/config.yaml`，不在 conf/runAll.yaml 中重复 |

## 架构

### conf/runAll.yaml 结构

```yaml
version: "1"

remote_docker:
  ssh_host: zcpu                        # SSH hostname（conf/ 中没有，保留）
  context: cpu-remote
  workspace: ~/gitClone/ramDisk/ram-mount
  sync:
    auto_before_start: true
  # host 和 portainer_port 自动从 conf/docker-infra/config.yaml 读取

logging:
  file_root: /tmp/runall-logs

observability:
  grafana_url: "http://${INFRA_HOST}:3000"
  loki_url: "http://${INFRA_HOST}:3100"
  trace_dashboard_uid: distributed-trace-view

groups:
  - name: infrastructure
    services:
      # 基础设施 — 保留显式 url/tcp（无 conf_app）
      - name: docker-redis
        start_command: "bash scripts/runall-remote-docker.sh stack up redis"
        stop_command: "bash scripts/runall-remote-docker.sh stack down redis"
        launch_mode: detach
        health_check:
          tcp: "${INFRA_HOST}:6379"
          timeout: 120
          retries: 30
        on_failure: exit
      # ... 其余基础设施服务同理

  - name: platform
    services:
      # 平台服务 — 使用 conf_app + health_path
      - name: saas-backend
        conf_app: django
        start_command: "bash scripts/runall-saas-backend.sh start"
        stop_command: "bash scripts/runall-saas-backend.sh stop"
        working_dir: task2app
        depends_on: [task-auth, git-oauth, docker-redis, task-sse]
        health_check:
          health_path: /api/health/
          liveness_path: /api/live/
          timeout: 180
          retries: 35
        on_failure: skip
      # ... 其余平台服务同理
```

### 服务迁移映射

**使用 `conf_app` + `health_path`（12 个服务）：**

| 服务 | conf_app | health_path | liveness_path |
|------|----------|-------------|---------------|
| task-auth | task-auth | /api/health/ | — |
| task-bill | task-bill | /api/health/ | — |
| ai-provider | ai-provider | /api/health/ | — |
| saas-backend | django | /api/health/ | /api/live/ |
| task-gateway | task-gateway | /api/health/ | — |
| task-agent-support | task-agent-support | /api/health/ | — |
| task-ai-endpoint | task-ai-endpoint | /api/health/ | — |
| task-container-gateway | task-container-gateway | /api/health/ | — |
| task-sse | task-sse | /health | — |
| taskFE | vue | /health | — |
| go-run-container | mock-run-container | /health | — |
| go-relay | relay-to-trae | /health | — |

**保留显式 `url`/`tcp`（22 个服务）：**

| 服务 | 原因 |
|------|------|
| docker-redis | TCP 探活 6379，无对应 conf/ |
| docker-kafka | Kafka UI 18080，无独立 conf/ |
| docker-portainer | Portainer 9000，无 conf/ |
| ai-monitor | Grafana 3000，无 conf/ |
| git-service | 远程 GitLab，探活 URL 为 `/users/sign_in` |
| git-oauth | conf/git-oauth/ 无 config.yaml（仅有生成片段） |
| task-events-* (16个) | domain-events 无统一 host/port 入口 |
| value-stream | 无对应 conf/ |

### 加载流程

```
LoadConfig("conf/runAll.yaml")
  ├── 1. YAML 解析
  ├── 2. fillDefaults()
  ├── 3. resolveInfraHost()          ← 读取 conf/docker-infra/config.yaml 的 host
  ├── 4. resolveConfApps()           ← 新增：遍历有 conf_app 的服务
  │     ├── 读取 conf/<conf_app>/config.yaml
  │     ├── 取 host 和 port
  │     ├── 拼接 url = "http://{host}:{port}{health_path}"
  │     └── 拼接 liveness_url = "http://{host}:{port}{liveness_path}"（如有）
  ├── 5. resolveInfraHostTemplates() ← ${INFRA_HOST} → 步骤3 读取的 host
  ├── 6. normalizeServiceLifecycleCommands()
  └── 7. validate()                  ← 增强：conf_app + url 互斥校验
```

### 校验增强

- `conf_app` 和 `health_check.url` 同时指定 → **报错**（互斥）
- `conf_app` 指定但 `health_path` 为空 → **报错**
- `conf_app` 指向的 `conf/<app>/config.yaml` 不存在 → **报错**
- 无 `conf_app` 且无 `url`/`tcp` → 保持现有报错

### Go 结构体变更

```go
type Service struct {
    // ... 现有字段保持不变 ...
    ConfApp      string `yaml:"conf_app"`       // 新增：指向 conf/<app>/
    HealthPath   string `yaml:"health_path"`    // 新增：健康检查路径
    LivenessPath string `yaml:"liveness_path"`  // 新增：启动探针路径
}
```

在 `HealthCheck` 解析后，由 `resolveConfApps()` 填充 `URL` 和 `LivenessURL`。

### 默认配置路径

- `main.go` 中 `--config` flag 默认值从 `"config.yaml"` 改为 `"conf/runAll.yaml"`
- `--config` flag 仍可覆盖

### 与现有系统的关系

- `SyncMonorepoConf`、`SyncConfReplicaToRemote` 无需改动 — 它们同步整个 `conf/` 目录，`conf/runAll.yaml` 自然被覆盖
- `scripts/conf-sync-all.sh` 无需改动
- `runAll/config.yaml`（runAll/ 内部示例）可考虑同步更新或标记 deprecated

## 实施步骤

1. 新建 `conf/runAll.yaml` — 从 `runAll.yaml` 迁移，添加 conf_app/health_path
2. 修改 `runAll/src/config.go` — 新增 `resolveInfraHost()` 和 `resolveConfApps()`
3. 修改 `runAll/src/main.go` — 默认配置路径改为 `conf/runAll.yaml`
4. 修改 `runAll/src/remote_docker.go` — 移除对 `remote_docker.host` YAML 字段的依赖
5. 添加/更新 Go 测试
6. 更新 `runAll/run.sh`、`runAll/build.sh` 中的默认路径
7. 运行 `test.sh` 验证
8. 删除根目录 `runAll.yaml`
9. 更新引用 `runAll.yaml` 的文档和 `.ai.md`

## 价值流影响

- 影响 `conf/` 配置管理流 — `conf/runAll.yaml` 成为编排配置的 SSOT
- 影响开发启动流程 — 启动命令从 `--config ../runAll.yaml` 变为 `--config conf/runAll.yaml`（或使用默认）
- 无新增领域概念，不影响现有 value stream steps
