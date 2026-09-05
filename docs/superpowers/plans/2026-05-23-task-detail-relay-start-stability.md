# Task Detail Relay Start Stability Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Eliminate user-visible startup instability on task-detail relay start by enforcing stable backend error contracts, frontend state/error convergence, and regression tests.

**Architecture:** Keep the existing startup flow (`env-prepare -> exchange-refresh -> refresh-access -> relay start`) and harden boundaries. Convert business failures into deterministic 4xx + `error_code` + `trace_id`, add frontend error-code mapping with explicit state transitions, and lock behavior with backend + Playwright regression coverage.

**Tech Stack:** Django + DRF, Python pytest, Vue composables/components, Playwright, relay proxy services.

---

### Task 1: Backend Error Contract Foundation (Token Exchange/Refresh)

**Files:**
- Modify: `task2app/Saas_project/cloud/views/container_runtime_token_views.py`
- Create: `task2app/Saas_project/core/outbound/errors.py` (only if reusable error enum/helper is absent)
- Test: `task2app/Saas_project/tests/test_container_runtime_tokens.py`

- [ ] **Step 1: Write failing backend contract tests (`error_code` + `trace_id`)**

```python
@pytest.mark.django_db
def test_exchange_refresh_returns_structured_business_error():
    # setup: stale/invalid access path
    client = APIClient()
    resp = client.post(exchange_url, {"access_token": "invalid", "business_api_endpoint": TEST_BUSINESS_API_ENDPOINT}, format="json")
    assert resp.status_code == 401
    assert resp.data["error_code"] == "TOKEN_ACCESS_INVALID"
    assert isinstance(resp.data.get("trace_id"), str) and resp.data["trace_id"]
```

- [ ] **Step 2: Run test to verify it fails first**

Run: `cd task2app/Saas_project && pytest tests/test_container_runtime_tokens.py -k "structured_business_error or trace_id" -v`  
Expected: FAIL because current response does not consistently include `error_code` and/or `trace_id`.

- [ ] **Step 3: Implement minimal response normalization helper in token views**

```python
def _token_error(detail: str, *, code: str, status_code: int, request: Request) -> Response:
    trace_id = str(getattr(request, "trace_id", "") or request.headers.get("X-Trace-Id") or "").strip()
    return Response(
        {"detail": detail, "error_code": code, "trace_id": trace_id},
        status=status_code,
    )
```

- [ ] **Step 4: Replace direct business `Response(...)` branches with `_token_error(...)`**

```python
if cfg is None:
    return _token_error(
        "无效的 access_token",
        code="TOKEN_ACCESS_INVALID",
        status_code=status.HTTP_401_UNAUTHORIZED,
        request=request,
    )
```

- [ ] **Step 5: Run targeted backend tests and confirm green**

Run: `cd task2app/Saas_project && pytest tests/test_container_runtime_tokens.py -k "exchange_refresh or refresh_access" -v`  
Expected: PASS for new contract assertions and existing lifecycle behaviors.

- [ ] **Step 6: Commit Task 1**

```bash
cd task2app
git add Saas_project/cloud/views/container_runtime_token_views.py Saas_project/tests/test_container_runtime_tokens.py
git commit -m "fix: standardize token exchange/refresh business error contract"
```

### Task 2: Relay Proxy Contract Propagation and Classification

**Files:**
- Modify: `task2app/Saas_project/cloud/services/relay_to_trae_proxy.py`
- Test: `task2app/Saas_project/tests/test_relay_to_trae_proxy.py`

- [ ] **Step 1: Write failing tests for proxy passthrough of `error_code` and `trace_id`**

```python
def test_relay_start_propagates_business_error_code(client, mocker):
    mocker.patch("cloud.services.relay_to_trae_proxy._relay_http_request", return_value=_mock_resp(
        status=403, json_body={"detail": "token mismatch", "error_code": "TOKEN_SCOPE_MISMATCH", "trace_id": "t-1"}
    ))
    resp = client.post(start_url, payload, format="json")
    assert resp.status_code in (403, 502)
    assert "TOKEN_SCOPE_MISMATCH" in str(resp.data)
```

- [ ] **Step 2: Run proxy tests to verify failure**

Run: `cd task2app/Saas_project && pytest tests/test_relay_to_trae_proxy.py -k "error_code or trace_id" -v`  
Expected: FAIL where proxy currently drops or rewrites structured fields.

- [ ] **Step 3: Implement proxy error passthrough for structured downstream business failures**

