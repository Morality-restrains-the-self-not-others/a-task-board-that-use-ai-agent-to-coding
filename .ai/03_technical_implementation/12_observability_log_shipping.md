# 可观测性 — 服务日志投递规范

## 基本信息
- 版本：1.2.2
- 创建日期：2026-07-01
- 最后修改：2026-07-16
- 维护者：Claude AI
- 适用范围：所有后端服务（Python/Go/Node.js）+ runAll 编排器

## 规则分类

### 核心规则
> 影响线上排障效率和可观测性的关键规则，必须严格遵守

---

## 0. 部署模式与日志架构总览

本规范覆盖两种部署模式，服务无论处于哪种模式下均须满足 §1 日志输出标准。

### Mode A: 本地开发（runAll 单机多进程）

```
┌──────────────────────────────────────────┐
│  runAll 编排器                             │
│  ├─ saas-backend       → logs/saas-backend.log
│  ├─ task-auth          → logs/task-auth.log
│  ├─ taskContainerGateway → logs/task-container-gateway.log
│  └─ ... (N 个进程)                         │
│                                           │
│  promtail-local (1 个容器)                  │
│  ├─ tails logs/*.log                      │
│  ├─ 每服务独立 scrape_config → job 标签     │
│  └─ push → Loki (:3100)                   │
└──────────────────────────────────────────┘
```

### Mode B: 服务独立部署（每服务独立容器/VM）

```
┌─────────────────┐  ┌─────────────────┐  ┌─────────────────┐
│ saas-backend    │  │ task-auth       │  │ taskGateway     │
│ ┌─────────────┐ │  │ ┌─────────────┐ │  │ ┌─────────────┐ │
│ │ app         │ │  │ │ app         │ │  │ │ app         │ │
│ │ stdout/stderr│ │  │ │ stdout/stderr│ │  │ │ stdout/stderr│ │
│ └──────┬──────┘ │  │ └──────┬──────┘ │  │ └──────┬──────┘ │
│ ┌──────▼──────┐ │  │ ┌──────▼──────┐ │  │ ┌──────▼──────┐ │
│ │ promtail    │ │  │ │ promtail    │ │  │ │ promtail    │ │
│ │ sidecar     │ │  │ │ sidecar     │ │  │ │ sidecar     │ │
│ └──────┬──────┘ │  │ └──────┬──────┘ │  │ └──────┬──────┘ │
└────────┼────────┘  └────────┼────────┘  └────────┼────────┘
         │                    │                    │
         └────────────────────┼────────────────────┘
                              │
                         ┌────▼────┐
                         │  Loki   │
                         │ (:3100) │
                         └────┬────┘
                              │
                         ┌────▼────┐
                         │ Grafana │
                         │ (:3000) │
                         └─────────┘
```

**核心差异**: Mode A 共享 promtail，Mode B 每服务自带 promtail sidecar。两种模式下 Loki 中的 `job` 标签均等于服务名称，Grafana 查询语句完全一致。

---

## 1. 日志输出标准（一级）

### 1.1 结构化 JSON 日志

所有服务的 **stdout/stderr 输出**必须使用结构化 JSON 格式，包含以下必填字段：

| 字段 | 类型 | 说明 | 示例 |
|------|------|------|------|
| `ts` | ISO8601 string | 日志时间戳 | `"2026-07-01T15:43:54Z"` |
| `level` | string (小写) | 日志级别 | `"info"`, `"warn"`, `"error"`, `"debug"` |
| `msg` | string | 日志消息 | `"request completed"` |
| `service` | string | 服务名称 | `"saas-backend"` |
| `trace_id` | string | 请求追踪 ID | `"web-1782920634053-ztjtzyrygk"` |
| `daydaymoney_service_id` | string | daydaymoney.yaml 中的稳定服务标识（推荐） | `"taskProjectService"` |
| `daydaymoney_tags` | string / array | 服务标签，Loki 可用逗号串；至少含 `svc:<service_id>`（推荐） | `"svc:taskProjectService,domain:project"` |

**Python 示例 (Django)**:
```python
import json, logging, time
from datetime import datetime, timezone

class StructuredFormatter(logging.Formatter):
    def format(self, record):
        return json.dumps({
            "ts": datetime.now(timezone.utc).strftime("%Y-%m-%dT%H:%M:%SZ"),
            "level": record.levelname.lower(),
            "msg": record.getMessage(),
            "service": "saas-backend",
            "trace_id": getattr(record, "trace_id", ""),
        })
```

