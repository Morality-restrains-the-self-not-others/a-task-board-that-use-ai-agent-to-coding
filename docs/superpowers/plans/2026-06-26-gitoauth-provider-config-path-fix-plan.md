# 实施计划: gitOauth Provider 配置加载路径修复

> 输入:
> - 设计文档: `.claude/skills/1-brainstorming-设计文档/design.md`
> - 价值流: `docs/superpowers/plans/2026-06-26-gitoauth-provider-config-path-fix-value-stream.md`
> - NFR 澄清: `docs/superpowers/plans/2026-06-26-gitoauth-provider-config-path-fix-nfr-clarification.md`
> - DDD: `docs/superpowers/plans/2026-06-26-gitoauth-provider-config-path-fix-ddd.md` (跳过)

## 任务总览

| # | 任务 | 文件 | 增量 | 依赖 |
|---|------|------|------|------|
| 1 | 修正提供者配置加载路径 | `gitOauth/config/port_config.py` | Increment 1 | — |
| 2 | 新增 `load_merged` 提供者加载测试 | `gitOauth/config/test_port_config.py` | Increment 1 | 1 |
| 3 | 添加路径回退兼容机制 | `gitOauth/config/port_config.py` | Increment 2 | 1 |
| 4 | 新增路径回退测试 | `gitOauth/config/test_port_config.py` | Increment 2 | 3 |
| 5 | DEBUG 模式错误透传 | `gitOauth/api/gitlab_browser_views.py`, `gitOauth/api/github_browser_views.py` | Increment 3 | 1 |

---

## 详细任务

### Task 1: 修正提供者配置加载路径 ☐

**文件:** `gitOauth/config/port_config.py:53`
**增量:** Increment 1 — Thin Slice
**依赖:** 无

**修改:**

```diff
-    prov_dir = root / "conf" / "git-oauth" / "providers"
+    prov_dir = root / "conf" / "auth" / "git-oauth" / "providers"
```

**验证:**
```bash
cd gitOauth && python -c "
from config.port_config import load_merged
cfg = load_merged()
assert cfg.get('gitOauth'), 'gitOauth catalog should not be empty'
print(f'Loaded {len(cfg[\"gitOauth\"])} provider(s)')
"
```

---

### Task 2: 新增 `load_merged` 提供者加载测试 ☐

**文件:** `gitOauth/config/test_port_config.py`
**增量:** Increment 1 — Thin Slice
**依赖:** Task 1

**新增测试用例:**

```python
class TestLoadMergedProviderConfigs:
    """Tests for provider config loading in load_merged()."""

    def test_loads_providers_from_correct_path(self):
        """load_merged() loads gitOauth providers from conf/auth/git-oauth/providers/."""
        with tempfile.TemporaryDirectory() as tmpdir:
            prov_dir = Path(tmpdir) / "conf" / "auth" / "git-oauth" / "providers"
            prov_dir.mkdir(parents=True)
            (prov_dir / "test-gitlab.yaml").write_text(
                "provider: gitlab\n"
                "service_provider: test-gitlab\n"
                "target:\n"
                "  website: http://example.com:8012\n"
                "  client_id: test-id\n"
                "  client_secret: test-secret\n"
                "  redirect_uri: http://localhost/callback/\n"
                "  scope: read_repository\n"
                "service:\n"
                "  allowedHost: http://127.0.0.1:8002\n"
                "  host: 127.0.0.1\n"
                "  port: 8002\n"
            )

            from config import port_config

            with patch.object(
                port_config, "_ram_mount_root", return_value=Path(tmpdir)
            ):
                result = port_config.load_merged()

            assert "gitOauth" in result
            assert len(result["gitOauth"]) == 1
```

**验证:**
```bash
cd gitOauth && python -m pytest config/test_port_config.py -v
```

---

### Task 3: 添加路径回退兼容机制 ☐

**文件:** `gitOauth/config/port_config.py` (在 `load_merged()` 中)
**增量:** Increment 2 — Essential Support
**依赖:** Task 1

**修改:** 在 `prov_dir` 检查后添加回退逻辑：

```python
prov_dir = root / "conf" / "auth" / "git-oauth" / "providers"
# Fallback: try old path for backward compatibility
if not prov_dir.is_dir():
    prov_dir = root / "conf" / "git-oauth" / "providers"
```

完整上下文（替换第 53-54 行）:

```python
    catalog: dict[str, Any] = {}
    prov_dir = root / "conf" / "auth" / "git-oauth" / "providers"
    if not prov_dir.is_dir():
        prov_dir = root / "conf" / "git-oauth" / "providers"
    if prov_dir.is_dir():
        for path in sorted(prov_dir.glob("*.yaml")):
            ...
```

**验证:**
```bash
cd gitOauth && python -c "
from pathlib import Path
import tempfile, os
# Create old path only, verify fallback works
# (manual verification — see test in Task 4)
print('Fallback mechanism in place')
"
```

---

### Task 4: 新增路径回退测试 ☐

**文件:** `gitOauth/config/test_port_config.py`
**增量:** Increment 2 — Essential Support
**依赖:** Task 3

**新增测试用例:**

```python
    def test_fallback_to_old_path_when_new_path_missing(self):
        """When conf/auth/git-oauth/providers/ is missing, falls back to conf/git-oauth/providers/."""
        with tempfile.TemporaryDirectory() as tmpdir:
            # Create only the old path
            old_prov_dir = Path(tmpdir) / "conf" / "git-oauth" / "providers"
            old_prov_dir.mkdir(parents=True)
            (old_prov_dir / "test-legacy.yaml").write_text(
                "provider: gitlab\n"
                "service_provider: legacy-gitlab\n"
                "target:\n"
                "  website: http://legacy.example.com\n"
                "  client_id: legacy-id\n"
                "  client_secret: legacy-secret\n"
                "  redirect_uri: http://localhost/callback/\n"
                "  scope: read_repository\n"
                "service:\n"
                "  allowedHost: http://127.0.0.1:8002\n"
                "  host: 127.0.0.1\n"
                "  port: 8002\n"
            )

            from config import port_config

            with patch.object(
                port_config, "_ram_mount_root", return_value=Path(tmpdir)
            ):
                result = port_config.load_merged()

            assert "gitOauth" in result
            assert len(result["gitOauth"]) == 1
```

**验证:**
```bash
cd gitOauth && python -m pytest config/test_port_config.py::TestLoadMergedProviderConfigs -v
```

---

### Task 5: DEBUG 模式错误透传 ☐

**文件:**
- `gitOauth/api/gitlab_browser_views.py` (修改 `_gateway_start_error`)
- `gitOauth/api/github_browser_views.py` (修改 `_gateway_start_error`)

**增量:** Increment 3 — Enhancement
**依赖:** Task 1

**修改 `_gateway_start_error` 函数签名与实现 (GitLab):**

`gitlab_browser_views.py:323-325`:

```diff
-def _gateway_start_error(code: str):
+def _gateway_start_error(code: str, reason: str = ""):
     from rest_framework.response import Response
-    return Response({"detail": f"无法启动 GitLab 授权（{code}）"}, status=503)
+    from django.conf import settings
+    detail = f"无法启动 GitLab 授权（{code}）"
+    if getattr(settings, "DEBUG", False) and reason:
+        detail = f"{detail} [debug: {reason}]"
+    return Response({"detail": detail}, status=503)
```

**修改调用处 (第 303 行):**

```diff
-return _gateway_start_error("bad_state")
+return _gateway_start_error("bad_state", reason=str(route_reason or "route_decision_failed"))
```

**同样修改 GitHub 版本:**

`github_browser_views.py:209-211` + 调用处。

**验证:**
```bash
# 在 DEBUG=True 环境下触发 OAuth start，确认响应体包含 [debug: ...]
cd gitOauth && python -m pytest api/tests.py -v -k "start"
```

---

## 执行顺序

```
Task 1 (路径修复) ──→ Task 2 (测试)
    │
    ├──→ Task 3 (回退) ──→ Task 4 (回退测试)
    │
    └──→ Task 5 (DEBUG 透传)
```

Tasks 1-2 构成 Thin Slice（最小可交付修复）。
Tasks 3-5 可并行执行（无相互依赖）。

## 风险与回滚

| 风险 | 缓解 |
|------|------|
| 路径修复后配置加载仍为空 | Task 2 单测在临时目录中验证加载逻辑 |
| 回退机制掩盖真实配置错误 | 仅在 `is_dir()` 检查前 fallback；两个路径都不存在时仍返回空 |
| DEBUG 透传泄露敏感信息 | `reason` 仅包含错误码（如 `missing_allowed_host`），不包含 token/secret |
