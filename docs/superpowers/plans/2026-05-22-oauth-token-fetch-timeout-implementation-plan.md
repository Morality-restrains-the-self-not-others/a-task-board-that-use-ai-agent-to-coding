# OAuth Token Fetch Timeout Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** 基于已批准 value stream，落地 OAuth Token 拉取链路的超时治理，确保失败可解释、卡点可定位、增量可验证。

**Architecture:** 先冻结 Domain 契约（事件、错误分类、仓储接口），再在 Application/Interface 层打通结构化错误和分段观测，最后补充策略增强（重试/部分成功）与回归验证。Domain 层不直接依赖 Django ORM/HTTP SDK。

**Tech Stack:** Python 3 (Django/DRF), Node.js (Express), pytest, node:test

---

## Skill Notice

I'm using the writing-plans skill to create the implementation plan.

## 输入基线

- design: `docs/superpowers/specs/2026-05-22-oauth-token-fetch-timeout-design.md`
- value stream: `docs/superpowers/plans/2026-05-22-oauth-token-fetch-timeout-value-stream.md`
- stream registry: `value-stream.yaml` (`oauth-token-fetch-timeout-governance`)

## 依赖顺序（必须遵守）

1. **Domain 阶段（先做）**：错误语义、领域事件、仓储接口  
2. **Application/Interface 阶段（后做）**：task2app 视图/服务与 onlineServiceJS 错误契约  
3. **Enhancement 阶段（最后）**：部分成功、重试与补充回归

## DDD 约束 Hard Gates

- [ ] `cloud/domain/**` 不导入 `django.*`、`cloud.models`、`requests`
- [ ] Repository interface 先定义，再写实现/调用
- [ ] Application 层只编排，不承载核心业务不变量
- [ ] 错误码与失败阶段在 Domain/Application 中统一定义，避免多处硬编码漂移

## DDD 结构校验（/4-plans 要求）

- [x] **分层结构符合要求**：任务按 `domain -> services/views -> integration` 排序。
- [x] **任务顺序符合要求**：先契约，后实现，再增强。
- [x] **接口先于实现**：仓储/事件接口先在 Domain 冻结。
- [x] **增量可验证**：每个任务都有可执行命令与预期结果。

---

### Task 1: Domain Contract Freeze（错误语义 + 领域事件）

**Files:**
- Create: `task2app/Saas_project/cloud/domain/events/oauth_token_fetch_failed.py`
- Modify: `task2app/Saas_project/cloud/domain/events/__init__.py`
- Modify: `task2app/Saas_project/cloud/domain/repositories/container_runtime_context_repository.py`
- Test: `task2app/Saas_project/tests/cloud/domain/test_oauth_token_fetch_failed_domain_event.py`

- [ ] **Step 1: 先写失败测试（事件字段与不可变性）**
- [ ] **Step 2: 运行测试确认失败**
  - Run: `cd task2app/Saas_project && ../activate_env.sh unit -- pytest tests/cloud/domain/test_oauth_token_fetch_failed_domain_event.py -q`
  - Expected: 因事件未定义或字段不匹配失败
- [ ] **Step 3: 最小实现 Domain 契约**
  - `OauthTokenFetchFailed`（建议 `frozen dataclass`）
  - 字段包含：`task_id`、`failed_stage`、`error_code`、`retryable`、`elapsed_ms`、`occurred_at`
  - 在 `ContainerRuntimeContextRepository` 增加记录/上报该事件的抽象方法
- [ ] **Step 4: 运行测试确认通过**
  - Run: `cd task2app/Saas_project && ../activate_env.sh unit -- pytest tests/cloud/domain/test_oauth_token_fetch_failed_domain_event.py -q`
  - Expected: PASS
- [ ] **Step 5: Commit（Domain first）**
  - Run: `git add task2app/Saas_project/cloud/domain/events task2app/Saas_project/cloud/domain/repositories task2app/Saas_project/tests/cloud/domain && git commit -m "feat(cloud-domain): add oauth token fetch failure contract"`

---

### Task 2: task2app Structured Error Contract（Thin Slice）

**Files:**
- Modify: `task2app/Saas_project/cloud/views/container_layer_github_oauth_views.py`
- Modify: `task2app/Saas_project/cloud/services/layer_github_oauth_tokens.py`
- Test: `task2app/Saas_project/tests/test_layer_github_oauth_tokens.py`

- [ ] **Step 1: 写失败测试（错误载荷结构）**
  - 断言失败响应包含：`error_code`、`failed_stage`、`retryable`、`detail_safe`