**Go 示例 (slog)**:
```go
import "log/slog"

func Init(serviceName string) {
    handler := slog.NewJSONHandler(os.Stdout, &slog.HandlerOptions{
        Level: slog.LevelInfo,
        ReplaceAttr: func(groups []string, a slog.Attr) slog.Attr {
            if a.Key == slog.TimeKey { a.Key = "ts" }
            if a.Key == slog.LevelKey { a.Key = "level" }
            if a.Key == slog.MessageKey { a.Key = "msg" }
            return a
        },
    })
    logger := slog.New(handler).With("service", serviceName)
    slog.SetDefault(logger)
}
```

### 1.2 traceId 透传

- HTTP 请求入口（APISIX/网关）生成或读取 `X-Trace-Id` 请求头
- 所有下游服务在接收请求时从 `X-Trace-Id` 头提取 traceId
- traceId 写入每条日志的 `trace_id` 字段
- 容器在线服务（onlineServiceJS）通过 `X-Trace-Id` 头接收并回传

**约束**: traceId 缺失时 `trace_id` 字段设为空字符串 `""`，不得省略

**前端报错展示**: 请求失败时，展示错误的 DOM 须设置 `data-traceId`（值为该次请求 traceId），见 [24_frontend_error_data_trace_id.md](../01_project_constraints/24_frontend_error_data_trace_id.md)。

**Agent 排障**: 错误带 `data-traceId` 时须**优先**按该 ID 检索 Loki（`{job=~".+"} | json | trace_id="<id>"`）并重建全链路路径，再改代码；短规范见 `.claude/skills/1-brainstorming-design-docs/references/traceid-log-first-diagnosis.md`。

---

## 2. 日志采集 — 部署模式自适应（一级）

### 2.1 原则

> **每个服务在 Loki 中拥有独立的 `job` 标签**，确保 Grafana 可按 `{job="service-name"}` 精确检索。
> **无论 Mode A 还是 Mode B，`job` 标签始终等于服务名称**，Grafana 查询无需因部署模式而修改。

### 2.1.1 规范路径与禁止旁路（一级）

| 要求 | 说明 |
|------|------|
| **规范路径** | `${RUNALL_LOG_ROOT:-logs}/<runAll-service-name>.log`（例：`logs/go-relay.log`、`logs/task-cloud-service.log`） |
| **启动方式** | 优先经 runAll UI/API/`--daemon` 拉起（自动 tee）；手动调试用 `bash runAll/scripts/run-with-canonical-log.sh <service> -- <cmd>` |
| **禁止** | `*.restart.log`、`*-run.log`、二进制 camelCase 旁路重定向（如 `go_relayToTrae.restart.log`）——主 scrape 不保证采集，易导致 Grafana 无日志 |
| **Promtail 兜底** | `generate-promtail-config.sh` 将已知旁路文件名映射到同一 `job`，仅作过渡；新启动仍须写规范路径 |
| **常驻采集** | `conf/runAll.yaml` 中 `promtail-local` 依赖 `ai-monitor`；清空可观测栈后须启动 infrastructure（或 `runall-local-promtail.sh up`） |

### 2.2 Mode A: 本地开发环境（runAll 单机多进程）

采用 **单一 promtail 实例 + 每服务独立 scrape_config** 模式：

```yaml
# AiMonitor/promtail/promtail-local.yaml（由 runAll 启动时自动生成）
scrape_configs:
  # 每个服务一个 scrape_config，job 标签 = 服务名
  - job_name: saas-backend
    static_configs:
      - targets: [localhost]
        labels:
          job: saas-backend
          __path__: /var/log/runall/saas-backend.log

  - job_name: task-auth
    static_configs:
      - targets: [localhost]
        labels:
          job: task-auth
          __path__: /var/log/runall/task-auth.log

  - job_name: task-container-gateway
    static_configs:
      - targets: [localhost]
        labels:
          job: task-container-gateway
          __path__: /var/log/runall/task-container-gateway.log

  # ... 每个服务一个 job_name
```

**特点**:
- 单一 promtail 容器，资源开销最小
- 服务以进程方式运行，日志经 tee → `logs/<service>.log`
- promtail 配置由 `runAll` 启动时自动生成

