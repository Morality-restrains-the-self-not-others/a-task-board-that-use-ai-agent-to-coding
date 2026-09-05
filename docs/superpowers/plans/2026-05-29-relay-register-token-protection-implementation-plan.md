# relay register 换票窗口保护 Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** 防止任务详情 `?relayToTrae=true` 直启链路中，`token-init/start` 后的 `relay register` 在 TEIP 换票窗口内清空 `refresh_token`，导致 access/refresh 失效。

**Architecture:** 在领域层引入 `TokenExchangePhase`（TEIP 判定）与 `ContainerTokenBootstrapPolicy`（bootstrap 决策）；应用层 `_replace_access_token_placeholder` 与 `relay_to_trae_register` 经策略门控；前端收敛冗余 register；go_relay 换票半失败快速失败；status-push 缓存在 token 轮换后失效。

**Tech Stack:** Django 4.x、Python 3.11+、pytest-django、Vue 3、Go（go_relayToTrae）、Playwright

**输入文档：**
- 设计：`docs/superpowers/specs/2026-05-29-relay-register-token-protection-design.md`
- 价值流：`docs/superpowers/plans/2026-05-29-relay-register-token-protection-value-stream.md`
- NFR：`docs/superpowers/plans/2026-05-29-relay-register-token-protection-nfr-clarification.md`
- 领域模型：`docs/superpowers/plans/2026-05-29-relay-register-token-protection-domain-model.md`

**工作目录：** `task2app/Saas_project`（pytest）；`task2app/front_project`（前端）；`go_relayToTrae`（Go）

---

## 文件结构

| 文件 | 职责 |
|------|------|
| `cloud/domain/value_objects/token_exchange_phase.py` | TEIP 相位值对象 |
| `cloud/domain/value_objects/relay_register_intent.py` | register 意图值对象 |
| `cloud/domain/exceptions/token_bootstrap_blocked.py` | TEIP 阻断异常 |
| `cloud/domain/entities/container_token_session.py` | 聚合扩展 `exchange_phase()` / `is_exchange_in_progress()` |
| `cloud/domain/services/container_token_bootstrap_policy.py` | bootstrap 决策领域服务 |
| `cloud/domain/events/token_audit_events.py` | 新增审计事件类型常量 |
| `cloud/services/mock_run_container.py` | `_replace_access_token_placeholder` TEIP 保护 |
| `cloud/services/relay_to_trae_proxy.py` | `relay_to_trae_register` register-only 降级 |
| `cloud/services/relay_to_trae_status.py` | token 轮换后 cfg 缓存失效 |
| `tests/test_relay_register_token_exchange_guard.py` | QS-01~04 集成单测 |
| `tests/domain/cloud/test_container_token_domain_model.py` | 领域单测扩展 |
| `tests/domain/cloud/test_container_token_bootstrap_policy.py` | 策略单测 |
| `front_project/app/src/components/ServerConfig.logic.vue` | 前端 register 收敛 |
| `go_relayToTrae/src/process.go` | 换票失败 fail-fast |

---

## Increment 1: Django TEIP 硬保护

### Task 1: `TokenExchangePhase` 值对象

**Files:**
- Create: `task2app/Saas_project/cloud/domain/value_objects/token_exchange_phase.py`
- Create: `task2app/Saas_project/tests/domain/cloud/test_token_exchange_phase.py`

- [ ] **Step 1: Write the failing test**

```python
# tests/domain/cloud/test_token_exchange_phase.py
from datetime import datetime

from cloud.domain.value_objects.token_exchange_phase import (
    TokenExchangePhase,
    derive_token_exchange_phase,
)


def test_derive_teip_when_refresh_only():
    phase = derive_token_exchange_phase(
        access_token="",
        refresh_token="refresh-abc",
        access_token_expires_at=None,
    )
    assert phase == TokenExchangePhase.EXCHANGE_IN_PROGRESS


def test_derive_bootstrapped_when_access_only():
    phase = derive_token_exchange_phase(
        access_token="access-abc",
        refresh_token="",
        access_token_expires_at=datetime(2026, 6, 1),
    )
    assert phase == TokenExchangePhase.BOOTSTRAPPED


def test_derive_active_when_both_present():
    phase = derive_token_exchange_phase(
        access_token="access-abc",
        refresh_token="refresh-abc",
        access_token_expires_at=datetime(2026, 6, 1),
    )
    assert phase == TokenExchangePhase.ACTIVE
```