- [ ] **Step 2: 运行测试确认失败**
  - Run: `cd task2app/Saas_project && ../activate_env.sh unit -- pytest tests/test_layer_github_oauth_tokens.py -k "error_contract or timeout" -v`
  - Expected: 断言失败（当前无统一结构）
- [ ] **Step 3: 最小实现结构化错误映射**
  - 统一 `timeout/network/other` 到 `error_code`
  - 统一设置 `failed_stage`（如 `binding_check` / `gitoauth_summary` / `gitoauth_access`）
  - 保证输出脱敏文案 `detail_safe`
- [ ] **Step 4: 运行测试确认通过**
  - Run: `cd task2app/Saas_project && ../activate_env.sh unit -- pytest tests/test_layer_github_oauth_tokens.py -k "error_contract or timeout" -v`
  - Expected: PASS
- [ ] **Step 5: Commit**
  - Run: `git add task2app/Saas_project/cloud/views/container_layer_github_oauth_views.py task2app/Saas_project/cloud/services/layer_github_oauth_tokens.py task2app/Saas_project/tests/test_layer_github_oauth_tokens.py && git commit -m "feat(cloud): add structured oauth token fetch error contract"`

---

### Task 3: Stage-level Observability（Core Value）

**Files:**
- Modify: `task2app/Saas_project/cloud/services/layer_github_oauth_tokens.py`
- Test: `task2app/Saas_project/tests/cloud/services/test_layer_github_oauth_tokens_observability.py`

- [ ] **Step 1: 写失败测试（分段日志与耗时）**
  - 断言包含 `entry/token_check/binding_check/gitoauth_summary/gitoauth_access/exit`
- [ ] **Step 2: 运行测试确认失败**
  - Run: `cd task2app/Saas_project && ../activate_env.sh unit -- pytest tests/cloud/services/test_layer_github_oauth_tokens_observability.py -q`
  - Expected: 分段字段或耗时断言失败
- [ ] **Step 3: 最小实现分段观测**
  - 每阶段记录 `trace_id`、`repo_count`、`elapsed_ms`、`status`
  - 将失败阶段写入统一错误载荷
- [ ] **Step 4: 运行测试确认通过**
  - Run: `cd task2app/Saas_project && ../activate_env.sh unit -- pytest tests/cloud/services/test_layer_github_oauth_tokens_observability.py -q`
  - Expected: PASS
- [ ] **Step 5: Commit**
  - Run: `git add task2app/Saas_project/cloud/services/layer_github_oauth_tokens.py task2app/Saas_project/tests/cloud/services/test_layer_github_oauth_tokens_observability.py && git commit -m "feat(cloud): add stage-level oauth fetch observability"`

---

### Task 4: onlineServiceJS Contract Bridging（Thin Slice 完成）

**Files:**
- Modify: `trae-agent/onlineServiceJS/src/layerGitOauthFetchTokenFiles.mjs`
- Modify: `trae-agent/onlineServiceJS/src/saasTaskCloud.mjs`
- Test: `trae-agent/onlineServiceJS/src/layerGitOauthFetchTokenFiles.test.mjs`

- [ ] **Step 1: 写失败测试（abort 映射结构化错误）**
  - 断言 `This operation was aborted` 被映射为 `UPSTREAM_TIMEOUT`（或约定码）
- [ ] **Step 2: 运行测试确认失败**
  - Run: `cd trae-agent/onlineServiceJS && node --test src/layerGitOauthFetchTokenFiles.test.mjs`
  - Expected: 新增断言失败
- [ ] **Step 3: 最小实现桥接逻辑**
  - 保留 120s 保护
  - 将上游结构化错误透传到前端响应
  - 当本地 abort 时补齐 `error_code/failed_stage/retryable`
- [ ] **Step 4: 运行测试确认通过**
  - Run: `cd trae-agent/onlineServiceJS && node --test src/layerGitOauthFetchTokenFiles.test.mjs`
  - Expected: PASS
- [ ] **Step 5: Commit**
  - Run: `git add trae-agent/onlineServiceJS/src/layerGitOauthFetchTokenFiles.mjs trae-agent/onlineServiceJS/src/saasTaskCloud.mjs trae-agent/onlineServiceJS/src/layerGitOauthFetchTokenFiles.test.mjs && git commit -m "feat(onlineServiceJS): map oauth fetch abort into structured errors"`

---

### Task 5: Safe Timeout/Retry + Partial Success（Enhancement）

