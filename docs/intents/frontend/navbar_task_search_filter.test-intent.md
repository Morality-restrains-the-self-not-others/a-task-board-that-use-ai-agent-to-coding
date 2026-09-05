# 测试意图：工作面板任务搜索过滤

## 对应功能意图

`navbar_task_search_filter.intent.md`

## 用例映射

| AC | 测试 | 类型 |
|----|------|------|
| AC1 | `WorkPanelHeader` 单行工具栏含搜索；`Navbar.ui` 不含搜索 | unit |
| AC1–2 | `NavbarTaskSearch` Vitest + Go search 扩展 | unit |
| AC3–4 | 跳转 URL helper Vitest；「打开任务」为 `a[href]` 指向 task-detail | unit |
| AC5 | Go search ACL / workspace 门禁 | unit |
| AC6 | 失败路径 data-traceId（沿用 showRequestError） | unit/手工 |
| 编号误粘贴 | `normalizeSearchText` / `TestSearchTasksAcceptsServicePrefixedTaskID`：`task-task_<id>` 可命中 | unit |
| 评论容器名 | `normalizeSearchText` / `TestNormalizeTaskSearchQuery` / `TestSearchTasksAcceptsCommentContainerName`：`task_<id>_cmt_<cmt>`、`task-task_<id>_cmt_<cmt>`、`task_task_<id>_cmt_<cmt>` 命中任务 | unit |
| 评论 id | `TestSearchTasksAcceptsCommentID`：`cmt_<id>` 命中所属任务 | unit |

## 变更记录

| 日期 | 内容 |
|------|------|
| 2026-07-20 | 初版 |
| 2026-07-22 | 增补 `task-task_` 前缀归一化用例 |
| 2026-08-16 | 「打开任务」href 指向 task-detail，回归：同 work-panel URL 不再被当成跳转目标 |
| 2026-08-17 | AC1 改为工作面板标题行挂载；增补 Navbar 不再挂载的回归测例 |
| 2026-08-17 | 增补评论容器名 / 评论 id 搜索用例 |
| 2026-09-02 | AC1 改为单行工具栏挂载（与机器摘要同一行） |
