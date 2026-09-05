# 自动调度安排 · 排队任务卡可管理 — Review

- 日期：2026-08-27
- 范围：本次 diff（queue-schedule 排队任务卡管理）

## 五轴审查

| 轴 | 结论 |
|----|------|
| Correctness | 搜索过滤 workspace + 已入队；未启用不 PATCH；离开需 confirm；刷新走 reload。单测 18/18 绿。 |
| Readability | 卡片抽出 QueueMembersCard；composable 仅负责 GET/PUT/PATCH。 |
| Architecture | 无新服务/表/API；复用 todos PATCH 与 search。无架构变更。 |
| Security | tid+wid+taskId 路径；无密钥；文本插值防 XSS；modal 非 window.confirm。 |
| Performance | 搜索 debounce 250ms、limit 20；无轮询。 |

## 安全审计清单

- [x] 无密钥在代码/日志（仅 task_id）
- [x] 用户输入经既有 search API；PATCH 仅 task id
- [x] 无 SQL
- [x] 输出文本插值
- [x] 鉴权沿用既有 PATCH
- [x] 错误不暴露内部栈
- [x] 无 SSRF（相对路径 apiFetch）

## CodeGraph

MCP `codegraph_explore` 不可用。调用链：QueueMembersCard → patchQueuedAutoRun → PATCH todos → enqueue/dequeue（既有）。

## simplify-and-harden

- Simplify：未另抽 search composable（Rule 0）。
- Harden：clickGuard + Idempotency-Key；search/join 错误 data-traceId。
- Document：卡片 Anti-Replay-OK 注释已写。

## 发现

无 Critical / Required。Nit：页面测例 WorkspaceSwitcher 仍打真实 fetch 噪音（既有）。

## 测试

- 新增 QueueMembersCard.test.js 5
- useWorkspaceQueueSchedule.test.js +2
- WorkspaceQueueSchedule.test.js +1
- 合计本次相关 18 passed / 0 failed