### 2.3 Mode B: 服务独立部署（每服务独立 promtail sidecar）

每个服务容器 **必须** 附带一个 promtail sidecar 容器，负责将自身日志推送至 Loki。

#### 2.3.1 Docker Compose 模板

```yaml
# docker-compose.<service-name>.yaml
version: "3.8"
services:
  <service-name>:
    image: <service-name>:latest
    container_name: <service-name>
    # ... 服务自身配置（ports, env, volumes, healthcheck）...
    logging:
      driver: json-file
      options:
        max-size: "50m"
        max-file: "3"

  <service-name>-promtail:                    # ← 每服务必须附带 promtail sidecar
    image: grafana/promtail:3.4.2
    container_name: <service-name>-promtail
    command:
      - -config.file=/etc/promtail/config.yaml
      - -config.expand-env=true
    environment:
      LOKI_PUSH_URL: ${LOKI_PUSH_URL:?required}   # 由部署环境注入
    volumes:
      - <service-name>-logs:/var/log/app:ro       # 共享日志卷
      - ./promtail/config.yaml:/etc/promtail/config.yaml:ro
      - promtail_<service-name>_data:/var/lib/promtail
    depends_on:
      <service-name>:
        condition: service_healthy                 # 等服务就绪后再启 promtail
    restart: unless-stopped
    healthcheck:
      test: ["CMD", "wget", "--spider", "-q", "http://localhost:9080/ready"]
      interval: 15s
      timeout: 5s
      retries: 3
      start_period: 10s

volumes:
  <service-name>-logs:
  promtail_<service-name>_data:
```

#### 2.3.2 独立部署 promtail 配置

每服务的 `promtail/config.yaml` 使用标准模板（与 Mode A 中 pipeline_stages 完全一致，仅 `__path__` 与 `job` 不同）：

```yaml
# promtail/config.yaml（随服务镜像打包或在部署时挂载）
server:
  http_listen_port: 9080
  grpc_listen_port: 0

positions:
  filename: /var/lib/promtail/positions.yaml

clients:
  - url: ${LOKI_PUSH_URL}
    # 重试与容错
    backoff_config:
      min_period: 500ms
      max_period: 10m
      max_retries: 20
    batchwait: 1s
    batchsize: 1048576

scrape_configs:
  - job_name: ${SERVICE_NAME}               # ← 部署时注入服务名
    static_configs:
      - targets: [localhost]
        labels:
          job: ${SERVICE_NAME}
          __path__: /var/log/app/*.log       # 服务日志挂载路径
    pipeline_stages:
      # 标准解析管道 — 与 Mode A 完全一致
      - regex:
          expression: '^\S+ \((?:stdout|stderr)\) (?P<payload>.*)$'
      - json:
          expressions:
            ts: ts
            level: level
            msg: msg
            service: service
            trace_id: trace_id
            traceId: traceId
          source: payload
      # 明文兜底：非 JSON 行（如 log.Fatalf）仍提取 level，避免 Grafana var-level=error 漏看
      - regex:
          expression: '(?i)\b(?P<plain_level>error|fatal|panic|warn(?:ing)?|info|debug)\b'
          source: payload
      - template:
          source: level
          template: '{{ if .level }}{{ .level }}{{ else if eq .plain_level "fatal" }}error{{ else if eq .plain_level "panic" }}error{{ else if eq .plain_level "warning" }}warn{{ else }}{{ .plain_level }}{{ end }}'
      - template:
          source: trace_id
          template: '{{ if .trace_id }}{{ .trace_id }}{{ else }}{{ .traceId }}{{ end }}'
      - labels:
          level:
          service:
      - structured_metadata:
          trace_id:
          msg:
      - output:
          source: payload
```

#### 2.3.3 部署时注入的环境变量

| 变量 | 说明 | 示例 |
|------|------|------|
| `LOKI_PUSH_URL` | Loki push 端点 | `http://10.2.150.119:3100/loki/api/v1/push` |
| `SERVICE_NAME` | 服务名（= `job` 标签） | `saas-backend` |

> 💡 **部署脚本职责**: 启动服务时自动注入 `SERVICE_NAME` 和 `LOKI_PUSH_URL` 环境变量到 promtail 容器。promtail 配置文件中使用 `${SERVICE_NAME}` / `${LOKI_PUSH_URL}` 占位符，由 `-config.expand-env=true` 在启动时展开。

