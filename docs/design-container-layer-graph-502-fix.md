# 设计文档: container-layer-graph 502 修复 — 路由归一化 + 容器可用性预检

- **状态**: 🎯 已设计，待实现
- **作者**: claude
- **日期**: 2026-07-01
- **TraceId**: `web-1782920634053-ztjtzyrygk`

---

## 1. 问题描述

### 现象

```
GET /api/tenant/{tid}/workspace/{wid}/task/{taskId}/cloud/compute/container-layer-graph/
→ 502 Bad Gateway
→ "容器接口返回错误: GET http://127.0.0.1:8765/api/layers -> HTTP 502; detail=Connection refused"
```

### 影响范围

- **受影响端点**: `container-layer-graph`（zTree 层级快照）
- **受影响用户**: 所有使用容器任务的用户，当容器内 `onlineServiceJS` 未运行时
- **严重度**: P1 — 核心功能不可用

---

## 2. 根因分析

### 2.1 完整请求链路

```
Browser (:4000) 
  │  X-Trace-Id: web-1782920634053-ztjtzyrygk
  ▼
APISIX (:18081)
  │  路由匹配: container-layer-graph 无专属路由
  │  → 落入 django-default (priority 0, /*)
  ▼
Django (:8001) — CloudComputeViewSet
  │  @action url_path='container-layer-graph'
  │  → forward_container_layer_graph()
  │    ├─ CloudServerConfig.objects.filter(task_id=...).order_by("-updated_at").first()
  │    ├─ server_url = cfg.server_url  (→ http://127.0.0.1:8765)
  │    └─ GET {server_url}/api/layers  ← ❌ Connection Refused (Errno 111)
  ▼
502 Bad Gateway
```

### 2.2 三层根因

| 层 | 问题 | 严重度 | 代码位置 |
|----|------|--------|---------|
| **运行时** | 容器内 `onlineServiceJS`（port 8765）未运行 | 🔴 直接原因 | `go_relayToTrae/src/process.go` |
| **路由** | `container-layer-graph` 无 APISIX 专属路由，走 django-default；而同级 `container-layer-git-commit` 有 priority 890 专属路由 | 🟡 架构不一致 | `taskGateway/routes/routes.yaml:45-51` |
| **可观测性** | 仅 `runall` job 在 Loki 中有日志，Django 业务日志未推送 | 🟡 排查困难 | Loki 可用 jobs: `["runall"]` |

### 2.3 路由不对称证据

| 端点 | APISIX 路由 | 上游服务 | 优先级 |
|------|-----------|---------|--------|
| `container-layer-git-commit` | ✅ `container-git-commit` | `taskContainerGateway` (:8014) | 890 |
| `container-layer-graph` | ❌ 无 | django-default → Django (:8001) | 0 |
| `container-layer-command` | ❌ 无 | django-default → Django (:8001) | 0 |
| `container-layer-git-push` | ❌ 无 | django-default → Django (:8001) | 0 |

> 除 `container-layer-git-commit` 外，所有 `container-layer-*` 端点均走 Django 默认路由。

### 2.4 错误生成代码

- **入口**: `cloud/views/cloud_compute_views.py:173-176` — `get_container_layer_graph`
- **转发逻辑**: `cloud/services/forward_container_layer_graph.py:27-151`
- **错误封装**: `cloud/services/container_forward_debug.py:26-38` — `build_upstream_http_error()`
- **HTTP 客户端**: `cloud/services/container_forward_requests.py` — `container_http_session()` (trust_env=False, threading.local)

---

## 3. 🔍 Trace 日志分析 (traceId: `web-1782920634053-ztjtzyrygk`)

### 查询结果

| 步骤 | 查询条件 | 时间范围 | 结果 |
|------|---------|---------|------|
| 主查询 | `{job=~".+"} \| json \| trace_id` | 1h | 0 条 |
| 回退 1 | `{job=~".+"} \|= "<ID>"` | 24h | 0 条 |
| D1 扩大 | `{job=~".+"} \|= "<ID>"` | 7d | 0 条 |
| D3 大小写 | `\|~ "(?i)<ID>"` | 7d | 0 条 |
| D7 部分匹配 | `\|~ "ztjtzyrygk"` | 7d | 0 条 |

### 诊断结论

- **Loki 数据可用性**: `runall` job 有日志（最近 1h 有 1 条 stream），但仅有编排器自身日志
- **可用 jobs**: `["runall"]` — Django、Go 服务日志**未推送至 Loki**
- **根因**: 业务服务日志采集管道未配置，traceId 在 Loki 中不可检索
- **对本次设计的影响**: 基于代码静态分析 + 错误响应结构进行根因定位；在方案中加入可观测性改进

---

## 4. 设计方案（方案 B + C 组合）

### 4.1 总体思路

