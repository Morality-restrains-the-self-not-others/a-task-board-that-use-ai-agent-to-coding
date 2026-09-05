# GitHub UID 字符串化与缓存治理 Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** 在 task2app + gitOauth + 前端链路中统一 `github_user_id` 为数字字符串，并强制 token 缓存 `access_token_expires_at` 非空，确保绑定、摘要、出站 PR 链路稳定一致。

**Architecture:** 先锁定并验证 `cloud/domain` 中的 UID/缓存契约，再将契约下沉到 Django 仓储与服务实现，随后统一 API/前端交互字段，最后执行跨服务回归。实现遵循薄切片递进：先 token/cache，再绑定与摘要，再内网治理，再出站异步链路。

**Tech Stack:** Django, DRF, pytest, Vue 3, Vitest, gitOauth internal API

---

## File Structure（按实现层次）

- `task2app/Saas_project/cloud/domain/...`：UID 值对象、缓存聚合、领域服务、事件与仓储接口（契约层）。
- `task2app/Saas_project/cloud/infrastructure/repositories/...`：领域仓储接口的 Django 实现（基础设施层）。
- `task2app/Saas_project/accounts/github_app_tokens.py`、`task2app/Saas_project/cloud/services/layer_github_oauth_tokens.py`：token/cache 读写与容器 OAuth 入口（应用层）。
- `task2app/Saas_project/projects/services/task_github_oauth_binding.py`、`task2app/Saas_project/projects/views/github_task_credential_views.py`：任务绑定/摘要接口收敛（接口层）。
- `gitOauth/api/github_internal_views.py`、`gitOauth/api/swagger_serializers.py`：上游内部接口协议一致性（跨服务接口层）。
- `task2app/front_project/app/src/views/UserGitSiteOAuthSettings.vue`、`task2app/front_project/app/src/components/task-detail/TaskDetailLinkedProjectsPanel.vue`、`task2app/front_project/app/src/composables/useTaskDetail.js`：前端字符串协议消费（前端层）。

---

### Task 1: 锁定 Domain 契约（DDD 优先）

**Files:**
- Modify: `task2app/Saas_project/cloud/domain/value_objects/github_user_id.py`
- Modify: `task2app/Saas_project/cloud/domain/entities/github_oauth_token_cache_entry.py`
- Modify: `task2app/Saas_project/cloud/domain/events/github_oauth_token_cache_upserted.py`
- Modify: `task2app/Saas_project/cloud/domain/repositories/github_oauth_token_cache_repository.py`
- Modify: `task2app/Saas_project/cloud/domain/services/github_uid_cache_governance_service.py`
- Test: `task2app/Saas_project/tests/domain/cloud/test_github_uid_cache_governance_domain_model.py`

- [ ] **Step 1: 先补失败测试，覆盖 UID 非法值与过期时间治理边界**

```python
@pytest.mark.parametrize("raw_value", ["", "abc", "1001-x", None])
def test_github_user_id_rejects_non_digit_values(raw_value) -> None:
    with pytest.raises(ValueError):
        GithubUserId(raw_value)

def test_ensure_cache_expiry_falls_back_to_default_ttl() -> None:
    now = datetime(2026, 1, 1, 0, 0, 0)
    expires_at = GithubUidCacheGovernanceService.ensure_cache_expiry(
        parsed_expires_at=None, now=now, default_ttl_seconds=300
    )
    assert expires_at == now + timedelta(seconds=300)
```

- [ ] **Step 2: 运行领域测试，确认当前失败点**

Run: `cd task2app/Saas_project && pytest tests/domain/cloud/test_github_uid_cache_governance_domain_model.py -q`
Expected: FAIL，报出契约断言不满足（至少 1 个失败）。

- [ ] **Step 3: 最小实现修复领域对象/服务，使测试通过**

```python
@dataclass(frozen=True)
class GithubUserId:
    value: str
    def __post_init__(self) -> None:
        normalized = str(self.value or "").strip()
        if not normalized or not normalized.isdigit():
            raise ValueError("github_user_id 必须为数字字符串")
        object.__setattr__(self, "value", normalized)
```

- [ ] **Step 4: 回跑领域测试并确认通过**

Run: `cd task2app/Saas_project && pytest tests/domain/cloud/test_github_uid_cache_governance_domain_model.py -q`
Expected: PASS，显示 `100%` 且无失败。

---

### Task 2: 实现基础设施仓储并接入 token/cache 路径

**Files:**
- Create: `task2app/Saas_project/cloud/infrastructure/repositories/django_github_oauth_token_cache_repository.py`
- Modify: `task2app/Saas_project/cloud/infrastructure/repositories/__init__.py`
- Modify: `task2app/Saas_project/accounts/github_app_tokens.py`
- Test: `task2app/Saas_project/tests/infrastructure/cloud/test_django_github_oauth_token_cache_repository.py`
- Test: `task2app/Saas_project/tests/test_layer_github_oauth_tokens.py`

