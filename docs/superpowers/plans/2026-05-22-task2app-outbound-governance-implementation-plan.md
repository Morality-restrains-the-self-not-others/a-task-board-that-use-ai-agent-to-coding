# Task2app Outbound Governance Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** 在 task2app 中落地可交付的出站调用治理：内网同步调用 1 秒硬超时、外网调用异步化并通过事件推送回传结果。

**Architecture:** 以 `cloud/domain` 作为业务契约层（实体/值对象/仓储接口/事件/领域服务），应用与基础设施层按契约实现。调用入口统一进入策略路由，`internal_sync` 强制超时保护，`external_async` 经任务状态机驱动并复用 SSE 事件通道回传。按 value stream 增量逐步扩展到更多外网链路。

**Tech Stack:** Python 3.9, Django, pytest, requests, Kafka event publisher (`SSE_MESSAGE`), 现有 `cloud/domain` DDD 分层。

---

## File Structure

- Domain contracts (already defined, first-class dependency):
  - `task2app/Saas_project/cloud/domain/entities/outbound_dispatch_job.py`
  - `task2app/Saas_project/cloud/domain/value_objects/outbound_call_route.py`
  - `task2app/Saas_project/cloud/domain/value_objects/internal_sync_timeout_policy.py`
  - `task2app/Saas_project/cloud/domain/repositories/outbound_dispatch_job_repository.py`
  - `task2app/Saas_project/cloud/domain/events/internal_sync_call_timed_out.py`
  - `task2app/Saas_project/cloud/domain/events/external_call_job_queued.py`
  - `task2app/Saas_project/cloud/domain/events/external_call_job_completed.py`
  - `task2app/Saas_project/cloud/domain/services/outbound_call_governance_service.py`
- Infrastructure + application integration:
  - `task2app/Saas_project/core/outbound/gateway.py`
  - `task2app/Saas_project/core/outbound/errors.py`
  - `task2app/Saas_project/core/outbound/external_async.py`
  - `task2app/Saas_project/accounts/github_app_tokens.py`
  - `task2app/Saas_project/cloud/services/github_pull_request_after_push.py`
- Tests:
  - `task2app/Saas_project/tests/domain/cloud/test_outbound_call_governance_domain_model.py`
  - `task2app/Saas_project/tests/core/test_outbound_gateway.py`
  - `task2app/Saas_project/tests/test_github_pr_after_layer_push_async.py`
  - `task2app/Saas_project/tests/cloud/services/test_layer_github_oauth_tokens_observability.py`

---

### Task 1: 锁定领域契约（Domain First Gate）

**Files:**
- Modify: `task2app/Saas_project/tests/domain/cloud/test_outbound_call_governance_domain_model.py`
- Test: `task2app/Saas_project/tests/domain/cloud/test_outbound_call_governance_domain_model.py`

- [ ] **Step 1: 补一个失败测试，约束 external_async 任务不能从 succeeded 再次迁移**

```python
def test_outbound_dispatch_job_rejects_invalid_status_transition():
    now = datetime.utcnow()
    scope = TaskScope(tenant_id="t", workspace_id="w", task_id="task")
    job = OutboundDispatchJob.create(
        scope=scope,
        route=OutboundCallRoute("external_async"),
        service="github_pulls",
        now=now,
    )
    job.mark_running(now=now)
    job.mark_succeeded(now=now)

    with pytest.raises(ValueError):
        job.mark_failed(detail="late error", now=now)
```

- [ ] **Step 2: 运行该测试并确认先失败（如果未失败，调整断言到真正的非法迁移点）**

Run: `cd task2app/Saas_project && pytest tests/domain/cloud/test_outbound_call_governance_domain_model.py::test_outbound_dispatch_job_rejects_invalid_status_transition -v`  
Expected: 先失败，提示状态迁移约束未覆盖或行为不一致。

- [ ] **Step 3: 在实体中最小实现状态迁移守卫**

```python
def _ensure_status(self, allowed: set[str]) -> None:
    if self.status not in self.ALLOWED_STATUS:
        raise ValueError("job status 非法")
    if self.status not in allowed:
        raise ValueError("job 状态迁移非法")
```

- [ ] **Step 4: 运行 domain 测试确认通过**

Run: `cd task2app/Saas_project && pytest tests/domain/cloud/test_outbound_call_governance_domain_model.py -v`  
Expected: PASS（全部通过）。

- [ ] **Step 5: 提交**

```bash
cd task2app
git add Saas_project/tests/domain/cloud/test_outbound_call_governance_domain_model.py Saas_project/cloud/domain/entities/outbound_dispatch_job.py
git commit -m "test(domain): lock outbound dispatch job transition invariants"
```

---

### Task 2: 内网同步 1 秒超时契约稳定化

**Files:**
- Modify: `task2app/Saas_project/tests/core/test_outbound_gateway.py`
- Modify: `task2app/Saas_project/core/outbound/gateway.py`
- Test: `task2app/Saas_project/tests/core/test_outbound_gateway.py`

