# 会话结束优化建议 Todo（编号与完成标记）

- **版本**：1.0.0
- **日期**：2026-07-17
- **适用范围**：整个 monorepo；所有 Cursor / Claude / Trae 智能体会话

## 目标

会话收尾时，把可落地的**优化建议**写入仓库内统一编号清单，并在后续会话真正做完某条时**标记为已完成**，避免建议只停留在对话末尾、无法跨会话追踪。

## 清单 SSOT

| 角色 | 路径 |
|------|------|
| **开放清单（pending SSOT）** | [`.learnings/OPTIMIZATION_TODOS.md`](../../.learnings/OPTIMIZATION_TODOS.md) |
| **已完成/取消归档** | [`.learnings/OPTIMIZATION_TODOS_COMPLETED.md`](../../.learnings/OPTIMIZATION_TODOS_COMPLETED.md) |
| 本规则（行为约束） | 本文 |
| Cursor 常驻短规则 | `.cursor/rules/session-optimization-todo.mdc` |

与 [`.learnings/LEARNINGS.md`](../../.learnings/LEARNINGS.md) 的区分：

| 文件 | 用途 |
|------|------|
| `LEARNINGS.md` | 失败/纠错/最佳实践等**经验沉淀**（self-improvement） |
| `OPTIMIZATION_TODOS.md` | 会话收尾产出的**可执行优化待办**（编号 + 完成状态） |

同一洞见可两边都记：LEARNINGS 写「为什么」，OPTIMIZATION_TODOS 写「要做什么」。

## 强制要求

### 1. 会话结束时追加优化建议

当会话达到**收尾态**（见下方触发条件）时，智能体**必须**：

1. 阅读 `.learnings/OPTIMIZATION_TODOS.md` 现有开放项，避免重复主题；
2. 使用 `python3 .learnings/_move_opt.py --next-id` 获取跨文件全局唯一编号（扫描所有 OPT 文件：pending/completed/archive/product-decisions），**禁止**手工自增编号；
3. 将本会话产生的、**可执行且非阻塞当前交付**的优化建议追加进该文件；
4. 为每条建议分配唯一编号（见编号规则）；
5. 初始状态为 `pending`；
6. 在对用户的最终答复中可用简短列表引用编号（如 `OPT-20260717-001`），**不得**只在聊天里写建议而不落盘。

**无实质建议时**：不伪造条目；可在最终答复中说明「本会话无新增优化待办」。

### 2. 完成时标记已完成并强制迁移

当某条 `OPT-*` 在本会话（或用户明确要求下）**已落地验证**时，智能体**必须**：

1. 将该条目 `**Status**` 改为 `completed`；
2. 填写 `**Completed**`（ISO-8601）与简短 `**Completion-Note**`（做了什么 / 关键路径）；
3. **禁止**删除历史条目（保留编号可追溯）；`cancelled` 仅用于明确作废且须写原因。
4. **【强制】立即迁移**：将完成的条目从开放清单移至 `.learnings/OPTIMIZATION_TODOS_COMPLETED.md`，开放清单仅保留 `pending` 项。迁移后触发归档检查（非当日条目按天归档至 `archive/completed/`）。**禁止只改 Status 不迁移**，completed 条目不得留在 pending 清单中。细则见 `.learnings/ai.md`「强制迁移流程」。

### 3. 会话开始时的轻量对齐（推荐且应做）

进入实现类任务前，宜扫一眼开放 `pending` 项：若当前工作范围**直接覆盖**某条，完成后按第 2 条标记；勿把无关开放项强行塞进本次任务。

## 触发条件（收尾态）

满足任一即视为应执行第 1 条：

- 用户任务已按约定完成，即将给出最终答复；
- `/goal` 或 goal-mode 输出 **Final Execution Overview**；
- ship / 交付收尾；
- 用户明确结束会话（如「就这样」「结束」）且本轮有可沉淀建议。

**不触发**：中途进度汇报、仅提问未交付、被中断后的「继续」中间步（续接完成后再写）。

## 编号规则

- 格式：`OPT-YYYYMMDD-NNN`
  - `YYYYMMDD`：写入当日（以会话环境「今天」为准）；
  - `NNN`：当日三位序号，从 `001` 起，**跨所有 OPT 文件**（pending/completed/archive/product-decisions）取最大 +1。
- **跨文件全局唯一**：编号必须跨 `OPTIMIZATION_TODOS.md`、`OPTIMIZATION_TODOS_COMPLETED.md`、`PRODUCT_DECISIONS.md` 及所有 `archive/completed/OPT_COMPLETED_*.md` 文件唯一，不得重复。
- **获取编号**：使用 `python3 .learnings/_move_opt.py --next-id [YYYYMMDD]`（不传日期则默认今日），该命令自动扫描所有文件返回下一个可用编号。**禁止**手工自增或仅扫描单个文件后分配。
- 编号一经分配**不复用**；状态变更不改编号。

## 条目字段（最小集）

```markdown
## [OPT-YYYYMMDD-NNN] pending

**Logged**: ISO-8601
**Priority**: low | medium | high
**Status**: pending
**Area**: frontend | backend | infra | tests | docs | rules | config | other

### Summary
一句话可执行描述

### Details
背景、为何值得做、验收要点（可选但推荐）

### Metadata
- Source: conversation | goal-overview | review | user
- Related Files: path/a, path/b
- Tags: tag1, tag2
```

完成时：将 `**Status**` 改为 `completed`，并追加：

```markdown
**Completed**: ISO-8601
**Completion-Note**: 一句话说明落地结果
```

**强制**把该条目从开放清单移至 [`.learnings/OPTIMIZATION_TODOS_COMPLETED.md`](../../.learnings/OPTIMIZATION_TODOS_COMPLETED.md)（见上方第 2 条第 4 款），编号与正文保留不变。开放清单 `OPTIMIZATION_TODOS.md` 仅保留 `pending` 项。**禁止只改 Status 不迁移**。

## 质量门槛（写入前自检）

- **可执行**：能落到具体文件/服务/流程，而非空泛「继续优化」。
- **非阻塞**：不是当前任务未完成的缺口（缺口应当场修，不应只写 todo）。
- **去重**：与已有 `pending` 的 Summary 语义相同则合并（可更新 Details），不新建编号。
- **编号唯一**：使用 `python3 .learnings/_move_opt.py --next-id` 获取跨文件全局唯一编号，禁止手工自增。
- **克制**：单会话通常 0～5 条；宁缺毋滥。

## 与相关规则的关系

- 不替代 `.learnings/LEARNINGS.md` / self-improvement 技能；
- 不替代失败经验库 `.ai/09_failure_experience/`；
- goal-mode「剩余优化建议」章节的内容**须同步写入**本清单（有建议时）；
- 与 `00_start` 会话接续兼容：续接完成后再执行收尾写入。
