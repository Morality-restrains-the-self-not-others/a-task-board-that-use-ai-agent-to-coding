# Relay Two-Step Async Start Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** 将 relay 直接启动流程改为“先 token-init，再异步 start(202 accepted)”并通过既有 SSE 收敛最终运行状态。

**Architecture:** SaaS 后端新增 `token-init` 接口做令牌初始化，`start` 接口只做异步受理与派发；后台线程调用 relay `/v1/start` 并把失败信息通过既有 `relay_to_trae_status` SSE 通道回传。前端改为两步请求，不再依赖 `start` 同步返回 `ui_url`。

**Tech Stack:** Django REST Framework, Vue 3, pytest, existing relay/status-push SSE pipeline

---

## DDD Structure Check (Backend)
- 领域优先顺序：先定义/扩展领域事件枚举，再改应用服务与接口层。
- 分层约束：
  - Domain: `cloud/domain/events/*`
  - Application Service: `cloud/services/relay_to_trae_proxy.py`
  - Interface: `cloud/views/cloud_compute_views.py`
- 仓储与基础设施不新增耦合；继续复用现有 `CloudServerConfig` 与审计 app service。

### Task 1: Define Domain-Level Event Contract First

**Files:**
- Modify: `task2app/Saas_project/cloud/domain/events/token_audit_events.py`
- Modify: `task2app/Saas_project/cloud/domain/events/__init__.py`
- Test: `task2app/Saas_project/tests/test_relay_to_trae_proxy.py`

- [ ] **Step 1: 写失败测试（新事件在链路中可被引用）**

```python
def test_relay_token_init_events_available():
    from cloud.domain.events import TokenAuditEventTypes
    assert TokenAuditEventTypes.RELAY_TOKEN_INIT_ATTEMPTED
    assert TokenAuditEventTypes.RELAY_TOKEN_INIT_SUCCEEDED
    assert TokenAuditEventTypes.RELAY_TOKEN_INIT_FAILED
```

- [ ] **Step 2: 运行测试并确认失败**

Run: `../activate_env.sh unit -- pytest tests/test_relay_to_trae_proxy.py -k "token_init_events_available" -v`  
Expected: `AttributeError` or missing enum failure

- [ ] **Step 3: 最小实现枚举与导出**

```python
class TokenAuditEventTypes:
    RELAY_TOKEN_INIT_ATTEMPTED = "relay_token_init_attempted"
    RELAY_TOKEN_INIT_SUCCEEDED = "relay_token_init_succeeded"
    RELAY_TOKEN_INIT_FAILED = "relay_token_init_failed"
```

- [ ] **Step 4: 运行测试并确认通过**

Run: `../activate_env.sh unit -- pytest tests/test_relay_to_trae_proxy.py -k "token_init_events_available" -v`  
Expected: PASS

- [ ] **Step 5: Commit**

```bash
git add task2app/Saas_project/cloud/domain/events/token_audit_events.py task2app/Saas_project/cloud/domain/events/__init__.py task2app/Saas_project/tests/test_relay_to_trae_proxy.py
git commit -m "feat: add relay token-init audit event contract"
```

### Task 2: Add Token-Init API in Relay Proxy Service

**Files:**
- Modify: `task2app/Saas_project/cloud/services/relay_to_trae_proxy.py`
- Modify: `task2app/Saas_project/cloud/services/__init__.py`
- Test: `task2app/Saas_project/tests/test_relay_to_trae_proxy.py`

- [ ] **Step 1: 写失败测试（token-init 返回 200 + token_initialized）**

```python
def test_relay_to_trae_token_init_returns_ok(monkeypatch):
    # mock runtime env builder and assert response payload includes token_initialized
    ...
```

- [ ] **Step 2: 运行测试并确认失败**

Run: `../activate_env.sh unit -- pytest tests/test_relay_to_trae_proxy.py -k "token_init_returns_ok" -v`  
Expected: missing function/route assertion failure

- [ ] **Step 3: 实现 token-init service 方法**

```python
def relay_to_trae_token_init(request, tenant_id=None, workspace_id=None, task_id=None):
    # validate context, build runtime env, audit attempted/succeeded/failed
    return Response({"status": "ok", "token_initialized": True, "task_id": tid, "env_preview": {...}})
```

- [ ] **Step 4: 运行测试并确认通过**

Run: `../activate_env.sh unit -- pytest tests/test_relay_to_trae_proxy.py -k "token_init" -v`  
Expected: PASS

- [ ] **Step 5: Commit**

```bash
git add task2app/Saas_project/cloud/services/relay_to_trae_proxy.py task2app/Saas_project/cloud/services/__init__.py task2app/Saas_project/tests/test_relay_to_trae_proxy.py
git commit -m "feat: add relay token-init endpoint service"
```

### Task 3: Convert Start to Async Accepted Flow

**Files:**
- Modify: `task2app/Saas_project/cloud/services/relay_to_trae_proxy.py`
- Test: `task2app/Saas_project/tests/test_relay_to_trae_proxy.py`
- Test: `task2app/Saas_project/tests/test_relay_to_trae_status.py`

- [ ] **Step 1: 写失败测试（start 返回 202 accepted）**

```python
def test_relay_to_trae_start_returns_accepted(monkeypatch):
    # expect status_code == 202 and response contains request_id/task_id
    ...
```

- [ ] **Step 2: 运行测试并确认失败**

