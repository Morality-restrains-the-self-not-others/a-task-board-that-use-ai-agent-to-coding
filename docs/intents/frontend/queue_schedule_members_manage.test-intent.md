# 测试意图：自动调度安排排队任务卡可加入/离开队列

## 测试目标

排队任务卡能搜索入队、未启用阻断、确认后出队，且写路径带幂等键、错误带 traceId、刷新会 GET。

## 测试分层

| 层 | 文件 | 覆盖 |
|----|------|------|
| 单元 | `useWorkspaceQueueSchedule.test.js` | PATCH + 刷新快照 |
| 组件 | `QueueMembersCard.test.js` | 搜索/入队/出队/门闩/traceId |
| 页面 | `WorkspaceQueueSchedule.test.js` | 刷新 GET、卡片入口 |

## 用例矩阵

| ID | 步骤 | 期望 |
|----|------|------|
| TP-1 | 未启用点加入 | 无 PATCH；提示调度设置 |
| TP-2 | 已启用搜索后加入 | PATCH true + Idempotency-Key + GET 快照 |
| TP-3 | 已在队任务不出现在搜索结果 | 结果不含该 task_id |
| TP-4 | 离开确认 | PATCH false |
| TP-5 | 离开取消 | 无 PATCH |
| TP-6 | PATCH 失败 | 错误节点 data-traceId |
| TP-7 | 点刷新 | GET queue-schedule |

## 数据与环境

jsdom + mock `window.apiFetch` / `modalService`。不启真实后端。

## 通过标准

上表全部绿；既有标题链接测例仍绿。
