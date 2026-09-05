# Task Detail Repo Clone Credentials Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** 让 `task-detail` 只在返回完整 `repo_clone_credentials` 时成功，缺失时前置失败并在容器侧输出可操作错误。

**Architecture:** 先冻结 `cloud/domain` 的“凭证覆盖契约”模型（VO/聚合根/事件/仓储接口/领域服务），再在 `cloud/views` 落地 409 错误契约，最后在 `onlineServiceJS` 做结构化错误翻译并补齐回归测试。领域层保持纯业务逻辑，不导入 ORM/HTTP/SDK。

**Tech Stack:** Python 3 (Django/DRF), Node.js (ESM), pytest, node:test

---

## Skill Notice

I'm using the writing-plans skill to create the implementation plan.

## 输入基线

- design: `docs/superpowers/specs/2026-05-24-task-detail-repo-clone-credentials-design.md`
- value stream: `docs/superpowers/plans/2026-05-24-task-detail-repo-clone-credentials-value-stream.md`
- ddd model target: `task2app/Saas_project/cloud/domain/**`
- stream registry: `value-stream.yaml` (`task-detail-repo-clone-credentials-contract`)

## 文件结构（先锁定边界）

- Domain
  - `task2app/Saas_project/cloud/domain/value_objects/repo_clone_credential_coverage.py`
  - `task2app/Saas_project/cloud/domain/entities/task_repo_clone_credentials_contract.py`
  - `task2app/Saas_project/cloud/domain/events/repo_clone_credentials_incomplete_detected.py`
  - `task2app/Saas_project/cloud/domain/repositories/task_repo_clone_credentials_repository.py`
  - `task2app/Saas_project/cloud/domain/services/task_repo_clone_credentials_guard_service.py`
- Interface / App
  - `task2app/Saas_project/cloud/views/container_task_detail_views.py`
  - `trae-agent/onlineServiceJS/src/bootstrap.mjs`
- Tests
  - `task2app/Saas_project/tests/domain/cloud/test_task_repo_clone_credentials_domain_model.py`
  - `task2app/Saas_project/tests/test_container_runtime_tokens.py`
  - `trae-agent/onlineServiceJS/src/bootstrap.cloneCredentials.test.mjs`

## 依赖顺序（DDD 约束）

1. Domain contract first（VO -> Entity/Aggregate -> Repository ABC -> Event -> Domain Service）
2. `task-detail` 接口实现 second
3. `onlineServiceJS` 错误桥接 third
4. 回归验证与合规检查 last

---

### Task 1: 冻结 Domain 契约（Thin Slice 前置）

**Files:**
- Create: `task2app/Saas_project/cloud/domain/value_objects/repo_clone_credential_coverage.py`
- Create: `task2app/Saas_project/cloud/domain/entities/task_repo_clone_credentials_contract.py`
- Create: `task2app/Saas_project/cloud/domain/events/repo_clone_credentials_incomplete_detected.py`
- Create: `task2app/Saas_project/cloud/domain/repositories/task_repo_clone_credentials_repository.py`
- Create: `task2app/Saas_project/cloud/domain/services/task_repo_clone_credentials_guard_service.py`
- Test: `task2app/Saas_project/tests/domain/cloud/test_task_repo_clone_credentials_domain_model.py`

- [ ] **Step 1: 先写失败测试（覆盖缺失仓库、事件触发、ABC 约束）**

```python
def test_repo_clone_credential_coverage_reports_missing_urls():
    coverage = RepoCloneCredentialCoverage(
        expected_repo_urls=["http://localhost:8012/demo/repo-a.git"],
        credential_repo_urls=[],
    )
    assert coverage.missing_repo_urls() == ("http://localhost:8012/demo/repo-a.git",)
```

- [ ] **Step 2: 运行测试确认失败**
  - Run: `cd task2app/Saas_project && pytest tests/domain/cloud/test_task_repo_clone_credentials_domain_model.py -q`
  - Expected: 因类/方法未定义失败

- [ ] **Step 3: 实现最小 Domain 模型**

```python
class TaskRepoCloneCredentialsContract:
    def detect_incomplete_event(self, *, occurred_at: datetime):
        missing = self.coverage.missing_repo_urls()
        if not missing:
            return None
        return RepoCloneCredentialsIncompleteDetected(
            scope=self.scope,
            missing_repo_urls=missing,
            occurred_at=occurred_at,
        )
```

- [ ] **Step 4: 运行测试确认通过**
  - Run: `cd task2app/Saas_project && pytest tests/domain/cloud/test_task_repo_clone_credentials_domain_model.py -q`
  - Expected: PASS

- [ ] **Step 5: Commit（Domain first）**
  - Run: `git add task2app/Saas_project/cloud/domain task2app/Saas_project/tests/domain/cloud/test_task_repo_clone_credentials_domain_model.py && git commit -m "feat(cloud-domain): add repo clone credential coverage contract"`

---

### Task 2: 落地 task-detail 严格契约（Increment 1）

**Files:**
- Modify: `task2app/Saas_project/cloud/views/container_task_detail_views.py`
- Test: `task2app/Saas_project/tests/test_container_runtime_tokens.py`

- [ ] **Step 1: 写失败测试（无凭证/部分凭证应返回 409）**

```python
def test_fetch_container_task_detail_returns_project_urls():
    r = client.post(url, {"access_token": access}, format="json")
    assert r.status_code == 409
    assert r.data["error_code"] == "REPO_CLONE_CREDENTIALS_INCOMPLETE"
```