- [ ] **Step 2: Run test to verify it fails**

Run: `cd task2app/Saas_project && python -m pytest tests/domain/cloud/test_token_exchange_phase.py -v`

Expected: FAIL `ModuleNotFoundError: token_exchange_phase`

- [ ] **Step 3: Write minimal implementation**

```python
# cloud/domain/value_objects/token_exchange_phase.py
from dataclasses import dataclass
from datetime import datetime
from typing import Optional


@dataclass(frozen=True)
class TokenExchangePhase:
    BOOTSTRAPPED = "bootstrapped"
    EXCHANGE_IN_PROGRESS = "teip"
    ACTIVE = "active"
    UNINITIALIZED = "uninitialized"

    value: str

    def __post_init__(self) -> None:
        raw = str(self.value or "").strip()
        if raw not in {
            self.BOOTSTRAPPED,
            self.EXCHANGE_IN_PROGRESS,
            self.ACTIVE,
            self.UNINITIALIZED,
        }:
            raise ValueError(f"非法 token 换票相位: {raw}")
        object.__setattr__(self, "value", raw)

    @property
    def is_exchange_in_progress(self) -> bool:
        return self.value == self.EXCHANGE_IN_PROGRESS


def derive_token_exchange_phase(
    *,
    access_token: str,
    refresh_token: str,
    access_token_expires_at: Optional[datetime],
) -> TokenExchangePhase:
    access = str(access_token or "").strip()
    refresh = str(refresh_token or "").strip()
    if refresh and not access and access_token_expires_at is None:
        return TokenExchangePhase(TokenExchangePhase.EXCHANGE_IN_PROGRESS)
    if access and not refresh:
        return TokenExchangePhase(TokenExchangePhase.BOOTSTRAPPED)
    if access and refresh:
        return TokenExchangePhase(TokenExchangePhase.ACTIVE)
    return TokenExchangePhase(TokenExchangePhase.UNINITIALIZED)
```

- [ ] **Step 4: Run test to verify it passes**

Run: `cd task2app/Saas_project && python -m pytest tests/domain/cloud/test_token_exchange_phase.py -v`

Expected: PASS (3 tests)

---

### Task 2: 扩展 `ContainerTokenSession` 聚合

**Files:**
- Create: `task2app/Saas_project/cloud/domain/exceptions/token_bootstrap_blocked.py`
- Modify: `task2app/Saas_project/cloud/domain/entities/container_token_session.py`
- Modify: `task2app/Saas_project/tests/domain/cloud/test_container_token_domain_model.py`

- [ ] **Step 1: Write the failing test**

