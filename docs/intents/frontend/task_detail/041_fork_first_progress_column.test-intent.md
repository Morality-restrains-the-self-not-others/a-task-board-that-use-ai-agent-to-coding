# 测试意图：Fork 后新任务落在进度第一列

## 对应意图
`041_fork_first_progress_column.intent.md`

## 测试目标

验证 Fork 创建 payload 使用工作区进度第一列，而不是源任务当前进度列。

## 测试分层

- 单元：`taskFE/app/src/composables/taskDetail/taskDetailEditing.test.js`
- E2E：`taskFE/tests/TaskDetail.fork-popup.playwright.test.js`

## 用例矩阵

| ID | 给定 | 当 | 则 |
|----|------|----|----|
| T1 | 源任务 `progress_column_id=col-wip`，选项第一列为 `col-todo` | `forkTask` | POST `progress_column_id === 'col-todo'` |
| T2 | 源任务在 `col-wip`，`progressStatusOptions=[]` 且无 fetch | `forkTask` | POST 不含源列 `col-wip`（省略或空） |
| T3 | Playwright：progress-system 两列，源任务在第二列 | 确认派生 | POST `progress_column_id` 为第一列 id |
| T4 | 选项为空但提供 `fetchProgressStatusOptions` 填入第一列 | `forkTask` | 先 fetch，POST 第一列 |

## 数据与环境

- Vitest node 环境；`apiFetch` mock。
- Playwright 纯 mock，无真实账号。

## 通过标准

```
cd taskFE/app && npx vitest run src/composables/taskDetail/taskDetailEditing.test.js
```

全绿。