#### 2.3.4 服务直写 JSON 到 stdout 时的简化管道

若服务直接输出 JSON 到 stdout（不经 tee），可省略 regex 阶段：

```yaml
    pipeline_stages:
      # 服务直写 JSON → 无需 tee regex 提取
      - json:
          expressions:
            ts: ts
            level: level
            msg: msg
            service: service
            trace_id: trace_id
      - labels:
          level:
          service:
      - structured_metadata:
          trace_id:
          msg:
```

### 2.4 两种模式对比

| 维度 | Mode A (runAll 开发) | Mode B (独立部署) |
|------|---------------------|-------------------|
| promtail 实例数 | **1** 个（共享） | **N** 个（每服务 1 个） |
| 日志传输方式 | 进程 tee → 文件 → promtail tail | stdout/stderr → Docker log driver → promtail tail 或直接 scrape |
| Loki `job` 标签 | `{service-name}` | `{service-name}` |
| 配置来源 | runAll 启动时自动生成 | 部署时模板渲染 + 环境变量注入 |
| 资源开销 | 最小（~30MB 内存） | 每服务 ~30-50MB 内存（可接受） |
| 故障隔离 | 共享 promtail 崩溃影响所有服务 | promtail 崩溃仅影响对应服务 |
| Grafana 查询 | `{job="saas-backend"}` | `{job="saas-backend"}` ← 完全一致 |

---

## 3. 日志采集配置自动生成（一级）

### 3.1 runAll 集成

`runAll` 启动时自动生成 promtail 配置：

```
runAll 启动流程:
  1. 读取 conf/runAll.yaml → groups[].services[].name
  2. 生成 AiMonitor/promtail/promtail-local.yaml（每服务一个 scrape_config）
  3. 启动 promtail-local 容器
  4. promtail 开始 tail logs/ 目录下各服务日志文件
```

### 3.2 配置生成脚本

```bash
# runAll/scripts/generate-promtail-config.sh
# 从 runAll.yaml 提取服务列表 → 生成 per-service scrape_configs
```

### 3.3 新服务接入检查清单

新增服务时，必须完成以下步骤。标记 `[A]` 仅适用于 Mode A（runAll），`[B]` 仅适用于 Mode B（独立部署），无标记则两者均适用。

- [ ] **[日志]** 服务日志输出使用结构化 JSON（见 §1.1）
- [ ] **[日志]** 日志包含 `ts`, `level`, `msg`, `service`, `trace_id` 字段
- [ ] **[A]** `conf/runAll.yaml` 中注册服务（`groups[].services[].name`）
- [ ] **[A]** 服务日志文件写入 `logs/<service-name>.log`
- [ ] **[A]** promtail 配置自动生成（由 runAll 启动流程处理）
- [ ] **[B]** 服务 `Dockerfile` 中日志输出到 stdout/stderr（不写文件）
- [ ] **[B]** `docker-compose.<service>.yaml` 包含 promtail sidecar（见 §2.3.1）
- [ ] **[B]** `promtail/config.yaml` 中 `job_name` = `SERVICE_NAME`（环境变量注入）
- [ ] **[B]** 部署脚本注入 `SERVICE_NAME` + `LOKI_PUSH_URL` 环境变量
- [ ] **[验证]** Grafana Explore `{job="<service-name>"}` 可检索到日志
- [ ] **[验证]** `{job="<service-name>"} | json | trace_id != ""` 可检索到带 traceId 的日志

---

## 4. Grafana 可观测性验证（一级）

### 4.1 新服务接入后验证

```bash
# 1. 检查 Loki 中是否有该服务日志
curl -s -G "${LOKI_URL}/loki/api/v1/query_range" \
  --data-urlencode 'query={job="<service-name>"}' \
  --data-urlencode 'limit=5' \
  --data-urlencode 'start='"$(date -u -d '10 minutes ago' +%s)000000000" \
  --data-urlencode 'end='"$(date -u +%s)000000000" | python3 -c "
import sys, json
d = json.load(sys.stdin)
results = d.get('data', {}).get('result', [])
print(f'{len(results)} streams found for job=<service-name>')
"

# 2. 检查 traceId 是否可检索
curl -s -G "${LOKI_URL}/loki/api/v1/query_range" \
  --data-urlencode 'query={job="<service-name>"} | json | trace_id != ""' \
  --data-urlencode 'limit=5' ...
```

