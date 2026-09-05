# 新建 Go 服务：必须同步注册到 runAll 编排

## 硬约束

**当创建新的 Go 服务（或任何需要 runAll 管理的后台进程）时，必须同步完成以下三项：**

### 1. `conf/runAll.yaml` — 添加服务条目

在对应的 `groups[].services[]` 中添加：

```yaml
- name: task-{name}
  conf_app: {conf-key}           # conf/{conf-key}/config.yaml 的端口配置
  build_command: "./build.sh"
  start_command: "./bin/{binary}"
  stop_command: "bash -c 'lsof -ti:{port} | xargs kill -9 2>/dev/null || true'"
  working_dir: {repo-dir}
  depends_on: [{dependencies}]   # 至少依赖 task-auth（鉴权）
  health_check:
    health_path: /api/health/
    timeout: 60
    retries: 15
  on_failure: skip
```

### 2. `build.sh` — 编译脚本

在服务仓库根目录创建 `build.sh`：

```bash
#!/usr/bin/env bash
set -euo pipefail
ROOT="$(cd "$(dirname "$0")" && pwd)"
cd "$ROOT"
mkdir -p bin
echo "==> Building {name} (./src -> bin/{binary})..."
go build -o bin/{binary} ./src
echo "==> Done: $ROOT/bin/{binary}"
```

### 3. `conf/{conf-key}/config.yaml` — 端口配置

```yaml
port: {port}
host: "0.0.0.0"
```

## 检查清单

创建新 Go 服务后，逐项确认：

- [ ] `conf/runAll.yaml` 已添加服务条目（含 build/start/stop/health_check）
- [ ] **健康检查端口来自监听 SSOT**，禁止从相邻服务复制 `health_check.url` 后只改 `name`（`.ai/01_project_constraints/52_runall_health_port_ssot.md`）
- [ ] 平台服务用 `conf_app` + `health_path`；task-events intent 的 `health_check.url` / `liveness_url` 端口必须等于 `conf/events/domain-events/<event>/config.yaml` 的 `intents.<intent>.port`，并同步 `intent_registry.go` 与 Prometheus file_sd
- [ ] 服务仓库根目录存在 `build.sh` 且可执行
- [ ] `conf/{conf-key}/config.yaml` 存在且端口不冲突
- [ ] 服务依赖 `depends_on: [task-auth]`（如需鉴权）或正确列出其他依赖
- [ ] **禁止**在业务 `main` 内嵌 `time.NewTicker` / sleep 扫表循环；周期工作须注册独立 `taskEvents` timer worker + 一次性 internal API（`.ai/01_project_constraints/51_no_service_internal_poll_loop.md`）
- [ ] 网关 `taskGateway/routes/routes.yaml` 已添加 upstream + routes
- [ ] `go build ./src` 通过
- [ ] migration 可执行（`go run ./src migrate` 或启动时自动 migrate）

## 动机

2026-07-25 创建 `taskReferral` 服务时遗漏了 runAll.yaml 注册，
导致新服务无法通过 runAll 一键启动。此规则防止同类遗漏。