```python
# append to test_container_token_domain_model.py
from cloud.domain.exceptions.token_bootstrap_blocked import TokenBootstrapBlocked
from cloud.domain.value_objects.token_exchange_phase import TokenExchangePhase


def test_exchange_refresh_sets_teip_phase():
    now = datetime(2026, 1, 1, 0, 0, 0)
    scope = TaskScope(tenant_id="t1", workspace_id="w1", task_id="task1")
    session = ContainerTokenSession.bootstrap(
        scope=scope,
        access_token=AccessToken(_token("d")),
        expires_at=now + timedelta(hours=1),
        now=now,
    )
    session.exchange_refresh(
        expected_access_token=AccessToken(_token("d")),
        refresh_token=RefreshToken(_token("r")),
        business_api_endpoint=BusinessApiEndpoint("https://biz.example/api"),
        now=now + timedelta(minutes=1),
    )
    assert session.exchange_phase() == TokenExchangePhase.EXCHANGE_IN_PROGRESS
    assert session.is_exchange_in_progress() is True


def test_assert_bootstrap_allowed_raises_in_teip():
    now = datetime(2026, 1, 1, 0, 0, 0)
    scope = TaskScope(tenant_id="t1", workspace_id="w1", task_id="task1")
    session = ContainerTokenSession.bootstrap(
        scope=scope,
        access_token=AccessToken(_token("e")),
        expires_at=now + timedelta(hours=1),
        now=now,
    )
    session.exchange_refresh(
        expected_access_token=AccessToken(_token("e")),
        refresh_token=RefreshToken(_token("f")),
        business_api_endpoint=BusinessApiEndpoint("https://biz.example/api"),
        now=now + timedelta(minutes=1),
    )
    with pytest.raises(TokenBootstrapBlocked):
        session.assert_bootstrap_allowed()
```

- [ ] **Step 2: Run test to verify it fails**

Run: `cd task2app/Saas_project && python -m pytest tests/domain/cloud/test_container_token_domain_model.py::test_exchange_refresh_sets_teip_phase -v`

Expected: FAIL `AttributeError: exchange_phase`

- [ ] **Step 3: Write minimal implementation**

```python
# cloud/domain/exceptions/token_bootstrap_blocked.py
class TokenBootstrapBlocked(Exception):
    """TEIP 窗口内禁止 bootstrap 签发 access。"""

    error_code = "TOKEN_EXCHANGE_IN_PROGRESS"
```

```python
# container_token_session.py — add imports and methods
from cloud.domain.exceptions.token_bootstrap_blocked import TokenBootstrapBlocked
from cloud.domain.value_objects.token_exchange_phase import (
    TokenExchangePhase,
    derive_token_exchange_phase,
)

def exchange_phase(self) -> TokenExchangePhase:
    return derive_token_exchange_phase(
        access_token=str(self.current_access_token or ""),
        refresh_token=str(self.current_refresh_token or ""),
        access_token_expires_at=self.access_token_expires_at,
    )

def is_exchange_in_progress(self) -> bool:
    return self.exchange_phase().is_exchange_in_progress

def assert_bootstrap_allowed(self) -> None:
    if self.is_exchange_in_progress():
        raise TokenBootstrapBlocked("换票进行中，禁止 bootstrap 签发 access")
```

- [ ] **Step 4: Run tests**

Run: `cd task2app/Saas_project && python -m pytest tests/domain/cloud/test_container_token_domain_model.py -v`

Expected: PASS

---

### Task 3: `ContainerTokenBootstrapPolicy` 领域服务

**Files:**
- Create: `task2app/Saas_project/cloud/domain/value_objects/relay_register_intent.py`
- Create: `task2app/Saas_project/cloud/domain/services/container_token_bootstrap_policy.py`
- Create: `task2app/Saas_project/tests/domain/cloud/test_container_token_bootstrap_policy.py`

- [ ] **Step 1: Write the failing test**

