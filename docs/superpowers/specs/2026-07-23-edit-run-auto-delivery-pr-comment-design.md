# 「改后执行」完成后自动交付 PR 并回填评论 — 设计

- **日期**: 2026-07-23
- **作者**: goal-mode / 0-auto-flow（自动采纳）
- **状态**: approved（goal-mode 跳过 USER GATE）
- **页面**: `…/task-detail/…` 评论内 ztree「改后执行」
- **基线**: `2026-07-13-auto-run-first-instruction-and-delivery-design.md`、`2026-07-22-auto-run-at-comment-pr-backfill-design.md`
- **python_api_approval**: n/a（不新增 Python 接口）

## 1. 问题

ztree「改后执行」（`container-job-edit-run`）仅删除旧 job、创建并启动新 job；**完成后不会**自动 commit / 推送 / 建 PR，也不会把 PR 写回评论回复。用户须手工点「提交」「推送并创建PR」。

期望：指令执行成功结束后，自动提交并创建 PR，再把 PR 及相关内容写成该评论下的 container_agent 回复。

## 2. 方案（采纳）

| 项 | 做法 |
|----|------|
| 标记 | Gateway createJob 始终带 `edit_run_delivery: true`；可选 `mounted_parent_comment_id` / `installed_image_id` |
| 前端 | edit-run 请求附带当前执行评论 `parent_comment_id` + 任务 `installed_image_id`（`container_image_id`） |
| 交付 | job `completed` 且 `edit_run_delivery` → 复用 `runAutoRunDelivery`，`force: true`（不受 `auto_run_delivery.done` 阻挡）；幂等用 `edit_run_delivery.<jobId>.done` |
| Agent 回复 | 无 `mounted_agent_comment_id` 时，容器用 `X-Access-Token` 创建 pending Agent（扩展 AIC create 鉴权），再 `/complete` 写入输出摘要 + PR 链接 |
| 文案 | 「修改指令后执行已完成…」与 auto_run「自动运行已完成…」区分 |
| 失败/中断 | 不交付、不建 PR 回复 |

## 3. 成功标准

| # | 标准 | 验证 |
|---|------|------|
| S1 | edit-run createJob 带 `edit_run_delivery` | Gateway Go 单测 |
| S2 | completed + edit_run_delivery → commit/push/PR | Node 单测（mock push） |
| S3 | 不受既有 `auto_run_delivery.done` 阻挡 | Node 单测 |
| S4 | parent 下 Agent 回复含 PR URL | Node 单测（mock create+complete） |
| S5 | failed/interrupted 不交付 | Node 单测 |
| S6 | 意图/测试意图与设计同步 | docs |

## 4. 架构交付物

不引入新服务边界，**不**新增 `docs/architecture/` 三类视图。

## 5. 爆炸半径

- Gateway edit-run 请求体向后兼容（新字段可选）
- onlineServiceJS 完成钩子：与 `auto_run_first` 并列触发交付
- taskAIComment：public create 增加 Access-Token 鉴权路径（与 stream/complete 对齐）
