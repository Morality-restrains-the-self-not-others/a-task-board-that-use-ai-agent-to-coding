---
name: 0-auto-flow
description: Use when the user wants to run the full development pipeline end-to-end automatically, or uses phrases like "全自动开发", "一键开发流程", "auto dev flow", "自动走完整流程", "从设计到交付全自动", "自动实现"
---

# 0-Auto-Flow (全自动开发流程)

## Overview
Orchestrates the complete **canonical 10-step** pipeline（SSOT：`.ai/11_ai_development/03_superpowers_workflow.md`）。编号与目录一一对应：`1-brainstorming-design-docs` … `10-ship`。

**Step 1 用户闸门（默认开启）：** 单独调用本技能时，brainstorming 后须取得用户确认再继续。  
**例外：** 若由 `/goal`（`goal-mode`）委托本流水线，则 **跳过** Step 1 `AskUserQuestion`，自动采用设计并进入 Step 2（见 `goal-mode` Mandatory Rules）。Steps 2–10 在两种入口下均零中间确认。

## When NOT to Use
- Single-file fixes or trivial changes (use individual step skills directly)
- Exploratory tasks with no implementation (brainstorming alone is sufficient)
- When the user explicitly wants fine-grained control over each step

## CodeGraph 代码知识图谱 — 代码理解横切（Step 1 / 8 / 9 强制）

> 工具：MCP `codegraph_explore`（唯一入口 — 自然语言问题或符号/文件名查询 → 一次返回相关符号**逐文件原文源码** + 符号间调用路径）。CLI 补充：`codegraph context <task>` / `query` / `callers` / `callees` / `impact` / `affected`。索引缺失时先 `codegraph init`，增量维护用 `codegraph sync`，核验用 `codegraph status`。

**进入 Step 1、Step 8、Step 9 时，若目标区域存在 `.codegraph/` 索引，先用 codegraph 建立代码理解（兜底：索引不可用时才退化 Grep/Ripgrep）：**

- **Step 1（Brainstorming）** — 设计前用 `codegraph_explore` 查询任务涉及的符号/文件，摸清既有实现、调用链与依赖面；返回的源码视为**已读取**（勿重复 Read 同文件）
- **Step 8（Build）** — 每个增量切片动手前，对切片涉及的符号执行 explore（先理解后实现）；跨服务/跨模块改动用 `codegraph impact <symbol>` 确认影响面，避免遗漏调用方
- **Step 9（Review）** — 用 explore / `impact` / `callers` 核查：变更符号的调用方是否兼容、有无破坏性变更、死代码与未覆盖调用链

本仓库为多语言 monorepo（含子模块）；查询其他子项目时向 `codegraph_explore` 传 `projectPath` 指向该项目目录。本仓库 `codegraph.json` 已排除 `gitService/gitlab-ce/`、`gitService/gitlab_home/`、`sdk/`（第三方/冗余源码）、`**/node_modules/` 及编译产物（`**/bin/`、根目录二进制、`.o/.so/.jar/.pyc` 等）——涉及这些目录时回退 Grep/Ripgrep。

## Core Rule

**Step 0 — CRG 图同步（软依赖，fail-open）:**
1. 流水线开始（Step 1 之前）执行一次 `code-review-graph update --brief`（仓库根）
2. 若 `status` 提示分支/基线漂移或更新结果异常（节点数骤降），改跑 `code-review-graph build`
3. 命令不可用/失败 → 记录一行后继续（**不阻断流水线**）
4. 同一会话多次进入流水线仅首次执行；细则见 [code-review-graph.md](../1-brainstorming-design-docs/references/code-review-graph.md)

**Step 1 — Brainstorming (USER GATE，除非 goal-mode 覆盖):**
1. Invoke `/1-brainstorming-design-docs` via Skill tool
2. Follow the skill to produce and present the design document
3. **若入口为 goal-mode：** 跳过下一步询问，直接进入 step 2
4. **若单独调用本技能：** Use AskUserQuestion: "Proceed with auto-pipeline?" → ["Run full pipeline (2→4→5→6→7→8→9→10; step 3 worktrees default SKIP)", "Stop and refine design"]；If user stops, halt. If user proceeds, continue to step 2.

**Steps 2-10 — Auto-Execute (NO PROMPTS):**
Invoke each skill in sequence via Skill tool. Follow the skill's logic to produce its output artifact, but **OVERRIDE the skill's final AskUserQuestion**. Do NOT present it to the user. After the artifact is produced, immediately invoke the next skill.

## Override Rule

