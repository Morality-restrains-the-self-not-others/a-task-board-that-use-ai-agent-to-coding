# runAll 健康/就绪端点 Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** 为 `config.yaml` 所列服务实现统一 Readiness 探针（200/503 + `checks` JSON），并更新 runAll 探针 URL。

**Architecture:** 各 bounded context 内联实现；Django 在 `core/health/`（Saas_project）或 `api/views.py`（gitOauth/ai-provider）做基础设施 ping；Go 扩展现有 `/health`；Vite 用 `configureServer` 中间件。领域层无健康逻辑。

**Tech Stack:** Django 4.x + DRF、confluent-kafka、redis-py、Go net/http、Vite 5

**Spec:** `docs/superpowers/specs/2026-05-19-runall-health-endpoints-design.md`（已批准）

**Build status:** ✅ 已完成（2026-05-19）  
**TDD status:** ✅ 已补强（2026-05-19）— 见 `tests/test_health_checks_unit.py`、503 场景、Go docker 注入测试

---

## File Map

| 文件 | 职责 |
|------|------|
| `task2app/Saas_project/core/health/checks.py` | DB/Kafka/Redis 基础设施检查 |
| `task2app/Saas_project/core/health/views.py` | 汇总 checks → 200/503 |
| `task2app/Saas_project/saas_project/urls.py` | 注册 `api/health/` |
| `task2app/Saas_project/tests/test_health_endpoint.py` | saas-backend 探针测试 |
| `task2app/Saas_Ai_Provider/apps/marketplace/health.py` | ai-provider checks + view |
| `task2app/Saas_Ai_Provider/provider/urls.py` | 注册 `api/health/` |
| `task2app/Saas_Ai_Provider/apps/marketplace/tests/test_health.py` | ai-provider 测试 |
| `gitOauth/api/health_checks.py` | gitOauth DB check |
| `gitOauth/api/views.py` | 扩展 `HealthView` |
| `gitOauth/api/tests.py` | HealthView 测试 |
| `task2app/front_project/app/vite.config.js` | `/health` 中间件 |
| `go_run_container/src/server.go` | docker check + 统一 JSON |
| `go_run_container/src/server_test.go` | 503 场景 |
| `go_relayToTrae/handlers.go` | 统一 JSON（Liveness） |
| `go_relayToTrae/handlers_test.go` | 响应字段断言 |
| `config.yaml` | 更新 `health_check.url` |

---

### Task 1: saas-backend — `core/health` 模块

**Files:**
- Create: `task2app/Saas_project/core/health/__init__.py`
- Create: `task2app/Saas_project/core/health/checks.py`
- Create: `task2app/Saas_project/core/health/views.py`
- Create: `task2app/Saas_project/tests/test_health_endpoint.py`
- Modify: `task2app/Saas_project/saas_project/urls.py`

- [ ] **Step 1: 写失败测试**

创建 `task2app/Saas_project/tests/test_health_endpoint.py`：

```python
import pytest
from django.test import Client
from unittest.mock import patch


@pytest.mark.django_db
def test_health_ok_memory_queue_skips_kafka_redis():
    client = Client()
    with patch("config.port_config.use_memory_message_queue", return_value=True):
        r = client.get("/api/health/")
    assert r.status_code == 200
    body = r.json()
    assert body["service"] == "saas-backend"
    assert body["ok"] is True
    assert body["checks"]["database"]["ok"] is True
    assert body["checks"]["kafka"]["skipped"] is True
    assert body["checks"]["redis"]["skipped"] is True


@pytest.mark.django_db
def test_health_503_when_database_fails():
    client = Client()
    with patch("core.health.checks.run_database_check", return_value={"ok": False, "error": "db down"}):
        with patch("config.port_config.use_memory_message_queue", return_value=True):
            r = client.get("/api/health/")
    assert r.status_code == 503
    assert r.json()["ok"] is False
```

- [ ] **Step 2: 运行测试确认失败**

```bash
cd task2app/Saas_project
../activate_env.sh unit -- pytest tests/test_health_endpoint.py -v
```

Expected: FAIL（404 或 import error）

- [ ] **Step 3: 实现 `checks.py`**