```
                    ┌──────────────────────────────┐
                    │     APISIX (:18081)           │
                    │                              │
  container-layer-* │  priority 890                │
  ─────────────────→│  upstream: taskContainerGateway
                    │  auth_mode: token            │
                    └──────────┬───────────────────┘
                               │
                    ┌──────────▼───────────────────┐
                    │  taskContainerGateway (:8014) │
                    │                               │
                    │  handleContainerCompute()     │
                    │  ├─ container-layer-git-commit│
                    │  ├─ container-layer-graph  🆕 │
                    │  └─ ... (其他 action)         │
                    │                               │
                    │  djangoResolveTarget()        │
                    │  ├─ 获取 server_url + token   │
                    │  ├─ 可用性预检 🆕             │
                    │  └─ forwardToOnlineService()  │
                    └──────────┬───────────────────┘
                               │
                    ┌──────────▼───────────────────┐
                    │  onlineServiceJS (:8765)      │
                    │  GET /api/layers              │
                    │  GET /api/jobs                │
                    └──────────────────────────────┘
```

### 4.2 变更 1: APISIX 路由归一化

**文件**: `taskGateway/routes/routes.yaml`

将 `container-git-commit` 路由扩展为 `container-layer` 通配路由，覆盖所有 `container-layer-*` 端点：

```yaml
# 修改前:
- id: container-git-commit
  priority: 890
  uris:
    - /api/tenant/*/workspace/*/task/*/cloud/compute/container-layer-git-commit*
    - /api/tenant/*/workspace/*/task/*/cloud/compute/container-layer-git-commit/*

# 修改后:
- id: container-layer
  priority: 890
  uris:
    - /api/tenant/*/workspace/*/task/*/cloud/compute/container-layer-*
    - /api/tenant/*/workspace/*/task/*/cloud/compute/container-layer-*/*
  upstream: taskContainerGateway
  auth_mode: token
```

**理由**: 所有 `container-layer-*` 端点（graph, git-commit, git-push, command, create, delete, files, ...）具有相同的认证和上游需求，统一路由避免不对称。

### 4.3 变更 2: taskContainerGateway 扩展 handler

**文件**: `taskContainerGateway/src/handlers.go`

在 `handleContainerCompute()` 的 switch 中添加 `container-layer-graph` action:

```go
switch match.Action {
case "container-layer-git-commit":
    // 已有逻辑: POST → forwardToOnlineService
case "container-layer-graph":  // 🆕
    // GET /api/layers + /api/jobs → merge → return
    handleContainerLayerGraph(w, r, match.Scope)
default:
    writeJSON(w, http.StatusNotImplemented, ...)
}
```

新增 `handleContainerLayerGraph()` 函数:

```go
func handleContainerLayerGraph(w http.ResponseWriter, r *http.Request, sc scope) {
    if r.Method != http.MethodGet {
        writeJSON(w, http.StatusMethodNotAllowed, ...)
        return
    }
    ctx := r.Context()
    
    // 1. 验证 session
    _, status, body := djangoValidateSession(ctx, r, sc)
    if status != http.StatusOK { ... }
    
    // 2. 获取容器 target（base_url + token）
    target, status, body := djangoResolveTarget(ctx, sc, "")
    if status != http.StatusOK { ... }
    
    // 3. 可用性预检
    if !checkOnlineServiceReady(ctx, target.BaseURL, target.AccessToken) {
        writeJSON(w, 503, map[string]string{
            "detail": "容器内在线服务未启动，请先启动容器",
            "container_status": "offline",
        })
        return
    }
    
    // 4. 并行请求 /api/layers + /api/jobs
    layersBody, lerr := fetchFromOnlineService(ctx, target, "/api/layers")
    jobsBody, jerr := fetchFromOnlineService(ctx, target, "/api/jobs")
    
    // 5. 合并返回
    ...
}
```

### 4.4 变更 3: Django 端降级 + 可用性预检

**文件**: `cloud/services/forward_container_layer_graph.py`

在 APISIX 路由归一化之后，Django 端的 `forward_container_layer_graph` 仍然作为**降级路径**保留。添加可用性预检：

```python
def _check_container_health(base_url: str, token: str, timeout: float = 5.0) -> bool:
    """TCP 连接预检：容器 onlineServiceJS 是否可达。"""
    import socket
    from urllib.parse import urlparse
    try:
        parsed = urlparse(base_url)
        host = parsed.hostname
        port = parsed.port or 8765
        with socket.create_connection((host, port), timeout=timeout):
            return True
    except OSError:
        return False

def forward_container_layer_graph(request, tenant_id, workspace_id, task_id):
    # ... 现有 CloudServerConfig 查询 ...
    
    # 🆕 可用性预检
    if not _check_container_health(base):
        return Response(
            {
                "detail": "容器内在线服务未启动，请先启动容器",
                "container_status": "offline",
                "server_url": base,
                "suggestion": "请前往任务详情页启动容器，或使用 relay-to-trae/start 启动 onlineServiceJS",
            },
            status=status.HTTP_503_SERVICE_UNAVAILABLE,
        )
    
    # ... 原有转发逻辑 ...
```

### 4.5 变更 4: 可观测性增强（建议）

- **Loki 采集**: 为 Django (`saas-backend`) 和 Go 服务添加 promtail scrape config，使业务日志推送至 Loki
- **结构化日志**: 确保所有 `container-layer-*` 请求在 taskContainerGateway 层输出包含 `trace_id` 的结构化日志