```python
# tests/domain/cloud/test_container_token_bootstrap_policy.py
from datetime import datetime, timedelta

from cloud.domain.entities.container_token_session import ContainerTokenSession
from cloud.domain.services.container_token_bootstrap_policy import (
    BootstrapAction,
    ContainerTokenBootstrapPolicy,
)
from cloud.domain.value_objects.access_token import AccessToken
from cloud.domain.value_objects.business_api_endpoint import BusinessApiEndpoint
from cloud.domain.value_objects.refresh_token import RefreshToken
from cloud.domain.value_objects.relay_register_intent import RelayRegisterIntent
from cloud.domain.value_objects.task_scope import TaskScope


def _teip_session() -> ContainerTokenSession:
    now = datetime(2026, 1, 1, 0, 0, 0)
    scope = TaskScope(tenant_id="t1", workspace_id="w1", task_id="task1")
    session = ContainerTokenSession.bootstrap(
        scope=scope,
        access_token=AccessToken("a" * 32),
        expires_at=now + timedelta(hours=1),
        now=now,
    )
    session.exchange_refresh(
        expected_access_token=AccessToken("a" * 32),
        refresh_token=RefreshToken("r" * 32),
        business_api_endpoint=BusinessApiEndpoint("https://biz.example/api"),
        now=now,
    )
    return session


def test_teip_issue_access_blocked():
    policy = ContainerTokenBootstrapPolicy()
    action = policy.resolve_bootstrap_action(
        session=_teip_session(),
        intent=RelayRegisterIntent.ISSUE_ACCESS_AND_REGISTER,
    )
    assert action == BootstrapAction.BLOCK


def test_teip_register_only_allowed():
    policy = ContainerTokenBootstrapPolicy()
    action = policy.resolve_bootstrap_action(
        session=_teip_session(),
        intent=RelayRegisterIntent.REGISTER_ONLY,
    )
    assert action == BootstrapAction.REGISTER_ONLY
```

- [ ] **Step 2: Run test to verify it fails**

Run: `cd task2app/Saas_project && python -m pytest tests/domain/cloud/test_container_token_bootstrap_policy.py -v`

Expected: FAIL `ModuleNotFoundError`

- [ ] **Step 3: Write minimal implementation**

```python
# cloud/domain/value_objects/relay_register_intent.py
from dataclasses import dataclass


@dataclass(frozen=True)
class RelayRegisterIntent:
    REGISTER_ONLY = "register_only"
    ISSUE_ACCESS_AND_REGISTER = "issue"

    value: str

    def __post_init__(self) -> None:
        raw = str(self.value or "").strip()
        if raw not in {self.REGISTER_ONLY, self.ISSUE_ACCESS_AND_REGISTER}:
            raise ValueError(f"非法 relay register 意图: {raw}")
        object.__setattr__(self, "value", raw)
```

```python
# cloud/domain/services/container_token_bootstrap_policy.py
from dataclasses import dataclass

from cloud.domain.entities.container_token_session import ContainerTokenSession
from cloud.domain.value_objects.relay_register_intent import RelayRegisterIntent
from cloud.domain.value_objects.token_exchange_phase import TokenExchangePhase


@dataclass(frozen=True)
class BootstrapAction:
    ALLOW = "allow"
    BLOCK = "block"
    REGISTER_ONLY = "register_only"

    value: str


class ContainerTokenBootstrapPolicy:
    def resolve_bootstrap_action(
        self,
        *,
        session: ContainerTokenSession,
        intent: RelayRegisterIntent,
    ) -> BootstrapAction:
        phase = session.exchange_phase()
        if phase == TokenExchangePhase.EXCHANGE_IN_PROGRESS:
            if intent.value == RelayRegisterIntent.REGISTER_ONLY:
                return BootstrapAction(BootstrapAction.REGISTER_ONLY)
            return BootstrapAction(BootstrapAction.BLOCK)
        return BootstrapAction(BootstrapAction.ALLOW)
```

- [ ] **Step 4: Run tests**

Run: `cd task2app/Saas_project && python -m pytest tests/domain/cloud/test_container_token_bootstrap_policy.py -v`

Expected: PASS

---

### Task 4: 审计事件类型扩展

**Files:**
- Modify: `task2app/Saas_project/cloud/domain/events/token_audit_events.py`
- Modify: `task2app/Saas_project/tests/domain/cloud/test_container_token_audit_domain_model.py`

- [ ] **Step 1: Write the failing test**

```python
# append to test_container_token_audit_domain_model.py
def test_token_bootstrap_blocked_event_type_exists():
    assert TokenAuditEventTypes.TOKEN_BOOTSTRAP_BLOCKED == "token_bootstrap_blocked"
    assert TokenAuditEventTypes.RELAY_REGISTER_REUSED_STATE == "relay_register_reused_state"
```

