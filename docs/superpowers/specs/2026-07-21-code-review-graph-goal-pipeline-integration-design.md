# 设计：将 code-review-graph 融合进 `/goal` 全流程

- **日期**: 2026-07-21
- **状态**: approved（goal-mode 自动批准）
- **前置**: `2026-07-21-code-review-graph-brainstorming-integration-design.md`（Step 1 + 工程化）
- **硬度**: 软依赖（全程）
- **架构制品**: 不更新（Agent 编排 / 技能文档）

## 1. 目标（SMART）

| # | 标准 | 完成 |
|---|------|------|
| 1 | `references/code-review-graph.md` 成为十步 × CRG 矩阵 SSOT | ✅ |
| 2 | `goal-mode` + `0-auto-flow` 显式编排 CRG | ✅ |
| 3 | Steps 2/4/5/6/7/8/9/10 技能含 CRG 指针与强度 | ✅ |
| 4 | `03_superpowers_workflow.md` 十步表标注 CRG | ✅ |
| 5 | README / logging-audit 交叉引用 | ✅ |
| 6 | 无运行时拓扑变更；不新增 Python API | ✅ |

## 2. 自主决策

| 决策 | 选择 | 理由 |
|------|------|------|
| 硬度 | 全流程软依赖 | 与 Step1 一致；不因缺包阻断 `/goal` |
| SSOT | 单 reference，各步短指针 | 避免工具表漂移 |
| Step 3 | 跳过 CRG | worktree 无结构分析价值 |
| Step 4/5 | 建议非强制 | 价值流/NFR 以业务为主，图为辅助 |
| Step 2/6/7/8/9/10 | 必须尝试 | 权限边、BC、任务、构建上下文、审查、交付门禁价值高 |

## 3. 业务意图 → 事件

| 意图 | 事件 | 例外 |
|------|------|------|
| （无） | — | 纯技能/文档编排，无服务端状态变更 |

## 4. 验收命令

```bash
rg -n "code-review-graph|Code Review Graph" \
  .claude/skills/goal-mode/SKILL.md \
  .claude/skills/0-auto-flow/SKILL.md \
  .claude/skills/{2-role-permission,4-value-stream,5-nfr,6-ddd,7-plans,8-build,9-review,10-ship}/SKILL.md \
  .ai/11_ai_development/03_superpowers_workflow.md
python3 db/scripts/ci/check_claude_skill_metadata.py
```
