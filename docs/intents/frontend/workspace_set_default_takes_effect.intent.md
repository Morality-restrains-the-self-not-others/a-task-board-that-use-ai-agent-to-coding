# 意图：工作空间「是否设为默认」须真正切换租户默认

## 背景与目标

租户设置「工作空间管理」(`settings/task-panel`) 编辑弹窗有「是否设为默认」。用户勾选并保存后，列表「默认」badge 仍钉在当前选中行（`is_current`），后端也不取消其它工作空间的 `is_default`，表现为「修改没有生效」。

目标：勾选并保存后，该工作空间成为租户唯一默认；列表 badge 跟 `is_default`，删除按钮仍按 `is_default` 隐藏。

## 范围与边界

- 范围：`WorkspaceSettingsTaskPanel` 列表 badge；`WorkspaceCreateEditModal` 勾选；`taskProjectService` 创建/更新工作空间的 `is_default`。
- 非目标：不改 `is_current`（成员当前选中工作空间 / `workspace_id` 查询参数）；不改工作空间切换器选中逻辑。

## 约束与风险

- 写操作仍须 `createClickGuard` + `Idempotency-Key`。
- 同一租户至多一个 `is_default=true`；跨租户互不影响。
- 无新 HTTP 路径、无新表、无新领域事件（沿用既有 workspace PUT/POST）。

## 业务意图 → 事件对照

| 业务意图 | 事件名 | MQ/契约 | 发布点 | 例外理由 |
|---------|--------|---------|--------|----------|
| 将工作空间设为租户默认 | — | — | taskProjectService PUT/POST `is_default` | 既有资源更新，无新领域事件契约 |

## 验收标准

1. 列表「默认」badge 只出现在 `is_default===true` 的行，即使另一行 `is_current===true`。
2. 勾选「是否设为默认」保存后，POST/PUT body 含 `is_default: true`。
3. 将 B 设为默认后，同租户 A 的 `is_default` 为 false；其它租户默认不变。
4. 默认工作空间仍无删除按钮。
