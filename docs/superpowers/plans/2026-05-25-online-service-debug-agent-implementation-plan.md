# onlineServiceJS DEBUG_AGENT Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** 在 `relayToTrae=true` 直启链路中默认注入 `DEBUG_AGENT=True`，并让 onlineServiceJS 在开关开启时完整记录入站/出站 `method/url/headers/body` 调试信息。

**Architecture:** 先按 DDD 合同冻结 `cloud/domain`（VO→Entity/Aggregate→Repository ABC→Events→Domain Service），再在 `onlineServiceJS` 与 `task2app/front_project` 落地开关与日志行为，最后补齐单测/E2E 与合规校验。调试日志仅由开关启用，默认模式保持兼容摘要日志。

**Tech Stack:** Python 3.9 (Django/pytest), Node.js ESM (node:test), Vue 3/Vitest, Playwright

---

## Skill Notice

I'm using the writing-plans skill to create the implementation plan.

## 输入基线

- 设计文档：`docs/superpowers/specs/2026-05-25-online-service-debug-agent-design.md`
- 价值流：`docs/superpowers/plans/2026-05-25-online-service-debug-agent-value-stream.md`
- NFR：`docs/superpowers/plans/2026-05-25-online-service-debug-agent-nfr-clarification.md`
- DDD 模型：`docs/superpowers/plans/2026-05-25-online-service-debug-agent-ddd-model.md`

## 文件结构（按职责）

- Domain
  - `task2app/Saas_project/cloud/domain/value_objects/debug_agent_flag.py`
  - `task2app/Saas_project/cloud/domain/value_objects/http_debug_payload.py`
  - `task2app/Saas_project/cloud/domain/entities/online_service_debug_log_entry.py`
  - `task2app/Saas_project/cloud/domain/repositories/online_service_debug_log_repository.py`
  - `task2app/Saas_project/cloud/domain/events/online_service_inbound_request_handled.py`
  - `task2app/Saas_project/cloud/domain/events/online_service_outbound_request_completed.py`
  - `task2app/Saas_project/cloud/domain/services/online_service_debug_log_service.py`
- Runtime / Relay
  - `trae-agent/onlineServiceJS/src/outboundReqLog.mjs`
  - `trae-agent/onlineServiceJS/src/server.mjs`
  - `trae-agent/onlineServiceJS/src/saasTaskCloud.mjs`
  - `trae-agent/onlineServiceJS/src/layerGitOauthPush.mjs`
  - `trae-agent/onlineServiceJS/src/reachability.mjs`
  - `trae-agent/onlineServiceJS/src/stagedCommitSuggest.mjs`
- Frontend Relay Direct Start
  - `task2app/front_project/app/src/utils/relayToTraeUtils.js`
  - `task2app/front_project/app/src/utils/relayToTraeUtils.test.js`
  - `task2app/playwright/front_project/tests/TaskDetail.relay-to-trae-direct-start.playwright.test.js`
- Domain Tests
  - `task2app/Saas_project/tests/domain/cloud/test_online_service_debug_log_domain_model.py`

## 依赖顺序（DDD 约束）

1. Domain layer first（VO → Entity/Aggregate → Repository ABC → Events → Domain Service）
2. Runtime logging behavior second（onlineServiceJS 出/入站日志）
3. Relay direct-start env injection third（前端默认值与透传）
4. Regression + compliance verification last

---

### Task 1: 冻结 Domain 合同（DDD First）

**Files:**
- Modify/Create: `task2app/Saas_project/cloud/domain/value_objects/debug_agent_flag.py`
- Modify/Create: `task2app/Saas_project/cloud/domain/value_objects/http_debug_payload.py`
- Modify/Create: `task2app/Saas_project/cloud/domain/entities/online_service_debug_log_entry.py`
- Modify/Create: `task2app/Saas_project/cloud/domain/repositories/online_service_debug_log_repository.py`
- Modify/Create: `task2app/Saas_project/cloud/domain/events/online_service_inbound_request_handled.py`
- Modify/Create: `task2app/Saas_project/cloud/domain/events/online_service_outbound_request_completed.py`
- Modify/Create: `task2app/Saas_project/cloud/domain/services/online_service_debug_log_service.py`
- Test: `task2app/Saas_project/tests/domain/cloud/test_online_service_debug_log_domain_model.py`

