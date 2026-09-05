# runAll 编排服务 — 统一健康/就绪端点设计

**Date:** 2026-05-19  
**Status:** approved (2026-05-19)  
**Scope:** `config.yaml` 所列各应用服务 + 根目录 `config.yaml` 探针更新  
**Related:** `runAll/README.md`、`config.yaml`、`task2app/conf/port_config.json`

## Summary

为 runAll 编排的各 HTTP 服务增加**统一就绪探针（Readiness）**：`GET` 返回 JSON，依赖就绪时 **200**，任一必需依赖失败时 **503**。runAll 仅使用此 URL 判断「可进入下一 DAG 层级」，不再借用 `/admin/`、业务 API 或 Vite 首页。

用户已确认采用 **方案 B（Readiness）**，非纯 Liveness。

## Motivation

| 问题 | 现状 | 目标 |
|------|------|------|
| saas-backend | 探针 `GET /admin/`（302 即过） | DB 可用后再标健康 |
| ai-provider | 探针 `GET /api/public/catalog/` | 不依赖业务数据，迁移/SQLite 就绪即可 |
| taskFE | 探针 `GET /` | 专用 `/health`，语义清晰 |
| git-oauth | 已有 `/api/health/`，仅 Liveness | 增加 SQLite 检查 |
| go-run-container / go-relay | 已有 `/health`，仅进程存活 | 容器栈对 Docker 做轻量就绪（可选） |
| 响应格式 | Go 与 Django 字段不一致 | 统一 `ok` + `checks` 结构 |

runAll 实现约束（不变）：`GET` + HTTP **2xx/3xx** 视为通过；**503 不通过**。因此就绪失败必须返回 **503**（不能 200 + `"ok": false`）。

## User-Confirmed Decisions

| 项 | 决定 |
|----|------|
| 探针深度 | **Readiness（B）**：Django 服务检查 DB；Kafka/Redis 按配置条件检查 |
| runAll 使用 URL | 各服务**单一**就绪 URL（不单独暴露 `/health/ready`） |
| 失败 HTTP 状态 | **503** + JSON 说明失败项 |
| 领域逻辑 | 健康检查**不进** domain 层（交付/基础设施关注点） |

## Approaches Considered

### 1. 各服务内联实现（推荐）

每个仓库按现有风格增加 view/handler；Saas_project 在 `core/health/` 放**基础设施适配器**（DB/Kafka/Redis ping），`saas_project/urls.py` 挂路由。

- **优点**：符合各 bounded context 独立部署；与 gitOauth、go_* 现有模式一致；改动面可控。
- **缺点**：Saas_Ai_Provider 与 Saas_project 各有一份薄 view（可接受）。

### 2. 共享 `task2app/common/health` 包

单一 Python 包供所有 Django 子项目 import。

- **优点**：DRY。
- **缺点**：跨项目耦合；gitOauth 路径深；违背「子项目可独立运行」习惯。**不采用。**

### 3. 独立 health-proxy  sidecar

单独进程聚合 Redis/Kafka HTTP 探针。

- **优点**：基础设施探针集中。
- **缺点**：runAll 需多管一个进程；Docker 内 Redis 仍无 HTTP。**不采用。**

**推荐：方案 1**，Go 服务在原 `/health` 上扩展 JSON，不新增路径。

## Unified HTTP Contract

### Request

```
GET /health          # Go 微服务（根路径）
GET /api/health/     # Django REST 服务（与 gitOauth 一致，末尾斜杠）
GET /health          # Vite 开发服（中间件，无尾斜杠亦可）
```

- 无需鉴权（与 go_run_container、gitOauth 现有 `/health` 一致）。
- 不写入审计日志（可选：`django.middleware` 排除或 logger 降级）。

### Response（200 — 就绪）

```json
{
  "service": "saas-backend",
  "ok": true,
  "checks": {
    "database": { "ok": true, "latency_ms": 3 },
    "kafka": { "ok": true, "skipped": false, "latency_ms": 12 },
    "redis": { "ok": true, "skipped": true }
  }
}
```

### Response（503 — 未就绪）

```json
{
  "service": "saas-backend",
  "ok": false,
  "checks": {
    "database": { "ok": true, "latency_ms": 2 },
    "kafka": { "ok": false, "error": "Connection refused", "skipped": false }
  }
}
```

| 字段 | 说明 |
|------|------|
| `service` | 与 runAll `services[].name` 或约定短名一致 |
| `ok` | 所有**未跳过**的 check 均为 true 时为 true |
| `checks.*.skipped` | `true` 表示本环境不依赖该项（如内存消息队列跳过 Kafka） |
| `checks.*.latency_ms` | 可选，便于 runAll UI / 排障 |

**超时**：单次请求内所有 check 合计预算 **≤ 3s**（避免 runAll 默认 30s 内重试风暴）。

## Per-Service Specification

### saas-backend（`task2app/Saas_project`）

| 项 | 值 |
|----|-----|
| URL | `http://127.0.0.1:8001/api/health/` |
| 实现位置 | `core/health/checks.py`（基础设施）、`core/health/views.py`（API）、`saas_project/urls.py` |
| Checks | |

**checks 规则：**

| Check | 条件 | 方法 |
|-------|------|------|
| `database` | 始终 | `django.db.connection.ensure_connection()` + `SELECT 1` |
| `kafka` | `use_memory_message_queue()` 为 false | `confluent_kafka.AdminClient` 对 `KAFKA_BOOTSTRAP_SERVERS`（默认 `localhost:9093`）`list_topics(timeout=2)` |
| `redis` | 非内存 pub/sub（`get_pubsub_client()` 为 Redis 实现） | `PING` localhost:6379（或 `port_config` 中 redis 段若未来有） |