- [ ] **Step 2: Run test to verify it fails**

Run: `cd task2app/Saas_project && python -m pytest tests/domain/cloud/test_container_token_audit_domain_model.py::test_token_bootstrap_blocked_event_type_exists -v`

Expected: FAIL `AttributeError`

- [ ] **Step 3: Add constants**

```python
# token_audit_events.py — append before STATUS_PUSH_OK
TOKEN_BOOTSTRAP_BLOCKED = "token_bootstrap_blocked"
RELAY_REGISTER_REUSED_STATE = "relay_register_reused_state"
```

- [ ] **Step 4: Run test**

Expected: PASS

---

### Task 5: `_replace_access_token_placeholder` TEIP 保护

**Files:**
- Modify: `task2app/Saas_project/cloud/services/mock_run_container.py:281-368`
- Test: `task2app/Saas_project/tests/test_relay_register_token_exchange_guard.py`（新建，本步先写第一个用例）

- [ ] **Step 1: Write the failing integration test (QS-01)**

```python
# tests/test_relay_register_token_exchange_guard.py
from datetime import timedelta

import pytest
from django.utils import timezone

from cloud.models import CloudServerConfig, Company
from cloud.services.mock_run_container import _replace_access_token_placeholder
from cloud.services.relay_to_trae_proxy import TASK2APP_ACCESS_TOKEN_PLACEHOLDER


@pytest.mark.django_db
def test_replace_placeholder_does_not_clear_refresh_during_teip():
    company = Company.objects.create(name="teip-guard-co")
    refresh = "teip-refresh-token-value"
    cfg = CloudServerConfig.objects.create(
        company=company,
        workspace_id="ws1",
        task_id="task-teip",
        platform="mock",
        region="mock-local",
        zone_id="mock-local-a",
        authorization_id="mock-auth",
        container_access_token="",
        container_refresh_token=refresh,
        container_access_token_expires_at=None,
    )
    env = _replace_access_token_placeholder(
        {"ACCESS_TOKEN": TASK2APP_ACCESS_TOKEN_PLACEHOLDER},
        tenant_id=str(company.id),
        workspace_id="ws1",
        task_id="task-teip",
        caller="relay_register",
    )
    cfg.refresh_from_db()
    assert cfg.container_refresh_token == refresh
    assert cfg.container_access_token == ""
    assert env["ACCESS_TOKEN"] == TASK2APP_ACCESS_TOKEN_PLACEHOLDER
```

- [ ] **Step 2: Run test to verify it fails**

Run: `cd task2app/Saas_project && python -m pytest tests/test_relay_register_token_exchange_guard.py::test_replace_placeholder_does_not_clear_refresh_during_teip -v`

Expected: FAIL — refresh 被清空

- [ ] **Step 3: Implement TEIP guard in `_replace_access_token_placeholder`**

在函数签名增加 `caller: str = ""`；在 `if cfg.container_refresh_token:` 分支前插入：

```python
from cloud.domain.value_objects.token_exchange_phase import derive_token_exchange_phase

phase = derive_token_exchange_phase(
    access_token=str(cfg.container_access_token or ""),
    refresh_token=str(cfg.container_refresh_token or ""),
    access_token_expires_at=cfg.container_access_token_expires_at,
)
if phase.is_exchange_in_progress:
    # TEIP：禁止清空 refresh / 重签 access；占位符原样返回
    return env
```

同时更新所有调用点传入 `caller`（`build_relay_to_trae_runtime_env` 等），register 路径传 `"relay_register"`。

- [ ] **Step 4: Run test**

Expected: PASS

---

### Task 6: `relay_to_trae_register` TEIP register-only 降级

**Files:**
- Modify: `task2app/Saas_project/cloud/services/relay_to_trae_proxy.py:725-841`
- Modify: `task2app/Saas_project/tests/test_relay_register_token_exchange_guard.py`

- [ ] **Step 1: Write the failing test (QS-01 + QS-03)**

