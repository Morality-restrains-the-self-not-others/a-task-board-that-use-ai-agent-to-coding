# Implementation Plan: 工作空间任务帖人读序号

> Design: `docs/superpowers/specs/2026-08-14-workspace-task-display-seq-design.md`

## Increment 1 — 发号 + JSON

- [x] `dataMigrate/taskTaskService/010_workspace_seq.sql`：仅当列不存在时清空任务帖域；建 `task_workspace_seq`；加 `workspace_seq` + UNIQUE
- [x] `nextWorkspaceSeq(tx, tenant, workspace)` + 单测（连续、并发）
- [x] `taskRecord` / `taskSelectCols` / `scanTask` / INSERT / `taskToJSON` 带 `workspace_seq`
- [x] `publishTaskCreated` payload 增补 seq；忽略 body 中的 seq
- [x] 结构化日志 `event=workspace_seq_allocated`
- [x] 扩展 `TestCreateAndListTasks`：seq=1；连续创建 1,2,3

## Increment 2 — FE 展示

- [x] `formatTaskDisplayNo` / `formatTaskIdTitleLabel` 优先 `workspace_seq`
- [x] `TaskCardIdBadge` 展示 `#N`；单击复制 `#N`
- [x] 更新 `taskIdDisplay.test.js` 与卡片单测

## Increment 3 — 搜索

- [x] `searchTasksInWorkspaces`：去 `#` 后纯数字则 OR `workspace_seq=?`（仍限传入的 workspace 列表）
- [x] `navbarTaskSearch` / `parentDeliverableFilter` 按 seq 匹配
- [x] 跨工作空间同号不串（用两 workspace 夹具）

## 事件契约

- `TASK_CREATED` 增加 `workspace_seq`（可选字段，向后兼容）
- 无新消费者

## 部署

- 9999 初始化数据库（禁止业务进程迁移）
- 精准重启 `task-task-service` + `taskFE`；SPA collectstatic
