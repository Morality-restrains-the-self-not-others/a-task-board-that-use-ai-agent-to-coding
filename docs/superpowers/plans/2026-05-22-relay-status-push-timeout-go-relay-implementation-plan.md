# Relay Status Push Timeout (Go Relay) Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** 修复 `go_relayToTrae` 在代理环境下的 status-push / token-exchange 超时问题，避免连续 missed ACK 误注销任务。

**Architecture:** 以“统一后端出站 no-proxy client”为核心改造点，在 Go relay 内集中封装 HTTP client 构建逻辑，并让 `push.go` 与 `token.go` 都走同一策略。领域层契约（`RelayTaskRegistration`、`RelayStatusPushRoutingService`）只作为策略与状态语义约束，基础设施实现在其后完成，保持分层边界。

**Tech Stack:** Go 1.22+, Python 3, pytest, go test, existing `go_relayToTrae` runtime

---

I'm using the writing-plans skill to create the implementation plan.

## Scope Check

- 本次只覆盖一个子系统：`go_relayToTrae` 后端出站网络策略修复（status-push + token-exchange）。
- 不改 `seq/ack` 协议，不改 Django 的 `TASK_API_ENDPOINT_ORIGIN` 选择逻辑，不新增 DB 字段。

## File Structure (before tasks)

- `go_relayToTrae/src/http_client.go`（Create）  
  统一封装 no-proxy 后端 HTTP client 构造，避免各处重复声明 `http.Client`。
- `go_relayToTrae/src/http_client_test.go`（Create）  
  验证 client transport 语义（Proxy 必须为 `nil`）。
- `go_relayToTrae/src/push.go`（Modify）  
  status-push 改为使用统一 client 构造函数。
- `go_relayToTrae/src/token.go`（Modify）  
  token-exchange / refresh-access 改为使用统一 client 构造函数。
- `go_relayToTrae/src/token_test.go`（Modify）  
  覆盖 default backend client 的 no-proxy 约束。
- `go_relayToTrae/src/push_test.go`（Modify）  
  覆盖 status-push 使用 no-proxy backend client 的约束。
- `task2app/Saas_project/tests/cloud/domain/test_relay_status_push_routing_service.py`（Create）  
  锁定 DDD 契约：`RelayStatusPushRoutingService` 选择 `no_proxy` 与 missed-ack 收敛行为。

## DDD Structure Check (Backend)

- 领域契约优先：先补领域服务契约测试（`cloud/domain/*`），再做基础设施实现（`go_relayToTrae/src/*`）。
- 仓储依赖方向不变：领域服务仅依赖 `domain/repositories` ABC，不引入 ORM/HTTP SDK。
- 事件契约不变：继续使用 `RelayOutboundRouteSelected` 与 `RelayTaskUnregisteredAfterMissedAck` 语义。
- 基础设施层（Go relay）实现完成后，只通过测试验证契约一致性，不把基础设施导入领域层。

### Task 1: Lock Domain Contract For Routing And ACK Convergence

**Files:**
- Create: `task2app/Saas_project/tests/cloud/domain/test_relay_status_push_routing_service.py`
- Test: `task2app/Saas_project/tests/cloud/domain/test_relay_status_push_routing_service.py`

- [ ] **Step 1: 写领域契约测试（路由必须选择 no_proxy）**

```python
from datetime import datetime

from cloud.domain.entities.relay_task_registration import RelayTaskRegistration
from cloud.domain.repositories.relay_task_registration_repository import RelayTaskRegistrationRepository
from cloud.domain.services.relay_status_push_routing_service import RelayStatusPushRoutingService
from cloud.domain.value_objects.task_scope import TaskScope


class _InMemoryRelayTaskRegistrationRepository(RelayTaskRegistrationRepository):
    def __init__(self, registration: RelayTaskRegistration):
        self._registration = registration

    def find_latest_by_scope(self, scope: TaskScope):
        if self._registration.scope.to_identity_key() == scope.to_identity_key():
            return self._registration
        return None

    def save(self, registration: RelayTaskRegistration):
        self._registration = registration
        return registration


def test_select_mode_returns_no_proxy_for_status_push():
    now = datetime.now()
    scope = TaskScope(tenant_id="t1", workspace_id="w1", task_id="task1")
    registration = RelayTaskRegistration.create(
        scope=scope,
        task_api_origin="http://api.daydaymoney.com",
        now=now,
    )
    repo = _InMemoryRelayTaskRegistrationRepository(registration)
    service = RelayStatusPushRoutingService(repo)

    event = service.select_mode(scope=scope, channel="status-push", now=now)
    assert event.mode.is_no_proxy() is True
```