```python
# task2app/Saas_project/core/health/checks.py
import os
import time
from typing import Any, Dict, Optional

from django.db import connection


def _elapsed_ms(start: float) -> int:
    return int((time.monotonic() - start) * 1000)


def _check(ok: bool, *, skipped: bool = False, latency_ms: Optional[int] = None, error: Optional[str] = None) -> Dict[str, Any]:
    out: Dict[str, Any] = {"ok": ok}
    if skipped:
        out["skipped"] = True
    if latency_ms is not None:
        out["latency_ms"] = latency_ms
    if error:
        out["error"] = error
    return out


def run_database_check() -> Dict[str, Any]:
    start = time.monotonic()
    try:
        connection.ensure_connection()
        with connection.cursor() as cursor:
            cursor.execute("SELECT 1")
        return _check(True, latency_ms=_elapsed_ms(start))
    except Exception as exc:
        return _check(False, latency_ms=_elapsed_ms(start), error=str(exc))


def run_kafka_check() -> Dict[str, Any]:
    from config.port_config import use_memory_message_queue

    if use_memory_message_queue():
        return _check(True, skipped=True)

    start = time.monotonic()
    try:
        from confluent_kafka.admin import AdminClient

        bootstrap = os.getenv("KAFKA_BOOTSTRAP_SERVERS", "localhost:9093")
        admin = AdminClient({"bootstrap.servers": bootstrap})
        admin.list_topics(timeout=2)
        return _check(True, latency_ms=_elapsed_ms(start))
    except Exception as exc:
        return _check(False, latency_ms=_elapsed_ms(start), error=str(exc))


def run_redis_check() -> Dict[str, Any]:
    from core.services.memory_pubsub import InMemoryPubSubClient
    from core.services.registry import get_pubsub_client

    if isinstance(get_pubsub_client(), InMemoryPubSubClient):
        return _check(True, skipped=True)

    start = time.monotonic()
    try:
        import redis

        client = redis.Redis(host="localhost", port=6379, db=0, socket_connect_timeout=2)
        client.ping()
        return _check(True, latency_ms=_elapsed_ms(start))
    except Exception as exc:
        return _check(False, latency_ms=_elapsed_ms(start), error=str(exc))


def run_all_checks() -> Dict[str, Any]:
    checks = {
        "database": run_database_check(),
        "kafka": run_kafka_check(),
        "redis": run_redis_check(),
    }
    ok = all(c.get("ok") or c.get("skipped") for c in checks.values())
    return {"service": "saas-backend", "ok": ok, "checks": checks}
```

- [ ] **Step 4: 实现 `views.py` 与路由**

`core/health/views.py`：

```python
from rest_framework.permissions import AllowAny
from rest_framework.response import Response
from rest_framework.views import APIView

from .checks import run_all_checks


class HealthView(APIView):
    authentication_classes = []
    permission_classes = [AllowAny]

    def get(self, request):
        payload = run_all_checks()
        status = 200 if payload["ok"] else 503
        return Response(payload, status=status)
```

`saas_project/urls.py` 顶部 import 后、其它 `api/` 路由附近添加：

```python
from core.health.views import HealthView

# ...
path("api/health/", HealthView.as_view(), name="api-health"),
```

`core/health/__init__.py` 可为空。

- [ ] **Step 5: 运行测试**

```bash
cd task2app/Saas_project
../activate_env.sh unit -- pytest tests/test_health_endpoint.py -v
```

Expected: PASS

- [ ] **Step 6: Commit**

```bash
git add task2app/Saas_project/core/health/ task2app/Saas_project/tests/test_health_endpoint.py task2app/Saas_project/saas_project/urls.py
git commit -m "feat(saas): add readiness health endpoint at /api/health/"
```

---

### Task 2: ai-provider — `/api/health/`

**Files:**
- Create: `task2app/Saas_Ai_Provider/apps/marketplace/health.py`
- Create: `task2app/Saas_Ai_Provider/apps/marketplace/tests/__init__.py`
- Create: `task2app/Saas_Ai_Provider/apps/marketplace/tests/test_health.py`
- Modify: `task2app/Saas_Ai_Provider/provider/urls.py`

- [ ] **Step 1: 写失败测试**

`apps/marketplace/tests/test_health.py`：

```python
from django.test import Client
import pytest


@pytest.mark.django_db
def test_ai_provider_health_ok():
    r = Client().get("/api/health/")
    assert r.status_code == 200
    body = r.json()
    assert body["service"] == "ai-provider"
    assert body["ok"] is True
    assert body["checks"]["database"]["ok"] is True
```

- [ ] **Step 2: 运行测试确认失败**

```bash
cd task2app/Saas_Ai_Provider
python3 manage.py test apps.marketplace.tests.test_health -v 2
```

Expected: FAIL

- [ ] **Step 3: 实现 `health.py`**

```python
# apps/marketplace/health.py
import time
from django.db import connection
from rest_framework.permissions import AllowAny
from rest_framework.response import Response
from rest_framework.views import APIView


def run_database_check():
    start = time.monotonic()
    try:
        connection.ensure_connection()
        with connection.cursor() as cursor:
            cursor.execute("SELECT 1")
        ms = int((time.monotonic() - start) * 1000)
        return {"ok": True, "latency_ms": ms}
    except Exception as exc:
        return {"ok": False, "error": str(exc)}


class HealthView(APIView):
    authentication_classes = []
    permission_classes = [AllowAny]

    def get(self, request):
        db = run_database_check()
        ok = db["ok"]
        payload = {
            "service": "ai-provider",
            "ok": ok,
            "checks": {"database": db},
        }
        return Response(payload, status=200 if ok else 503)
```