### 4.2 可用 job 标签巡检

```bash
# 列出所有 job → 与 conf/runAll.yaml 服务列表对比 → 发现缺失
curl -s "${LOKI_URL}/loki/api/v1/label/job/values" | \
  python3 -c "import sys,json; [print(j) for j in json.load(sys.stdin)['data']]"
```

---

## 5. 反模式与禁止项

| 反模式 | 问题 | 正确做法 |
|--------|------|---------|
| 纯文本日志（无结构化） | traceId/level/service 无法提取 | 所有日志使用 JSON 格式（§1.1） |
| `log.Fatalf` / 明文 stderr 致命错误 | Grafana `var-level=error` 漏看启动失败 | Go 使用 `tracelog.Fatal` / `tracelog.Fatalf`（或等价 JSON）后 `os.Exit` |
| 日志写到随机路径 | promtail 无法发现 | 统一写入 `logs/<service-name>.log` |
| `trace_id` 字段缺失 | 无法按 traceId 检索 | 始终包含 `trace_id` 字段（缺失时设空字符串） |
| 多个服务共用同一 `job` 标签 | 无法按服务隔离查询 | 每服务独立 `job` 标签（§2.2） |
| `level` 字段使用非标准值 | Grafana 面板无法按级别过滤 | 仅使用 `debug/info/warn/error` |
| HTTP 请求/响应 body 写入日志 | 敏感信息泄露、日志膨胀 | 仅记录请求元数据（method/url/status/duration） |

---

## 6. 故障排查

### 6.1 服务日志未出现在 Loki

1. 确认日志文件存在: `ls -la logs/<service-name>.log`
2. 确认日志格式: `head -5 logs/<service-name>.log` — 应为 JSON
3. 确认 promtail 运行: `docker ps | grep promtail`
4. 确认 promtail 配置包含该服务: 检查 `AiMonitor/promtail/promtail-local.yaml`
5. 检查 promtail 自身日志: `docker logs aimonitor-promtail-local`

### 6.2 traceId 检索为空

1. 确认服务日志 JSON 中包含 `trace_id` 字段
2. 确认字段名一致（snake_case `trace_id`，非 `traceId`）
3. 确认 pipeline stages 正确解析 JSON 路径
4. 验证: `{job="<service>"} | json | trace_id != ""` 查询

### 6.3 Grafana `var-level=error` 看不到启动失败

**典型根因：** 服务用 `log.Fatalf` / 明文 stderr 报错，未写 JSON `level` 字段；旧版 dashboard 用流选择器 `{level=~"error"}`，无 `level` label 的行被排除。

**排查：**

1. 本地日志是否含明文错误: `rg -i 'error|fatal|panic' logs/<service>.log | tail`
2. Loki 是否已采集但无 level 标签: `{job="<service>"} |= "Config error"` 有结果，而 `{job="<service>", level="error"}` 为空
3. Promtail pipeline 是否含 `plain_level` 兜底（`generate-promtail-config.sh`）
4. Trace Log Journey 面板 LogQL 是否在过滤前 `regexp` + `label_format` 合并明文 level

**修复方向：** 服务侧改为结构化 JSON（§1.1）；采集侧保留明文 level 兜底；dashboard 不依赖「level 标签必须已存在」。

---

## 变更历史

| 日期 | 版本 | 变更 |
|------|------|------|
| 2026-07-16 | 1.2.2 | `tracelog.Fatal`；Journey `service`/`job` 收窄；value-stream 缺失 test_file 纠路径或 planned |
| 2026-07-16 | 1.2.1 | Promtail 明文 level 兜底；§6.3 Grafana level=error 漏看排查；Trace Log Journey LogQL 合并 extracted_level |
| 2026-07-02 | 1.1.0 | 新增 §0 部署模式与日志架构总览（Mode A/B）；§2 重构为部署模式自适应；新增独立部署 Docker Compose 模板、环境变量注入规范、每服务 promtail sidecar 要求 |
| 2026-07-01 | 1.0.0 | 初始版本：结构化日志标准 + 每服务独立 promtail job 标签 + 自动配置生成 |
