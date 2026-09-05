# onlineServiceJS init.log Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** 在 `trae-agent/onlineServiceJS` 启动阶段新增 `logs/init.log` 快照日志，记录本次启动环境变量（支持可选白名单 `INIT_LOG_ENV_KEYS`），并满足“写失败不阻断启动”。

**Architecture:** 以现有 DDD 合同为约束（`InitLogEnvKeysPolicy` / `OnlineServiceRuntimeEnv` / `OnlineServiceInitSnapshot` / `OnlineServiceInitLogService`），在 `onlineServiceJS` 增加启动日志应用服务层与基础设施写盘适配；`server.mjs` 只负责调用，不承载复杂采集逻辑。默认全量采集，配置白名单时只记录白名单键。

**Tech Stack:** Node.js ESM (node:test), Python 3.9 (pytest), Django domain contract verification

---

## Skill Notice

I'm using the writing-plans skill to create the implementation plan.

## 输入基线

- 设计文档：`docs/superpowers/specs/2026-05-25-online-service-init-log-design.md`
- NFR 文档：`docs/superpowers/plans/2026-05-25-online-service-init-log-nfr-clarification.md`
- DDD 产物：
  - `task2app/Saas_project/cloud/domain/value_objects/init_log_env_keys_policy.py`
  - `task2app/Saas_project/cloud/domain/value_objects/online_service_runtime_env.py`
  - `task2app/Saas_project/cloud/domain/entities/online_service_init_snapshot.py`
  - `task2app/Saas_project/cloud/domain/repositories/online_service_init_snapshot_repository.py`
  - `task2app/Saas_project/cloud/domain/events/online_service_bootstrapping_started.py`
  - `task2app/Saas_project/cloud/domain/events/online_service_init_logged.py`
  - `task2app/Saas_project/cloud/domain/services/online_service_init_log_service.py`

## 文件结构（实施目标）

- Runtime implementation (`trae-agent/onlineServiceJS/src`)
  - `server.mjs`（接入启动日志调用）
  - `paths.mjs`（复用 `logsDir()`，无需新增路径函数）
  - `initLog.mjs`（新增：启动快照采集 + best-effort 写盘）
- Runtime tests (`trae-agent/onlineServiceJS/src`)
  - `initLog.test.mjs`（新增：全量采集/白名单/JSON 行格式）
  - `server.initLog.test.mjs`（新增：写失败不阻断启动的行为）
- Domain verification (`task2app/Saas_project/tests/domain/cloud`)
  - `test_online_service_init_log_domain_model.py`（已存在，计划中要求回归）

## 依赖顺序（DDD 约束）

1. **Domain contract first:** 先确认 DDD 合同测试通过（不改 domain 语义）
2. **Infrastructure implementation second:** 实现 `onlineServiceJS` 启动日志写盘
3. **Entrypoint wiring third:** 将 `server.mjs/main()` 接入 `initLog`，保持 best-effort
4. **Verification last:** Node 单测 + domain 回归 + DDD 合规

---

### Task 1: 锁定领域契约基线（DDD 先行校验）

**Files:**
- Verify: `task2app/Saas_project/tests/domain/cloud/test_online_service_init_log_domain_model.py`

- [ ] **Step 1: 回归领域测试，锁定契约不漂移**
  - Run: `cd task2app/Saas_project && pytest tests/domain/cloud/test_online_service_init_log_domain_model.py -q`
  - Expected: PASS
- [ ] **Step 2: 确认无需修改 domain 代码（仅作为 runtime 实现约束）**
- [ ] **Step 3: Commit（如有文档/注释微调）**
  - Run: `git add task2app/Saas_project/tests/domain/cloud/test_online_service_init_log_domain_model.py && git commit -m "test(domain): lock init-log contract baseline"`

---

### Task 2: 新增 init.log 启动快照实现（Thin Slice）

**Files:**
- Create: `trae-agent/onlineServiceJS/src/initLog.mjs`
- Test: `trae-agent/onlineServiceJS/src/initLog.test.mjs`

- [ ] **Step 1: 先写失败测试**
  - 覆盖点：
    - 默认全量采集 `process.env`
    - `INIT_LOG_ENV_KEYS` 白名单模式
    - 输出为 JSON Lines（每次一行，含 `ts/event/pid/port/env`）
- [ ] **Step 2: 运行测试确认失败**
  - Run: `cd trae-agent/onlineServiceJS && node --test src/initLog.test.mjs`
  - Expected: FAIL（模块未实现）
- [ ] **Step 3: 实现 `initLog.mjs`**
  - 提供 `buildInitSnapshot()` 与 `appendInitLogBestEffort()`（命名可微调）
  - 使用 `logsDir()` + `fs.appendFileSync` 写入 `logs/init.log`
  - 白名单解析规则与 DDD 一致（逗号分隔，去空格）