- [ ] **Step 2: 写领域契约测试（连续 missed-ack 达阈值触发注销事件）**

```python
def test_record_ack_result_emits_unregistered_event_on_threshold():
    now = datetime.now()
    scope = TaskScope(tenant_id="t1", workspace_id="w1", task_id="task1")
    registration = RelayTaskRegistration.create(
        scope=scope,
        task_api_origin="http://api.daydaymoney.com",
        now=now,
        ack_threshold=2,
    )
    repo = _InMemoryRelayTaskRegistrationRepository(registration)
    service = RelayStatusPushRoutingService(repo)

    first = service.record_ack_result(scope=scope, ack_succeeded=False, now=now)
    second = service.record_ack_result(scope=scope, ack_succeeded=False, now=now)

    assert first is None
    assert second is not None
    assert second.threshold == 2
```

- [ ] **Step 3: 运行测试并确认通过（锁定领域契约）**

Run: `cd task2app/Saas_project && python3 -m pytest tests/cloud/domain/test_relay_status_push_routing_service.py -v`  
Expected: PASS（2 passed）

- [ ] **Step 4: Commit**

```bash
git add task2app/Saas_project/tests/cloud/domain/test_relay_status_push_routing_service.py
git commit -m "test: lock relay routing domain contract for no-proxy and missed-ack convergence"
```

### Task 2: Add Failing Go Tests For Unified Backend No-Proxy Client

**Files:**
- Create: `go_relayToTrae/src/http_client_test.go`
- Modify: `go_relayToTrae/src/token_test.go`
- Modify: `go_relayToTrae/src/push_test.go`
- Test: `go_relayToTrae/src/http_client_test.go`
- Test: `go_relayToTrae/src/token_test.go`
- Test: `go_relayToTrae/src/push_test.go`

- [ ] **Step 1: 添加编译失败测试，声明目标函数 `newBackendHTTPClient`**

```go
package main

import (
	"net/http"
	"testing"
	"time"
)

func TestNewBackendHTTPClientDisablesProxy(t *testing.T) {
	client := newBackendHTTPClient(2 * time.Second)
	transport, ok := client.Transport.(*http.Transport)
	if !ok {
		t.Fatalf("expected *http.Transport, got %T", client.Transport)
	}
	if transport.Proxy != nil {
		t.Fatalf("expected nil Proxy function, got non-nil")
	}
}
```

- [ ] **Step 2: 在 token/push 测试中增加“必须复用统一后端 client”断言**

```go
func TestDefaultHTTPClientDisablesProxy(t *testing.T) {
	transport, ok := defaultHTTPClient.Transport.(*http.Transport)
	if !ok {
		t.Fatalf("expected *http.Transport, got %T", defaultHTTPClient.Transport)
	}
	if transport.Proxy != nil {
		t.Fatalf("expected nil Proxy function on defaultHTTPClient")
	}
}
```

```go
func TestBackendHTTPClientFactoryDisablesProxy(t *testing.T) {
	client := newBackendHTTPClient(time.Second)
	transport, ok := client.Transport.(*http.Transport)
	if !ok {
		t.Fatalf("expected *http.Transport, got %T", client.Transport)
	}
	if transport.Proxy != nil {
		t.Fatalf("expected nil Proxy function")
	}
}
```

- [ ] **Step 3: 运行测试并确认失败（Red）**

Run: `cd go_relayToTrae && go test ./... -run "Test(NewBackendHTTPClientDisablesProxy|DefaultHTTPClientDisablesProxy|BackendHTTPClientFactoryDisablesProxy)" -v`  
Expected: FAIL（`undefined: newBackendHTTPClient` 或 Transport 断言失败）

- [ ] **Step 4: Commit（仅测试）**

```bash
git add go_relayToTrae/src/http_client_test.go go_relayToTrae/src/token_test.go go_relayToTrae/src/push_test.go
git commit -m "test: add failing no-proxy backend client contract for go relay"
```

