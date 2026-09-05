# gitOauth GitLab Provider Config Compatibility Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** 修复 GitLab OAuth 在项目详情页触发后回落 `gitlab=bad_state` 的问题，使 `repo_url=http://localhost:8012/...` 稳定命中 `gitlab:local-gitlab` 并跳转到 GitLab `oauth/authorize`。

**Architecture:** 以领域层路由契约为核心：`OauthProviderRouteRule` + `OauthProviderRoutingCatalog` + `OauthAuthorizeRouteDomainService` 负责纯决策；基础设施层负责从 `task2app/conf/port_config.json` 读取并归一化 provider 配置（兼容 list/dict）；接口层仅做 token 解码、调用领域服务、错误映射与日志。

**Tech Stack:** Python 3.9, Django, PyJWT, Playwright, domain-first layering

---

## File Structure

- Domain (已建模，作为契约)
  - `gitOauth/api/domain/value_objects/oauth_provider_key.py`
  - `gitOauth/api/domain/value_objects/repo_origin.py`
  - `gitOauth/api/domain/value_objects/oauth_authorize_target.py`
  - `gitOauth/api/domain/entities/oauth_provider_route_rule.py`
  - `gitOauth/api/domain/entities/oauth_provider_routing_catalog.py`
  - `gitOauth/api/domain/repositories/oauth_provider_route_rule_repository.py`
  - `gitOauth/api/domain/services/oauth_authorize_route_domain_service.py`
  - `gitOauth/api/domain/events/oauth_authorize_route_resolved.py`
  - `gitOauth/api/domain/events/oauth_authorize_route_rejected.py`
- Infrastructure / Application (本计划实施对象)
  - Modify: `gitOauth/config/provider_registry.py`
  - Modify: `gitOauth/api/provider_configs.py`
  - Create: `gitOauth/api/infrastructure/repositories/port_config_oauth_provider_route_rule_repository.py`
  - Modify: `gitOauth/api/gitlab_browser_views.py`
  - Modify: `gitOauth/api/tests.py`（或拆分新测试文件）
  - Modify: `gitOauth/api/health_checks.py`（可观测增强）

---

### Task 1: 领域契约回归守卫（先锁定模型不退化）

**Files:**
- Test: `gitOauth/api/tests.py`

- [ ] **Step 1: 为新领域模型补充纯领域测试（无 Django/ORM 依赖）**
  - `OauthProviderKey` 规范化与非空校验
  - `RepoOrigin.from_repo_url` 解析与非法 URL 拒绝
  - `OauthProviderRoutingCatalog.resolve` 的确定性（同输入同输出）
  - `OauthAuthorizeRouteDomainService.resolve` 的 `resolved/rejected` 二值结果

- [ ] **Step 2: 运行领域测试**

Run:
`cd gitOauth && python3 -m pytest api/tests.py -k "oauth_provider or authorize_route" -q`

Expected: PASS

---

### Task 2: 基础设施层实现 provider 配置双结构归一化

**Files:**
- Modify: `gitOauth/config/provider_registry.py`
- Modify: `gitOauth/api/provider_configs.py`
- Create: `gitOauth/api/infrastructure/repositories/port_config_oauth_provider_route_rule_repository.py`
- Test: `gitOauth/api/tests.py`

- [ ] **Step 1: 写失败测试（dict 结构 `gitOauth` 配置应能产出 gitlab 规则）**
  - 给出最小 dict 配置样例（`http://localhost:8012` -> `service_provider=local-gitlab`）
  - 断言 `list_rules(provider="gitlab")` 非空
  - 断言规则包含 `provider_key=gitlab:local-gitlab`、`authorize_origin=http://localhost:8012`

- [ ] **Step 2: 运行测试确认失败**

Run:
`cd gitOauth && python3 -m pytest api/tests.py -k "provider_registry_dict_shape" -q`

Expected: FAIL（实现前）

- [ ] **Step 3: 最小实现（不改业务语义）**
  - 在 `provider_registry.py` 扩展归一化：支持 list 与 dict
  - 在新仓储实现中把归一化结果映射为领域实体 `OauthProviderRouteRule`
  - 保持 `provider/service_provider/allowedHost/client_id/redirect_uri/scope` 统一语义

- [ ] **Step 4: 运行测试确认通过**

Run:
`cd gitOauth && python3 -m pytest api/tests.py -k "provider_registry_dict_shape or provider_registry_list_shape" -q`

Expected: PASS

---

### Task 3: 接口层接线到领域服务并保持错误分类稳定

**Files:**
- Modify: `gitOauth/api/gitlab_browser_views.py`
- Modify: `gitOauth/api/provider_configs.py`
- Test: `gitOauth/api/tests.py`

