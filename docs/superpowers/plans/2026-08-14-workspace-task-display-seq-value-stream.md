# Value Stream: 工作空间任务帖人读序号

> Derived from design: `docs/superpowers/specs/2026-08-14-workspace-task-display-seq-design.md`

## Value Summary

工作区成员在看板看到同一工作空间内从 `#1` 起的有序任务编号，可复制、可搜索。

## Related Value Streams

- **task-management / todo-crud**：modification — 创建/列表增加 `workspace_seq`；展示从后 6 位改为 `#N`
- **todo-fork-from**：extension — 派生自文案改用 `#N`

## End-to-End Flow

建帖 → 事务发号 → 落库 `workspace_seq` → JSON/`TASK_CREATED` → 看板 `#N` → 搜索 `#N` 命中

## Value Increments

### Increment 1: 发号 + JSON（Thin Slice）
**Value to user:** 创建后响应带 `workspace_seq`，连续建帖为 1,2,3
**Scope:** DDL + 发号 + create/list/detail JSON；存量清空
**Business intents → events:** 创建 → `TASK_CREATED`（增补 seq）
**Depends on:** nothing

### Increment 2: 看板展示与复制
**Value to user:** 卡片显示 `#N`；单击复制 `#N`，双击复制技术 ID
**Scope:** `TaskCardIdBadge` / `formatTaskIdTitleLabel`
**Business intents → events:** 纯展示，无事件
**Depends on:** Increment 1

### Increment 3: 搜索与父任务/派生自
**Value to user:** `#12` 在当前工作空间命中；父任务标签为 `#N 标题`
**Scope:** 后端搜索 + `navbarTaskSearch` + `parentDeliverableFilter`
**Business intents → events:** 搜索为纯查询
**Depends on:** Increment 1–2