**Files:**
- Modify: `task2app/Saas_project/accounts/github_app_tokens.py`
- Modify: `task2app/Saas_project/cloud/services/layer_github_oauth_tokens.py`
- Test: `task2app/Saas_project/tests/test_layer_git_push_auth_context.py`
- Test: `task2app/Saas_project/tests/test_layer_github_oauth_tokens.py`

- [ ] **Step 1: 写失败测试（部分成功与重试边界）**
  - 单仓失败不影响已成功仓库返回
  - 非幂等路径不做重试
- [ ] **Step 2: 运行测试确认失败**
  - Run: `cd task2app/Saas_project && ../activate_env.sh unit -- pytest tests/test_layer_git_push_auth_context.py tests/test_layer_github_oauth_tokens.py -k "partial or retry" -v`
  - Expected: 断言失败
- [ ] **Step 3: 最小实现增强**
  - `summary/access` 设置明确 connect/read timeout
  - 仅对可重试错误执行有限重试（如 1 次）
  - 返回 `token_files + partial_error`
- [ ] **Step 4: 运行测试确认通过**
  - Run: `cd task2app/Saas_project && ../activate_env.sh unit -- pytest tests/test_layer_git_push_auth_context.py tests/test_layer_github_oauth_tokens.py -k "partial or retry" -v`
  - Expected: PASS
- [ ] **Step 5: Commit**
  - Run: `git add task2app/Saas_project/accounts/github_app_tokens.py task2app/Saas_project/cloud/services/layer_github_oauth_tokens.py task2app/Saas_project/tests/test_layer_git_push_auth_context.py task2app/Saas_project/tests/test_layer_github_oauth_tokens.py && git commit -m "feat(cloud): add safe retry and partial success for oauth token fetch"`

---

### Task 6: Value Stream Verification（oauth-token-fetch-timeout-governance）

**Files:**
- Test: `task2app/Saas_project/tests/test_layer_github_oauth_tokens.py`
- Test: `task2app/Saas_project/tests/cloud/services/test_layer_github_oauth_tokens_observability.py`
- Test: `task2app/Saas_project/tests/cloud/domain/test_container_github_oauth_token_fetch_observed.py`
- Test: `task2app/Saas_project/tests/test_layer_git_push_auth_context.py`

- [ ] **Step 1: 验证 step1 `oauth-error-contract-thin-slice`**
  - Run: `cd task2app/Saas_project && ../activate_env.sh unit -- pytest tests/test_layer_github_oauth_tokens.py -k "error_contract" -v`
  - Expected: PASS
- [ ] **Step 2: 验证 step2 `oauth-stage-observability`**
  - Run: `cd task2app/Saas_project && ../activate_env.sh unit -- pytest tests/cloud/services/test_layer_github_oauth_tokens_observability.py -v`
  - Expected: PASS
- [ ] **Step 3: 验证 step3 `oauth-binding-domain-safety`**
  - Run: `cd task2app/Saas_project && ../activate_env.sh unit -- pytest tests/cloud/domain/test_container_github_oauth_token_fetch_observed.py -v`
  - Expected: PASS
- [ ] **Step 4: 验证 step4 `oauth-multi-repo-enhancement`**
  - Run: `cd task2app/Saas_project && ../activate_env.sh unit -- pytest tests/test_layer_git_push_auth_context.py -v`
  - Expected: PASS
- [ ] **Step 5: 组合回归**
  - Run: `cd task2app/Saas_project && ../activate_env.sh unit -- pytest tests/test_layer_github_oauth_tokens.py tests/cloud/services/test_layer_github_oauth_tokens_observability.py tests/test_layer_git_push_auth_context.py -v`
  - Expected: PASS

---

## 执行前后校验清单

- [ ] 先完成 Task 1 再进入 Task 2（Domain first）
- [ ] 执行 `rg "from cloud\\.models|import django|import requests" task2app/Saas_project/cloud/domain` 结果为空
- [ ] `value-stream.yaml` 的 `oauth-token-fetch-timeout-governance` 至少前 4 个 active 环节有对应测试通过
- [ ] `onlineServiceJS` 返回错误不再仅有裸文本 `This operation was aborted`

## 与 value stream 对齐说明

- Increment 1（thin slice）：Task 2 + Task 4
- Increment 2（core value）：Task 3
- Increment 3（essential support）：Task 5（timeout/retry）
- Increment 4（enhancement）：Task 5（partial success）
- Increment 5（future async）：本计划仅保留接口占位，不在本轮实施
