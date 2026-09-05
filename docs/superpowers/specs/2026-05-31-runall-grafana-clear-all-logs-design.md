# runAll 一键清空 Grafana 可观测数据（日志 / 指标 / Trace）

**日期：** 2026-05-31  
**状态：** 待批准（v2 — 含 Prometheus + Tempo）  
**关联：**
- `2026-05-28-centralized-logs-grafana-traceid-design.md`（Loki + Promtail + Grafana 栈）
- `2026-05-29-grafana-distributed-tracing-runbook.md`（Distributed Trace View）
- `2026-05-31-task-container-gateway-vite-proxy-grafana-observability-design.md`（Trace Log Explore 排障）
- runAll 现有 per-service「清空日志」仅清内存缓冲（`/api/logs/clear`）

---

## 1. 背景与目标

### 1.1 问题

本地开发中，AiMonitor 栈持续积累三类可观测数据：

| 类型 | 路径 | AI 查询入口 | 问题 |
|------|------|-------------|------|
| **日志** | tee → Promtail → Loki | Trace Log Explore / Loki Explore | 7 天 retention，旧 trace 日志污染上下文 |
| **Trace** | OTel → Tempo | Tempo Explore / Distributed Trace View | 旧 span 树、traceId 搜索仍命中历史 |
| **指标** | scrape + Tempo remote_write → Prometheus | Grafana 健康/延迟面板 | 历史 scrape 与 span-metrics 使 AI 误判当前状态 |

runAll Web UI 的 per-service「清空日志」**仅清内存**，tee 文件与 AiMonitor 各后端均不受影响。

**核心诉求：** 在 runAll Web UI 提供**一键**操作，使 Grafana 侧日志、Trace、指标查询均回到「仅含清空之后新产生数据」的干净状态，避免 AI Agent 浪费上下文在过时 observability 数据上。

### 1.2 目标

1. runAll 可观测栏（`#observability-bar`）增加 **「清空 Grafana 可观测数据」** 按钮。
2. 一次操作同步清理：
   - **runAll 本地：** 全部服务内存日志缓冲 + tee 源文件
   - **Loki：** 索引/块 + Promtail 读取位点
   - **Tempo：** WAL + blocks + metrics_generator 本地存储
   - **Prometheus：** TSDB（含 blackbox scrape、Tempo span-metrics remote_write）
3. 操作后 Grafana 查询（LogQL / Tempo traceId / PromQL）**不再返回清空前历史**（允许秒级延迟）。
4. **保留** Grafana 仪表盘/数据源/provisioning 配置（`grafana_data` 不删）。

### 1.3 非目标

- 生产环境 retention / 合规删除策略。
- 替换 per-service「清空日志」（保留；一键清空为 observability 层增强）。
- 自动定时清理。
- 停止 runAll 编排的业务服务。

---

## 2. 现状与缺口

### 2.1 数据流

```
日志:  runAll tee → /tmp/runall-logs/*.log → Promtail (job=runall) → Loki
Trace: 服务 OTEL → otel-collector → Tempo (:3200)
指标:  blackbox scrape + Tempo metrics_generator remote_write → Prometheus (:9090)
展示:  Grafana (:3000) ← Loki / Tempo / Prometheus datasources
```

| 组件 | Volume | 现有清空 | 缺口 |
|------|--------|----------|------|
| runAll 内存 | — | per-service `Clear(name)` | 无「全部」入口 |
| tee 文件 | host `/tmp/runall-logs` | ❌ Clear 未截断文件 | Promtail 可读旧内容 |
| Loki | `loki_data` | ❌ | Explore 返回 7 天历史 |
| Promtail | `promtail_data` (positions) | ❌ | truncate 后位点可能错乱 |
| Tempo | `tempo_data` | ❌ | trace 搜索仍命中旧数据 |
| Prometheus | `prometheus_data` | ❌ | 指标/span-metrics 历史残留 |

### 2.2 为何不用各组件 Delete API 作为主路径

| 组件 | API 方案 | 不适合 MVP 的原因 |
|------|----------|-------------------|
| Loki | `POST /loki/api/v1/delete` | 默认 24h cancel period；非即时 |
| Tempo | 无「删全部 trace」简易 API | 需停服清 volume 或 blocklist 运维 |
| Prometheus | `admin/tsdb/delete_series` | 高基数 match 慢且不彻底；remote_write 历史难精确删 |

**结论：** 本地 dev 采用 **具名容器 stop → 删除 data volumes → 按依赖顺序 start**，与 compose 架构一致、即时生效。

---

## 3. 价值流影响

| 流 | 影响 |
|----|------|
| **platform-centralized-logging** | **扩展** — 步骤从「仅 Loki」升级为全栈 observability reset |
| increment2-grafana-trace-dashboard / Distributed Trace View | 无结构变更；清空后 dashboard 空态正常 |
| increment3-loki-retention / tempo block_retention | 互补 — retention 管被动过期；本功能管主动会话重置 |