- [ ] **Step 1: 写失败测试（开关解析、载荷约束、聚合构建、服务门控）**
- [ ] **Step 2: 运行失败测试**
  - Run: `cd task2app/Saas_project && pytest tests/domain/cloud/test_online_service_debug_log_domain_model.py -q`
  - Expected: FAIL（类或行为未定义）
- [ ] **Step 3: 实现最小领域模型代码（不导入 ORM/HTTP/SDK）**
- [ ] **Step 4: 运行测试确认通过**
  - Run: `cd task2app/Saas_project && pytest tests/domain/cloud/test_online_service_debug_log_domain_model.py -q`
  - Expected: PASS
- [ ] **Step 5: Commit**
  - Run: `git add task2app/Saas_project/cloud/domain task2app/Saas_project/tests/domain/cloud/test_online_service_debug_log_domain_model.py && git commit -m "feat(cloud-domain): add debug-agent logging domain contract"`

---

### Task 2: onlineServiceJS 入站请求完整日志（Increment 2）

**Files:**
- Modify: `trae-agent/onlineServiceJS/src/outboundReqLog.mjs`
- Modify: `trae-agent/onlineServiceJS/src/server.mjs`

- [ ] **Step 1: 写/补失败测试（若无现成测试，先以最小 node:test 补中间件行为断言）**
- [ ] **Step 2: 运行测试确认失败**
  - Run: `cd trae-agent/onlineServiceJS && npm test -- --runInBand`
  - Expected: FAIL（缺少 DEBUG_AGENT 入站日志行为）
- [ ] **Step 3: 实现 `DEBUG_AGENT` 判定 + 入站中间件记录 request/response 全量字段**
- [ ] **Step 4: 运行测试确认通过**
  - Run: `cd trae-agent/onlineServiceJS && npm test -- --runInBand`
  - Expected: PASS
- [ ] **Step 5: Commit**
  - Run: `git add trae-agent/onlineServiceJS/src/outboundReqLog.mjs trae-agent/onlineServiceJS/src/server.mjs && git commit -m "feat(onlineServiceJS): add DEBUG_AGENT inbound full logging"`

---

### Task 3: onlineServiceJS 出站请求完整日志（Increment 3）

**Files:**
- Modify: `trae-agent/onlineServiceJS/src/saasTaskCloud.mjs`
- Modify: `trae-agent/onlineServiceJS/src/layerGitOauthPush.mjs`
- Modify: `trae-agent/onlineServiceJS/src/reachability.mjs`
- Modify: `trae-agent/onlineServiceJS/src/stagedCommitSuggest.mjs`
- Modify: `trae-agent/onlineServiceJS/src/server.mjs`

- [ ] **Step 1: 为关键 fetch 路径写/补失败测试（至少覆盖 postJson + 一个外部 API 调用）**
- [ ] **Step 2: 运行测试确认失败**
  - Run: `cd trae-agent/onlineServiceJS && node --test src/**/*.test.mjs`
  - Expected: FAIL（缺少出站 debug 字段）
- [ ] **Step 3: 实现 `method/url/headers/body` 与响应全量记录（不脱敏、不截断）**
- [ ] **Step 4: 运行测试确认通过**
  - Run: `cd trae-agent/onlineServiceJS && node --test src/**/*.test.mjs`
  - Expected: PASS
- [ ] **Step 5: Commit**
  - Run: `git add trae-agent/onlineServiceJS/src/saasTaskCloud.mjs trae-agent/onlineServiceJS/src/layerGitOauthPush.mjs trae-agent/onlineServiceJS/src/reachability.mjs trae-agent/onlineServiceJS/src/stagedCommitSuggest.mjs trae-agent/onlineServiceJS/src/server.mjs && git commit -m "feat(onlineServiceJS): add DEBUG_AGENT outbound full logging"`