- [ ] **Step 1: 写失败测试（local-gitlab token 必须 302 到 localhost:8012/oauth/authorize）**
  - 构造合法 start token（`service_provider=local-gitlab`）
  - 调用 `/api/accounts/gitlab/oauth/start/`
  - 断言 `Location` 为 `http://localhost:8012/oauth/authorize?...`
  - 断言 `client_id` 与 `redirect_uri` 来自 local-gitlab 配置

- [ ] **Step 2: 写失败测试（配置缺失仍返回 bad_state，不抛 500）**
  - 清空/模拟缺失规则
  - 断言回跳 `gitlab=bad_state`
  - 断言响应非 500

- [ ] **Step 3: 运行测试确认失败**

Run:
`cd gitOauth && python3 -m pytest api/tests.py -k "gitlab_oauth_start_local_route or gitlab_oauth_start_missing_config" -q`

Expected: FAIL（实现前）

- [ ] **Step 4: 最小实现**
  - `gitlab_browser_views.py` 使用 `OauthAuthorizeRouteDomainService`
  - 将 `resolved` 映射为 authorize URL 参数
  - 将 `rejected.reason` 映射到现有 `bad_state` 分支（保持前端兼容）

- [ ] **Step 5: 运行测试确认通过**

Run:
`cd gitOauth && python3 -m pytest api/tests.py -k "gitlab_oauth_start_local_route or gitlab_oauth_start_missing_config" -q`

Expected: PASS

---

### Task 4: 可观测与健康信号增强（NFR L2）

**Files:**
- Modify: `gitOauth/api/gitlab_browser_views.py`
- Modify: `gitOauth/api/health_checks.py`
- Test: `gitOauth/api/tests.py`

- [ ] **Step 1: 写失败测试（失败日志字段完整）**
  - 模拟 start 失败
  - 断言日志包含 `provider/service_provider/reason/trace_id`（可通过 logger mock）

- [ ] **Step 2: 写失败测试（健康接口包含 GitLab 配置计数）**
  - 调用 `/api/health/`
  - 断言响应包含 `gitlab_provider_config_count`

- [ ] **Step 3: 运行测试确认失败**

Run:
`cd gitOauth && python3 -m pytest api/tests.py -k "gitlab_start_observability or health_provider_count" -q`

Expected: FAIL（实现前）

- [ ] **Step 4: 最小实现并验证**

Run:
`cd gitOauth && python3 -m pytest api/tests.py -k "gitlab_start_observability or health_provider_count" -q`

Expected: PASS

---

### Task 5: 端到端回归（Playwright + 合规检查）

**Files:**
- Modify: `task2app/playwright/front_project/tests/ProjectDetail.branch-preview-local-git.playwright.test.js`（必要时新增断言）
- Optional Create: `task2app/playwright/front_project/tests/ProjectDetail.gitlab-oauth-local-route.playwright.test.js`

- [ ] **Step 1: 增加 Playwright 回归断言**
  - 断言 `start` 返回的 `authorize_url` 可用
  - 断言二跳 Location 指向 `localhost:8012/oauth/authorize`
  - 断言 `client_id/redirect_uri` 为 local-gitlab 配置值

- [ ] **Step 2: 运行 Playwright 目标用例**

Run:
`cd task2app/playwright && npx playwright test -c front_project/playwright.config.js front_project/tests/ProjectDetail.branch-preview-local-git.playwright.test.js --project=chromium`

Expected: PASS

- [ ] **Step 3: 运行 DDD/BDD 合规检查**

Run:
`cd /Users/task2app/gitClone/ramDisk/ram-mount && python3 scripts/ci/check_ddd_bdd_compliance.py`

Expected: `DDD/BDD 合规检查通过。`

---

### Task 6: 全量验证清单（提交前）

- [ ] `cd gitOauth && python3 -m pytest api/tests.py -q`
- [ ] `cd task2app/playwright && npx playwright test -c front_project/playwright.config.js front_project/tests/ProjectDetail.branch-preview-local-git.playwright.test.js --project=chromium`
- [ ] `cd /Users/task2app/gitClone/ramDisk/ram-mount && python3 scripts/ci/check_ddd_bdd_compliance.py`

Expected: 全部通过；且手工点击项目页 OAuth 不再出现 `gitlab=bad_state`。

---

## Done Definition

- GitLab OAuth start 在 `local-gitlab` 场景下稳定跳转 GitLab 授权页
- dict/list 两种 provider 配置结构均可解析并通过测试
- `bad_state` 仅作为已分类故障回退，不再由“配置空”误触发
- 失败日志可诊断，健康接口可提前暴露配置风险
- Playwright 回归 + DDD 合规检查全部通过