```python
@pytest.mark.django_db
def test_relay_register_teip_register_only_preserves_refresh_and_succeeds(monkeypatch):
    from unittest.mock import MagicMock

    from cloud.services.relay_to_trae_proxy import (
        TASK2APP_ACCESS_TOKEN_PLACEHOLDER,
        relay_to_trae_register,
    )
    from tests.test_relay_to_trae_proxy import _make_task_with_image

    company, workspace, todo, _ = _make_task_with_image(tenant_name="teip-register-only")
    refresh = "teip-register-refresh"
    cfg = CloudServerConfig.objects.create(
        company=company,
        workspace_id=str(workspace.id),
        task_id=str(todo.id),
        platform="mock",
        region="mock-local",
        zone_id="mock-local-a",
        authorization_id="mock-auth",
        container_access_token="",
        container_refresh_token=refresh,
        container_access_token_expires_at=None,
    )
    monkeypatch.setattr(
        "cloud.services.relay_to_trae_proxy._relay_worker_url",
        lambda: "http://127.0.0.1:8797",
    )
    captured = {}

    class FakeResp:
        status_code = 200
        content = b'{"status":"ok"}'
        def json(self):
            return {"status": "ok"}

    def fake_request(method, path, **kwargs):
        captured["json"] = kwargs.get("json")
        return FakeResp()

    monkeypatch.setattr("cloud.services.relay_to_trae_proxy._relay_http_request", fake_request)
    req = MagicMock()
    req.data = {
        "task_api_endpoint_origin": "http://api.daydaymoney.com",
        "access_token": TASK2APP_ACCESS_TOKEN_PLACEHOLDER,
    }
    resp = relay_to_trae_register(
        req,
        tenant_id=str(company.id),
        workspace_id=str(workspace.id),
        task_id=str(todo.id),
    )
    cfg.refresh_from_db()
    assert resp.status_code == 200
    assert cfg.container_refresh_token == refresh
    assert "access_token" not in captured["json"] or not captured["json"].get("access_token")
```

- [ ] **Step 2: Run test to verify it fails**

Expected: FAIL — 400 missing access_token 或 refresh 被清空

- [ ] **Step 3: Implement register-only path in `relay_to_trae_register`**

在 `_access_token_needs_server_issue` 分支前：

```python
from cloud.domain.services.container_token_bootstrap_policy import (
    BootstrapAction,
    ContainerTokenBootstrapPolicy,
)
from cloud.domain.value_objects.relay_register_intent import RelayRegisterIntent
from cloud.domain.value_objects.task_scope import TaskScope
from cloud.infrastructure.repositories.django_container_token_session_repository import (
    DjangoContainerTokenSessionRepository,
)

register_only = bool(body.get("register_only"))
intent = RelayRegisterIntent(
    RelayRegisterIntent.REGISTER_ONLY if register_only
    else RelayRegisterIntent.ISSUE_ACCESS_AND_REGISTER
)
cfg = CloudServerConfig.objects.filter(company_id=tenant, task_id=tid).order_by("-updated_at").first()
if cfg is not None:
    repo = DjangoContainerTokenSessionRepository()
    session = repo.find_by_scope(TaskScope(tenant_id=tenant, workspace_id=wid, task_id=tid))
    if session is not None:
        action = ContainerTokenBootstrapPolicy().resolve_bootstrap_action(session=session, intent=intent)
        if action.value == BootstrapAction.BLOCK:
            append_container_token_audit_event(..., event_type=TokenAuditEventTypes.TOKEN_BOOTSTRAP_BLOCKED, error_code="TOKEN_EXCHANGE_IN_PROGRESS")
            action = BootstrapAction(BootstrapAction.REGISTER_ONLY)  # 自动降级
        if action.value == BootstrapAction.REGISTER_ONLY:
            # 省略 access_token，仅转发 scope + origin
            payload = {"tenant_id": tenant, "workspace_id": wid, "task_id": tid, "task_api_endpoint_origin": task_api_origin}
            append_container_token_audit_event(..., event_type=TokenAuditEventTypes.RELAY_REGISTER_REUSED_STATE)
            # forward to /v1/register without access_token
            ...
            return api_resp
```

