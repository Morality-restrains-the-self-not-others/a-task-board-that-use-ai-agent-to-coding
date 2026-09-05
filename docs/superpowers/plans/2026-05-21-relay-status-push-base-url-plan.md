# Relay Status Push Base URL Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** 修复 relayToTrae 状态上报目标 URL 选择错误，避免 `onlineServiceJS` 实际已启动但任务页显示“未启动”。

**Architecture:** 在 `SettingsManager.get_relay_task_api_base_url()` 增加更稳健的 origin 选择：优先使用 `vue.apiBaseUrl` 中可达的非回环地址（如 API 网关），否则保持回环 Django 走 `internalApiBase` 的现有行为。通过单测覆盖“优先公网”和“回退内网”两条路径，确保后续配置调整不回归。

**Tech Stack:** Python 3, Django settings manager, pytest

---

## DDD Structure Check (Backend)
- 本需求属于配置解析与应用层连通性修复，不新增实体/值对象/聚合/仓储接口/领域事件。
- 分层保持不变：仅修改 `core/config`（配置层）与 `tests`（验证层），不引入基础设施依赖到 domain。
- 任务顺序仍遵循“测试先行（契约）→ 实现 → 回归验证”。

### Task 1: Add Failing Contract Test For Relay Base URL Priority

**Files:**
- Modify: `task2app/Saas_project/tests/test_port_config_merge.py`
- Test: `task2app/Saas_project/tests/test_port_config_merge.py`

- [ ] **Step 1: 写失败测试（当 `vue.apiBaseUrl` 为公网地址时应优先返回）**

```python
def test_get_relay_task_api_base_url_prefers_frontend_api_base_url(monkeypatch):
    from core.config import settings_manager

    monkeypatch.setattr(
        settings_manager,
        "_config",
        {
            "django": {
                "host": "localhost",
                "port": 8001,
                "allowedHost": "https://api.example.test",
                "internalApiBase": "http://127.0.0.1:8001",
            },
            "vue": {
                "host": "localhost",
                "port": 4000,
                "apiBaseUrl": "https://gateway.example.test",
            },
        },
    )
    assert (
        settings_manager.get_relay_task_api_base_url()
        == "https://gateway.example.test"
    )
```

- [ ] **Step 2: 运行测试并确认失败**

Run: `python3 -m pytest task2app/Saas_project/tests/test_port_config_merge.py -k "prefers_frontend_api_base_url" -v`  
Expected: FAIL，当前实现返回 `http://127.0.0.1:8001`

- [ ] **Step 3: Commit**

```bash
git add task2app/Saas_project/tests/test_port_config_merge.py
git commit -m "test: cover relay task api base url priority to frontend api base"
```

### Task 2: Implement URL Origin Normalization And Selection Logic

**Files:**
- Modify: `task2app/Saas_project/core/config/settings_manager.py`
- Test: `task2app/Saas_project/tests/test_port_config_merge.py`

- [ ] **Step 1: 最小实现 origin 归一化方法（仅接受合法 http/https origin）**

```python
@staticmethod
def _normalize_origin(raw: str) -> str:
    text = str(raw or "").strip().rstrip("/")
    if not text:
        return ""
    candidate = text if "://" in text else f"http://{text}"
    parsed = urlparse(candidate)
    if parsed.scheme not in ("http", "https") or not parsed.netloc:
        return ""
    return f"{parsed.scheme}://{parsed.netloc}"
```

- [ ] **Step 2: 修改 relay base URL 选择逻辑**

```python
def get_relay_task_api_base_url(self) -> str:
    frontend_api_base = self._normalize_origin(
        self.get_frontend_config().get("apiBaseUrl", "")
    )
    if frontend_api_base:
        parsed = urlparse(frontend_api_base)
        if not self._is_loopback_host(parsed.hostname or ""):
            return frontend_api_base
    backend_config = self.get_backend_config()
    host = str(backend_config.get("host") or "").strip()
    if self._is_loopback_host(host):
        return self.get_internal_task_api_base_url()
    return self.get_public_task_api_base_url()
```

- [ ] **Step 3: 运行针对性测试并确认通过**

Run: `python3 -m pytest task2app/Saas_project/tests/test_port_config_merge.py -k "relay_task_api_base_url" -v`  
Expected: PASS（含“loopback 回退”与“frontend 优先”两个用例）

- [ ] **Step 4: Commit**

```bash
git add task2app/Saas_project/core/config/settings_manager.py task2app/Saas_project/tests/test_port_config_merge.py
git commit -m "fix: prioritize frontend api base for relay status push target"
```

### Task 3: Run Safety Regression For Relay Status Proxy

**Files:**
- Test: `task2app/Saas_project/tests/test_relay_to_trae_proxy.py`
- Test: `task2app/Saas_project/tests/test_port_config_merge.py`

- [ ] **Step 1: 运行配置与代理相关回归测试**

Run: `python3 -m pytest task2app/Saas_project/tests/test_port_config_merge.py task2app/Saas_project/tests/test_relay_to_trae_proxy.py -k "relay" -v`  
Expected: PASS，无新增失败

- [ ] **Step 2: 本地链路冒烟验证（可选但推荐）**

Run: `curl -sS -H "X-Relay-To-Trae-Secret: dev-secret" "http://127.0.0.1:8797/v1/status"`  
Expected: JSON 返回中 `running/online_service_up` 与实际服务状态一致，且不出现持续 `status push failed ... timeout`

- [ ] **Step 3: Commit**

```bash
git add -A
git commit -m "chore: verify relay status push url selection regression"
```

### Task 4: Document Operational Verification Steps

**Files:**
- Modify: `docs/runbooks/oauth-refresh-push-timeout-troubleshooting.md`

- [ ] **Step 1: 补充一节“任务页显示未启动但 onlineServiceJS 可访问”排查流程**

```markdown
## 任务页显示「onlineServiceJS 未启动」但 8765 可访问

1. 检查 relay 状态：
   `curl -sS -H "X-Relay-To-Trae-Secret: dev-secret" http://127.0.0.1:8797/v1/status`
2. 若 logs 出现连续 `status push failed ... timeout`，检查 `get_relay_task_api_base_url()` 选择值。
3. 本地 Django 回环 + 前端公网 API 网关时，应优先使用 `vue.apiBaseUrl` 作为 relay 上报目标。
```

- [ ] **Step 2: 提交文档**

```bash
git add docs/runbooks/oauth-refresh-push-timeout-troubleshooting.md
git commit -m "docs: add relay status push mismatch troubleshooting"
```
