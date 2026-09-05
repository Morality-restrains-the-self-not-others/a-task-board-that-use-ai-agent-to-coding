# OAuth Structured Error Contract Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** 将 OAuth 拉取失败从“裸 `detail` 文本”升级为结构化错误契约，并在 `task2app` API 层实际返回 `error_code` / `failed_stage` / `retryable` / `detail_safe`，同时保持旧字段兼容。

**Architecture:** 先固化领域层错误语义（分类结果作为单一真相源），再接入应用服务 helper，最后在视图层透传并补齐 API 级测试。保持 Domain 纯净，Infrastructure 仅做接口最小实现（no-op）。

**Tech Stack:** Python 3, Django/DRF, pytest, DDD layered architecture

---

## Skill Notice

I'm using the writing-plans skill to create the implementation plan.

## 输入基线

- 设计（已确认）：`docs/superpowers/specs/2026-05-22-oauth-token-fetch-timeout-design.md`
- 当前审查结论：结构化错误函数已存在但未接入 API 响应
- 目标链路：
  - `cloud/domain/services/oauth_token_failure_classifier.py`
  - `cloud/services/layer_github_oauth_tokens.py`
  - `cloud/views/container_layer_github_oauth_views.py`

## 依赖顺序（必须遵守）

1. **Domain 语义先行**：分类结果与文案映射保持一致
2. **Service 接口收敛**：由 helper 统一生产错误契约
3. **View 对外返回**：结构化字段透传到 API 响应
4. **回归与合规**：测试与 DDD/BDD 合规检查

## DDD 约束 Hard Gates

- [ ] `cloud/domain/**` 不导入 `django.*`、`cloud.models`、`requests`
- [ ] 错误码与文案来源统一（不得双轨推断）
- [ ] Repository 抽象不回退（no-op 可保留）
- [ ] View 层只做编排和协议转换，不复制业务分类规则

## DDD 结构校验（/4-plans 要求）

- [x] **分层结构符合要求**：Domain -> Services -> Views 顺序明确
- [x] **任务顺序符合要求**：先修语义一致性，再接 API 返回
- [x] **接口先于实现**：沿用既有 Domain 契约，先通过 Service helper 对齐
- [x] **可验证**：每个任务都有命令与期望结果

---

### Task 1: 统一错误语义来源（Domain + Service）

**Files:**
- Modify: `task2app/Saas_project/cloud/domain/services/oauth_token_failure_classifier.py`
- Modify: `task2app/Saas_project/cloud/services/layer_github_oauth_tokens.py`
- Test: `task2app/Saas_project/tests/cloud/domain/test_oauth_token_failure_classifier.py`
- Test: `task2app/Saas_project/tests/test_layer_github_oauth_tokens.py`

- [ ] **Step 1: 写失败测试（语义一致性）**
  - 断言：`aborted` -> `UPSTREAM_GITOAUTH_TIMEOUT` 且 `detail_safe` 为“超时可重试”文案
  - 断言：`BINDING_MISSING` 不可重试
- [ ] **Step 2: 运行测试确认失败（Red）**
  - Run: `cd task2app/Saas_project && ../activate_env.sh unit -- pytest tests/cloud/domain/test_oauth_token_failure_classifier.py tests/test_layer_github_oauth_tokens.py -k "timeout or binding or error_contract" -v`
  - Expected: 至少一个断言失败（错误码/文案来源不一致）
- [ ] **Step 3: 最小实现（Green）**
  - 在 `build_oauth_error_contract` 中仅使用分类结果映射 `detail_safe`
  - 避免再次调用独立 `_oauth_error_kind` 造成双轨语义
- [ ] **Step 4: 复跑测试确认通过**
  - Run: `cd task2app/Saas_project && ../activate_env.sh unit -- pytest tests/cloud/domain/test_oauth_token_failure_classifier.py tests/test_layer_github_oauth_tokens.py -k "timeout or binding or error_contract" -v`
  - Expected: PASS

---

### Task 2: 视图层接入结构化错误契约（API Thin Slice）

**Files:**
- Modify: `task2app/Saas_project/cloud/views/container_layer_github_oauth_views.py`
- Test: `task2app/Saas_project/tests/cloud/view_test/test_container_layer_github_oauth_views.py` (new)

- [ ] **Step 1: 新增 API 失败契约测试（Red）**
  - 场景：`resolve_github_auth_by_repo_for_container_task` 返回 `({}, err)`
  - 断言响应 JSON 包含：
    - `ok: false`
    - `detail`
    - `detail_safe`
    - `error_code`
    - `failed_stage`
    - `retryable`
    - `github_auth_by_repo: {}`
- [ ] **Step 2: 运行测试确认失败**
  - Run: `cd task2app/Saas_project && ../activate_env.sh unit -- pytest tests/cloud/view_test/test_container_layer_github_oauth_views.py -v`
  - Expected: 断言失败（当前视图未返回结构化字段）
- [ ] **Step 3: 最小实现视图透传**
  - 仅在 `err and not tokens` 分支返回结构化字段
  - 保持 `detail` 兼容旧调用方
  - HTTP 状态维持现有 `409`（本轮不调整语义状态码）
- [ ] **Step 4: 复跑测试确认通过**
  - Run: `cd task2app/Saas_project && ../activate_env.sh unit -- pytest tests/cloud/view_test/test_container_layer_github_oauth_views.py -v`
  - Expected: PASS

---

### Task 3: 回归与合规验证

**Files:**
- Test: `task2app/Saas_project/tests/test_layer_github_oauth_tokens.py`
- Test: `task2app/Saas_project/tests/cloud/domain/test_oauth_token_fetch_failed_domain_event.py`
- Test: `task2app/Saas_project/tests/cloud/domain/test_oauth_token_failure_classifier.py`
- Test: `task2app/Saas_project/tests/cloud/view_test/test_container_layer_github_oauth_views.py`

- [ ] **Step 1: 运行目标回归组**
  - Run: `cd task2app/Saas_project && ../activate_env.sh unit -- pytest tests/test_layer_github_oauth_tokens.py tests/cloud/domain/test_oauth_token_fetch_failed_domain_event.py tests/cloud/domain/test_oauth_token_failure_classifier.py tests/cloud/view_test/test_container_layer_github_oauth_views.py -v`
  - Expected: PASS
- [ ] **Step 2: 运行 DDD/BDD 合规检查**
  - Run: `cd /Users/task2app/gitClone/ramDisk/ram-mount && python scripts/ci/check_ddd_bdd_compliance.py`
  - Expected: `DDD/BDD 合规检查通过`
- [ ] **Step 3: 手工接口抽样验证（可选）**
  - Run: `curl -i -X POST "http://127.0.0.1:8765/api/layers/<layer_id>/git/oauth-fetch-token-files?access_token=<token>" -H "X-Access-Token: <token>" -H "Content-Type: application/json" -d '{}'`
  - Expected: 失败时响应体含结构化错误字段，而非仅 `detail`

---

## 执行前后校验清单

- [ ] API 失败响应字段完整且兼容旧字段 `detail`
- [ ] `aborted` 场景错误码为 `UPSTREAM_GITOAUTH_TIMEOUT`
- [ ] `detail_safe` 与 `error_code` 语义一致
- [ ] Domain 层未引入基础设施依赖
- [ ] 所有新增测试通过

## 风险与回滚

- **风险：** 旧前端/调用方若依赖失败响应的最小字段，可能受新增字段影响（低风险，因保留 `detail`）。
- **回滚策略：**
  - 回滚 `container_layer_github_oauth_views.py` 的结构化响应改动；
  - 保留 domain/service 契约改动不影响旧行为。