### Task 3: Implement Unified No-Proxy Backend Client In Go Relay

**Files:**
- Create: `go_relayToTrae/src/http_client.go`
- Modify: `go_relayToTrae/src/push.go`
- Modify: `go_relayToTrae/src/token.go`
- Test: `go_relayToTrae/src/http_client_test.go`
- Test: `go_relayToTrae/src/token_test.go`
- Test: `go_relayToTrae/src/push_test.go`

- [ ] **Step 1: 新增统一 client 工厂（no-proxy backend client）**

```go
package main

import (
	"net"
	"net/http"
	"time"
)

func newBackendHTTPClient(timeout time.Duration) *http.Client {
	return &http.Client{
		Transport: &http.Transport{
			Proxy: nil,
			DialContext: (&net.Dialer{
				Timeout: 5 * time.Second,
			}).DialContext,
		},
		Timeout: timeout,
	}
}
```

- [ ] **Step 2: 修改 `push.go`，status-push 使用统一 client**

```go
client := newBackendHTTPClient(time.Duration(pushTimeoutSec * float64(time.Second)))

resp, err := client.Do(req)
if err != nil {
	appendLog(fmt.Sprintf("[relayToTrae] status push failed: %v", err))
	return false
}
```

- [ ] **Step 3: 修改 `token.go`，default client 走统一工厂**

```go
var defaultHTTPClient = newBackendHTTPClient(30 * time.Second)
```

```go
clientWithTimeout := newBackendHTTPClient(timeout)
```

- [ ] **Step 4: 运行针对性测试并确认通过（Green）**

Run: `cd go_relayToTrae && go test ./... -run "Test(NewBackendHTTPClientDisablesProxy|DefaultHTTPClientDisablesProxy|BackendHTTPClientFactoryDisablesProxy)" -v`  
Expected: PASS

- [ ] **Step 5: Commit**

```bash
git add go_relayToTrae/src/http_client.go go_relayToTrae/src/push.go go_relayToTrae/src/token.go
git commit -m "fix: disable proxy for go relay backend status and token calls"
```

### Task 4: Full Regression For Relay Status Convergence

**Files:**
- Test: `go_relayToTrae/src/*.go`
- Test: `task2app/Saas_project/tests/test_relay_to_trae_status.py`
- Test: `task2app/Saas_project/cloud/view_test/test_relay_token_audit_status_push_view.py`

- [ ] **Step 1: 运行 Go 全量测试**

Run: `cd go_relayToTrae && go test ./... -v`  
Expected: PASS（all go relay tests passed）

- [ ] **Step 2: 运行 Django status-push 关键回归**

Run: `cd task2app/Saas_project && python3 -m pytest tests/test_relay_to_trae_status.py cloud/view_test/test_relay_token_audit_status_push_view.py -v`  
Expected: PASS（status-push ACK 与审计链路不回归）

- [ ] **Step 3: 手工冒烟验证（代理环境下）**

Run: `HTTP_PROXY=http://127.0.0.1:9 HTTPS_PROXY=http://127.0.0.1:9 ./go_relayToTrae/bin/go_relayToTrae`  
Expected: relay 启动后 status-push 不再持续出现 `context deadline exceeded`，且不会触发 `5 consecutive missed ACKs` 误注销

- [ ] **Step 4: Commit**

```bash
git add -A
git commit -m "test: verify relay status convergence under proxy environment"
```

## Self-Review

- **Spec coverage:** 已覆盖设计文档三项核心要求：统一 no-proxy、覆盖 status/token 两条链路、测试回归保障。
- **Placeholder scan:** 计划中无 TBD/TODO/“自行处理”类占位语句。
- **Type consistency:** 统一使用 `RelayStatusPushRoutingService`、`RelayTaskRegistration`、`RelayOutboundMode` 命名；Go 侧统一使用 `newBackendHTTPClient`。

## Execution Handoff

Plan complete and saved to `docs/superpowers/plans/2026-05-22-relay-status-push-timeout-go-relay-implementation-plan.md`.

Two execution options:

1. **Subagent-Driven (recommended)** - 我按任务逐个派发子代理执行，并在每个任务后回审  
2. **Inline Execution** - 我在当前会话按任务顺序直接执行，按检查点回报

Which approach?
