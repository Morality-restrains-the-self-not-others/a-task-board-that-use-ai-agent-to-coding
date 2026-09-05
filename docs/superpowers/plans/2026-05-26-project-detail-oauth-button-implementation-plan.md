# ProjectDetail OAuth Routing Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** 让项目详情页仓库行 `OAuth 授权` 入口稳定走 `repo_url` 对应 provider 配置，并在配置缺失时返回可观测错误，同时保持前后端回归通过。

**Architecture:** 基于已完成的 `accounts/domain` 路由聚合（值对象 + 聚合根 + 领域服务）实现“领域优先”落地：先补领域测试，再实现基础设施仓储适配器，再在 `github_app_views` 接入领域服务。最后补 API/前端回归，确保不回退到 `DJANGO_GITOAUTH_BASE` 且配置双结构兼容。

**Tech Stack:** Django REST Framework, pytest, Vitest, Playwright, Python dataclasses/ABC

---

## File Structure

- Create: `task2app/Saas_project/accounts/tests/domain/test_git_oauth_authorize_route_domain_model.py`
- Create: `task2app/Saas_project/accounts/infrastructure/repositories/django_git_oauth_provider_routing_repository.py`
- Create: `task2app/Saas_project/accounts/infrastructure/repositories/__init__.py`
- Modify: `task2app/Saas_project/accounts/github_app_views.py`
- Modify: `task2app/Saas_project/accounts/git_oauth_providers.py`
- Modify: `task2app/Saas_project/tests/test_github_app_start_redirect_uri.py`
- Modify: `task2app/front_project/app/src/views/ProjectDetail.test.js`
- Verify: `task2app/front_project/app/src/views/ProjectDetail.vue`

---

### Task 1: 领域层路由模型测试先行

**Files:**
- Test: `task2app/Saas_project/accounts/tests/domain/test_git_oauth_authorize_route_domain_model.py`

- [ ] **Step 1: 写失败测试（RepoUrl + Catalog + RouteService）**

```python
def test_route_service_resolves_local_gitlab_repo():
    repo = InMemoryRoutingRepo(
        rules=("http://localhost:8012", "gitlab:local-gitlab", "http://localhost:8002"),
    )
    service = GitOauthAuthorizeRouteService(repo)
    resolved, rejected = service.resolve(
        provider="gitlab",
        repo_url="http://localhost:8012/ljy/somanyad",
        occurred_at=datetime.utcnow(),
    )
    assert rejected is None
    assert resolved is not None
    assert resolved.provider_key.value == "gitlab:local-gitlab"
    assert resolved.service_base == "http://localhost:8002"
```

- [ ] **Step 2: 运行单测确认失败**

Run: `cd task2app/Saas_project && pytest -q accounts/tests/domain/test_git_oauth_authorize_route_domain_model.py -k resolves_local`  
Expected: FAIL（提示仓储桩或断言字段不匹配）

- [ ] **Step 3: 补齐测试集（拒绝场景 + 值对象校验）**

```python
def test_route_service_rejects_when_no_rule():
    service = GitOauthAuthorizeRouteService(InMemoryRoutingRepo(rules=()))
    resolved, rejected = service.resolve(
        provider="gitlab",
        repo_url="http://localhost:8012/ljy/somanyad",
        occurred_at=datetime.utcnow(),
    )
    assert resolved is None
    assert rejected.reason == "no_matching_provider_route"
```

- [ ] **Step 4: 运行通过**

Run: `cd task2app/Saas_project && pytest -q accounts/tests/domain/test_git_oauth_authorize_route_domain_model.py`  
Expected: PASS

- [ ] **Step 5: Commit**

```bash
cd task2app
git add Saas_project/accounts/tests/domain/test_git_oauth_authorize_route_domain_model.py
git commit -m "test: add domain coverage for oauth route resolution"
```

---

### Task 2: 基础设施仓储适配器（domain/repositories -> settings）

**Files:**
- Create: `task2app/Saas_project/accounts/infrastructure/repositories/django_git_oauth_provider_routing_repository.py`
- Create: `task2app/Saas_project/accounts/infrastructure/repositories/__init__.py`

- [ ] **Step 1: 先写适配器测试（失败）**

```python
def test_django_routing_repository_builds_rules_from_settings(settings):
    settings.GIT_OAUTH_PROVIDER_CONFIGS = {
        "gitlab": [{
            "provider_key": "gitlab:local-gitlab",
            "allowedHost": "http://localhost:8012",
            "service_base": "http://localhost:8002",
        }]
    }
    repo = DjangoGitOauthProviderRoutingRepository()
    rules = repo.list_rules(provider="gitlab")
    assert len(rules) == 1
    assert rules[0].allowed_origin == "http://localhost:8012"
```

