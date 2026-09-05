# 价值流：导航栏任务搜索过滤

- **日期**: 2026-07-20
- **设计**: `docs/superpowers/specs/2026-07-20-navbar-task-search-filter-design.md`

## 最小价值增量

用户在任意租户页导航栏输入关键词 → 看到匹配任务 → 一键进入工作面板或指定任务。

## 端到端步骤

1. 用户聚焦导航搜索框并输入 q
2. 前端防抖；解析成员名 → `assignee_ids`；调用 `tasks/search`
3. 服务端按 title/id/owner/assignee + ACL 返回 results
4. 下拉展示；用户选「工作面板」或「打开任务」
5. 路由跳转；若带 `task_id`，WorkPanel 打开详情

## 测试点（须有对应用例）

| ID | 测试点 | 类型 | 用例落点 |
|----|--------|------|----------|
| TP-1 | q 匹配 title | Go unit | `TestSearchTasks` 扩展 |
| TP-2 | q 匹配 id 后缀/片段 | Go unit | `TestSearchTasks` |
| TP-3 | q/assignee_ids 匹配负责人 | Go unit | `TestSearchTasksByAssignee` |
| TP-4 | 无 workspace 权限不返回 | Go unit | 既有/补充 |
| TP-5 | Navbar 防抖搜索与下拉渲染 | Vitest | `NavbarTaskSearch.test.js` |
| TP-6 | 跳转 work-panel / task_id | Vitest | 导航 helper 单测 |
| TP-7 | WorkPanel 深链打开详情 | Vitest | composable/小测 |

## 价值流图

更新 `docs/flows/value-stream-test-integration.wsd`：在工作面板泳道增加「导航栏任务搜索」节点与 TP-1…TP-7。
