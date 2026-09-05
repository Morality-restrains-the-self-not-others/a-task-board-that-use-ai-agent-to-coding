---
name: 9-review
description: "Step 9 - Code Review: review against the plan, block on serious issues."
---

# /9-review — 代码审查

Invoke the Superpowers `requesting-code-review` skill: review changes against the plan, catch issues before they compound.

## CRG 影响面核查（软依赖，fail-open）

审查开始前，若 `.code-review-graph/graph.db` 可用（`code-review-graph status` 正常），用图核查变更符号：

- `code-review-graph impact <symbol|file>` — 爆炸半径：变更符号的全部直接/间接依赖方
- `code-review-graph dead-code` — 变更面内无调用者的死代码
- `code-review-graph search/query` — 调用方兼容性与遗漏调用链

发现未覆盖的调用方 → 补测或标注；图缺失/命令不可用 → 记录一行 `CRG unavailable` 后按常规路径审查。细则：[code-review-graph.md](../1-brainstorming-design-docs/references/code-review-graph.md)

## DDD Context (Backend Work)

For backend code review, invoke the `/6-ddd` skill in compliance audit mode to run a DDD
compliance audit: scan domain-layer imports, check aggregate boundaries, verify event
flow (not direct calls), confirm repository abstraction, and check layering. Run:

```bash
python runAll/scripts/ci/check_ddd_bdd_compliance.py
```

Note: this root command is a unified entrypoint. It runs:
- Python DDD/BDD compliance in `task2app/scripts/ci/check_ddd_bdd_compliance.py`
- Go DDD compliance for `valueStream/domain` in `valueStream/scripts/ci/check_go_ddd_compliance.py`

Block on: domain files importing infrastructure, business logic in application layer,
missing repository interfaces, cross-aggregate direct calls, or **accepted business intents
that never publish a corresponding domain event to the message queue** (unless the intent
doc documents a valid no-event exception).

### 业务意图 → 事件投递审计（强制）

对照 `docs/intents/*.intent.md` 中的「意图 → 事件」表与实现：

- [ ] 每个非例外业务意图在成功路径上有 `publish` / `send_event` / `PublishEvent`
- [ ] 事件名与意图文档对照一致（过去式）
- [ ] 测试覆盖「意图发生 → 事件被投递」
- [ ] 跨边界副作用（邮件/SSE/下游）由事件消费者触发，而非命令路径同步直调
- [ ] 事件消费者经 `eventbin.RunIntent` + 与业务重复边界同粒度的幂等键；有重放测例（元规则 49 / ADR-0015）

缺失时：

```
🔴 Intent→Event Gap: <意图名> — 无对应 MQ 事件投递
   → 下一步: /6-ddd 补事件契约 → /7-plans → /8-build
```

### Domain Model Issues Found During Review

If review discovers domain model problems (wrong aggregate boundaries, missing events,
incorrect context partitioning), do NOT patch them in the implementation layer. Instead:

```
8-review 发现问题 → 重新调用 5-ddd 修正领域模型 → 更新 6-plans → 7-build 修复实现
```

The domain model is the source of truth. Implementation fixes that bypass the domain
model create drift that compounds over time.

## Log Audit（强制内建步骤）

代码审查**必须包含日志审计**。在审查每条变更路径时，逐项检查以下清单。完整规范和代码示例见 `/logging-audit` 技能。

若变更涉及**前端请求错误展示**：检查错误 DOM 是否挂载 `data-traceId`（元规则 24）；若审查中复现到带 ID 的失败，须先按该 ID 查日志重建路径再评判根因（见 `1-brainstorming-design-docs/references/traceid-log-first-diagnosis.md`）。

### A. 覆盖率检查

- [ ] 每个 `except` 块（不含 `pass`）是否有 ERROR/WARN 日志？
- [ ] 每个外部 HTTP/gRPC/DB 调用是否有请求前 DEBUG + 响应后 INFO？
- [ ] 每个关键状态变更（业务实体 C/U/D）是否有 INFO 日志？
- [ ] 每个认证拒绝点是否有 WARN 日志（含 who/what/why）？
- [ ] 每个后台任务是否有 started/completed/failed 日志？
- [ ] 每个资源创建/删除是否有 INFO 日志？

### B. 质量检查