- [ ] **Step 2: 运行确认失败**

Run: `cd task2app/Saas_project && pytest -q accounts/tests/domain/test_git_oauth_authorize_route_domain_model.py -k builds_rules`  
Expected: FAIL（仓储类不存在）

- [ ] **Step 3: 实现最小适配器**

```python
class DjangoGitOauthProviderRoutingRepository(GitOauthProviderRoutingRepository):
    def list_rules(self, *, provider: str) -> tuple[OAuthProviderRouteRule, ...]:
        rows = (getattr(settings, "GIT_OAUTH_PROVIDER_CONFIGS", {}) or {}).get(provider, [])
        built = []
        for idx, row in enumerate(rows):
            if not isinstance(row, dict):
                continue
            base = str(row.get("service_base") or "").strip()
            if not base:
                host = str(row.get("host") or "").strip()
                port = row.get("port")
                if host and port not in (None, ""):
                    base = f"http://{host}:{int(port)}"
            if not base:
                continue
            built.append(
                OAuthProviderRouteRule(
                    id=f"{provider}:{idx}",
                    allowed_origin=str(row.get("allowedHost") or "").strip(),
                    provider_key=OAuthProviderKey(provider=provider, service_provider=str(row.get("service_provider") or "default")),
                    service_base=GitOauthServiceBase(base),
                )
            )
        return tuple(built)
```

- [ ] **Step 4: 运行相关测试通过**

Run: `cd task2app/Saas_project && pytest -q accounts/tests/domain/test_git_oauth_authorize_route_domain_model.py -k "builds_rules or resolves_local"`  
Expected: PASS

- [ ] **Step 5: Commit**

```bash
cd task2app
git add Saas_project/accounts/infrastructure/repositories/
git commit -m "feat: add django repository adapter for oauth route rules"
```

---

### Task 3: 接入 API 起始路由（GitHub/GitLab start）

**Files:**
- Modify: `task2app/Saas_project/accounts/github_app_views.py`
- Test: `task2app/Saas_project/tests/test_github_app_start_redirect_uri.py`

- [ ] **Step 1: 先写失败测试（API 应用领域服务）**

```python
def test_gitlab_start_uses_domain_route_resolution(api_client, auth_user):
    api_client.force_authenticate(user=auth_user)
    response = api_client.get("/api/accounts/gitlab/app/start/", {"repo_url": "http://localhost:8012/group/repo"})
    body = response.json()
    assert response.status_code == 200
    assert urlparse(body["authorize_url"]).netloc == "localhost:8002"
```

- [ ] **Step 2: 运行测试确认失败**

Run: `cd task2app/Saas_project && pytest -q tests/test_github_app_start_redirect_uri.py -k uses_domain_route_resolution`  
Expected: FAIL（旧逻辑未通过领域服务）

- [ ] **Step 3: 最小实现（注入领域服务 + 适配器）**

```python
repo = DjangoGitOauthProviderRoutingRepository()
route_service = GitOauthAuthorizeRouteService(repo)
resolved, rejected = route_service.resolve(
    provider="gitlab",
    repo_url=repo_url,
    occurred_at=datetime.utcnow(),
)
if rejected is not None:
    return Response({"detail": "服务端未配置 Git OAuth 服务根 URL..."}, status=503)
base = resolved.service_base
```

- [ ] **Step 4: 回归现有 start 用例**

Run: `cd task2app/Saas_project && pytest -q tests/test_github_app_start_redirect_uri.py`  
Expected: PASS（包含 no-fallback 约束）

- [ ] **Step 5: Commit**

```bash
cd task2app
git add Saas_project/accounts/github_app_views.py Saas_project/tests/test_github_app_start_redirect_uri.py
git commit -m "feat: route oauth start via domain service"
```

---

### Task 4: 配置归一化与 provider 解析一致性

**Files:**
- Modify: `task2app/Saas_project/accounts/git_oauth_providers.py`
- Modify: `task2app/Saas_project/saas_project/settings.py`（仅当测试发现缺字段时）

- [ ] **Step 1: 写失败测试（对象结构 + key 作为 allowedHost）**

```python
def test_provider_inference_accepts_dict_key_allowed_host(settings):
    settings.GIT_OAUTH_PROVIDER_CONFIGS = {
        "gitlab": [{
            "allowedHost": "http://localhost:8012",
            "service_provider": "local-gitlab",
            "provider_key": "gitlab:local-gitlab",
            "service_base": "http://localhost:8002",
        }]
    }
    provider, sp = resolve_provider_from_repo_url("http://localhost:8012/ljy/somanyad")
    assert provider == "gitlab"
    assert sp == "local-gitlab"
```