- [ ] **Step 1: 先写仓储实现测试（find_valid/save/clear）**

```python
def test_find_valid_returns_latest_unexpired_entry(db):
    repo = DjangoGithubOauthTokenCacheRepository()
    entry = repo.find_valid(user_id=1, github_user_id=GithubUserId("1001"))
    assert entry is None  # 初始为空
```

- [ ] **Step 2: 运行仓储测试，确认失败**

Run: `cd task2app/Saas_project && pytest tests/infrastructure/cloud/test_django_github_oauth_token_cache_repository.py -q`
Expected: FAIL（仓储类不存在或行为断言失败）。

- [ ] **Step 3: 新增 Django 仓储并在 `accounts/github_app_tokens.py` 里替换直连 ORM 逻辑**

```python
class DjangoGithubOauthTokenCacheRepository(GithubOauthTokenCacheRepository):
    def find_valid(self, *, user_id: int, github_user_id: GithubUserId):
        ...
    def save(self, entry: GithubOauthTokenCacheEntry):
        ...
    def clear(self, *, user_id: int, github_user_id: GithubUserId | None = None) -> None:
        ...
```

- [ ] **Step 4: 运行仓储+token 服务测试验证基础设施接入**

Run: `cd task2app/Saas_project && pytest tests/infrastructure/cloud/test_django_github_oauth_token_cache_repository.py tests/test_layer_github_oauth_tokens.py -q`
Expected: PASS；缓存读取、过期兜底与 upsert 行为一致。

---

### Task 3: 收敛 task2app 接口层字符串协议（绑定 + 摘要）

**Files:**
- Modify: `task2app/Saas_project/projects/services/task_github_oauth_binding.py`
- Modify: `task2app/Saas_project/projects/views/github_task_credential_views.py`
- Modify: `task2app/Saas_project/accounts/github_app_tokens.py`
- Test: `task2app/Saas_project/tests/test_github_task_repo_oauth_binding.py`
- Test: `task2app/Saas_project/tests/test_fetch_gitoauth_credential_summary_for_user.py`

- [ ] **Step 1: 先写失败测试，要求接口仅接受/返回字符串 `github_user_id`**

```python
def test_github_credential_approve_rejects_non_string_uid(client, user):
    resp = client.post(url, {"github_user_id": "abc"}, content_type="application/json")
    assert resp.status_code == 400
```

- [ ] **Step 2: 运行绑定与摘要测试，确认失败**

Run: `cd task2app/Saas_project && pytest tests/test_github_task_repo_oauth_binding.py tests/test_fetch_gitoauth_credential_summary_for_user.py -q`
Expected: FAIL，体现 UID 类型/归一化不一致。

- [ ] **Step 3: 修正服务与视图中的 UID 归一化流程，仅保留字符串协议**

```python
github_user_id = normalize_github_user_id(request.data.get("github_user_id"))
if github_user_id is None:
    return Response({"detail": "github_user_id 必填且必须为字符串"}, status=400)
```

- [ ] **Step 4: 回跑接口测试并确认通过**

Run: `cd task2app/Saas_project && pytest tests/test_github_task_repo_oauth_binding.py tests/test_fetch_gitoauth_credential_summary_for_user.py -q`
Expected: PASS；绑定选择、摘要 connections、主账号 fallback 全部稳定。

---

### Task 4: 对齐 gitOauth 协议与前端消费层

**Files:**
- Modify: `gitOauth/api/github_internal_views.py`
- Modify: `gitOauth/api/swagger_serializers.py`
- Create: `gitOauth/api/tests/test_github_uid_string_contract.py`
- Modify: `task2app/front_project/app/src/views/UserGitSiteOAuthSettings.vue`
- Modify: `task2app/front_project/app/src/components/task-detail/TaskDetailLinkedProjectsPanel.vue`
- Modify: `task2app/front_project/app/src/composables/useTaskDetail.js`

- [ ] **Step 1: 为 gitOauth 新增失败测试，校验 `github_user_id` 序列化为字符串**

```python
def test_summary_for_user_returns_string_github_user_id(client):
    resp = client.post("/api/internal/github/oauth/user-credential/summary-for-user/", {"user_id": 1})
    assert isinstance(resp.json().get("github_user_id"), (str, type(None)))
```

- [ ] **Step 2: 运行 gitOauth 测试，确认失败**

Run: `cd gitOauth && python manage.py test api.tests.test_github_uid_string_contract -v 2`
Expected: FAIL（返回字段类型与断言不符或未覆盖）。

- [ ] **Step 3: 调整 gitOauth 视图/Serializer 与前端 normalize 逻辑，统一字符串 UID**

