# 设计文档：将 Promtail 纳入 runAll 统一编排

## 问题

日志流水线在 Promtail 环节断裂：runAll 管理的服务持续产出日志到 `/tmp/runall-logs/*.log`，但 Promtail 需要手动启动，
每次重启 runAll 后极易遗忘，导致 Grafana Trace Log Explorer 仪表板显示为空。

## 现有架构

```
┌─── Mac 本地 ────────────────────┐      ┌─── 远程服务器 183.250.1.132 ────┐
│  runAll                          │      │                                 │
│    ├─ docker-redis (通过脚本)     │      │  AiMonitor compose 栈            │
│    ├─ docker-kafka               │      │    ├─ Loki :3100                 │
│    ├─ ai-monitor (AiMonitor/run.sh)│    │    ├─ Grafana :3000              │
│    ├─ saas-backend               │      │    ├─ Prometheus :9090           │
│    ├─ go-relay                   │      │    ├─ Tempo :3200                │
│    ├─ ... (20+ 服务)              │      │    └─ OTEL Collector :4317/4318 │
│    └─ ❌ promtail-local 缺失      │      │                                 │
│                                  │      │                                 │
│  /tmp/runall-logs/*.log          │      │                                 │
│  (日志堆积，无人搬运)             │      │  Loki: 空数据                     │
└──────────────────────────────────┘      └─────────────────────────────────┘
```

关键约束：**Promtail 必须在 Mac 本地运行**，因为它 tail 本地日志文件。远程 Loki 无法访问 Mac 文件系统。

## 现状复盘

| 层面 | 已有的 | 缺失的 |
|------|--------|--------|
| runAll.yaml | `observability.local_promtail: true` 标记 | 未注册为 service，未被 Go 代码消费 |
| 启动脚本 | `runAll/scripts/runall-local-promtail.sh` (功能完整) | 手动 CLI，不在任何自动化流程中被调用 |
| 配置解析 | 脚本从 `conf/infra/docker-infra/config.yaml` 读 `host` | 该文件无顶层 `host`，解析必然失败 |
| 领域模型 | `ObservabilityStorageResetter.PromtailReset` 字段 | 只用于 reset 流程，不管理生命周期 |
| AiMonitor/run.sh | 提示用户手动启动 Promtail | 不自动启动 |

## 方案选择

### 方案 A：注册为 runAll infrastructure service（✅ 推荐）

将 `promtail-local` 加入 `conf/runAll.yaml` 的 `infrastructure` group，与 `docker-redis`、`ai-monitor` 等
同级管理。使用已有的 `runall-local-promtail.sh up/down/status` 作为 start/stop/health 命令。

**优点**：
- 与现有模式完全一致（Docker 容器通过 shell 脚本管理是 runAll 的既有模式）
- 自动获得依赖管理、健康检查、级联启停、TUI 可视化
- 最小改动，复用已有脚本

**缺点**：
- 需要 `desktop-linux` Docker context 可用

### 方案 B：runAll Go 代码自动启动

在 runAll 启动时，Go 代码读取 `local_promtail: true`，以子进程方式调用 `runall-local-promtail.sh up`。

**优点**：
- 用户无感知，零配置

**缺点**：
- Promtail 生命周期与 runAll 主进程耦合太紧（runAll 退出 → Promtail 也退出？）
- 无法独立控制 Promtail（若想单独重启 Promtail 需要重启 runAll）
- 需要修改 Go 代码，改动面比方案 A 大

### 方案 C：集成到 AiMonitor/run.sh 中

让 `AiMonitor/run.sh` 也负责启动本地 Promtail。

**优点**：
- 集中管理监控栈

**缺点**：
- AiMonitor 启动脚本运行在远程 Docker context，Promtail 需要本地 `desktop-linux` context
- 两个 context 切换容易出错
- 违反关注点分离：AiMonitor 不应感知本地日志搬运

### 结论：选择方案 A

方案 A 与现有架构最一致。`docker-redis`、`docker-kafka`、`ai-monitor` 已经通过 shell 脚本
作为 runAll service 管理。Promtail 加入后，`infrastructure` group 的依赖关系变为：

```
docker-redis ──→ ai-monitor ──→ promtail-local
                              └─→ (其他服务产出日志后，Promtail 推送到 Loki)
```

## 详细设计

### 1. 修改 `conf/runAll.yaml`

在 `infrastructure` group 新增 `promtail-local` service：

```yaml
      - name: promtail-local
        conf_app: infra/ai-monitor          # 复用 ai-monitor 配置（获取 host）
        start_command: "bash runAll/scripts/runall-local-promtail.sh up"
        stop_command: "bash runAll/scripts/runall-local-promtail.sh down"
        working_dir: .
        depends_on: [ai-monitor]             # Loki 必须先就绪
        health_check:
          url: "http://${INFRA_HOST}:3100/ready"
          timeout: 60
          retries: 10
        on_failure: skip
```

### 2. 修复 `runAll/scripts/runall-local-promtail.sh` 配置解析 bug

**Bug**：第 34 行从 `conf/infra/docker-infra/config.yaml` 读 `host`，该文件无顶层 `host` 字段。

**修复**：改为从 `conf/infra/ai-monitor/config.yaml` 读取（已有 `host: 127.0.0.1`），
或更优：直接从 `runAll.yaml` 的 `observability.loki_url` 解析。

```diff
-  local infra_yaml="${ROOT}/conf/infra/docker-infra/config.yaml"
+  local infra_yaml="${ROOT}/conf/infra/ai-monitor/config.yaml"
```

### 3. 可选：增强健康检查

当前 Promtail 容器运行健康但推送可能失败（网络不通、Loki 拒绝等）。
可选增强：health check 脚本检查 Promtail 是否成功推送了最近日志。

```bash
# 检查 Loki 中最近 5 分钟内是否有 runall job 的日志
curl -s "${LOKI_URL}/loki/api/v1/query_range?query={job=\"runall\"}&limit=1&start=$(date -u -d '5 min ago' +%s)000000000" | jq -e '.data.result | length > 0'
```

此项为可选项，V1 可跳过。

### 4. 更新 `AiMonitor/run.sh` 提示信息

`print_ready_hint()` 中增加说明 Promtail 现在由 runAll 自动管理。

## 领域概念清单

| 概念 | 类型 | 说明 |
|------|------|------|
| PromtailLocal | ManagedService (runAll) | 本地日志搬运进程，tail → parse → push |
| ObservabilityStack | Bounded Context | 监控栈（Loki/Tempo/Prometheus/Grafana/Promtail） |
| LogShipper | Domain Service | 日志搬运的抽象（当前实现：Promtail） |

## 影响范围

| 文件 | 改动 |
|------|------|
| `conf/runAll.yaml` | infrastructure group 新增 `promtail-local` service 定义 |
| `runAll/scripts/runall-local-promtail.sh` | 修复 LOKI_PUSH_URL 自动解析路径（line 34） |
| `AiMonitor/run.sh` | 更新 `print_ready_hint()` 说明 Promtail 已自动管理 |

## 验收

1. runAll 启动后，`promtail-local` 作为 infrastructure 服务自动启动
2. `docker ps` 可见 `aimonitor-promtail-local` 容器 running
3. `curl http://183.250.1.132:3100/loki/api/v1/label` 返回 `job` 标签包含 `runall`
4. Grafana Trace Log Explorer 仪表板显示日志
5. runAll stop 时，Promtail 级联停止
