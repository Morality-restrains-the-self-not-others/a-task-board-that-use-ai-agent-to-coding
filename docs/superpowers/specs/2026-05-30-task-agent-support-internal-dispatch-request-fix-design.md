# 设计文档：taskAgentSupport internal_dispatch Request 类型修复

**日期**：2026-05-30  
**状态**：已实现（0-auto-flow Step 7）  
**关联现象**：relayToTrae 直启 `exchange-refresh` 返回 `HTTP 500 {"detail":"internal error"}`，go-relay `/v1/start` 连带 400  
**关联规格**：`docs/superpowers/specs/2026-05-29-relay-to-trae-startup-reliability-design.md`（T2 已做 503 分支，但未覆盖本缺陷）

---

## 1. 背景与已验证根因

### 1.1 用户可见故障

任务详情 `?relayToTrae=true` 点击「启动」：

```
[django relay start] rejected request_id=... status=400
[relayToTrae] token-exchange: exchange-refresh attempt 1/2 failed: HTTP 500: {"detail": "internal error"}
[relayToTrae] start failed: token exchange failed
```

### 1.2 Grafana / Loki 证据

`saas-backend` 日志 `task_agent_support internal dispatch failed action=exchange-refresh` 的 `exc` 字段为：

```
AssertionError: The `request` argument must be an instance of `django.http.HttpRequest`,
not `rest_framework.request.Request`.
```

本地对 `:8011` 复现：POST `exchange-refresh` 仍返回 `{"detail":"internal error"}`。

### 1.3 代码根因

`cloud/task_agent_support/internal_dispatch.py` 中 `_build_drf_request()` 将 `RequestFactory` 产出的 **Django HttpRequest** 预先包装为 **DRF Request**，再传给 `@api_view` 装饰的 inbound 视图。视图 `dispatch()` 会再次 `initialize_request()`，导致双重包装并触发 AssertionError；`except Exception` 统一映射为 500 `internal error`。

---

## 2. 目标与非目标

### 2.1 目标

1. `exchange-refresh`、`refresh-access`、`register-reachability`、`heartbeat`、`relay-status-push` 经 internal API 转发时**不再**因 Request 类型错误返回 500。
2. 保留既有 T2 行为：`OperationalError`（SQLite locked）→ 503 + `RELAY_DOWNSTREAM_BUSY`（若已在分支中，合并到同一修复 PR）。
3. 增加 pytest：internal `exchange-refresh` 在合法 envelope 下**不**返回 500 `internal error`（可用 mock 或最小 DB 夹具）。
4. 部署说明：修复后须**重启**占用 8001 的 saas-backend（避免热重载 `port already in use` 导致旧进程继续服务）。

### 2.2 非目标

- 不改造 taskAgentSupport Go 网关路由。
- 不修改 go-relay token-exchange 重试策略（换票成功后自然恢复）。
- 不在本轮实现 Phase 2 batch internal API。
- 不处理令牌业务 401/403（修复后才会暴露真实鉴权错误）。

---

## 3. 价值流影响

| 流 | 步骤 | 影响 |
|----|------|------|
| `task-detail-runtime-relay` | relay 直启 / token-exchange | internal 转发恢复后 go-relay 可完成换票并启动 onlineServiceJS |
| `relay-token-audit-full-chain-eventization` | exchange-refresh 审计 | 500 消失后审计事件可正常写入 |
| `relay-register-token-exchange-guard` | 换票窗口 TEIP | 依赖 exchange-refresh 成功 |

**字段**：`cloud_cloudserverconfig.container_access_token`、`container_refresh_token`、`business_api_endpoint`（换票成功后写入）。

---

## 4. 领域概念（供 DDD）

| 概念 | 说明 |
|------|------|
| `TaskAgentSupportInternalDispatch` | taskAgentSupport → Django internal 信封转发 |
| `ContainerTokenExchange` | exchange-refresh 领域服务（已有） |
| `RelayStartupSession` | relay 直启编排（不变） |

---

## 5. 方案对比

### 方案 A — 传入原始 HttpRequest（推荐）

- `_build_drf_request` 改名为 `_build_inbound_http_request`，返回 `django_request`（不包 `Request()`）。
- `view_func(django_request, tenant_id=..., ...)`，由 DRF 视图自行 `initialize_request`。

**优点**：一行级修复、与 DRF 约定一致、覆盖全部 action。  
**缺点**：无。

### 方案 B — 调用视图底层逻辑函数

- 跳过 HTTP 层，直接 import service 层函数。

**优点**：无 Request 问题。  
**缺点**：重复路由/鉴权逻辑，与「复用 inbound 视图」设计背离。

### 方案 C — APIRequestFactory

- 使用 `rest_framework.test.APIRequestFactory`。

**优点**：测试友好。  
**缺点**：生产路径仍应使用 Django HttpRequest + 视图入口，改动大于 A。

**推荐方案 A**。

---

## 6. 详细设计（方案 A）

### 6.1 `internal_dispatch.py`

```python
def _build_inbound_http_request(body: dict[str, Any], trace_id: str):
    django_request = _request_factory.post('/', data=json.dumps(body or {}), content_type='application/json')
    if trace_id:
        django_request.META['HTTP_X_TRACE_ID'] = trace_id
    return django_request

# dispatch 内：
http_request = _build_inbound_http_request(body, trace_id)
drf_response = view_func(http_request, tenant_id=..., workspace_id=..., task_id=...)
```

保留 `OperationalError` → 503 分支（与 2026-05-29 设计一致）；其余未预期异常仍 `logger.exception` + `error_code=INTERNAL_DISPATCH_ERROR`。

### 6.2 测试

新建或扩展 `tests/test_task_agent_support_internal_dispatch.py`：

- `exchange-refresh`：合法 envelope + secret → status **≠ 500**（可为 401/403/400，取决于 token 夹具）。
- 回归：`register-reachability`、`heartbeat` 各一条 smoke。

### 6.3 验收

```bash
cd task2app/Saas_project && pytest tests/test_task_agent_support_internal_dispatch.py -q
# 重启 saas-backend 后：
curl -X POST http://127.0.0.1:8011/api/tenant/.../task/.../cloud/server-container-token/exchange-refresh/ \
  -H 'Content-Type: application/json' -d '{"access_token":"...","business_api_endpoint":"http://127.0.0.1:8765/api"}'
# 期望：非 {"detail":"internal error"} 的 500
```

relay 直启：任务 `848546827193511936` 点击启动，Grafana trace 内 exchange-refresh 为 200/401 等可诊断状态，非裸 500。

---

## 7. 风险与缓解

| 风险 | 缓解 |
|------|------|
| 旧 Django 进程未重启 | 文档 + Step 9 验收注明重启 runAll saas-backend |
| 修复后暴露真实 401（无效预埋 token） | 属预期；用户需重新 mockStart / 刷新 access |
| 与 T2 503 分支合并冲突 | 单文件小 diff，一并审查 |

---

## 8. 自检

- [x] 根因与 Loki 堆栈一致  
- [x] 范围限定 internal_dispatch，无无关重构  
- [x] 价值流 `task-detail-runtime-relay` 已标注  
- [x] 非目标明确（不含 batch API / go 网关）
