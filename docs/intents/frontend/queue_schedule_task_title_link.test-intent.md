# 测试意图：自动调度安排排队任务标题可点击进入任务详情

## 对应用户意图

`queue_schedule_task_title_link.intent.md`

## 测试目标

排队任务标题必须是可复制、可中键打开的真实链接，指向任务详情页。

## 测试分层

- 组件单测：`taskFE/app/src/views/WorkspaceQueueSchedule.test.js`
- 浏览器核对：打开 `/tenant/:tenant/queue-schedule/?workspace_id=…` 点击标题

## 用例矩阵

| ID | 场景 | 期望 | 自动化 |
|----|------|------|--------|
| TP-1 | 快照含 `task_id`+`title` | 标题为 `a`，href=`/tenant/t1/workspace/ws9/task-detail/ta/`，文本含标题 | `WorkspaceQueueSchedule.test.js` |
| TP-2 | 标题为空 | 链接文本为「（无标题）」，href 仍按 `task_id` 指向详情 | `WorkspaceQueueSchedule.test.js` |
| TP-3 | 成员缺少 `task_id` | 标题为 `span`，无 `href="#"` | `WorkspaceQueueSchedule.test.js` |

## 数据与环境

Vitest + jsdom；mock `apiFetch` 返回 queue-schedule 快照。无需 MQ。

## 通过标准

上表 TP-1～TP-3 全绿；浏览器中标题可左键进入任务详情。

## 价值流锚点

`docs/flows/value-stream-test-integration.wsd` → `WQSTL`  
`conf/value-stream.yaml` → `workspace-queue-schedule-task-title-link`