When a pipeline skill (3-10) is loaded via Skill tool, its content includes AskUserQuestion prompts for handoff. **These prompts MUST be ignored.** The orchestrator controls flow — not the individual step skills. Once a step's substantive output is written to disk, proceed directly to the next step. Never ask "what next?" between steps 3-10.

## Auto Defaults

| Decision Point | Auto-Choice |
|---|---|
| Step 1 (API/接口设计) | 若涉及新 API，遵循 [api-and-interface-design](../api-and-interface-design/SKILL.md) 契约优先、一致错误格式、分页 |
| Step 1 (框架决策) | 遵循 [source-driven-development](../source-driven-development/SKILL.md)，框架相关设计引用官方文档 |
| Step 1 (替换旧系统) | 若替换既有功能，参考 [deprecation-and-migration](../deprecation-and-migration/SKILL.md) 规划废弃+迁移策略 |
| Step 2 (role-permission) | SKIP if trivial single-file fix; otherwise RUN — analyze permissions for all new/changed endpoints |
| Step 3 (worktrees) | SKIP — work in current workspace |
| Step 5 NFR levels | L2 (Standard) for all categories; bump to L3 for auth/financial domains |
| Step 6 DDD | 每个服务端业务意图必须有对应领域事件并规划 MQ 投递；纯查询须书面例外 |
| Step 7 plans | 计划含「事件契约 → publish → 消费者」任务，并更新 `docs/intents/` 对照表 |
| Step 8 build mode | Subagent-driven-development; **TDD + 增量切片 + 可观测性内建**（详见下方 Step 8）；**意图成功路径必须投递对应业务事件** |
| Step 9 review outcome | **五轴审查** + **安全审计** + **Log Audit** + **Intent→Event 投递审计**（详见下方 Step 9）；Auto-fix critical issues once, re-review; if still critical → stop |
| Step 10 ship method | **Pre-launch checklist** + **Feature flag** + **Rollback plan**（详见下方 Step 10）+ **直接合入 `main` 并 `git push origin main`**（禁止默认 `gh pr create`；见 `/10-ship`） |

## Sequence

```
/1-brainstorming-design-docs → [USER GATE†] → /2-role-permission → [/3-worktrees SKIP] → /4-value-stream → /5-nfr → /6-ddd → /7-plans → /8-build → /9-review → /10-ship
```

† goal-mode 入口跳过 USER GATE。旧同号目录（`2-worktrees`、`3-plans`、`4-build`、`5-tdd`、`6-tdd`、`6-review`、`7-ship`）仅为重定向，禁止当权威步骤加载。

## Step 1 — Brainstorming 补充约束

除 `/1-brainstorming-design-docs` 本身的设计产出外，本步骤需额外执行以下检查：

### 1a. API 契约审查
若设计涉及新增/修改 API 端点，遵循 [api-and-interface-design](../api-and-interface-design/SKILL.md)：
- [ ] 契约先于实现（类型签名即 spec）
- [ ] 错误格式一致（`{ error: { code, message, details } }`）
- [ ] 列表端点有分页
- [ ] 新增字段可选（向后兼容）
- [ ] URL 遵循 `/api/${serviceName}/${funcName}/key/value/...` 模式

### 1b. 框架决策溯源
涉及 Go/Django/Vue 框架选型时，遵循 [source-driven-development](../source-driven-development/SKILL.md)：从 go.mod / pyproject.toml / package.json 检测版本 → 拉取对应版本文档 → 设计基于文档而非记忆。

### 1c. 废弃规划
若设计替换既有系统/API，遵循 [deprecation-and-migration](../deprecation-and-migration/SKILL.md)：规划 advisory vs compulsory 废弃、迁移路径、Expand/Contract 数据库策略。

### 1d. 代码基线理解（CodeGraph 横切）
设计前对任务涉及符号执行 `codegraph_explore`（详见上方 CodeGraph 横切段），设计必须基于既有代码现状而非假设。

## Step 8 — Build（内建质量门禁）

Build 阶段执行实现代码，内建以下硬门禁：

### 8a. TDD 测例先行（硬门禁）
- Red → Green → Refactor 循环，**禁止先写业务再补测**
- 每个增量切片对应一组测试

### 8b. 增量切片纪律（原则内建）
- 垂直切片优先（穿越全栈的完整功能路径）
- 每个切片 <100 行、独立可测试、独立可提交
- 保持项目在每一切片后编译通过+测试全绿

### 8c. 框架代码溯源（硬门禁）
写 Go/Django/Vue 框架相关代码前，遵循 [source-driven-development](../source-driven-development/SKILL.md) DETECT→FETCH→IMPLEMENT→CITE 流程。**禁止凭记忆写 API 调用，必须验证官方文档。**