调整 payload 校验：`register_only` / TEIP 降级时允许无 `access_token`。

- [ ] **Step 4: Run tests**

Run: `cd task2app/Saas_project && python -m pytest tests/test_relay_register_token_exchange_guard.py -v`

Expected: PASS

---

### Task 7: 完整换票回归 (QS-02)

**Files:**
- Modify: `task2app/Saas_project/tests/test_relay_register_token_exchange_guard.py`

- [ ] **Step 1: Write end-to-end test**

```python
@pytest.mark.django_db
def test_exchange_refresh_then_register_teip_then_refresh_access_still_works(monkeypatch):
    """QS-02: TEIP 期间 register 后 refresh-access 仍 200。"""
    # 复用 test_container_runtime_tokens 的 exchange + refresh 流程，
    # 在中间插入 relay_to_trae_register(placeholder) mock
    ...
```

- [ ] **Step 2: Run full regression**

Run:
```bash
cd task2app/Saas_project && python -m pytest \
  tests/test_relay_register_token_exchange_guard.py \
  tests/test_container_runtime_tokens.py::test_exchange_refresh_then_refresh_access_updates_db \
  tests/test_relay_to_trae_proxy.py::test_relay_to_trae_register_reuses_existing_token_when_refresh_exists \
  -v
```

Expected: ALL PASS

- [ ] **Step 3: Commit Increment 1**

```bash
git add cloud/domain cloud/services/mock_run_container.py cloud/services/relay_to_trae_proxy.py tests/
git commit -m "fix(relay): block bootstrap during TEIP to protect refresh token"
```

---

## Increment 2: 前端 register 语义收敛

### Task 8: 删除 `startRelayToTrae` 末尾冗余 register

**Files:**
- Modify: `task2app/front_project/app/src/components/ServerConfig.logic.vue:1697`

- [ ] **Step 1: Remove redundant call**

删除：
```javascript
await registerRelayToTraeTask({ issueTokenOnServer: true })
```
（位于 `startRelayToTrae` 中 `props.resumeContainerHeartbeatForRelayStart?.()` 之后）

- [ ] **Step 2: Manual smoke**

打开任务详情 `?relayToTrae=true`，直启后 Network 面板：`start` 202 之后不应再有带 `issueTokenOnServer` 的 register。

---

### Task 9: `fetchRelayToTraeServiceStatus` 使用 register-only

**Files:**
- Modify: `task2app/front_project/app/src/components/ServerConfig.logic.vue:1393-1428,1511`

- [ ] **Step 1: Extend `registerRelayToTraeTask`**

```javascript
/** @param {{ issueTokenOnServer?: boolean, registerOnly?: boolean }} [options] */
const registerRelayToTraeTask = async ({ issueTokenOnServer = false, registerOnly = false } = {}) => {
  ...
  if (registerOnly) {
    body.register_only = true
    delete body.access_token  // 或不发送 access_token
  }
```

- [ ] **Step 2: Change health path**

```javascript
// line ~1511
await registerRelayToTraeTask({ registerOnly: true })
```

- [ ] **Step 3: Run Playwright regression**

Run:
```bash
cd task2app/playwright/front_project && npx playwright test \
  tests/TaskDetail.relay-to-trae-no-invalid-access-token.playwright.test.js \
  tests/TaskDetail.relay-to-trae-direct-start.playwright.test.js
```

Expected: PASS；日志不含 `无效的 access_token`

- [ ] **Step 4: Commit Increment 2**

```bash
git add task2app/front_project/app/src/components/ServerConfig.logic.vue
git commit -m "fix(relay-ui): stop redundant register after start; use register_only on health"
```

---

## Increment 3: 关联加固