- [ ] **Step 2: 运行测试确认失败**
  - Run: `cd task2app/Saas_project && pytest tests/test_container_runtime_tokens.py -k "task_detail" -v`
  - Expected: 断言失败（当前仍返回 200 或无缺失列表）

- [ ] **Step 3: 最小实现覆盖率校验与错误载荷**

```python
missing_repo_credentials = sorted(expected_repo_urls - set(repo_clone_credentials.keys()))
if missing_repo_credentials:
    return Response(
        {
            "detail": "任务仓库克隆凭证不完整，请先在任务详情完成仓库授权绑定",
            "error_code": "REPO_CLONE_CREDENTIALS_INCOMPLETE",
            "trace_id": _resolve_trace_id(request),
            "missing_repo_credentials": missing_repo_credentials,
        },
        status=status.HTTP_409_CONFLICT,
    )
```

- [ ] **Step 4: 运行测试确认通过**
  - Run: `cd task2app/Saas_project && pytest tests/test_container_runtime_tokens.py -k "task_detail" -v`
  - Expected: PASS

- [ ] **Step 5: Commit**
  - Run: `git add task2app/Saas_project/cloud/views/container_task_detail_views.py task2app/Saas_project/tests/test_container_runtime_tokens.py && git commit -m "feat(cloud): enforce complete task-detail clone credentials contract"`

---

### Task 3: onlineServiceJS 错误桥接（Increment 2）

**Files:**
- Modify: `trae-agent/onlineServiceJS/src/bootstrap.mjs`
- Test: `trae-agent/onlineServiceJS/src/bootstrap.cloneCredentials.test.mjs`

- [ ] **Step 1: 写失败测试（识别结构化 error_code 并输出可操作文案）**

```javascript
test('buildTaskDetailBootstrapError renders actionable message for incomplete credentials', () => {
  const err = new Error('HTTP 409 ... {"error_code":"REPO_CLONE_CREDENTIALS_INCOMPLETE","missing_repo_credentials":["http://localhost:8012/demo/repo-a.git"]}');
  const wrapped = buildTaskDetailBootstrapError(err);
  assert.match(wrapped.message, /未返回完整 repo_clone_credentials/);
});
```

- [ ] **Step 2: 运行测试确认失败**
  - Run: `cd trae-agent/onlineServiceJS && node --test src/bootstrap.cloneCredentials.test.mjs`
  - Expected: 新增断言失败

- [ ] **Step 3: 最小实现结构化错误转换**

```javascript
try {
  detail = await postJson(`${prefix}/server-container-token/task-detail/`, { access_token: newAccess }, timeoutSec);
} catch (e) {
  throw buildTaskDetailBootstrapError(e);
}
```

- [ ] **Step 4: 运行测试确认通过**
  - Run: `cd trae-agent/onlineServiceJS && node --test src/bootstrap.cloneCredentials.test.mjs`
  - Expected: PASS

- [ ] **Step 5: Commit**
  - Run: `git add trae-agent/onlineServiceJS/src/bootstrap.mjs trae-agent/onlineServiceJS/src/bootstrap.cloneCredentials.test.mjs && git commit -m "feat(onlineServiceJS): translate incomplete task-detail credentials into actionable error"`

---

### Task 4: 增量回归与价值流验证（Increment 3）

**Files:**
- Test: `task2app/Saas_project/tests/domain/cloud/test_task_repo_clone_credentials_domain_model.py`
- Test: `task2app/Saas_project/tests/test_container_runtime_tokens.py`
- Test: `trae-agent/onlineServiceJS/src/bootstrap.cloneCredentials.test.mjs`
- Modify: `value-stream.yaml`（如步骤/字段需要对齐）

- [ ] **Step 1: 运行 Domain 验证**
  - Run: `cd task2app/Saas_project && pytest tests/domain/cloud/test_task_repo_clone_credentials_domain_model.py -q`
  - Expected: PASS

- [ ] **Step 2: 运行 task-detail 契约验证**
  - Run: `cd task2app/Saas_project && pytest tests/test_container_runtime_tokens.py -k "task_detail" -v`
  - Expected: PASS

- [ ] **Step 3: 运行 onlineServiceJS 桥接验证**
  - Run: `cd trae-agent/onlineServiceJS && node --test src/bootstrap.cloneCredentials.test.mjs`
  - Expected: PASS

- [ ] **Step 4: 运行 DDD 合规检查**
  - Run: `python3 task2app/scripts/ci/check_ddd_bdd_compliance.py`
  - Expected: `DDD/BDD 合规检查通过。`

- [ ] **Step 5: Commit（verification batch）**
  - Run: `git add value-stream.yaml docs/superpowers/specs/2026-05-24-task-detail-repo-clone-credentials-design.md docs/superpowers/plans/2026-05-24-task-detail-repo-clone-credentials-value-stream.md docs/superpowers/plans/2026-05-24-task-detail-repo-clone-credentials-implementation-plan.md && git commit -m "docs: add implementation plan and value-stream alignment for task-detail credential contract"`

---

## 自检清单（计划质量）

- [ ] 每个任务都可独立验证（有命令 + 预期输出）
- [ ] 任务顺序满足 DDD 依赖（Domain 先于 Interface）
- [ ] 无 “TODO/TBD/后续补充” 占位符
- [ ] 错误码 `REPO_CLONE_CREDENTIALS_INCOMPLETE` 在后端与容器侧语义一致
- [ ] `value-stream.yaml` 与实现用例一致（步骤与测试文件可追溯）

## 与 value increments 对齐

- Increment 1（thin slice）→ Task 1 + Task 2
- Increment 2（core value）→ Task 3
- Increment 3（regression safety）→ Task 4
