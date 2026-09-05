# auto_run 挂载 @ 评论并回填 PR — 设计

- **日期**: 2026-07-22
- **作者**: goal-mode / 0-auto-flow（自动采纳）
- **状态**: approved（goal-mode 跳过 USER GATE）
- **页面**: `…/task-detail/task_13757169132576941867/`
- **基线**: `2026-07-13-auto-run-first-instruction-and-delivery-design.md`、`2026-07-14-container-image-at-mention-design.md`
- **python_api_approval**: n/a（不新增 Python 接口）

## 1. 问题

自动运行（`auto_run=true`）静默执行首指令并交付 PR，**不出现在任务评论时间线**；用户无法在 Feed 中看到「这次自动运行」及其产物 PR。

期望：

1. 自动运行触发时创建一条 `@镜像` 人类评论；
2. 将本次自动运行挂载到该评论对应的 `container_agent` 回复节点；
3. 交付创建 PR 后，把 PR 链接回填到该 Agent 回复（`assistant_response`）。

## 2. 方案（采纳）

| 项 | 做法 |
|----|------|
| 创建 @ 评论 | TTS `triggerTaskAutoRun` 成功启服前调用 `ensureAutoRunAtComment`：插入人类评论（`mentions[]`）+ `notifyContainerAgentPending`；**不**发布 `TASK_COMMENT_IMAGE_MENTIONED`（避免二次 start-vm） |
| ContextPack | `at_mention_run.source = "auto_run"`；trigger content = title+description |
| Kickoff | `source===auto_run` 时走 auto_run 首指令（非 at_mention job），并把 `agent_comment_id` / `parent_comment_id` 写入 job |
| Job 挂载 | `jobsRuntime.createJob` 持久化 `mounted_agent_comment_id`、`mounted_parent_comment_id` |
| PR 回填 | `runAutoRunDelivery` 成功后从 `pushResult.repos[].pr.html_url` 提取链接，`POST …/container-agent-comments/{id}/complete`（`X-Access-Token`）写入 `assistant_response` |
| 幂等 | 同任务已存在 `source=auto_run` 且非终态 Agent → 复用；首指令/交付标志不变 |

## 3. 成功标准

| # | 标准 | 验证 |
|---|------|------|
| S1 | auto_run 启服前有人类 @ 评论 + pending Agent | TTS Go 单测 |
| S2 | 不发布 `TASK_COMMENT_IMAGE_MENTIONED` | TTS 单测 |
| S3 | kickoff：`source=auto_run` → auto_run job 且带 mount ids | Node 单测 |
| S4 | 交付成功后 Agent `assistant_response` 含 PR URL | Node 单测（mock complete） |
| S5 | 意图/测试意图与设计同步 | docs |

## 4. 🕸️ Code Review Graph 分析

- **graph_status**: unavailable（MCP `code-review-graph` 未注册到当前会话；CLI 未用全量图）
- **探测路径**: MCP fail → 定向探索（explore agent + rg/Read）
- **触点**: TTS `auto_run`/`aic_client`；onlineServiceJS kickoff/delivery；taskAIComment complete
- **爆炸半径**: 评论创建与 kickoff 优先级；禁止误触二次 start-vm
- **与 Archimate 对照**: 未对照（无新 Application_Component；复用既有评论/Agent/auto_run 交付）
- **对本次设计的影响**: 复用 @mention 数据模型与 complete API，仅加 `source=auto_run` 分流

## 5. 架构交付物

本变更不引入新服务边界，**不**新增 `docs/architecture/` PlantUML/ArchiMate/Mermaid 视图。
