# 设计：将 code-review-graph 融合进 /1-brainstorming-design-docs

- **日期**: 2026-07-21
- **状态**: approved
- **作者**: claude
- **硬度**: 软依赖（MCP/CLI 不可用则记录后继续）
- **融合深度**: 3 — 技能 + MCP + ignore + Cursor hook + daemon 文档 + CI Action
- **架构制品**: 不更新 `docs/architecture/`（无运行时服务拓扑变更）

## 1. 问题与目标

头脑风暴已强制读取 Archimate 与（有 traceId 时）Loki，但缺少**代码结构图**（调用/导入/社区/爆炸半径）作为设计输入，大仓扫读成本高、落点易漏。

目标：在 Step 1 中以 [code-review-graph](https://github.com/tirth8205/code-review-graph) 提供 token 友好的结构上下文，并完成仓库级可安装、可刷新、可 CI 评论的工程化。

## 2. 行为设计

### 2.1 技能时机

Archimate current 读完之后、方案定稿之前；若存在 traceId，**先**日志节再 CRG。

### 2.2 调用链

见 `.claude/skills/1-brainstorming-design-docs/references/code-review-graph.md`。

### 2.3 设计文档

强制章节 `## 🕸️ Code Review Graph 分析`（非代码任务可用 `skipped_non_code`）。

## 3. 工程化清单

| # | 交付物 | 说明 |
|---|--------|------|
| 1 | `references/code-review-graph.md` | 细则 SSOT |
| 2 | `1-brainstorming-design-docs/SKILL.md` | 横切节 + 加载指针 |
| 3 | `.mcp.json` | `code-review-graph` server |
| 4 | `.gitignore` / `.code-review-graphignore` | 图数据与索引排除 |
| 5 | `.cursor/hooks.json` + `crg-update.sh` | afterFileEdit 增量更新，fail-open |
| 6 | `.github/workflows/code-review-graph.yml` | PR sticky comment；`fail-on-risk` 默认关闭 |
| 7 | `.ai/11_ai_development/03_*` / `02_*` | 交叉引用 |

Daemon：文档化 `crg-daemon`，不强制本机常驻进程进仓。

## 4. 非目标

- 不以 CRG 替代 Archimate 三类伴生制品
- 不默认云 embedding
- 不以无 CRG 阻止设计定稿
- 不把 `fail-on-risk` 设为合并门禁（可后续 OPT）

## 5. 业务意图 → 事件

| 业务意图 | 事件 | 例外理由 |
|---------|------|---------|
| （无） | — | 纯 Agent 工具链 / 文档与 CI 配置，无服务端状态变更 |

## 6. 价值流影响

无 `value-stream.yaml` 业务步骤变更；不新增用户可见价值流。

## 7. 验收

- [x] 技能与 reference 已落地；触及代码时须写 `🕸️` 节或合法 skip/unavailable
- [x] `.mcp.json` 含 CRG（`.cursor/crg-mcp-serve.sh`）；需本机安装包并重启 Cursor 后发现工具
- [x] `.code-review-graph/` 已写入根 `.gitignore`（`git check-ignore` 通过）
- [x] `.github/workflows/code-review-graph.yml` 已添加（`fail-on-risk: none`）
- [x] `.cursor/hooks/crg-update.sh` fail-open（缺二进制时 exit 0）
- [ ] 本机 `uvx/pipx install` + `build` + Cursor 重启后实机冒烟（受环境代理限制，交付时未完成）

## 8. 批准记录

- **总体设计**: approved（用户 2026-07-21 选「批准本设计并开始落地」）
- **融合深度**: 3
- **硬度**: A 软依赖

## 后续

全流程融合见 `2026-07-21-code-review-graph-goal-pipeline-integration-design.md`（`/goal` / `0-auto-flow` 十步矩阵）。