```python
def _relay_error_response(resp: requests.Response) -> Response:
    try:
        body = resp.json()
    except Exception:
        body = {}
    detail = str(body.get("detail") or body.get("message") or "") or f"relayToTrae 返回 HTTP {resp.status_code}"
    payload = {"status": "error", "message": detail}
    if body.get("error_code"):
        payload["error_code"] = body["error_code"]
    if body.get("trace_id"):
        payload["trace_id"] = body["trace_id"]
    return Response(payload, status=status.HTTP_502_BAD_GATEWAY if resp.status_code >= 500 else resp.status_code)
```

- [ ] **Step 4: Ensure internal exception paths also emit stable `error_code`**

```python
except Exception as exc:
    return Response(
        {"status": "error", "message": f"调用 relayToTrae 失败：{exc}", "error_code": "RELAY_DOWNSTREAM_UNAVAILABLE"},
        status=status.HTTP_502_BAD_GATEWAY,
    )
```

- [ ] **Step 5: Run relay proxy tests and verify pass**

Run: `cd task2app/Saas_project && pytest tests/test_relay_to_trae_proxy.py -v`  
Expected: PASS, including new contract-propagation assertions.

- [ ] **Step 6: Commit Task 2**

```bash
cd task2app
git add Saas_project/cloud/services/relay_to_trae_proxy.py Saas_project/tests/test_relay_to_trae_proxy.py
git commit -m "fix: propagate structured relay startup error codes"
```

### Task 3: Frontend Startup State Convergence by Error Code

**Files:**
- Modify: `task2app/front_project/app/src/composables/useTaskDetail.js`
- Modify: `task2app/front_project/app/src/components/ServerConfig.logic.vue`
- Modify: `task2app/front_project/app/src/composables/taskDetail/establishSSEConnection.js`
- Test: `task2app/playwright/front_project/tests/TaskDetail.relay-to-trae-start-stop-button.playwright.test.js`
- Test: `task2app/playwright/front_project/tests/TaskDetail.relay-to-trae-no-invalid-access-token.playwright.test.js`

- [ ] **Step 1: Write failing Playwright assertion for startup error mapping**

```javascript
await expect
  .poll(async () => await page.locator('[data-testid="relay-to-trae-status-row"]').innerText())
  .not.toContain('获取任务详情失败（HTTP 500）');
```

- [ ] **Step 2: Run target Playwright test to confirm failure mode**

Run: `cd task2app/playwright && npx playwright test -c front_project/playwright.config.js front_project/tests/TaskDetail.relay-to-trae-start-stop-button.playwright.test.js --project=chromium`  
Expected: FAIL on current unstable/error-text behavior.

- [ ] **Step 3: Add frontend error-code mapping utility**

```javascript
const RELAY_START_ERROR_MESSAGES = {
  TOKEN_ACCESS_EXPIRED: '凭证已过期，正在准备重试',
  TOKEN_ACCESS_INVALID: '凭证无效，请重新准备启动环境',
  TOKEN_SCOPE_MISMATCH: '任务上下文不一致，请刷新页面后重试',
  TOKEN_EXCHANGE_ALREADY_DONE: '检测到已完成换票，正在走补偿流程',
  BUSINESS_API_ENDPOINT_INVALID: '业务端点配置异常，请检查配置后重试',
  RELAY_DOWNSTREAM_UNAVAILABLE: 'relay 服务暂不可用，请稍后重试',
};
```

- [ ] **Step 4: Use mapped message and explicit state transitions in startup handlers**

```javascript
function resolveRelayStartError(errorCode, fallbackMessage) {
  return RELAY_START_ERROR_MESSAGES[errorCode] || fallbackMessage || '启动失败，请重试';
}

setRelayState('degraded');
setRelayError(resolveRelayStartError(error_code, message));
```

- [ ] **Step 5: Ensure SSE reconciliation can recover `starting -> running` after retriable failures**

```javascript
if (event?.relay_payload?.running) {
  setRelayState('running');
  setRelayError('');
}
```

- [ ] **Step 6: Re-run Playwright subset and verify pass**

Run: `cd task2app/playwright && npx playwright test -c front_project/playwright.config.js front_project/tests/TaskDetail.relay-to-trae-start-stop-button.playwright.test.js front_project/tests/TaskDetail.relay-to-trae-no-invalid-access-token.playwright.test.js --project=chromium`  
Expected: PASS with no page-level 500 text and stable running transition.

- [ ] **Step 7: Commit Task 3**