`provider/urls.py`：

```python
from apps.marketplace.health import HealthView

urlpatterns = [
    path("api/health/", HealthView.as_view(), name="api-health"),
    # ... existing ...
]
```

- [ ] **Step 4: 运行测试通过**

```bash
cd task2app/Saas_Ai_Provider
python3 manage.py test apps.marketplace.tests.test_health -v 2
```

- [ ] **Step 5: Commit**

```bash
git add task2app/Saas_Ai_Provider/apps/marketplace/health.py task2app/Saas_Ai_Provider/apps/marketplace/tests/ task2app/Saas_Ai_Provider/provider/urls.py
git commit -m "feat(ai-provider): add /api/health/ readiness endpoint"
```

---

### Task 3: git-oauth — 扩展 HealthView

**Files:**
- Create: `gitOauth/api/health_checks.py`
- Modify: `gitOauth/api/views.py`
- Modify: `gitOauth/api/swagger_serializers.py`（可选：为 `checks` 加字段）
- Modify: `gitOauth/api/tests.py`

- [ ] **Step 1: 写失败测试**

在 `gitOauth/api/tests.py` 末尾添加：

```python
class HealthViewTests(TestCase):
    def setUp(self):
        self.client = Client()

    def test_health_ok_includes_database_check(self):
        r = self.client.get("/api/health/")
        self.assertEqual(r.status_code, 200)
        body = r.json()
        self.assertTrue(body["ok"])
        self.assertEqual(body["service"], "gitOauth")
        self.assertTrue(body["checks"]["database"]["ok"])
```

- [ ] **Step 2: 运行测试确认失败**

```bash
cd gitOauth
python3 manage.py test api.tests.HealthViewTests -v 2
```

- [ ] **Step 3: 实现**

`api/health_checks.py` — 复制 saas 的 `run_database_check` 逻辑（独立文件，不跨项目 import）。

`api/views.py` — 修改 `HealthView.get`：

```python
from .health_checks import run_database_check

def get(self, request):
    header_base = request.headers.get("X-GitOauth-Allowed-Host")
    db = run_database_check()
    ok = db["ok"]
    payload = {
        "service": "gitOauth",
        "ok": ok,
        "checks": {"database": db},
        "public_base_url": header_base or settings.GITOAUTH_ALLOWED_HOST,
        "from_nginx_header": bool(header_base),
    }
    return Response(payload, status=200 if ok else 503)
```

- [ ] **Step 4: 运行测试通过**

- [ ] **Step 5: Commit**

```bash
git add gitOauth/api/
git commit -m "feat(gitOauth): extend /api/health/ with database readiness"
```

---

### Task 4: taskFE — Vite `/health`

**Files:**
- Modify: `task2app/front_project/app/vite.config.js`

- [ ] **Step 1: 在 `defineConfig` 的 `return` 中增加 `configureServer`**

在 `plugins: [...]` 后添加：

```javascript
    configureServer(server) {
      server.middlewares.use('/health', (req, res, next) => {
        if (req.method !== 'GET' && req.method !== 'HEAD') {
          return next()
        }
        res.statusCode = 200
        res.setHeader('Content-Type', 'application/json')
        res.end(JSON.stringify({ service: 'taskFE', ok: true, checks: {} }))
      })
    },
```

- [ ] **Step 2: 手动验证**

```bash
cd task2app/front_project/app && npm run dev
# 另一终端：
curl -s http://127.0.0.1:4000/health
```

Expected: `{"service":"taskFE","ok":true,"checks":{}}`

- [ ] **Step 3: Commit**

```bash
git add task2app/front_project/app/vite.config.js
git commit -m "feat(vue): add /health liveness endpoint for runAll"
```

---

### Task 5: go-run-container — Docker readiness

**Files:**
- Modify: `go_run_container/src/server.go`
- Modify: `go_run_container/src/server_test.go`

- [ ] **Step 1: 更新失败测试**

在 `TestHealthEndpoint` 中断言新 JSON 形状；新增 `TestHealthEndpointDockerUnavailable`（通过注入或 build tag 可选；最小实现：mock `exec` 较难，改为集成测试文档 + 单元测试 `aggregateHealth` 纯函数）。

推荐：抽出 `func healthPayload(dockerOK bool) map[string]any` 并测试聚合逻辑。