---

## 5. 🏛️ 架构变更影响

- **迭代版本**: v3 🎯 target
- **迭代名称**: container-layer-graph 502 修复 — 路由归一化 + 可用性预检
- **变更明细**:
  - 🟡 [MODIFIED] APISIX 路由: `container-git-commit` → `container-layer`（扩大匹配范围）
  - 🟡 [MODIFIED] taskContainerGateway: 新增 `container-layer-graph` action handler
  - 🟡 [MODIFIED] Django `forward_container_layer_graph`: 新增 TCP 可用性预检
  - 🟢 [NEW] `checkOnlineServiceReady`: 容器在线服务健康检查

---

## 6. 领域概念清单

| 概念 | 类型 | 说明 |
|------|------|------|
| `CloudServerConfig` | Entity | 任务容器服务器配置，包含 `server_url` + `container_access_token` |
| `ContainerLayer` | Entity | 容器内代码变更层，通过 `onlineServiceJS /api/layers` 管理 |
| `ContainerJob` | Entity | 容器内执行任务，通过 `onlineServiceJS /api/jobs` 管理 |
| `onlineServiceJS` | Infrastructure Service | 容器内运行的 Node.js 在线服务（port 8765），提供 layers/jobs API |
| `taskContainerGateway` | Application Service | Go 网关（port 8014），统一代理容器出站请求 |
| **Bounded Context**: `cloud` (云平台与容器)、`task` (任务协作) |

---

## 7. 价值流影响分析

| 维度 | 影响 |
|------|------|
| 受影响流 | `cloud-compute`（cloud/tests/test_cloud_compute.py） |
| 新流需求 | `container-layer-routing` — 覆盖所有 container-layer-* 端点的路由一致性 |
| 字段变更 | 无数据库字段变更 |
| 测试影响 | 新增: `taskContainerGateway` handler 单元测试; 修改: `test_relay_to_trae_proxy.py`（路由变更后更新 mock） |
| 跨流依赖 | 无新增跨流依赖 |

---

## 8. 实施计划

### Increment 1: APISIX 路由归一化（最小改动，最高影响）

- [ ] 1.1 修改 `taskGateway/routes/routes.yaml`：将 `container-git-commit` 扩展为 `container-layer` 通配路由
- [ ] 1.2 同步更新 `taskGateway/apisix/apisix.yaml`
- [ ] 1.3 验证: 所有 `container-layer-*` 端点正确路由至 `taskContainerGateway`

### Increment 2: taskContainerGateway 扩展

- [ ] 2.1 在 `handlers.go` 中添加 `container-layer-graph` action case
- [ ] 2.2 实现 `handleContainerLayerGraph()` — GET layers + jobs 合并逻辑
- [ ] 2.3 添加 `checkOnlineServiceReady()` TCP 预检函数
- [ ] 2.4 编写单元测试 `handlers_test.go`

### Increment 3: Django 降级增强

- [ ] 3.1 在 `forward_container_layer_graph.py` 中添加 `_check_container_health()`
- [ ] 3.2 更新错误响应格式（503 + suggestion）
- [ ] 3.3 更新 Django 端测试

### Increment 4: 可观测性（可选）

- [ ] 4.1 为 Django/Go 服务配置 promtail scrape
- [ ] 4.2 确保 traceId 在各层透传

---

## 9. 风险与回滚

| 风险 | 缓解 |
|------|------|
| taskContainerGateway handler bug 导致所有 container-layer-* 端点不可用 | 先在 staging 验证；保留 APISIX 路由优先级可快速回滚 |
| TCP 预检超时影响用户体验 | 设置 3-5s 短超时；预检失败时降级为原 502 行为 |
| Go 端合并 layers+jobs 逻辑与 Python 端不一致 | 与现有 `forward_container_layer_graph` 对比测试；JSON schema 校验 |

---

## 10. 相关文件索引

| 文件 | 用途 |
|------|------|
| `taskGateway/routes/routes.yaml` | APISIX 路由定义（需修改） |
| `taskGateway/apisix/apisix.yaml` | APISIX declarative config（需同步） |
| `taskContainerGateway/src/handlers.go` | Go 网关 handler（需扩展） |
| `taskContainerGateway/src/django_client.go` | Go 端 Django 内部 API 调用 + forwardToOnlineService |
| `task2app/Saas_project/cloud/views/cloud_compute_views.py` | Django ViewSet（container-layer-graph entry） |
| `task2app/Saas_project/cloud/services/forward_container_layer_graph.py` | Django 转发逻辑（需添加预检） |
| `task2app/Saas_project/cloud/services/container_forward_debug.py` | 错误封装 `build_upstream_http_error` |
| `task2app/Saas_project/cloud/services/container_forward_requests.py` | HTTP Session（trust_env=False） |
| `go_relayToTrae/src/process.go` | onlineServiceJS 进程管理 |
| `trae-agent/onlineServiceJS/` | 容器内在线服务（提供 /api/layers） |