Run: `../activate_env.sh unit -- pytest tests/test_relay_to_trae_proxy.py -k "start_returns_accepted" -v`  
Expected: returns 200 currently

- [ ] **Step 3: 最小实现异步派发**

```python
def relay_to_trae_start(...):
    request_id = uuid4().hex
    threading.Thread(target=_dispatch_relay_start_async, kwargs={...}, daemon=True).start()
    return Response({"status": "accepted", "request_id": request_id, "task_id": tid}, status=202)
```

- [ ] **Step 4: 实现异步失败 SSE 回传与审计**

```python
publish_relay_to_trae_status_sse(task_id=tid, relay_payload={"error": detail, "running": False, "online_service_up": False})
```

- [ ] **Step 5: 运行测试并确认通过**

Run: `../activate_env.sh unit -- pytest tests/test_relay_to_trae_proxy.py tests/test_relay_to_trae_status.py -k "relay_to_trae_start or relay_to_trae_status" -v`  
Expected: PASS

- [ ] **Step 6: Commit**

```bash
git add task2app/Saas_project/cloud/services/relay_to_trae_proxy.py task2app/Saas_project/tests/test_relay_to_trae_proxy.py task2app/Saas_project/tests/test_relay_to_trae_status.py
git commit -m "feat: make relay start async accepted with failure SSE fallback"
```

### Task 4: Wire New API Route in Cloud Compute ViewSet

**Files:**
- Modify: `task2app/Saas_project/cloud/views/cloud_compute_views.py`
- Test: `task2app/Saas_project/tests/test_relay_to_trae_proxy.py`

- [ ] **Step 1: 写失败测试（view 层暴露 token-init action）**

```python
def test_cloud_compute_exposes_relay_token_init_route():
    # reverse or route mapping assertion
    ...
```

- [ ] **Step 2: 运行测试并确认失败**

Run: `../activate_env.sh unit -- pytest tests/test_relay_to_trae_proxy.py -k "token_init_route" -v`  
Expected: route not found

- [ ] **Step 3: 添加 action 与 service 调用**

```python
@action(detail=False, methods=['post'], url_path='relay-to-trae/token-init')
def post_relay_to_trae_token_init(self, request, tenant_id=None, workspace_id=None, task_id=None):
    return relay_to_trae_token_init(request, tenant_id, workspace_id, task_id)
```

- [ ] **Step 4: 运行测试并确认通过**

Run: `../activate_env.sh unit -- pytest tests/test_relay_to_trae_proxy.py -k "token_init_route" -v`  
Expected: PASS

- [ ] **Step 5: Commit**

```bash
git add task2app/Saas_project/cloud/views/cloud_compute_views.py task2app/Saas_project/tests/test_relay_to_trae_proxy.py
git commit -m "feat: expose relay token-init API route"
```

### Task 5: Update Frontend to Two-Step Start

**Files:**
- Modify: `task2app/front_project/app/src/components/ServerConfig.logic.vue`
- Modify: `task2app/front_project/app/src/utils/relayToTraeUtils.js`
- Test: `task2app/front_project/app/src/utils/relayToTraeUtils.test.js`

- [ ] **Step 1: 写失败测试（two-step payload + start accepted）**

```javascript
it('builds token-init then start payload for relay direct start', () => {
  // assert payload contains context ids and env origin keys
})
```

- [ ] **Step 2: 运行测试并确认失败**

Run: `npm run test -- "src/utils/relayToTraeUtils.test.js"`  
Expected: missing helper behavior

- [ ] **Step 3: 实现前端两步调用与 UI 状态切换**

```javascript
await apiFetch(relayToTraeApiUrl('token-init/'), {...})
const startResp = await apiFetch(relayToTraeApiUrl('start/'), {...})
if (startResp.status === 202) relayToTraeMessage.value = '已受理启动请求，等待状态推送...'
```

- [ ] **Step 4: 运行前端测试并确认通过**

Run: `npm run test -- "src/utils/relayToTraeUtils.test.js"`  
Expected: PASS

- [ ] **Step 5: Commit**

```bash
git add task2app/front_project/app/src/components/ServerConfig.logic.vue task2app/front_project/app/src/utils/relayToTraeUtils.js task2app/front_project/app/src/utils/relayToTraeUtils.test.js
git commit -m "feat: switch relay direct start to token-init plus async start"
```

### Task 6: Full Verification and Regression

**Files:**
- Test: `task2app/Saas_project/tests/test_relay_to_trae_proxy.py`
- Test: `task2app/Saas_project/tests/test_relay_to_trae_status.py`
- Test: `task2app/front_project/app/src/utils/relayToTraeUtils.test.js`

- [ ] **Step 1: 运行后端关键回归**

Run: `cd task2app/Saas_project && ../activate_env.sh unit -- pytest tests/test_relay_to_trae_proxy.py tests/test_relay_to_trae_status.py -v`  
Expected: all pass

- [ ] **Step 2: 运行前端关键回归**

Run: `cd task2app/front_project/app && npm run test -- "src/utils/relayToTraeUtils.test.js"`  
Expected: all pass

- [ ] **Step 3: 手工验证启动不再 pending**

```text
打开 task-detail relayDirect 页面 -> 点击“启动”
期望：先触发 token-init，再 start 返回 202；UI 显示“启动中”，最终由 SSE 变为运行中或失败。
```

- [ ] **Step 4: Commit**

```bash
git add -A
git commit -m "test: verify relay two-step async start regressions"
```

