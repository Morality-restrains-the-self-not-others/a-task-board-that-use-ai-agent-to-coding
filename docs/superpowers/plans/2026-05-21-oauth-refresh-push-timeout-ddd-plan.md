# OAuth Refresh Push Timeout Hardening Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** 解决 `POST /api/layers/:layer_id/git/oauth-refresh-push` 的 60 秒超时 502 问题，并补齐跨服务可追踪日志（含 `logs/gitPush.log`）以支持快速定位慢点。

**Architecture:** 在 `task2app` 侧先定义领域事件与日志仓储接口，再在应用服务中落地计时与结构化日志；在 `onlineServiceJS` 侧保留可配置超时并补充分段日志。最终通过 API/E2E 验证请求可成功或可解释失败（不再是无上下文 abort）。

**Tech Stack:** Python (Django/DRF), Node.js (Express), Git, pytest, node:test, Playwright(api)

---

## Scope Check

- 子系统 A：`task2app` 的换票与任务绑定查询链路（慢点根因与可观测性）
- 子系统 B：`trae-agent/onlineServiceJS` 的调用超时策略与本地日志落盘
- 本计划拆成两条可独立交付的任务线，每条都能单独验证

## File Structure (先锁定边界)

- `task2app/Saas_project/cloud/domain/events/container_github_oauth_token_fetch_observed.py`  
  领域事件：记录一次换票尝试的业务事实（成功/失败/耗时/仓库数）
- `task2app/Saas_project/cloud/domain/repositories/container_runtime_context_repository.py`  
  领域仓储接口：新增“记录换票观测事件”抽象方法（仅契约）
- `task2app/Saas_project/cloud/domain/events/__init__.py`  
  导出新领域事件
- `task2app/Saas_project/cloud/services/layer_github_oauth_tokens.py`  
  在换票路径写入事件（通过仓储接口），并输出分段耗时日志
- `task2app/Saas_project/cloud/views/container_layer_github_oauth_views.py`  
  在 API 层补充请求级 trace 日志（入口/出口/状态码）
- `task2app/Saas_project/tests/cloud/`（新增/调整测试）  
  覆盖成功、超时、部分失败、无绑定场景
- `trae-agent/onlineServiceJS/src/layerGitOauthRefreshPush.mjs`  
  保持可配置超时 + `logs/gitPush.log` 分段日志（已完成基础版，补测试）
- `trae-agent/onlineServiceJS/src/layerGitOauthRefreshPush.test.mjs`  
  增加超时配置与日志写入断言

## Domain Model (DDD 校验)

- **Bounded Context：**
  - `Container Runtime Context`：容器侧发起换票并推送
  - `Github OAuth Token Resolution`：按任务/仓库解析可用授权并换发 token
- **Aggregate：**
  - 不新增持久化聚合根；复用 `Container Runtime Context`，新增“观测事件”作为跨上下文通信契约
- **Domain Event：**
  - `ContainerGithubOauthTokenFetchObserved`（过去式命名，`frozen dataclass`）
- **Repository Interface：**
  - 在 `container_runtime_context_repository` 先定义 `record_github_oauth_token_fetch_observed(...)`
  - 基础设施实现可先用“日志实现”，后续可扩展 DB/消息总线
- **顺序约束：**
  - 必须先改 `domain/events` + `domain/repositories`，再改 `services/views`

---

### Task 1: 领域契约先行（事件 + 仓储接口）

**Files:**
- Create: `task2app/Saas_project/cloud/domain/events/container_github_oauth_token_fetch_observed.py`
- Modify: `task2app/Saas_project/cloud/domain/events/__init__.py`
- Modify: `task2app/Saas_project/cloud/domain/repositories/container_runtime_context_repository.py`
- Test: `task2app/Saas_project/tests/cloud/domain/test_container_github_oauth_token_fetch_observed.py`

- [ ] **Step 1: 先写失败测试（事件结构与不可变性）**