- [ ] 日志消息是否包含足够的定位信息（ID、状态、操作人）？
- [ ] 错误日志是否使用 `exc_info=True` / `logger.exception()` 记录 traceback？
- [ ] 是否有硬编码的敏感信息（搜索 `password`, `secret`, `token`, `key`）？
- [ ] 日志级别是否使用合理（ERROR 不是 WARN，INFO 不是 DEBUG）？

### C. 噪音检查

- [ ] 循环内是否有 INFO 日志？（高频路径用 DEBUG）
- [ ] 是否有 `log.info("here")` / `log.info("test")` 等无意义日志？
- [ ] 是否有被注释掉的日志语句？（要么恢复，要么删除）

### 审计输出格式

发现日志缺失时，与代码问题并列报告：

```
🟡 Log Gap: <文件名>:<行号> — <缺少什么日志>
   → 下一步: 回到 Step 8 补齐日志后重新审查
```

## Next Steps: Mapping Risks to Follow-up Skills

After review identifies risks, always state which skill should be used to resolve each
risk. Never leave the user guessing what to do next.

| Risk Type | Next Skill | Rationale |
|---|---|---|
| Domain model wrong (aggregate boundaries, missing events, context partitioning) | `/6-ddd` → `/7-plans` → `/8-build` | Domain model is source of truth; implementation fixes bypassing it create drift |
| Business intent without MQ event publish | `/6-ddd` → `/7-plans` → `/8-build` | Intent must map to a published domain event; update `docs/intents/` |
| Design / architecture flawed (wrong pattern, missing abstraction, coupling) | `/1-brainstorming-design-docs` → `/3-worktrees` | Re-think the design from scratch, isolate in a worktree |
| NFR / quality concerns (performance, security, scalability, reliability, idempotency) | `/5-nfr` → `/6-ddd` | Clarify quality requirements before touching domain model；写路径须有「幂等性审视」表 |
| Log gap / missing or insufficient logs | `/8-build` | Add missing logs per `/logging-audit` checklist, then re-review |
| Implementation bug / logic error | `/8-build` | Fix with TDD: write failing test → fix → refactor |
| Missing tests / test gaps | `/8-build` | Add tests via TDD, not after the fact |
| Plan deviation (implemented something not in plan, or skipped plan items) | `/7-plans` → `/8-build` | Update plan to reflect reality, then rebuild aligned parts |
| Code quality / over-complexity / duplication | `simplify` | Refactor for reuse, quality, and efficiency |
| License / compliance issue | `/0-license-compliance-check` | Verify license compatibility before proceeding |

**Output format:** When reporting each risk, append the recommendation inline:

```
🔴 Risk: <description>
   → 下一步: /X-技能名 来解决 <具体问题>
```

To proceed, invoke the **requesting-code-review** skill via the Skill tool.

## 优化建议落盘（强制）

审查中发现的**可执行的后续优化建议**（非阻塞性问题、技术债、改进方向），必须在审查结束前写入 `.learnings/OPTIMIZATION_TODOS.md`：

- **编号格式**：`OPT-YYYYMMDD-NNN`（如 `OPT-20260725-001`）
- **初始状态**：`pending`
- **每条必含**：标题、触发场景、建议方案、优先级
- **禁止**：只口头列出优化建议而不落盘。发现一条写一条。

落盘路径（仓库根目录）：`/tmp/ram-work/.learnings/OPTIMIZATION_TODOS.md`

## 完成后 — 下一步选择

审查完成后，使用 `AskUserQuestion` 工具让用户一键选择下一步：

```
header: "下一步"
question: "代码审查完成。下一步做什么？"
multiSelect: false
options:
  1. label: "交付上线 (审查通过)"
     description: "验证、合并/PR、清理工作树、记录经验"
  2. label: "修复实现问题"
     description: "用 TDD 修复 Bug 或实现问题"
  3. label: "修正领域模型"
     description: "领域模型有误，需要重新 DDD 建模"
  4. label: "重新设计"
     description: "架构或设计有根本性问题，从头脑风暴重新开始"
```

- 用户选 1 → 调用 `/10-ship`
- 用户选 2 → 调用 `/8-build`
- 用户选 3 → 调用 `/6-ddd`
- 用户选 4 → 调用 `/1-brainstorming-design-docs`
