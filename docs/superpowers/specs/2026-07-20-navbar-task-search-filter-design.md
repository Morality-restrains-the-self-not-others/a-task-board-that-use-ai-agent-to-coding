# 导航栏任务搜索过滤

- **日期**: 2026-07-20
- **状态**: 已批准（/goal 自动采纳推荐方案）
- **迭代**: navbar-task-search-filter
- **作者**: claude
- **页面来源**: `/tenant/{tid}/work-panel/` 工作面板标题行 + 看板任务编号
- **2026-08-17**: 搜索框从顶部导航栏迁至工作面板标题行（标题与工作空间切换之间）

## 问题陈述

工作面板用户需要在顶部导航快速按**任务编号 / 标题 / 负责人**过滤任务，并从结果中：

1. 跳转到对应工作空间的工作面板；或
2. 跳转到指定任务（详情页 / 面板内打开）。

现有能力：计费页已调用 `GET /api/tenant/{tid}/tasks/search/?q=`（仅 title/id）；工作面板内筛选模态不在导航栏；无负责人匹配与导航深链。

## 架构理解（基线）

- tip：**v41 ✅ current**（application-integration）
- 路径：Vue SPA → gateway → **taskTaskService** `tasks/search`（已有）
- 本迭代：**扩展既有搜索匹配维度 + Navbar UI**；不新增微服务、不改拓扑 → **不写架构 target**

📋 版本历史：v41 current — 任务子树状态；v40 — 排队自动执行节奏。

## 推荐方案（⭐）

| 层 | 约定 |
|----|------|
| API | 扩展 `GET /api/tenant/{tid}/tasks/search/`：`q` 匹配 **title / id / owner_id / operator_id / assignee company_member_id**；可选 `assignee_ids`（逗号分隔）精确并入 OR（assignee/owner/operator）；响应增加 `owner`、`operator`、`assignees` |
| 前端解析负责人名 | Navbar 侧拉取 `company_members`，名称模糊匹配后把 member id 传入 `assignee_ids`，与 `q` 一并查询 |
| UI | `NavbarTaskSearch`：已登录且有租户时显示输入框；防抖 250ms；下拉结果；每行「工作面板」「打开任务」 |
| 跳转 | 工作面板：`/tenant/{tid}/work-panel/?workspace_id={ws}`；指定任务：同路径加 `task_id={id}`，WorkPanel 打开详情模态；亦可直达 `task-detail` 路由 |
| 权限 | **fail-closed**：仅 `workspaces?mine=1` 返回的工作空间可搜；指定 `workspace_id` 必须在该集合内；未登录 401；权限服务不可用 503（禁止扫全库兜底） |
| 事件 | 纯查询，无新 MQ 业务事件（书面例外） |

### 不做

- 不替换工作面板内既有「筛选」模态
- 不新增 Django 公网接口
- 不在 Navbar 做全量 todos 客户端过滤

## 备选（未采用）

| 方案 | 理由 |
|------|------|
| A. 仅客户端过滤当前看板 todos | 无法跨工作空间；负责人名不全 |
| B. 新建专用 search 微服务 | 过重；已有 TTS search |
| C. 仅深链 task-detail，不支持面板内打开 | 与「跳转任务面板」期望不符 |

## 业务意图 → 事件对照

| 业务意图 | 事件名 | 发布点 | 消费者 | 例外理由 |
|---------|--------|--------|--------|---------|
| 导航栏任务搜索 | — | — | — | 只读查询，无聚合状态变更 |
| 打开任务/工作面板 | — | — | — | 纯前端导航 |

## 验收标准

1. 工作面板标题行可见任务搜索输入框（`data-testid="navbar-task-search"`）；顶部导航栏不再挂载。
2. 输入编号片段 / 标题关键词可返回匹配任务列表。
3. 输入负责人（成员）名称可返回其作为 assignee/owner 的任务。
4. 结果可点击「工作面板」进入对应 `workspace_id` 的工作面板。
5. 结果可点击「打开任务」进入该任务详情页 `/tenant/{tid}/workspace/{ws}/task-detail/{id}/`（真实 `<a href>`；禁止对当前 work-panel 再 `router.push` 同 URL）。
6. 无权限工作空间的任务不出现在结果中（沿用 API 门禁）。
7. 请求失败时错误 UI 带 `data-traceId`（若有）。
8. Vitest 覆盖搜索匹配与 Navbar 交互关键路径；Go 单测覆盖 assignee/owner 匹配。
9. 粘贴误带服务前缀的编号（`task-task_<digits>`）或 `#task_<digits>` 时，与规范 id `task_<digits>` 同等可搜（前后端 `normalize*`）。
10. 粘贴评论容器名 / 云实例名 `task_<digits>_cmt_<commentId>`（含 `task-task_`、`task_task_` 双前缀）或规范评论 id `cmt_<id>` 时，命中所属任务。

## 范围

- `taskTaskService` search 扩展
- `task2app/front_project` Navbar + WorkPanel `task_id` 深链
- intents / 价值流测试点
- 不含：跨租户搜索、未登录搜索、系统管理员导航