### 8d. Git 原子提交（硬门禁）
每切片提交遵循 [git-workflow-and-versioning](../git-workflow-and-versioning/SKILL.md)：
- 提交类型前缀（feat/fix/refactor/test/docs/chore）
- 解释 why 而非 what
- 保存点模式：每切片一提交

### 8e. 可观测性内建（硬门禁）
遵循 [observability-and-instrumentation](../observability-and-instrumentation/SKILL.md)：
- 结构化日志（JSON、稳定 event name、correlation ID 贯穿）
- 禁止 `console.log` / `fmt.Println` / `print()` 用于生产日志
- 每个新 API 端点+外部依赖有 RED 指标（Rate/Errors/Duration 直方图）
- 跨服务调用传播 trace context
- 禁止日志含密钥/token/PII
- Build 完成前 staging 验证：结构化日志确认、指标出现、trace 不中断

### 8f. 前端浏览器验证
若涉及前端变更，遵循 [webapp-testing](../webapp-testing/SKILL.md)，通过 Playwright 脚本或 Chrome DevTools MCP 验证浏览器行为、截图对比、console 无错误。

### 8g. 代码理解先行（CodeGraph 横切）
每个切片动手前对涉及符号执行 `codegraph_explore`（返回源码视为已读取）；跨模块改动先 `codegraph impact <symbol>` 圈定影响面（详见上方 CodeGraph 横切段）。

## Step 9 — Review（五轴审查 + 安全审计）

### 9a. 五轴审查框架
| 轴 | 检查要点 |
|---|---|
| **Correctness** | 代码符合 spec？边界条件？错误路径？测试验证正确行为？ |
| **Readability** | 命名清晰？控制流直观？死代码？"聪明"技巧应简化？ |
| **Architecture** | 遵循既有模式？循环依赖？抽象物有所值？功能泄漏到共享模块？ |
| **Security** | 详见 [security-and-hardening](../security-and-hardening/SKILL.md) |
| **Performance** | N+1 查询？无界循环？缺失分页？热路径大对象？ |

### 9b. 安全审计清单（必检）
引用 [security-and-hardening](../security-and-hardening/SKILL.md) 完整审查清单：
- [ ] 无密钥/token 在代码或日志中
- [ ] 所有用户输入在系统边界验证
- [ ] SQL 查询参数化（无字符串拼接）
- [ ] 输出编码防 XSS
- [ ] 认证+授权覆盖每个受保护端点
- [ ] 安全头配置（CSP/HSTS/X-Frame-Options）
- [ ] CORS 限制到已知 origin（无 `*`）
- [ ] 认证端点有速率限制
- [ ] 错误响应不暴露内部细节
- [ ] 依赖审计无 reachable critical/high 漏洞（Go: `govulncheck`; Python: `pip-audit`）
- [ ] 服务端 URL 获取有 allowlist（防 SSRF）

### 9c. 简化与加固（后审查清理）
审查完成后运行 [simplify-and-harden](../simplify-and-harden/SKILL.md) 三 pass：
- **Simplify**: 死代码、命名、控制流收紧（仅限本次 diff 范围内的文件）
- **Harden**: 输入验证缺口、注入向量、认证缺失、密钥泄露
- **Document**: 最多 5 条决策注释

### 9d. 审查输出
- 所有发现按 **Critical（阻断合并）> Required（必须修）> Nit（可选）** 分级
- Critical + Required：auto-fix once → re-review → 若仍有 → STOP
- 输出审查报告含：发现数、严重度分布、修复状态

### 9e. 影响面核查（CodeGraph 横切）
用 `codegraph_explore` / `codegraph impact` / `callers` 核查变更符号：调用方兼容性、破坏性变更、死代码、遗漏调用链；未覆盖的调用方必须补测或标注（详见上方 CodeGraph 横切段）。

## Step 10 — Ship（安全交付）

### 10a. Pre-Launch Checklist（自动验证）
- [ ] 所有测试通过（单元/集成/E2E）
- [ ] 构建成功无警告
- [ ] Lint + 类型检查通过
- [ ] 安全审计清单（Step 9b）全部 ✅
- [ ] 可观测性验证（Step 8e）全部 ✅
- [ ] 无调试语句残留
- [ ] 数据库迁移已准备（含回滚脚本）
- [ ] CHANGELOG 已更新（遵循 [git-workflow-and-versioning](../git-workflow-and-versioning/SKILL.md) 的 changelog 规范）
- [ ] ADR 已写（如有架构决策）