```go
func TestHealthPayloadAllOk(t *testing.T) {
    p := healthPayload(true)
    if !p["ok"].(bool) {
        t.Fatal("expected ok")
    }
}
```

- [ ] **Step 2: 实现 `healthPayload` + `handleHealth`**

```go
func healthPayload(dockerOK bool, dockerErr string) map[string]interface{} {
    checks := map[string]interface{}{}
    if dockerOK {
        checks["docker"] = map[string]interface{}{"ok": true}
    } else {
        checks["docker"] = map[string]interface{}{"ok": false, "error": dockerErr}
    }
    ok := dockerOK
    return map[string]interface{}{
        "service": "go-run-container",
        "ok":      ok,
        "checks":  checks,
    }
}

func (s *server) handleHealth(w http.ResponseWriter, r *http.Request) {
    dockerOK, dockerErr := checkDockerAvailable()
    payload := healthPayload(dockerOK, dockerErr)
    status := 200
    if !payload["ok"].(bool) {
        status = 503
    }
    writeJSON(w, status, payload)
}
```

`checkDockerAvailable` 使用 `exec.CommandContext` + 2s timeout 执行 `docker info`。

- [ ] **Step 3: 运行测试**

```bash
cd go_run_container/src && go test -run TestHealth -v
```

- [ ] **Step 4: Commit**

```bash
git add go_run_container/src/
git commit -m "feat(go_run_container): readiness /health with docker check"
```

---

### Task 6: go-relay — 统一 JSON（Liveness）

**Files:**
- Modify: `go_relayToTrae/handlers.go`
- Modify: `go_relayToTrae/handlers_test.go`

- [ ] **Step 1: 更新 `handleHealth`**

```go
func handleHealth(w http.ResponseWriter, r *http.Request) {
    writeJSON(w, 200, map[string]interface{}{
        "service": "go-relay",
        "ok":      true,
        "checks":  map[string]interface{}{},
    })
}
```

- [ ] **Step 2: 更新 `handlers_test.go` 中断言 `service` 字段**

- [ ] **Step 3: `go test ./...`**

```bash
cd go_relayToTrae && go test ./... -run Health -v
```

- [ ] **Step 4: Commit**

```bash
git add go_relayToTrae/
git commit -m "chore(go-relay): align /health JSON with runAll contract"
```

---

### Task 7: 更新 `config.yaml`

**Files:**
- Modify: `config.yaml`（仓库根）

- [ ] **Step 1: 替换探针 URL**

```yaml
      - name: saas-backend
        health_check:
          url: "http://127.0.0.1:8001/api/health/"

      - name: taskFE
        health_check:
          url: "http://127.0.0.1:4000/health"

      - name: ai-provider
        health_check:
          url: "http://127.0.0.1:8010/api/health/"
```

`git-oauth`、`go-run-container`、`go-relay` URL 不变。

- [ ] **Step 2: Commit**

```bash
git add config.yaml
git commit -m "chore(runAll): point health checks to readiness endpoints"
```

---

### Task 8: 端到端验证

- [ ] **Step 1: 构建 runAll**

```bash
cd runAll && go build -o runAll .
```

- [ ] **Step 2: 单独 curl 各端点**（服务已手动或 runAll 拉起）

```bash
curl -sf http://127.0.0.1:8001/api/health/ | head -c 200
curl -sf http://127.0.0.1:8010/api/health/ | head -c 200
curl -sf http://127.0.0.1:8002/api/health/ | head -c 200
curl -sf http://127.0.0.1:4000/health
curl -sf http://127.0.0.1:8796/health
curl -sf http://127.0.0.1:8797/health
```

- [ ] **Step 3: runAll 前台试跑**

```bash
./runAll --config ../config.yaml
```

打开 `http://localhost:9999`，确认各服务绿点；saas-backend 在 DB 未迁移时应 503 直至 migrate 完成。

---

## Spec Coverage Self-Review

| Spec 要求 | Task |
|-----------|------|
| saas-backend DB/Kafka/Redis readiness | Task 1 |
| ai-provider DB only | Task 2 |
| git-oauth 扩展 | Task 3 |
| vue Liveness `/health` | Task 4 |
| go-run-container Docker | Task 5 |
| go-relay JSON 对齐 | Task 6 |
| config.yaml 更新 | Task 7 |
| 503 on failure | Tasks 1–3, 5 |
| ≤3s check budget | `list_topics(timeout=2)`, redis `socket_connect_timeout=2` |
| DDD — 无 domain 代码 | 全部在 health/checks/views |

## Out of Scope（计划内不实现）

- runAll TCP 探针
- Saas_email / mock_run_container / onlineServiceJS
- 设计 doc 中已注释的 docker-infra