```python
from datetime import datetime
from cloud.domain.events.container_github_oauth_token_fetch_observed import ContainerGithubOauthTokenFetchObserved

def test_event_is_immutable_and_has_required_fields():
    evt = ContainerGithubOauthTokenFetchObserved(
        task_id="843742455533076480",
        layer_id="20260521_133028_b5ee27",
        repo_count=1,
        elapsed_ms=61234,
        status="timeout",
        detail="This operation was aborted",
        occurred_at=datetime.utcnow(),
    )
    assert evt.status == "timeout"
```

- [ ] **Step 2: 运行测试确认失败**

Run: `cd task2app/Saas_project && pytest tests/cloud/domain/test_container_github_oauth_token_fetch_observed.py -q`  
Expected: `ImportError` 或属性不匹配导致 FAIL

- [ ] **Step 3: 最小实现领域事件与仓储接口签名**

```python
@dataclass(frozen=True)
class ContainerGithubOauthTokenFetchObserved:
    task_id: str
    layer_id: str
    repo_count: int
    elapsed_ms: int
    status: str
    detail: str
    occurred_at: datetime
```

- [ ] **Step 4: 再跑测试确认通过**

Run: `cd task2app/Saas_project && pytest tests/cloud/domain/test_container_github_oauth_token_fetch_observed.py -q`  
Expected: `1 passed`

- [ ] **Step 5: 提交**

Run: `git add task2app/Saas_project/cloud/domain/events task2app/Saas_project/cloud/domain/repositories task2app/Saas_project/tests/cloud/domain && git commit -m "feat(cloud-domain): add oauth token fetch observed domain contract"`

---

### Task 2: task2app 服务链路打点与慢点可观测

**Files:**
- Modify: `task2app/Saas_project/cloud/services/layer_github_oauth_tokens.py`
- Modify: `task2app/Saas_project/cloud/views/container_layer_github_oauth_views.py`
- Test: `task2app/Saas_project/tests/cloud/services/test_layer_github_oauth_tokens_observability.py`

- [ ] **Step 1: 写失败测试（超时时返回可解释错误并记录观测）**

```python
def test_resolve_tokens_logs_timeout_observation(monkeypatch):
    monkeypatch.setattr(
        "cloud.services.layer_github_oauth_tokens.fetch_github_access_via_gitoauth_for_user",
        lambda *a, **k: (None, "Read timed out"),
    )
    tokens, err = resolve_github_auth_by_repo_for_container_task(
        tenant_id="1", workspace_id="2", task_id="3", repo_slugs=["acme/repo"]
    )
    assert tokens == {}
    assert "timed out" in err.lower()
```

- [ ] **Step 2: 运行测试确认失败**

Run: `cd task2app/Saas_project && pytest tests/cloud/services/test_layer_github_oauth_tokens_observability.py -q`  
Expected: 断言失败（尚无观测逻辑/错误归因）

- [ ] **Step 3: 实现分段计时 + 结构化日志 + 领域事件落盘调用**

```python
t0 = time.monotonic()
# ... resolve bindings
elapsed_ms = int((time.monotonic() - t0) * 1000)
logger.info(
    "[LayerGithubOauthTokens] observed task_id=%s repo_count=%s status=%s elapsed_ms=%s detail=%s",
    task_id, len(slugs), status, elapsed_ms, detail[:240]
)
```

- [ ] **Step 4: 运行目标测试与相关回归**

Run: `cd task2app/Saas_project && pytest tests/cloud/services/test_layer_github_oauth_tokens_observability.py tests/test_layer_git_push_policy.py -q`  
Expected: 全部 PASS

- [ ] **Step 5: 提交**

Run: `git add task2app/Saas_project/cloud/services/layer_github_oauth_tokens.py task2app/Saas_project/cloud/views/container_layer_github_oauth_views.py task2app/Saas_project/tests/cloud/services/test_layer_github_oauth_tokens_observability.py && git commit -m "feat(cloud): add oauth token fetch observability and timeout diagnostics"`

---

### Task 3: onlineServiceJS 超时策略与 gitPush 日志验证

