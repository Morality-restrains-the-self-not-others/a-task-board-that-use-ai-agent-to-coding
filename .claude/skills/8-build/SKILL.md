---
name: 8-build
description: "Step 8 - Build with TDD: execute the plan using test-driven development — red → green → refactor, task by task."
---

# /8-build — 构建（测试驱动）

Implement each task from the plan. Build and TDD are not separate phases — TDD is how building is done.

## Methodology: TDD Cycle

For each task in the plan:
1. **Red** — write a failing test that defines the expected behavior
2. **Green** — write the minimal implementation to pass the test
3. **Refactor** — clean up while keeping tests green

Invoke the Superpowers `test-driven-development` skill for TDD discipline, and
`subagent-driven-development` or `executing-plans` for parallel/serial task execution.

## Execution Handoff

Before starting implementation, present clickable options using `AskUserQuestion`:

```
AskUserQuestion with:
  header: "执行方式"
  question: "Plan is ready. How should I execute the build?"
  multiSelect: false
  options:
    1. label: "Subagent-Driven (推荐)"
       description: "每个任务派发独立子代理，任务间审查，快速迭代"
    2. label: "Inline Execution"
       description: "在当前会话中使用 executing-plans 执行，批量执行+检查点审查"
```

**If user selects "Subagent-Driven (推荐)":**
- **REQUIRED SUB-SKILL:** Use superpowers:subagent-driven-development
- Fresh subagent per task + two-stage review

**If user selects "Inline Execution":**
- **REQUIRED SUB-SKILL:** Use superpowers:executing-plans
- Batch execution with checkpoints for review

## Logging（强制内建步骤）

**每条实现代码必须伴随必要日志。** 日志不是事后补的——它和功能代码一起写。

实施每条 plan task 时，同步检查以下 6 类关键路径是否已写日志：

| # | 关键路径 | 日志要求 |
|---|---------|---------|
| 1 | **错误路径** | 每个 `except` 块（不含 `pass`）必须有 ERROR/WARN 日志 + `exc_info=True` |
| 2 | **外部调用** | 每个 HTTP/gRPC/DB 调用：请求前 DEBUG，响应后 INFO（含 status_code + duration_ms） |
| 3 | **状态变更** | 每个关键业务实体的 C/U/D 操作必须有 INFO 日志（含 from/to 状态） |
| 4 | **认证授权** | 每次 auth 拒绝必须有 WARN 日志（含 who/what/why） |
| 5 | **后台任务** | 每个 job 必须有 started + completed/failed 日志 |
| 6 | **资源生命周期** | 每个资源创建/删除必须有 INFO 日志 |
| 7 | **事件消费幂等** | 重复投递只执行一次副作用；`Seen` 命中必须 `warn` `idempotency skip`（键指纹，禁止完整键） |

### 禁止记录

- 密码 / Token / Secret / API Key → 永远不记录
- 完整 PII（身份证、银行卡、手机号）→ 脱敏处理
- Session ID 明文 → 只记录 hash 前 8 位
- 大体积 body（上传文件、Base64）→ 只记录 size

### 日志格式要求

每条日志必须包含：`timestamp`、`level`、`message`、`trace_id`。推荐包含：`request_id`、`user_id`、`duration_ms`。

完整规范和代码示例见 `/logging-audit` 技能。

**构建中遇运行时报错且带 `data-traceId`**：先按该 ID 检索 Loki/Grafana 重建全链路路径，再改实现（见 `.claude/skills/1-brainstorming-design-docs/references/traceid-log-first-diagnosis.md`）。禁止未查日志凭前端文案盲改。

## DDD Context (Backend Work)

During backend implementation, follow the domain model produced by `/6-ddd`:

- Domain-layer files (entities, value objects, aggregates, repository interfaces, domain events) are implemented first, as pure Python with no infrastructure imports
- Infrastructure-layer files (repository implementations, event publishers, external adapters) implement the ABC interfaces defined in domain
- Aggregate roots enforce invariants; all external access goes through the root
- Cross-aggregate communication uses domain events, not direct calls
- **业务意图 → MQ 事件（强制）**：实现每个业务意图命令/用例时，必须在状态变更成功后经事件总线端口向消息队列投递对应业务事件；测试须断言事件被发布（可用 `InMemoryEventPublisher`）。禁止只做同步副作用。细则：`.ai/08_prompt_management/01_intent_driven_development.md`
- **事件消费幂等（强制）**：对应消费者必须走 `eventbin.RunIntent` + `IdempotentDispatchService`；重放测例断言副作用只发生一次。细则：`.ai/01_project_constraints/54_event_consumer_idempotency.md`

### Test Structure

Write tests against the domain model:

- **Aggregate tests** in `tests/domain/` — test invariants, commands, events through root
- **Value object tests** — equality, immutability, validation
- **Domain event tests** — correct data, raised at right time; **intent acceptance publishes the mapped event**
- **Repository tests** in `tests/infrastructure/` — against real DB, not mocks
- Use `InMemoryEventPublisher` / `InMemoryEmailSender` to isolate domain tests
- Stack tests by layer: domain tests run fast and often; infrastructure tests on demand

The domain model (repository interfaces, domain service interfaces) in `domain/` is the
contract — tests verify against it. If tests reveal gaps in the domain model, go back
to `/6-ddd` to fix the model, then return to build.

### Compliance Check

Run the CI compliance script locally before committing:

```bash
python runAll/scripts/ci/check_ddd_bdd_compliance.py
```

Note: this root command is a unified wrapper that runs both:
- `task2app/scripts/ci/check_ddd_bdd_compliance.py` (Python DDD/BDD)
- `valueStream/scripts/ci/check_go_ddd_compliance.py` (Go DDD for `valueStream/domain`)

To proceed, invoke the **test-driven-development** and **subagent-driven-development** (recommended) skills via the Skill tool.

## 完成后 — 下一步选择

所有任务构建完成、测试全绿后，使用 `AskUserQuestion` 工具让用户一键选择下一步：

```
header: "下一步"
question: "构建已完成，所有测试通过。下一步做什么？"
multiSelect: false
options:
  1. label: "代码审查 (推荐)"
     description: "对照计划和领域模型审查变更，检查合规性"
  2. label: "继续构建"
     description: "还有未完成的任务，继续 TDD 构建"
```

- 用户选 1 → 调用 `/9-review`
- 用户选 2 → 继续执行本技能（TDD 构建）