**受影响 step（建议登记）：**

| name | status | test_file（建议） | fields |
|------|--------|-------------------|--------|
| increment4-runall-clear-all-observability | planned | `runAll/src/infrastructure/observability_storage_reset_test.go` | `ai-monitor.loki.chunks_cleared` |
| | | | `ai-monitor.tempo.blocks_cleared` |
| | | | `ai-monitor.prometheus.tsdb_cleared` |
| | | | `runall.runtime.log_file_truncated` |
| | | | `runall.runtime.memory_log_cleared` |

**跨流依赖：** 依赖 `ai-monitor` 已启动（相关容器存在）；**不**依赖业务服务重启。

---

## 4. 领域概念清单（供 /5-ddd）

| 概念 | 边界 | 说明 |
|------|------|------|
| **ObservabilityStackReset** | runAll 可观测性 | 一次一键清空的聚合根（日志+指标+trace） |
| **LocalLogSource** | runAll | tee 目录及内存缓冲 |
| **LokiStorageReset** | AiMonitor | `loki_data` + compactor |
| **PromtailCursorReset** | AiMonitor | `promtail_data` positions |
| **TempoTraceStorageReset** | AiMonitor | `tempo_data`（wal/blocks/generator） |
| **PrometheusTSDBReset** | AiMonitor | `prometheus_data` |

**领域事件：**

- `AllObservabilityDataCleared`
- `ObservabilityResetPartialFailure` — 返回逐步明细

**Bounded Context：** `platform-observability`（runAll UI/API）↔ `ai-monitor-infra`（Docker volume 生命周期）

---

## 5. 方案对比

### 方案 A：volume 级重置（推荐）

runAll `POST /api/observability/clear-all` 编排：

1. 清全部 service 内存日志 + truncate tee 文件
2. 调用 `AiMonitor/scripts/reset_observability_storage.sh`：
   - **Stop（采集先停）：** promtail → otel-collector
   - **Stop（存储）：** loki → tempo → prometheus
   - **Remove volumes：** `loki_data`, `promtail_data`, `tempo_data`, `prometheus_data`
   - **Start（依赖序）：** prometheus → tempo → otel-collector → loki → promtail
   - 等待 Loki `/ready`、Prometheus `/-/healthy`、Tempo `/ready`（可选）

| 优点 | 缺点 |
|------|------|
| 日志/Trace/指标 **一次性**干净 | 需 docker CLI |
| 秒级生效 | 清空期间 Grafana 查询短暂不可用 |
| 不碰 `grafana_data` | ai-monitor `managed` 模式需具名容器操作（见 §6.2） |

### 方案 B：分项 API 删除（Loki delete + Prom delete_series + Tempo 保留）

| 优点 | 缺点 |
|------|------|
| 部分组件可不重启 | 三套 API 语义/延迟不一致 |
| | Tempo 难彻底清空 |
| | 实现与验收复杂 |

### 方案 C：整栈 `compose down -v && compose up`

| 优点 | 缺点 |
|------|------|
| 最简单 | 会删掉 **grafana_data**（若用 `-v` 全卷）或 kill managed 前台进程 |
| | 影响面过大 |

**推荐：方案 A**，精确删除 4 个 data volume，保留 Grafana 配置。

---

## 6. 推荐设计

### 6.1 UI

```html
<button id="obs-clear-all" class="logs-panel-btn clear-logs-btn" type="button">
  清空 Grafana 可观测数据
</button>
```

**交互：**

1. `confirm()`：「将清空：① 所有服务内存/tee 日志 ② Loki 日志 ③ Tempo Trace ④ Prometheus 指标（含 span-metrics）。Grafana 仪表盘配置保留。数秒内查询将不再显示历史。是否继续？」
2. `POST /api/observability/clear-all`
3. 成功 alert 摘要；logs panel 若打开则 refresh
4. partial 失败时列出逐步状态

### 6.2 API

```
POST /api/observability/clear-all
```

**Response 200：**

```json
{
  "status": "ok",
  "memory_services_cleared": 12,
  "files_truncated": 12,
  "loki_reset": "ok",
  "promtail_reset": "ok",
  "tempo_reset": "ok",
  "prometheus_reset": "ok"
}
```

**Response（partial）：** 同上，`status: "partial"`，失败步为 `"error: ..."` 字符串。

**ai-monitor managed 模式：** 使用**具名容器**（与 compose `container_name` 一致），避免 `compose down` 终止 runAll 托管的前台进程：

