# `.claude/skills/` — Agent 技能目录

权威位置与维护约定见上级 [`.claude/README.md`](../README.md)。  
交付流水线 **编号 SSOT**：[`.ai/11_ai_development/03_superpowers_workflow.md`](../../.ai/11_ai_development/03_superpowers_workflow.md)。

## 权威十步（实现类任务）

| 步 | 目录 | 说明 |
|----|------|------|
| 0 | `0-auto-flow` | 编排全流程（非业务步） |
| 1 | `1-brainstorming-design-docs` | 设计；别名 `1-brainstorming` |
| 2 | `2-role-permission` | 角色权限 |
| 3 | `3-worktrees` | Worktree；全自动默认可 SKIP |
| 4 | `4-value-stream` | 价值流 |
| 5 | `5-nfr` | NFR 澄清（含路径分片键 + 幂等性硬门禁） |
| 6 | `6-ddd` | 领域建模 |
| 7 | `7-plans` | 实施计划 |
| 8 | `8-build` | 构建 + TDD |
| 9 | `9-review` | 审查 |
| 10 | `10-ship` | 交付 |

默认序列（跳过 Step 3）：

```
1 → 2 → 4 → 5 → 6 → 7 → 8 → 9 → 10
```

持久目标入口：`goal-mode`（可覆盖 Step 1 用户闸门，见该技能）。

## 废弃同号目录（薄重定向，勿当权威步）

| 旧目录 | 转去 |
|--------|------|
| `2-worktrees` | `3-worktrees` |
| `3-plans` | `7-plans` |
| `4-build` | `8-build` |
| `5-tdd` / `6-tdd` | `8-build` |
| `6-review` | `9-review` |
| `7-ship` | `10-ship` |

## 常用横切技能（非步号）

| 目录 | 用途 |
|------|------|
| `archimate` | ArchiMate / 架构制品 |
| `codegraph`（MCP） | 代码知识图谱 — `codegraph_explore` 一步返回符号源码+调用路径（goal-mode / 0-auto-flow Step 1/8/9 使用；索引在 `.codegraph/`，本仓库符号链接至真实磁盘） |
| `simplify-and-harden` | 完成后简化加固 |
| `logging-audit` | 日志审计（build/review 引用） |
| `1-brainstorming-design-docs/references/traceid-log-first-diagnosis.md` | 有 `data-traceId` 时**先 Loki 重建路径**（排障横切；pua/diagnose/webapp-testing 共用） |
| `self-improvement` | 经验沉淀 → `.learnings/` |
| `intent-framed-agent` / `plan-interview` | 意图框定与对齐 |
| `ddd-driven-design` | DDD 执行约束（配合 `6-ddd`） |
| `ai-coding-discipline` / `ddia-principles` / `software-design-philosophy-skill` | 编码与设计原则 |
| `*-design-guidelines` / `frontend-design` / `web-design-guidelines` | 平台/Web UI（按任务加载，勿全局常驻） |
| `design-md` | 产品视觉 SSOT：仓库根 `DESIGN.md`（Stitch 格式）。[awesome-design-md](https://github.com/VoltAgent/awesome-design-md) 只作可本地化模板，禁止整份覆盖品牌文件。taskFE 不以 `frontend-design` 覆盖本基线 |
| `template` | 脚手架占位 — **勿触发** |

编写新技能：`.ai/11_ai_development/02_agent_skills_authoring.md`。