- [ ] **Step 2: 运行确认失败**

Run: `cd task2app/Saas_project && pytest -q tests/test_git_oauth_providers.py -k dict_key_allowed_host`  
Expected: FAIL（若解析遗漏）

- [ ] **Step 3: 实现最小修复**

```python
def _infer_provider_from_configured_allowed_hosts(repo_url: str | None) -> str | None:
    # 只比较 origin/hostname，不依赖 UI 传入格式
    ...
```

- [ ] **Step 4: 跑 provider 相关测试**

Run: `cd task2app/Saas_project && pytest -q tests/test_git_oauth_providers.py tests/test_github_app_start_redirect_uri.py`  
Expected: PASS

- [ ] **Step 5: Commit**

```bash
cd task2app
git add Saas_project/accounts/git_oauth_providers.py Saas_project/saas_project/settings.py Saas_project/tests/test_git_oauth_providers.py
git commit -m "fix: align provider inference with dict-based gitOauth config"
```

---

### Task 5: 前端路径回归（项目详情页）

**Files:**
- Modify: `task2app/front_project/app/src/views/ProjectDetail.test.js`
- Verify: `task2app/front_project/app/src/views/ProjectDetail.vue`

- [ ] **Step 1: 增加失败测试（local repo start 参数完整）**

```javascript
expect(String(startCall[0])).toContain('/api/accounts/gitlab/app/start/?')
expect(String(startCall[0])).toContain('repo_url=http%3A%2F%2Flocalhost%3A8012%2Fljy%2Fsomanyad')
expect(String(startCall[0])).toContain('return_key=')
expect(String(startCall[0])).toContain('next=')
```

- [ ] **Step 2: 运行前端单测确认失败**

Run: `cd task2app/front_project/app && npm run test -- src/views/ProjectDetail.test.js`  
Expected: FAIL（断言与当前 mock 不一致）

- [ ] **Step 3: 修正测试与最小代码**

```javascript
if (path.startsWith('/api/accounts/gitlab/app/start/')) {
  return { ok: true, json: async () => ({ authorize_url: 'https://oauth.example/gitlab-start' }) }
}
```

- [ ] **Step 4: 运行前端回归**

Run: `cd task2app/front_project/app && npm run test -- src/views/ProjectDetail.test.js`  
Expected: PASS

- [ ] **Step 5: Commit**

```bash
cd task2app
git add front_project/app/src/views/ProjectDetail.test.js front_project/app/src/views/ProjectDetail.vue
git commit -m "test: cover project detail oauth start params for local gitlab"
```

---

### Task 6: 端到端与合规收尾

**Files:**
- Verify: `task2app/playwright/front_project/tests/ProjectDetail.branch-preview-local-git.playwright.test.js`
- Verify: `task2app/Saas_project/accounts/domain/**/*.py`

- [ ] **Step 1: 运行 DDD 合规检查**

Run: `cd task2app && python scripts/ci/check_ddd_bdd_compliance.py`  
Expected: `DDD/BDD 合规检查通过。`

- [ ] **Step 2: 运行后端关键回归**

Run: `cd task2app/Saas_project && pytest -q tests/test_github_app_start_redirect_uri.py tests/test_project_branches_gitlab_auth_guard.py`  
Expected: PASS

- [ ] **Step 3: 运行前端关键回归**

Run: `cd task2app/front_project/app && npm run test -- src/views/ProjectDetail.test.js`  
Expected: PASS

- [ ] **Step 4: 运行定向 Playwright**

Run: `cd task2app/playwright && npx playwright test -c front_project/playwright.config.js --grep "GitLab 仓库行应显示 OAuth 授权按钮并发起 start 请求"`  
Expected: PASS

- [ ] **Step 5: 最终提交**

```bash
cd task2app
git add Saas_project/accounts Saas_project/tests front_project/app/src/views playwright/front_project/tests docs/superpowers/plans/2026-05-26-project-detail-oauth-button-implementation-plan.md
git commit -m "feat: stabilize project-detail oauth routing by repo provider config"
```

---

## Self-Review

- Spec coverage: 覆盖了价值流 3 个增量（薄切片入口、provider 路由、配置兼容回归）。
- Placeholder scan: 无 TBD/TODO/“后续补充”占位语句。
- Type consistency: 统一使用 `RepoUrl`、`OAuthProviderKey`、`GitOauthServiceBase` 与 `GitOauthAuthorizeRouteService` 命名。