- [ ] **Step 1: 增加失败测试，验证 `timeout<=0` 会回退到 1.0**

```python
def test_internal_sync_non_positive_timeout_falls_back_to_hard_limit():
    session = _DummySession()
    call_internal_sync(
        InternalSyncCall(
            service="unit-test",
            method="GET",
            url="http://internal.example.local/ping",
            timeout_seconds=0,
        ),
        session=session,  # type: ignore[arg-type]
    )
    assert session.last_kwargs["timeout"] == 1.0
```

- [ ] **Step 2: 运行单测确认失败**

Run: `cd task2app/Saas_project && pytest tests/core/test_outbound_gateway.py::test_internal_sync_non_positive_timeout_falls_back_to_hard_limit -v`  
Expected: FAIL，提示超时回退策略未锁定。

- [ ] **Step 3: 在网关实现最小修复（只改超时归一化逻辑）**

```python
def _resolve_internal_timeout(timeout_seconds: float | None) -> float:
    raw = INTERNAL_SYNC_HARD_TIMEOUT_SECONDS if timeout_seconds is None else float(timeout_seconds)
    if raw <= 0:
        return INTERNAL_SYNC_HARD_TIMEOUT_SECONDS
    return min(raw, INTERNAL_SYNC_HARD_TIMEOUT_SECONDS)
```

- [ ] **Step 4: 回归 gateway 测试**

Run: `cd task2app/Saas_project && pytest tests/core/test_outbound_gateway.py -v`  
Expected: PASS。

- [ ] **Step 5: 提交**

```bash
cd task2app
git add Saas_project/tests/core/test_outbound_gateway.py Saas_project/core/outbound/gateway.py
git commit -m "fix(core): enforce internal sync timeout fallback policy"
```

---

### Task 3: gitoauth 内网调用全面纳管 internal_sync

**Files:**
- Modify: `task2app/Saas_project/accounts/github_app_tokens.py`
- Modify: `task2app/Saas_project/tests/cloud/services/test_layer_github_oauth_tokens_observability.py`
- Test: `task2app/Saas_project/tests/cloud/services/test_layer_github_oauth_tokens_observability.py`

- [ ] **Step 1: 增加失败测试，断言 internal timeout 映射为可解释错误**

```python
def test_summary_timeout_maps_to_internal_timeout_message(monkeypatch):
    monkeypatch.setattr(
        oauth_service,
        "_request_gitoauth_internal_sync",
        lambda **_kwargs: (None, "internal_timeout"),
    )
    summary, err = oauth_service.fetch_gitoauth_credential_summary_for_user(1)
    assert summary is None
    assert err == "gitoauth internal timeout"
```

- [ ] **Step 2: 跑单测确认失败**

Run: `cd task2app/Saas_project && pytest tests/cloud/services/test_layer_github_oauth_tokens_observability.py::test_summary_timeout_maps_to_internal_timeout_message -v`  
Expected: FAIL。

- [ ] **Step 3: 在调用封装里统一 timeout/network/upstream 错误映射**

```python
if err == "internal_timeout":
    return None, "gitoauth internal timeout"
if r is None:
    return None, err
```

- [ ] **Step 4: 运行 observability + oauth 相关测试**

Run: `cd task2app/Saas_project && pytest tests/cloud/services/test_layer_github_oauth_tokens_observability.py tests/test_layer_github_oauth_tokens.py -v`  
Expected: PASS。

- [ ] **Step 5: 提交**

```bash
cd task2app
git add Saas_project/accounts/github_app_tokens.py Saas_project/tests/cloud/services/test_layer_github_oauth_tokens_observability.py
git commit -m "refactor(accounts): route gitoauth internal calls via governed sync gateway"
```

---

### Task 4: GitHub PR 外网调用异步化闭环

**Files:**
- Modify: `task2app/Saas_project/cloud/services/github_pull_request_after_push.py`
- Modify: `task2app/Saas_project/core/outbound/external_async.py`
- Modify: `task2app/Saas_project/tests/test_github_pr_after_layer_push_async.py`
- Test: `task2app/Saas_project/tests/test_github_pr_after_layer_push_async.py`

- [ ] **Step 1: 增加失败测试，校验返回体必须包含 `queued/job_id/job_status`**

```python
assert out["queued"] is True
assert out["job_id"] == "job_123"
assert out["job_status"] == "queued"
assert out["service"] == "github_pull_request_create"
```

- [ ] **Step 2: 跑异步测试确认失败**

Run: `cd task2app/Saas_project && pytest tests/test_github_pr_after_layer_push_async.py -v`  
Expected: FAIL（字段缺失或值不一致）。

- [ ] **Step 3: 在 service 层只保留入队与立即返回（去除同步直连外网）**