| 容器 | 作用 |
|------|------|
| `aimonitor-promtail` | 日志采集 |
| `aimonitor-otel-collector` | trace 转发 |
| `aimonitor-loki` | 日志存储 |
| `aimonitor-tempo` | trace 存储 |
| `aimonitor-prometheus` | 指标存储 |

**Volume 命名：** 以 `docker volume ls` 实际为准（通常为 `{project}_loki_data` 等）；脚本内通过 compose project label 解析或文档化常量。

### 6.3 Stop / Start 顺序

```
Stop:  promtail → otel-collector → loki → tempo → prometheus
Rm:    loki_data, promtail_data, tempo_data, prometheus_data
Start: prometheus → tempo → otel-collector → loki → promtail
```

**说明：**

- 先停 **otel-collector**，避免 reset 期间继续写入 Tempo。
- **Prometheus 先于 Tempo 启动**：Tempo `metrics_generator` 配置 `remote_write` 到 Prometheus；Prometheus 需先就绪接收 remote write。
- **Grafana / blackbox-exporter 不 stop**：无 state volume 需清；blackbox 仅转发探测，指标存于 Prometheus TSDB（已清）。

### 6.4 代码结构

| 文件 | 变更 |
|------|------|
| `runAll/src/ui.go` | `POST /api/observability/clear-all` |
| `runAll/src/domain/observability_stack_reset.go` | 编排本地 + 远程 reset |
| `runAll/src/infrastructure/file_service_log_sink.go` | `Truncate` / `TruncateAll` |
| `runAll/src/infrastructure/tee_service_log_repository.go` | `Clear()` 同步截断文件 |
| `runAll/src/infrastructure/observability_storage_reset.go` | exec reset 脚本 |
| `runAll/src/status.html` | 按钮 + `clearAllObservability()` |
| `AiMonitor/scripts/reset_observability_storage.sh` | 四 volume reset + 健康等待 |
| `AiMonitor/scripts/test_reset_observability_storage.py` | 脚本结构/顺序单测 |

**配置（可选）：**

```yaml
observability:
  grafana_url: "http://127.0.0.1:3000"
  loki_url: "http://127.0.0.1:3100"
  reset_script: "../../AiMonitor/scripts/reset_observability_storage.sh"
```

### 6.5 验收标准

| # | 场景 | 期望 |
|---|------|------|
| AC1 | LogQL `{job="runall"}` | 空或仅清空后新日志 |
| AC2 | Tempo Explore traceId 搜索 | 清空前 trace **不可查** |
| AC3 | Prometheus `runall-health` 等指标 | 清空前时间序列不可查（新 scrape 后重建） |
| AC4 | Distributed Trace View | 无旧 trace 瀑布图 |
| AC5 | runAll logs panel | 全部服务 tail 为空 |
| AC6 | 清空后业务继续运行 | 新日志/trace/指标正常写入 |
| AC7 | ai-monitor 未启动 | API 明确报错；本地内存/文件仍清理 |
| AC8 | Grafana 登录与 dashboard | **仍可访问**（grafana_data 保留） |
| AC9 | per-service「清空日志」 | tee 文件同步截断（修复现有缺口） |

### 6.6 安全与范围

- localhost dev-only；与 runAll 其他 mutating API 同级。
- 不清 `grafana_data`；不 stop `aimonitor-grafana`。
- 文档注明：勿对含生产数据的 Prometheus/Tempo 实例使用。

---

## 7. 实施切片（供 /6-plans）

1. **Fix tee Clear** — 文件截断 + 单测
2. **AiMonitor script** — `reset_observability_storage.sh`（四 volume + 顺序）+ Python 结构测试
3. **Domain + API** — `ObservabilityStackResetService` + `/api/observability/clear-all`
4. **UI** — observability bar 按钮 + 确认文案
5. **value-stream.yaml** — `increment4-runall-clear-all-observability`
6. **验收** — `verify_trace_stack.py` 扩展或独立 `verify_observability_reset.py`

---

## 8. 决策记录

| 问题 | 决策 |
|------|------|
| 是否清 Tempo trace？ | **是** — 删 `tempo_data` |
| 是否清 Prometheus 指标？ | **是** — 删 `prometheus_data`（含 span-metrics remote_write） |
| 是否清 Grafana 配置？ | **否** — 保留 `grafana_data` |
| 是否 stop 业务服务？ | **否** |
| 按钮/API 命名 | UI「清空 Grafana 可观测数据」；API `POST /api/observability/clear-all` |

---

## 9. 摘要

在 runAll 可观测栏增加一键按钮，通过 **内存/tee 清空 + Loki/Promtail/Tempo/Prometheus 四个 data volume 重置** 实现 Grafana 侧日志、Trace、指标会话级「归零」，避免 AI 查询过时 observability 数据浪费上下文。采用具名容器 stop/start，保留 Grafana 配置卷；顺带修复 per-service 清空不截断 tee 文件的缺口。