```javascript
const uid = rawUid == null ? '' : String(rawUid).trim()
return { github_user_id: uid || null }
```

- [ ] **Step 4: 回跑 gitOauth 与前端测试**

Run: `cd gitOauth && python manage.py test api.tests.test_github_uid_string_contract -v 2`
Expected: PASS。

Run: `cd task2app/front_project/app && npm run test -- --runInBand=false`
Expected: PASS，GitHub 连接列表与仓库绑定下拉对字符串 UID 渲染正确。

---

### Task 5: 覆盖内网超时治理与出站异步链路

**Files:**
- Modify: `task2app/Saas_project/cloud/services/layer_github_oauth_tokens.py`
- Modify: `task2app/Saas_project/cloud/services/github_pull_request_after_push.py`
- Test: `task2app/Saas_project/tests/cloud/services/test_layer_github_oauth_tokens_observability.py`
- Test: `task2app/Saas_project/tests/test_github_pr_after_layer_push_async.py`

- [ ] **Step 1: 先补失败测试，覆盖 timeout/network 分支与异步 PR UID 透传**

```python
def test_build_oauth_error_contract_for_timeout():
    contract = build_oauth_error_contract(detail="timed out", failed_stage="gitoauth_summary")
    assert contract["error_code"] == "UPSTREAM_GITOAUTH_TIMEOUT"
```

- [ ] **Step 2: 运行观测与异步链路测试，确认失败**

Run: `cd task2app/Saas_project && pytest tests/cloud/services/test_layer_github_oauth_tokens_observability.py tests/test_github_pr_after_layer_push_async.py -q`
Expected: FAIL，提示阶段分类或 UID 透传不一致。

- [ ] **Step 3: 修正失败阶段推断与异步 PR token 选择逻辑，保证字符串 UID 贯穿**

```python
selected_github_user_id = selected_github_user_id_for_repo(...)
access, err = fetch_github_access_via_gitoauth_for_user(
    int(uid), github_user_id=selected_github_user_id
)
```

- [ ] **Step 4: 回跑测试确认通过**

Run: `cd task2app/Saas_project && pytest tests/cloud/services/test_layer_github_oauth_tokens_observability.py tests/test_github_pr_after_layer_push_async.py -q`
Expected: PASS；错误契约分类与出站链路 UID 协议稳定。

---

### Task 6: 全链路回归与发布前门禁

**Files:**
- Verify only (no code): 已修改文件集合
- Verify migrations: `task2app/Saas_project/accounts/migrations/0029_github_uid_str_and_non_null_expires_at.py`, `task2app/Saas_project/projects/migrations/0038_taskgithubrepooauthbinding_uid_to_char.py`

- [ ] **Step 1: 运行 value stream 对应最小回归集**

Run: `cd task2app/Saas_project && pytest tests/test_layer_github_oauth_tokens.py tests/test_github_task_repo_oauth_binding.py tests/test_fetch_gitoauth_credential_summary_for_user.py tests/cloud/services/test_layer_github_oauth_tokens_observability.py tests/test_github_pr_after_layer_push_async.py -q`
Expected: PASS，覆盖 4 个增量主路径。

- [ ] **Step 2: 校验 Django 迁移状态无漂移**

Run: `cd task2app/Saas_project && python manage.py makemigrations --check --dry-run`
Expected: `No changes detected`。

- [ ] **Step 3: 校验 gitOauth 与前端基础构建/测试**

Run: `cd gitOauth && python manage.py check`
Expected: `System check identified no issues`。

Run: `cd task2app/front_project/app && npm run build`
Expected: build 成功，无类型/语法错误。

- [ ] **Step 4: 发布前人工验收清单（手测）**

Run: `task2app 任务详情 -> 绑定仓库账号 -> 发起 layer push -> 观察 PR 创建结果`
Expected: 前后端展示的 `github_user_id` 全为字符串；未出现 `access_token_expires_at=null` 缓存记录。

---

## Self-Review（计划自检）

- [x] **Spec coverage:** 已覆盖 value stream 4 个增量：薄切片 token/cache、绑定摘要一致性、内网超时治理、出站异步透传。
- [x] **Placeholder scan:** 计划中无 `TODO/TBD/implement later/类似 Task N` 占位表达；每个任务含明确文件、命令、预期结果。
- [x] **Type consistency:** 统一使用 `github_user_id: str`（数字字符串）；跨任务命名保持 `normalize_github_user_id`、`GithubUserId`、`access_token_expires_at` 一致。

## Execution Notes

- 本计划仅定义实施步骤，不改动现有 plan 文件，不在本阶段提交代码。
- 执行阶段进入 `/6-build-构建`，按任务顺序逐项打勾并保留命令输出证据。
