# [运行时] 工作面板任务搜索：粘贴评论容器名 `task_<id>_cmt_<cmt>` 搜不到

## 基本信息

- 版本：1.0.0
- 案例编号：FE-20260817-0095
- 录入日期：2026-08-17
- 最后更新：2026-08-17
- 录入人：Cursor Agent

## 现象

- 页面：工作面板标题行任务搜索（`input#navbar-task-search-input`）
- 从云控制台 / 容器名 / 日志粘贴 `task_877071722828820480_cmt_877071748669927424` →「无匹配任务」
- 输入任务 id `task_877071722828820480` 或后几位数字 → 能搜到同一任务

## 环境与上下文

- 前端：`NavbarTaskSearch.vue` → `GET /api/tasks/search/tenant_id/{tid}/?q=`
- 任务主键：`genID("task")` → `task_<digits>`
- 评论主键：`genID("cmt")` → `cmt_<digits>`
- 评论级容器 / ECS InstanceName：`{taskId}_{commentId}` → `task_<digits>_cmt_<digits>`

## 根因

1. 搜索对 `q` 做 `id LIKE %q%`（`_` 已转义为字面量）。容器名比任务 id 更长，且含 `_cmt_` 后缀，不是库内 `id` 子串 → 0 命中。
2. 既有归一化只处理 `task-task_<digits>` / `#`，未剥离 `_cmt_<commentId>`。
3. 评论 id 本身不在 `task_tasks` 上，单独粘贴 `cmt_<id>` 同样无法命中。

## 修复

1. **搜索归一化**（前后端）：`task_<digits>_cmt_*` / `task-task_<digits>_cmt_*` / `task_task_<digits>_cmt_*` → `task_<digits>`。
2. **评论表精确匹配**：`q` 为 `cmt_<id>` 或可从容器名抽出评论 id 时，`EXISTS task_comments.id=?`（仍受 workspace ACL 约束）。

## 验收

```bash
cd taskTaskService/src && go test -count=1 -run 'TestNormalizeTaskSearchQuery|TestCommentIDFromSearchQuery|TestSearchTasksAcceptsCommentContainerName|TestSearchTasksAcceptsCommentID' .
cd taskFE/app && npx vitest run src/utils/navbarTaskSearch.test.js src/components/NavbarTaskSearch.test.js
```

## 预防

- 从云控制台 / 容器名粘贴的标识须先还原为规范 `task_<digits>` 或 `cmt_<id>`，禁止直接拿整段 InstanceName 做 `id LIKE`。
- 新增容器名变体时，同步扩展 `normalizeTaskSearchQuery` 与 `normalizeSearchText`。
