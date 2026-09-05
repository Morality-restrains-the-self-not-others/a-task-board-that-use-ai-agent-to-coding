# relayToTrae Token 审计全链路事件化 Implementation Plan

> **Derived from value stream:** `docs/superpowers/plans/2026-05-20-relay-token-audit-full-chain-events-value-stream.md`  
> **Spec:** `docs/superpowers/specs/2026-05-20-relay-token-audit-events-design.md`

**Goal:** 将 relay `register/start` 的审计语义升级为全链路三态（`attempted/succeeded/failed`），确保审计时间线与真实调用结果一致，并支持失败分类排障。

**Architecture (DDD-first):**
- 先收敛领域事件契约（事件类型与语义映射）。
- 再落应用层编排（写入时机、失败分类、响应路径一致性）。
- 最后补齐集成消费语义与全量验证（timeline/valueStream/DDD gate）。

**Tech Stack:** Django 4.x + DRF + pytest

---

## Task 1: 领域事件契约先行（Domain First）

**Files:**
- Modify: `task2app/Saas_project/cloud/domain/events/token_audit_events.py`
- Modify: `task2app/Saas_project/cloud/domain/events/__init__.py`
- Modify: `task2app/Saas_project/tests/domain/cloud/test_container_token_audit_domain_model.py`（若需补事件语义断言）

- [ ] **Step 1: 写失败测试（事件契约）**
  - 断言新增事件类型存在：`RELAY_REGISTER_ATTEMPTED/SUCCEEDED/FAILED`、`RELAY_START_ATTEMPTED/SUCCEEDED/FAILED`。
  - 断言兼容别名 `RELAY_REGISTER/RELAY_START` 指向 `*_SUCCEEDED`（避免旧逻辑断裂）。

- [ ] **Step 2: 运行测试确认失败**
```bash
cd task2app/Saas_project
../activate_env.sh unit -- pytest tests/domain/cloud/test_container_token_audit_domain_model.py -v
```
Expected: FAIL（新增事件常量尚未定义）

- [ ] **Step 3: 最小实现领域常量与兼容映射**
  - 仅在领域事件常量层定义语义，不在 domain 层引入 HTTP/ORM 细节。

- [ ] **Step 4: 回跑领域测试**
```bash
cd task2app/Saas_project
../activate_env.sh unit -- pytest tests/domain/cloud/test_container_token_audit_domain_model.py -v
```
Expected: PASS

---

## Task 2: Register/Start 全链路编排（Application Layer）

**Files:**
- Modify: `task2app/Saas_project/cloud/services/relay_to_trae_proxy.py`
- Modify: `task2app/Saas_project/tests/test_relay_to_trae_proxy.py`

- [ ] **Step 1: 写失败测试（按链路顺序断言）**
  - `register` 异常路径：必须 `attempted -> failed`。
  - `start` 异常路径：必须 `attempted -> failed`。
  - `register` 2xx 非法响应体：必须 `attempted -> failed`（`relay_response_invalid`）。
  - `start` 成功路径：必须 `attempted -> succeeded`。

- [ ] **Step 2: 运行测试确认失败**
```bash
cd task2app/Saas_project
../activate_env.sh unit -- pytest tests/test_relay_to_trae_proxy.py -k "relay_to_trae_register or relay_to_trae_start" -v
```
Expected: FAIL（尚未按三态链路写审计）

- [ ] **Step 3: 最小实现链路编排**
  - 请求发起前写 `*_ATTEMPTED`。
  - `Exception`、`HTTP>=400`、`2xx+invalid body` 写 `*_FAILED` 并补 `error_code/error_detail`。
  - 最终成功响应后写 `*_SUCCEEDED`。
  - 保持“审计写失败不阻断主链路”原则。

- [ ] **Step 4: 回归 relay 代理测试**
```bash
cd task2app/Saas_project
../activate_env.sh unit -- pytest tests/test_relay_to_trae_proxy.py -v
```
Expected: PASS

---

## Task 3: 集成语义对齐（DB 可观测视角）

**Files:**
- Modify: `task2app/Saas_project/tests/test_container_token_audit_integration.py`
- Modify: `task2app/Saas_project/cloud/services/get_container_token_audit_timeline.py`（如需调整输出分组）
- Modify: `task2app/Saas_project/tests/test_container_token_audit_timeline_api.py`（如需补语义回归）

- [ ] **Step 1: 写集成失败测试**
  - `register` 成功路径落库 `attempted + succeeded`。
  - `register` 非法响应路径落库 `attempted + failed`，且 `error_code=relay_response_invalid`。
  - timeline 查询返回时不泄露明文 token。

- [ ] **Step 2: 运行集成测试确认失败**
```bash
cd task2app/Saas_project
../activate_env.sh unit -- pytest tests/test_container_token_audit_integration.py tests/test_container_token_audit_timeline_api.py -v
```
Expected: FAIL（语义尚未完全对齐）

- [ ] **Step 3: 最小实现消费语义**
  - timeline 层保持与三态事件一致。
  - 保留旧 `relay_register/relay_start` 历史事件兼容读取（可映射为 succeeded）。

- [ ] **Step 4: 回跑集成测试**
```bash
cd task2app/Saas_project
../activate_env.sh unit -- pytest tests/test_container_token_audit_integration.py tests/test_container_token_audit_timeline_api.py -v
```
Expected: PASS

---

## Task 4: 全量验证与价值流闭环

- [ ] **Step 1: 运行本次改动相关回归集合**
```bash
cd task2app/Saas_project
../activate_env.sh unit -- pytest \
  tests/test_relay_to_trae_proxy.py \
  tests/test_container_token_audit_integration.py \
  tests/test_container_token_audit_timeline_api.py \
  tests/test_container_token_audit_app_service.py -v
```

- [ ] **Step 2: 运行 DDD/BDD 合规检查**
```bash
cd /Users/task2app/gitClone/ramDisk/ram-mount
python scripts/ci/check_ddd_bdd_compliance.py
```

- [ ] **Step 3: 验证 valueStream 配置**
```bash
cd /Users/task2app/gitClone/ramDisk/ram-mount/valueStream
go test ./...
```

---

## DDD Gate Self-Review

- [ ] 领域事件契约先于应用层编排改动（Task 1 在 Task 2 前）
- [ ] `domain/` 仅表达业务语义，不引入 ORM/HTTP/requests 依赖
- [ ] 应用层通过领域事件常量驱动，不硬编码散落字符串
- [ ] 基础设施层不承载业务判定（仅做持久化映射）
- [ ] 全链路事件均不落明文 token

---

## Value Stream Coverage Matrix

- [ ] Increment 1 `register-full-chain-thin-slice` -> Task 2 + Task 3
- [ ] Increment 2 `start-full-chain-core` -> Task 2
- [ ] Increment 3 `timeline-semantics-alignment` -> Task 3
- [ ] Increment 4 `failure-classification-enhancement` -> Task 2 + Task 4