### Task 10: go_relay 换票半失败 fail-fast (QS-05)

**Files:**
- Modify: `go_relayToTrae/src/process.go:276-282`
- Create: `go_relayToTrae/src/process_token_exchange_test.go`（若尚无）

- [ ] **Step 1: Write failing test**

```go
func TestStartOnlineServiceFailsWhenRefreshAccessFailsAfterExchange(t *testing.T) {
    // mock performTokenExchange returns error after exchange succeeded
    // assert startOnlineService returns error, not fallback token
}
```

- [ ] **Step 2: Remove dangerous fallback**

将：
```go
if err != nil {
    appendLog(fmt.Sprintf("[relayToTrae] token exchange failed, using original token: %v", err))
```
改为：
```go
if err != nil {
    return nil, fmt.Errorf("token exchange failed: %w", err)
}
```

- [ ] **Step 3: Run Go tests**

Run: `cd go_relayToTrae && go test ./src/... -v -run TokenExchange`

Expected: PASS

---

### Task 11: status-push cfg 缓存失效

**Files:**
- Modify: `task2app/Saas_project/cloud/services/relay_to_trae_status.py`
- Modify: `task2app/Saas_project/cloud/views/container_runtime_token_views.py`（exchange/refresh 成功后调用失效）
- Test: `task2app/Saas_project/tests/test_relay_to_trae_status.py`

- [ ] **Step 1: Add `invalidate_status_push_cfg_cache(access_token: str)`**

```python
def invalidate_status_push_cfg_cache(access_token: str) -> None:
    token = str(access_token or "").strip()
    if not token:
        return
    with _CFG_CACHE_LOCK:
        _CFG_CACHE.pop(token, None)
```

- [ ] **Step 2: Call after exchange_refresh / refresh_access in views**

在 `exchange_server_container_refresh_token` 与 `refresh_server_container_access_token` 成功写库后，对旧 access digest 调用 `invalidate_status_push_cfg_cache(old_access)`。

- [ ] **Step 3: Write test**

```python
def test_status_push_cache_invalidated_after_token_rotation():
  # seed cache → rotate token → lookup with old access returns None (not stale 200)
```

- [ ] **Step 4: Run tests**

Run: `cd task2app/Saas_project && python -m pytest tests/test_relay_to_trae_status.py -v`

- [ ] **Step 5: Commit Increment 3**

```bash
git add go_relayToTrae/src/process.go cloud/services/relay_to_trae_status.py cloud/views/container_runtime_token_views.py tests/
git commit -m "fix(relay): fail-fast on token exchange; invalidate status-push cfg cache"
```

---

## Self-Review

| 规格要求 | 对应 Task |
|----------|-----------|
| TEIP 判定与 exchange_refresh 字段一致 | Task 1, 5 |
| register 不在 start 后重签 token | Task 6, 8 |
| precheck TEIP 返回 409 | Task 5（caller=`relay_precheck` 时 raise/409，需在 Task 5 补充 precheck 分支） |
| 审计 `token_bootstrap_blocked` | Task 4, 6 |
| 价值流回归 | Task 7, 9 |
| go_relay 无 fallback | Task 10 |
| status-push 缓存失效 | Task 11 |

**Gap 补充（Task 5 内）：** `caller="relay_precheck"` 且 TEIP 时，应抛出 `TokenBootstrapBlocked` 或返回 409，而非静默保留占位符。在 `_issue_relay_access_token` 调用链增加相同策略检查。

---

## 验收清单

- [ ] QS-01: `exchange_refresh` 后 `relay_register(placeholder)` → refresh 不变
- [ ] QS-02: 完整换票 + 期间 register-only → `refresh_access` 200
- [ ] QS-03: TEIP register → HTTP 200，无 500
- [ ] QS-04: 审计含 `token_bootstrap_blocked` 或 `relay_register_reused_state`
- [ ] QS-05: go_relay 日志无 `using original token`
- [ ] Playwright: 直启日志无 `无效的 access_token`