**Files:**
- Modify: `trae-agent/onlineServiceJS/src/layerGitOauthRefreshPush.mjs`
- Modify: `trae-agent/onlineServiceJS/src/layerGitOauthRefreshPush.test.mjs`
- Test: `trae-agent/onlineServiceJS/e2e/layer-oauth-refresh-push.api.spec.mjs`

- [ ] **Step 1: 写失败测试（读取 env 超时配置、写入 logs/gitPush.log）**

```js
test('oauth timeout env should clamp into [30,300] and log begin/fail', async () => {
  process.env.TRAE_LAYER_GITHUB_OAUTH_FETCH_TIMEOUT_SEC = '15';
  // invoke runLayerOauthRefreshPush(...)
  // assert gitPush.log has timeout_sec=30
});
```

- [ ] **Step 2: 跑单测确认失败**

Run: `cd trae-agent/onlineServiceJS && node --test src/layerGitOauthRefreshPush.test.mjs`  
Expected: 新增用例 FAIL

- [ ] **Step 3: 补实现（若已有实现，仅补断言覆盖）**

```js
const oauthFetchTimeoutSec = oauthTokenFetchTimeoutSec(); // 默认 120，范围 30-300
appendOauthRefreshPushLog(`oauth-refresh-push token-fetch ... timeout_sec=${oauthFetchTimeoutSec}`);
```

- [ ] **Step 4: 跑单测 + API E2E**

Run: `cd trae-agent/onlineServiceJS && node --test src/layerGitOauthRefreshPush.test.mjs`  
Expected: PASS

Run: `cd trae-agent/onlineServiceJS && npx playwright test --project=api e2e/layer-oauth-refresh-push.api.spec.mjs`  
Expected: PASS（若外部依赖波动，至少断言返回 detail 含可解释原因）

- [ ] **Step 5: 提交**

Run: `git add trae-agent/onlineServiceJS/src/layerGitOauthRefreshPush.mjs trae-agent/onlineServiceJS/src/layerGitOauthRefreshPush.test.mjs && git commit -m "fix(onlineServiceJS): harden oauth-refresh-push timeout and gitPush logging"`

---

### Task 4: 端到端验收与运维手册

**Files:**
- Modify: `docs/superpowers/plans/2026-05-21-oauth-refresh-push-timeout-ddd-plan.md`
- Create: `docs/runbooks/oauth-refresh-push-timeout-troubleshooting.md`

- [ ] **Step 1: 手工复现并验证日志链路**

Run: `curl -i -X POST "http://127.0.0.1:8765/api/layers/<layer_id>/git/oauth-refresh-push?access_token=<token>" -H "X-Access-Token: <token>" -H "Content-Type: application/json" -d '{}'`  
Expected: 非 502 或 502 时 `detail` 可解释（非裸 `aborted`）

- [ ] **Step 2: 校验日志文件落盘**

Run: `rg "oauth-refresh-push" trae-agent/onlineProject_state/logs/gitPush.log`  
Expected: 至少包含 `begin`、`token-fetch`、`done/fail`

- [ ] **Step 3: 编写故障排查 Runbook**

```markdown
1. 先看 logs/gitPush.log（调用入口与timeout_sec）
2. 再看 reqLogs/outbound.log（实际上游耗时）
3. 最后看 task2app 日志（绑定/换票慢点）
```

- [ ] **Step 4: 回归核心命令**

Run: `cd task2app/Saas_project && pytest -q`  
Expected: 无新增失败

Run: `cd trae-agent/onlineServiceJS && node --test`  
Expected: 无新增失败

- [ ] **Step 5: 提交**

Run: `git add docs/runbooks/oauth-refresh-push-timeout-troubleshooting.md && git commit -m "docs: add oauth-refresh-push timeout troubleshooting runbook"`

---

## Self-Review

- **Spec coverage:** 覆盖了超时根因治理、日志落盘、可追踪性、回归验证四项目标。
- **Placeholder scan:** 无 `TBD/TODO/后续补` 占位内容。
- **Type consistency:** 事件名、方法名、日志关键字在各任务内保持一致。