内存消息队列模式（当前 `port_config.json` 默认 `django.messageQueue.memory: true`）：`kafka`、`redis` 标记 `skipped: true`，仅 DB 决定就绪。

### ai-provider（`task2app/Saas_Ai_Provider`）

| 项 | 值 |
|----|-----|
| URL | `http://127.0.0.1:8010/api/health/` |
| Checks | `database` only（SQLite `db.sqlite3`） |
| 说明 | 独立 Django 项目；**不**检查 Kafka（镜像市场无领域事件队列） |

### git-oauth（`gitOauth/`）

| 项 | 值 |
|----|-----|
| URL | `http://127.0.0.1:8002/api/health/`（不变） |
| 改动 | 扩展现有 `HealthView`：增加 `database` check；保留 `public_base_url` 等字段；`ok` 由 checks 汇总 |
| 兼容 | 200 响应可保留原字段，新增 `checks` 对象 |

### taskFE（Vite dev）

| 项 | 值 |
|----|-----|
| URL | `http://127.0.0.1:4000/health` |
| 深度 | **Liveness only**（无后端 DB） |
| 实现 | `vite.config.js` → `configureServer` 中间件，`200` + `{"service":"taskFE","ok":true,"checks":{}}` |

### go-run-container

| 项 | 值 |
|----|-----|
| URL | `http://127.0.0.1:8796/health`（不变） |
| 改动 | 在 `handleHealth` 中增加可选 `docker` check：`exec.Command("docker", "info")` 或 `docker version` 超时 2s |
| 失败 | Docker 不可用 → **503** |

### go-relay（`go_relayToTrae`）

| 项 | 值 |
|----|-----|
| URL | `http://127.0.0.1:8797/health`（不变） |
| 深度 | **Liveness only**（onlineServiceJS 由 `/v1/start` 按需拉起，不在就绪时检查） |
| 改动 | 统一 JSON：`{"service":"go-relay","ok":true,"checks":{}}`（字段对齐，行为不变） |

### docker-infra（已注释）

保持 **Kafka UI** `http://127.0.0.1:8080` 作为栈级 HTTP 探针；不在本设计内新增 Redis HTTP 端点。若启用 compose，建议在文档注明：Redis 就绪由 compose `depends_on` + UI 间接保证。

## Domain Model（DDD）

健康检查**不属于**任何业务聚合，定位为**交付层横切能力**：

| 层级 | 职责 |
|------|------|
| **API / 接口层** | `HealthView`：汇总 checks、映射 HTTP 200/503 |
| **基础设施层** | `run_database_check()`、`run_kafka_check()`、`run_redis_check()`：仅 I/O，无业务规则 |
| **领域层** | **无**健康相关代码 |
| **Bounded Context** | saas-backend、ai-provider、gitOauth 各自维护本 context 的 health 模块；不跨 context 调用领域服务 |

与 DDD-03 一致：check 函数可 import `django.db`、`confluent_kafka`、`redis`，但放在 `core/health/`（基础设施），不放入 `*/domain/**`。

## config.yaml Updates

```yaml
# 示例片段（端口以 port_config.json 为准）
- name: saas-backend
  health_check:
    url: "http://127.0.0.1:8001/api/health/"

- name: ai-provider
  health_check:
    url: "http://127.0.0.1:8010/api/health/"

- name: taskFE
  health_check:
    url: "http://127.0.0.1:4000/health"

# git-oauth、go-run-container、go-relay：URL 不变，行为升级为 readiness
```

## Error Handling

- 任一**未跳过**的 check 异常 → 该项 `ok: false`，记录 `error` 字符串（不泄露堆栈）。
- 视图层捕获总异常 → 503 + `{"ok":false,"error":"internal"}`。
- Kafka/Redis 检查失败**不**触发 Django 500 页面（仅 JSON）。

## Testing

| 服务 | 测试 |
|------|------|
| saas-backend | `APIClient.get("/api/health/")`：mock DB ok；`use_memory_message_queue` patch 下 kafka/redis skipped |
| ai-provider | 同上，仅 database |
| git-oauth | 扩展 `HealthView` 测试 |
| go_* | 现有 `TestHealthEndpoint` 增加 503 场景（docker 失败 mock） |
| vue | 可选：文档说明手动 `curl`；E2E 非必须 |

## Out of Scope

- runAll 支持 TCP/非 HTTP 探针
- Saas_email、mock_run_container Python、trae onlineServiceJS（未列入当前 `config.yaml`）
- Kubernetes liveness/readiness 双探针分离（本地 runAll 单 URL 即可）
- 价值流服务 valueStream（独立工具链）

## Implementation Order

1. **saas-backend** `core/health` + url + tests  
2. **ai-provider** `/api/health/`  
3. **git-oauth** 扩展现有 HealthView  
4. **vue** Vite 中间件  
5. **go-run-container** / **go-relay** JSON 对齐 + docker check  
6. 更新根目录 **`config.yaml`** 探针 URL  
7. 本地 `./runAll --config config.yaml` 验证 DAG

## Open Questions（实现前可默认）

1. saas-backend 健康 URL 用 `/api/health/` 还是根路径 `/health/`？**默认 `/api/health/`**（与 API 前缀一致）。  
2. go-relay 是否在就绪时检查 Docker？**默认否**（仅 go-run-container 检查）。

---

**Next step after approval:** invoke `writing-plans` skill for checkbox implementation plan.