```bash
cd task2app
git add front_project/app/src/composables/useTaskDetail.js front_project/app/src/components/ServerConfig.logic.vue front_project/app/src/composables/taskDetail/establishSSEConnection.js playwright/front_project/tests/TaskDetail.relay-to-trae-start-stop-button.playwright.test.js playwright/front_project/tests/TaskDetail.relay-to-trae-no-invalid-access-token.playwright.test.js
git commit -m "fix: converge relay startup UI state by error code"
```

### Task 4: Observability and CI Guardrail Completion

**Files:**
- Modify: `task2app/Saas_project/cloud/tests/test_container_token_lifecycle_audit_events.py`
- Modify: `task2app/Saas_project/cloud/view_test/test_relay_token_audit_status_push_view.py`
- Modify: `task2app/.github/workflows/` (or existing CI pipeline file used by repo)
- Modify: `value-stream.yaml` (only if stream step/test mapping needs explicit update)

- [ ] **Step 1: Add failing audit-chain test for failed startup with trace continuity**

```python
def test_relay_start_failure_audit_chain_contains_trace_id():
    # trigger relay start failure
    events = ContainerTokenAuditEvent.objects.filter(task_id=task_id).order_by("created_at")
    assert any(e.event_type == "relay_start_failed" for e in events)
    assert all(e.trace_id for e in events if e.event_type.startswith("relay_start"))
```

- [ ] **Step 2: Run audit-related tests and verify failure first**

Run: `cd task2app/Saas_project && pytest cloud/tests/test_container_token_lifecycle_audit_events.py cloud/view_test/test_relay_token_audit_status_push_view.py -v`  
Expected: FAIL for missing or inconsistent trace continuity assertions.

- [ ] **Step 3: Implement minimal audit emission/completion to satisfy test**

```python
append_container_token_audit_event(
    tenant_id=tenant,
    workspace_id=wid,
    task_id=tid,
    event_type=TokenAuditEventTypes.RELAY_START_FAILED,
    source_component="django",
    trace_id=request_id,
    error_code=error_code,
    error_detail=detail,
)
```

- [ ] **Step 4: Add CI required commands for backend+frontend relay startup regressions**

```yaml
- name: Relay token contract tests
  run: cd task2app/Saas_project && pytest tests/test_container_runtime_tokens.py tests/test_relay_to_trae_proxy.py -v

- name: Relay startup playwright smoke
  run: cd task2app/playwright && npx playwright test -c front_project/playwright.config.js front_project/tests/TaskDetail.relay-to-trae-start-stop-button.playwright.test.js --project=chromium
```

- [ ] **Step 5: Run complete targeted verification**

Run: `cd task2app/Saas_project && pytest tests/test_container_runtime_tokens.py tests/test_relay_to_trae_proxy.py cloud/tests/test_container_token_lifecycle_audit_events.py -v && cd ../playwright && npx playwright test -c front_project/playwright.config.js front_project/tests/TaskDetail.relay-to-trae-start-stop-button.playwright.test.js --project=chromium`  
Expected: PASS for all selected verification suites.

- [ ] **Step 6: Commit Task 4**

```bash
cd task2app
git add Saas_project/cloud/tests/test_container_token_lifecycle_audit_events.py Saas_project/cloud/view_test/test_relay_token_audit_status_push_view.py .github/workflows
git commit -m "test: enforce relay startup stability guardrails in CI"
```

### Task 5: Final Verification and Release Notes

**Files:**
- Modify: `docs/runbooks/` (or existing runbook location for startup troubleshooting)
- Modify: `docs/superpowers/specs/2026-05-23-task-detail-relay-start-stability-design.md` (only if implementation deltas require note)

- [ ] **Step 1: Execute end-to-end manual smoke from task-detail with real local flow**

Run: `open "http://localhost:4000/tenant/<tenant>/workspace/<workspace>/task-detail/<task>/?relayToTrae=true"` then click start once  
Expected: No page-level HTTP 500; state converges to running or deterministic error message with trace id.

- [ ] **Step 2: Capture troubleshooting commands into runbook**

```bash
curl -s http://127.0.0.1:8797/health
curl -s http://127.0.0.1:8797/v1/status
pytest tests/test_container_runtime_tokens.py -k exchange_refresh -v
```

- [ ] **Step 3: Add release-risk note and rollback switches**

```markdown
- Rollback order: frontend error mapping -> proxy passthrough -> token contract strictness
- Verify trace_id continuity before and after rollback
```

- [ ] **Step 4: Final commit**

```bash
cd task2app
git add docs
git commit -m "docs: add relay startup stability verification and rollback notes"
```