```python
job = enqueue_external_call(
    task_id=str(task_id),
    service="github_pull_request_create",
    executor=lambda: _create_pull_request_external_sync(...),
)
return {
    "queued": True,
    "job_id": job.job_id,
    "service": "github_pull_request_create",
    "job_status": job.status,
    "compare_url": compare_url,
}
```

- [ ] **Step 4: 全量回归 PR 路径测试**

Run: `cd task2app/Saas_project && pytest tests/test_github_pr_after_layer_push_env.py tests/test_github_pr_after_layer_push_async.py -v`  
Expected: PASS。

- [ ] **Step 5: 提交**

```bash
cd task2app
git add Saas_project/cloud/services/github_pull_request_after_push.py Saas_project/core/outbound/external_async.py Saas_project/tests/test_github_pr_after_layer_push_async.py
git commit -m "feat(cloud): switch github PR creation to external async dispatch"
```

---

### Task 5: 按 value stream 扩展 cloud-integration 外网链路（最小一条）

**Files:**
- Modify: `task2app/Saas_project/cloud/services/mock_run_container.py` 或一个确定的 cloud 外网调用入口文件
- Modify: `task2app/Saas_project/tests/test_mock_run_container_service.py`（或对应入口测试）
- Test: 对应测试文件

- [ ] **Step 1: 写失败测试，验证该外网调用不阻塞主请求且返回可追踪 job 信息**

```python
def test_external_cloud_call_returns_async_job(monkeypatch):
    monkeypatch.setattr("core.outbound.enqueue_external_call", lambda **_: SimpleNamespace(job_id="job_x", status="queued"))
    out = call_target_function(...)
    assert out["queued"] is True
    assert out["job_id"] == "job_x"
```

- [ ] **Step 2: 运行该测试确认失败**

Run: `cd task2app/Saas_project && pytest tests/test_mock_run_container_service.py::test_external_cloud_call_returns_async_job -v`  
Expected: FAIL。

- [ ] **Step 3: 对目标外网调用点套用 `external_async` 模式最小实现**

```python
job = enqueue_external_call(
    task_id=str(task_id),
    service="cloud_platform_external_call",
    executor=lambda: _run_external_call_sync(...),
)
return {"queued": True, "job_id": job.job_id, "job_status": job.status}
```

- [ ] **Step 4: 运行目标测试 + 相关回归**

Run: `cd task2app/Saas_project && pytest tests/test_mock_run_container_service.py -v`  
Expected: PASS。

- [ ] **Step 5: 提交**

```bash
cd task2app
git add Saas_project/cloud/services/mock_run_container.py Saas_project/tests/test_mock_run_container_service.py
git commit -m "feat(cloud): migrate one cloud external path to async governance"
```

---

### Task 6: 治理收口与合规门禁

**Files:**
- Modify: `value-stream.yaml`
- Modify: `docs/superpowers/plans/2026-05-22-task2app-outbound-governance-value-stream.md`
- Modify: `docs/superpowers/plans/2026-05-22-task2app-outbound-governance-rollout.md`

- [ ] **Step 1: 校准 value stream 状态（active/planned）与测试文件映射**

```yaml
- name: task2app-outbound-governance
  steps:
    - name: outbound-internal-timeout-thin-slice
      status: active
      test_file: tests/core/test_outbound_gateway.py
```

- [ ] **Step 2: 运行语法与价值流工具校验**

Run: `python3 -c "import yaml, pathlib; yaml.safe_load(pathlib.Path('value-stream.yaml').read_text(encoding='utf-8')); print('YAML_OK')"`  
Expected: `YAML_OK`

- [ ] **Step 3: 运行 DDD 合规检查**

Run: `python3 task2app/scripts/ci/check_ddd_bdd_compliance.py`  
Expected: `DDD/BDD 合规检查通过。`

- [ ] **Step 4: 运行集中回归（本计划涉及的最小测试集）**

Run: `cd task2app/Saas_project && pytest tests/domain/cloud/test_outbound_call_governance_domain_model.py tests/core/test_outbound_gateway.py tests/test_github_pr_after_layer_push_async.py tests/cloud/services/test_layer_github_oauth_tokens_observability.py -v`  
Expected: PASS。

- [ ] **Step 5: 提交**

```bash
cd task2app
git add ../value-stream.yaml ../docs/superpowers/plans/2026-05-22-task2app-outbound-governance-value-stream.md ../docs/superpowers/plans/2026-05-22-task2app-outbound-governance-rollout.md
git commit -m "docs(value-stream): align outbound governance increments and verification gates"
```

---

## Self-Review

- Spec coverage: 覆盖了 thin slice（internal timeout + external async）到扩展与硬化四个增量。
- Placeholder scan: 无 `TODO/TBD/later` 占位语句。
- Type consistency: 计划中统一使用 `OutboundDispatchJob`, `OutboundCallRoute`, `InternalSyncTimeoutPolicy`, `OutboundDispatchJobRepository`, `OutboundCallGovernanceService`。

Plan complete. Save this file and execute in `/6-build-构建`.

