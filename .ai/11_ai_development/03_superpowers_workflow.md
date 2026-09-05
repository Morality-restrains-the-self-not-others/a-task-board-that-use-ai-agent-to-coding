# 统一 Agent 交付工作流（本仓库单一流程）

> **文件说明**：路径仍为 `03_superpowers_workflow.md` 以保持历史链接有效；正文即 **唯一** 交付流水线。吸收 [obra/superpowers](https://github.com/obra/superpowers) 与 [garrytan/gstack](https://github.com/garrytan/gstack) 的社区命名，**编号以本仓库十步为准**，不再维护并行「第二套编号」。

## 基本信息

- 版本：3.0.0
- 创建日期：2026-05-11
- 最后修改：2026-08-19
- 维护者：与 `00_ai_development.md` 同步维护

## 来源与许可

本流程吸收 **Superpowers**（[obra/superpowers](https://github.com/obra/superpowers)，MIT）与 **gstack sprint** 命名（[garrytan/gstack](https://github.com/garrytan/gstack)，MIT），经本项目裁剪为 **单轨十步**（含角色权限、价值流、NFR、DDD 等本仓库扩展步）。不替代 **BDD/DDD、意图金字塔、`docs/intents/`** 等强制规则；冲突时以 **`.ai/`、`project_rules.md`** 为准。

可选：个人可安装上游 Superpowers / gstack 插件以复用 slash 命令；**团队默认以本文件与 `.claude/skills/` 为准**。

## 默认自动执行（无需用户点名）

- 凡属 **实现类任务**（新增或改变业务语义、API、UI、持久化、跨多文件逻辑、或依赖新/改测例者），智能体须在会话内 **自觉按下方十步推进**（可按专文裁剪跳过 Step 3 等），**不必**等待用户说出 *Superpowers*、*gstack*、*Think* 等词。可在首次进入实现阶段时用一句话声明正按 **「统一交付流水线」** 执行即可。
- **等价行为**：未安装上游插件时，**不**要求调用上游 `Skill` 工具；须在对应阶段主动加载本仓库 **`.claude/skills/`** 与 **`.ai`** 专文并落实该步产出。
- **用户显式要求跳过某步**：须在对话中简要记录偏离理由；**不得**用于规避支付/安全/Git 等核心约束；若跳过设计，仍须满足 `docs/intents/` 与测试意图的既有强制要求。
- **编排入口**：全自动见 `.claude/skills/0-auto-flow`；持久目标见 `.claude/skills/goal-mode`（后者可覆盖 Step 1 用户闸门，见该技能）。

## 适用范围与裁剪

| 情形 | 是否默认走满十步 |
|------|------------------|
| 新功能、跨模块改动、长会话、易返工需求 | **是**（与 `intent-framed-agent`、`context-surfing` 等组合）；Step 3 worktree 可 SKIP |
| 单点 typo、重命名、注释、无行为变更的格式调整、纯配置键名在同文件内替换 | **否**（可裁剪中间步；仍遵守 Git、pre-commit、相关测试与合规） |

## 权威十步流水线（SSOT）

与 `.claude/skills/0-auto-flow` 的 Sequence **编号一一对应**。目录名即步骤号；**禁止**再把上游 Superpowers 的步号当作本仓库步骤号。

| 步 | 技能目录 | 目标 | 本项目要点 |
|----|----------|------|------------|
| 1 | `1-brainstorming-design-docs`（别名 `1-brainstorming`） | 澄清意图与约束，产出设计 | `docs/intents/`；架构变更须 `.puml`+`.archimate`+`.mermaid.md`；Python 新 API 额外审批门 |
| 2 | `2-role-permission` | 角色/权限边界 | 新/改 endpoint 与数据访问路径 |
| 3 | `3-worktrees` | 大改隔离 | **默认可 SKIP**（`0-auto-flow` 默认当前工作区） |
| 4 | `4-value-stream` | 端到端价值流增量 | `docs/superpowers/plans/*-value-stream.md` |
| 5 | `5-nfr` | 非功能支撑程度 | 质量场景 → 喂给 DDD；**硬门禁**：路径分片键（元规则 43 / `48_nfr_path_shard_id_scalability.md`）；副作用路径幂等性（元规则 48 / `53_nfr_idempotency.md`） |
| 6 | `6-ddd` | 领域建模 | 端口-适配器；业务意图 → MQ 事件；消费幂等键与业务重复边界同粒度（元规则 49） |
| 7 | `7-plans` | 可勾选实施计划 | 含事件契约 → publish → 消费者任务 |
| 8 | `8-build` | TDD 构建 | 红→绿→重构；必要日志；意图成功路径投递事件；消费者重放测例 |
| 9 | `9-review` | 对照计划审查 | Log Audit + Intent→Event 审计 |
| 10 | `10-ship` | 验证、**直接合入 main + push origin**、清理、复盘 | 禁止默认 `gh pr create`；公网 SPA 构建勾选；合入后拆 `*-wt` 并删已合并 `feat/*`（约束 21 / ADR-0019） |

**CRG（code-review-graph）标注**：Step 1 与 Step 9 **必须尝试**用图（详见 `.claude/skills/1-brainstorming-design-docs/references/code-review-graph.md`）；Step 0 在流水线开始时自动 `update` 同步 `.code-review-graph/graph.db`；Steps 2/6/7/8/10 建议；Step 3 跳过。软依赖：缺图/不可用记录后继续，不阻断交付。

**默认全自动序列**（Step 3 跳过时）：

```
/1-brainstorming-design-docs → /2-role-permission → /4-value-stream → /5-nfr → /6-ddd → /7-plans → /8-build → /9-review → /10-ship
```

技能目录索引见 [`.claude/skills/README.md`](../../.claude/skills/README.md)。

## 与上游「七步 / sprint」概念映射（非第二套编号）

| Superpowers / gstack 概念 | 本仓库落点 |
|---------------------------|------------|
| *brainstorming* / **Think** | Step 1 |
| *using-git-worktrees* | Step 3 |
| *writing-plans* / **Plan** | Step 7 |
| *subagent-driven-development* / *executing-plans* / **Build** | Step 8 |
| *test-driven-development*（红绿重构子环） | **内嵌于** Step 8（无独立步号） |
| *requesting-code-review* / **Review** | Step 9 |
| *finishing-a-development-branch* / **Test · Ship · Reflect** | Step 10 |
| （本仓库扩展）角色权限 / 价值流 / NFR / DDD | Steps **2 / 4 / 5 / 6** |

## 废弃同号目录（薄重定向）

下列目录**仅保留兼容旧链接**，正文须立即转去权威技能；**禁止**再按其 frontmatter「Step N」解释流水线位置：

| 旧目录 | 转去 |
|--------|------|
| `2-worktrees` | `3-worktrees` |
| `3-plans` | `7-plans` |
| `4-build` | `8-build` |
| `5-tdd` | `8-build` |
| `6-tdd` | `8-build` |
| `6-review` | `9-review` |
| `7-ship` | `10-ship` |

## 哲学（内化表述）

- **测试先行**：与仓库 BDD/TDD 一致。  
- **系统化优于临场发挥**：疑难走 `.ai/09_failure_experience/` 与 **diagnose** 闭环。  
- **优先减复杂度**：YAGNI。  
- **以可运行证据为准**：与 *verification-before-completion* 一致。

## 与上游「Skill 工具必调」的边界

上游 Superpowers `using-superpowers` 中「凡有 1% 可能须先调 Skill」面向带专有 Skill 工具的宿主。**本仓库**以本文件十步 + 按需读 `.ai` / `.claude/skills` **实现同一纪律**，且不依赖用户逐句触发关键词。

## 调试与完成门槛（穿插步骤）

| 上游技能 | 核心要求 | 本项目落点 |
|----------|----------|------------|
| **systematic-debugging** | 先根因再改代码 | [`.ai/09_failure_experience/00_failure_experience.md`](../09_failure_experience/00_failure_experience.md)；`project_rules` **diagnose**；可选 `ce-debug`；**有 `data-traceId` 时先 Loki 重建路径**（[traceid-log-first-diagnosis.md](../../.claude/skills/1-brainstorming-design-docs/references/traceid-log-first-diagnosis.md)） |
| **verification-before-completion** | 声称完成前须有**本轮**验证输出 | `.ai/05_testing_quality/`、`测试.ai.md`；`.ai/06_execution_monitoring/`；CI |

第 8～10 步与上表叠加时：**无复现、无验证输出则不得宣称修复或交付完成**。

## 规则冲突处理

与 `.ai/01_project_constraints/`、`00_project_constraints.md`、支付合规、Git 禁止 `--no-verify` 等冲突时，以核心约束为准。

## 变更日志

- 2026-08-19：版本 **3.0.3** - Step 10 改为默认直接合入 `main` + `git push origin`（禁止默认 `gh pr create`；ADR-0019 / 约束 21）
- 2026-08-18：版本 **3.0.2** - Step 6/8 增补事件消费幂等落地（元规则 49 / ADR-0015）
- 2026-07-20：版本 **3.0.1** - systematic-debugging 落点增补：有 `data-traceId` 时先 Loki 重建路径（`traceid-log-first-diagnosis.md`）
- 2026-07-18：版本 **3.0.0** - **权威十步 SSOT**：与 `0-auto-flow` 编号对齐；上游七步改为概念映射；废弃同号目录表；TDD 明确内嵌 Step 8
- 2026-07-15：版本 **2.0.1** - Ship 步落点增补：公网 SPA collectstatic 勾选（与 simplify-and-harden Pass 4、`/10-ship` 对齐）
- 2026-05-11：版本 **2.0.0** - **单轨合并**：七步表并入 gstack sprint 命名；新增「默认自动执行」；废弃与 `04` 的并行流程说明（`04` 改为重定向）
- 2026-05-11：版本 1.1.1 - gstack 命名指针（并入 2.0.0）
- 2026-05-11：版本 1.1.0 - 「调试与完成门槛」
- 2026-05-11：版本 1.0.0 - Superpowers 初版