### 10b. Feature Flag Strategy
- 大功能放在 feature flag 后面：`deploy off → team/beta → 5% → 25% → 100% → clean up`
- 每个 flag 有 owner 和过期日（全量后 2 周内清理）

### 10c. Rollback Plan
每次部署前必须有回滚方案（详见 [ci-cd-and-automation](../ci-cd-and-automation/SKILL.md) 灰度发布阈值）：
- 触发条件（error rate >2x baseline、P95 latency >50%、数据完整性）
- 回滚步骤（feature flag 关闭 <1分钟 / 代码回滚 <5分钟 / DB 回滚 <15分钟）
- 通知对象

### 10d. CI/CD 集成
遵循 [ci-cd-and-automation](../ci-cd-and-automation/SKILL.md)：
- 质量门禁全部通过才允许合入 `main`（lint→test→build→security audit；本地 pre-commit / commit-msg 不可跳过）
- 使用 Docker service containers 运行集成测试
- 交付说明（commit message / 会话收尾）含：变更摘要、测试结果、回滚方案要点

### 10e. 直接合入 main 并推送（强制）
- **禁止**默认 `gh pr create` /「只开 PR 不 merge」
- 有 `feat/*`：`checkout main` → `merge feat/<name>`（或等价拣选）→ **`git push origin main`**
- 已在 `main` 上开发：提交后直接 **`git push origin main`**
- 多仓：先推子仓 `origin/main`，再推 meta（规则 32）
- 推送成功后跑分支/worktree 清理（约束 21）；若同主题仍有开放 PR，`gh pr close` 并注明已合入 main
- 打版本标签遵循 [git-workflow-and-versioning](../git-workflow-and-versioning/SKILL.md) 语义化版本规范

## Error Handling

| Situation | Action |
|---|---|
| Step 2-7 fails to produce output file | Stop, report the error, ask user |
| Step 8 test failure | Auto-fix up to 3 attempts per failing test; if still failing → stop and report |
| Step 9 critical or important issues | Auto-fix once, re-review; if still critical/important → stop and report |
| Step 10 ship fails | Stop, report error, ask user |
| Any step produces empty/missing artifact | Stop, report which file is missing |

## Progress Reporting

After each step completes, report one line:
```
[0-auto-flow] Step 2/10 (role-permission) done → proceeding to step 4 (value-stream)
[0-auto-flow] Step 4/10 (value-stream) done → proceeding to step 5 (NFR)
[0-auto-flow] Step 5/10 (NFR) done → proceeding to step 6 (DDD)
...
[0-auto-flow] All steps complete.
```

## Artifact Chain

Each step consumes the previous step's output. Track paths across steps:

```
docs/superpowers/specs/<date>-<topic>-design.md          ← Step 1
docs/superpowers/specs/<date>-<topic>-permission-analysis.md ← Step 2
docs/superpowers/plans/<date>-<topic>-value-stream.md    ← Step 4
docs/superpowers/plans/<date>-<topic>-nfr-clarification.md ← Step 5
{module}/domain/{entities,value_objects,repositories,...}  ← Step 6
docs/superpowers/plans/<date>-<topic>-plan.md            ← Step 7
(implementation files on disk)                            ← Step 8
(review report + security audit + simplify-harden output) ← Step 9
(main merged + origin pushed + launch checklist + rollback plan) ← Step 10
```

## Red Flags

- Don't skip a step because "it's simple" — each step produces artifacts consumed by the next
- Don't reorder the sequence
- Don't ask "should I continue?" or present AskUserQuestion between steps 3-9. **Exception (constraint 53 / ADR-0021):** keeping or restoring superseded logic requires in-session human approval; deleting old paths does not.
- Don't enter PlanMode between steps — the plan is auto-generated at step 6
- If the user interrupts mid-pipeline, stop and report current step + completed artifacts
- Don't start implementation before design approval at step 1 gate
- Don't commit code before step 8 (build) — prior steps only produce design artifacts
- Don't ship without pre-launch checklist (Step 10a) and rollback plan (Step 10c)
- Don't skip security audit (Step 9b) — even "simple" changes can introduce vulnerabilities
- Don't merge code with console.log/print() debugging statements
- Don't ship features without observability instrumentation (Step 8e)
- Don't write framework-specific code without verifying official docs (Step 8c — [source-driven-development](../source-driven-development/SKILL.md))
- Don't create API endpoints without contract-first design (Step 1a — [api-and-interface-design](../api-and-interface-design/SKILL.md))
- Don't default to `gh pr create` or leave unmerged PRs — Step 10 must merge to `main` and `git push origin main`