- [ ] **Step 4: 运行测试确认通过**
  - Run: `cd trae-agent/onlineServiceJS && node --test src/initLog.test.mjs`
  - Expected: PASS
- [ ] **Step 5: Commit**
  - Run: `git add trae-agent/onlineServiceJS/src/initLog.mjs trae-agent/onlineServiceJS/src/initLog.test.mjs && git commit -m "feat(onlineServiceJS): add startup init.log snapshot writer"`

---

### Task 3: 在 server 启动流程接入 init.log（可用性 L2）

**Files:**
- Modify: `trae-agent/onlineServiceJS/src/server.mjs`
- Test: `trae-agent/onlineServiceJS/src/server.initLog.test.mjs`

- [ ] **Step 1: 写失败测试（写盘异常不阻断启动）**
  - 场景：
    - `appendInitLogBestEffort()` 抛异常时，`main()` 仍继续 `listen`
    - 正常情况下启动时会调用一次 init.log 写入
- [ ] **Step 2: 运行测试确认失败**
  - Run: `cd trae-agent/onlineServiceJS && node --test src/server.initLog.test.mjs`
  - Expected: FAIL（尚未接入）
- [ ] **Step 3: 在 `main()` 早期接入 init.log**
  - 时机：`runBootstrapTokenExchangeOnly()` 前
  - 规则：best-effort，不得 `process.exit`
- [ ] **Step 4: 运行测试确认通过**
  - Run: `cd trae-agent/onlineServiceJS && node --test src/server.initLog.test.mjs`
  - Expected: PASS
- [ ] **Step 5: Commit**
  - Run: `git add trae-agent/onlineServiceJS/src/server.mjs trae-agent/onlineServiceJS/src/server.initLog.test.mjs && git commit -m "feat(onlineServiceJS): log startup env snapshot without blocking startup"`

---

### Task 4: 联合回归与兼容性验证

**Files:**
- Verify: `trae-agent/onlineServiceJS/src/server.debugAgentInbound.test.mjs`
- Verify: `trae-agent/onlineServiceJS/src/saasTaskCloud.debugAgentOutbound.test.mjs`
- Verify: `trae-agent/onlineServiceJS/src/stagedCommitSuggest.debugAgentOutbound.test.mjs`
- Verify: `task2app/Saas_project/tests/domain/cloud/test_online_service_init_log_domain_model.py`

- [ ] **Step 1: 跑 init-log + debug 相关 Node 测试**
  - Run: `cd trae-agent/onlineServiceJS && node --test src/initLog.test.mjs src/server.initLog.test.mjs src/server.debugAgentInbound.test.mjs src/saasTaskCloud.debugAgentOutbound.test.mjs src/stagedCommitSuggest.debugAgentOutbound.test.mjs`
  - Expected: PASS
- [ ] **Step 2: 跑 domain 回归测试**
  - Run: `cd task2app/Saas_project && pytest tests/domain/cloud/test_online_service_init_log_domain_model.py -q`
  - Expected: PASS
- [ ] **Step 3: 人工验证日志文件生成（可选）**
  - Run: `cd trae-agent/onlineServiceJS && ONLINE_PROJECT_STATE_ROOT=../onlineProject_state ONLINE_SERVICE_JS_SKIP_MAIN=1 node --test src/initLog.test.mjs`
  - Expected: PASS，且测试目录中生成 `logs/init.log`

---

### Task 5: 合规门禁与计划归档

**Files:**
- Verify: `scripts/ci/check_ddd_bdd_compliance.py`
- Doc: `docs/superpowers/plans/2026-05-25-online-service-init-log-implementation-plan.md`

- [ ] **Step 1: 执行 DDD 合规检查**
  - Run: `cd /Users/task2app/gitClone/ramDisk/ram-mount/task2app && python scripts/ci/check_ddd_bdd_compliance.py`
  - Expected: `DDD/BDD 合规检查通过。`
- [ ] **Step 2: 归档计划文档（如需单独提交）**
  - Run: `git add docs/superpowers/plans/2026-05-25-online-service-init-log-implementation-plan.md && git commit -m "docs: add onlineServiceJS init-log implementation plan"`

---

## 自检清单

- [ ] 计划任务顺序满足 DDD 依赖（domain 基线 → runtime 实现 → wiring → 回归）
- [ ] 每个任务都有可执行命令和期望结果
- [ ] 覆盖 NFR 关键点（可观测性 L3、可用性 L2、可维护性 L2）
- [ ] 明确“写失败不阻断启动”的验证任务
- [ ] 无 TODO/TBD 占位项