---

### Task 4: relay 直启默认注入 DEBUG_AGENT（Increment 1）

**Files:**
- Modify: `task2app/front_project/app/src/utils/relayToTraeUtils.js`
- Test: `task2app/front_project/app/src/utils/relayToTraeUtils.test.js`
- Test: `task2app/playwright/front_project/tests/TaskDetail.relay-to-trae-direct-start.playwright.test.js`

- [ ] **Step 1: 写失败测试（默认 env 包含 `DEBUG_AGENT=True`，start payload 透传）**
- [ ] **Step 2: 运行失败测试**
  - Run: `cd task2app/front_project/app && npm run -s test -- src/utils/relayToTraeUtils.test.js`
  - Expected: FAIL（未包含 DEBUG_AGENT）
- [ ] **Step 3: 实现默认 env key/value 及透传行为**
- [ ] **Step 4: 运行单测确认通过**
  - Run: `cd task2app/front_project/app && npm run -s test -- src/utils/relayToTraeUtils.test.js`
  - Expected: PASS
- [ ] **Step 5: 运行 E2E 断言（可针对单文件）**
  - Run: `cd task2app && npx playwright test playwright/front_project/tests/TaskDetail.relay-to-trae-direct-start.playwright.test.js`
  - Expected: PASS（start payload 中 `env.DEBUG_AGENT === "True"`）
- [ ] **Step 6: Commit**
  - Run: `git add task2app/front_project/app/src/utils/relayToTraeUtils.js task2app/front_project/app/src/utils/relayToTraeUtils.test.js task2app/playwright/front_project/tests/TaskDetail.relay-to-trae-direct-start.playwright.test.js && git commit -m "feat(task-detail): default DEBUG_AGENT for relay direct start"`

---

### Task 5: 全链路验证与合规门禁（Increment 4）

**Files:**
- Verify: `task2app/Saas_project/tests/domain/cloud/test_online_service_debug_log_domain_model.py`
- Verify: `task2app/front_project/app/src/utils/relayToTraeUtils.test.js`
- Verify: `task2app/playwright/front_project/tests/TaskDetail.relay-to-trae-direct-start.playwright.test.js`
- Verify: `scripts/ci/check_ddd_bdd_compliance.py`

- [ ] **Step 1: 跑领域模型测试**
  - Run: `cd task2app/Saas_project && pytest tests/domain/cloud/test_online_service_debug_log_domain_model.py -q`
  - Expected: PASS
- [ ] **Step 2: 跑前端单测**
  - Run: `cd task2app/front_project/app && npm run -s test -- src/utils/relayToTraeUtils.test.js`
  - Expected: PASS
- [ ] **Step 3: 跑 relay 直启 E2E**
  - Run: `cd task2app && npx playwright test playwright/front_project/tests/TaskDetail.relay-to-trae-direct-start.playwright.test.js`
  - Expected: PASS
- [ ] **Step 4: 跑 DDD 合规检查**
  - Run: `cd /Users/task2app/gitClone/ramDisk/ram-mount && python scripts/ci/check_ddd_bdd_compliance.py`
  - Expected: `DDD/BDD 合规检查通过。`
- [ ] **Step 5: Commit（verification batch）**
  - Run: `git add docs/superpowers/plans/2026-05-25-online-service-debug-agent-implementation-plan.md && git commit -m "docs: add DEBUG_AGENT implementation execution plan"`

---

## 自检清单

- [ ] 计划覆盖 value stream 的四个 increment
- [ ] 任务顺序满足 DDD 依赖（领域层优先）
- [ ] 每个任务都有可执行验证命令和期望结果
- [ ] 无 TODO/TBD 占位符
- [ ] 关闭 `DEBUG_AGENT` 的兼容行为在任务中有验证
