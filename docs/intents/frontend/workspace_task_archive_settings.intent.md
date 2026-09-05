# 意图：工作空间「套餐设置」打开任务存档档位

## 背景与目标

租户设置页 `/tenant/:tenant/settings/task-panel/` 工作空间行的「套餐设置」按钮会发出 `archive`，本应打开「套餐设置 · 任务存档时间」模态。前端用 `isTaskArchiveFeatureEnabled = false` 关掉了模态，点击变成空操作；行上还写着「存档：该功能暂未开放」。用户感知为点击报错/无响应。

目标：点击「套餐设置」打开已有存档档位模态，可 PATCH `task_archive_tier`；列表展示当前档位文案，不再假装功能未开放。

## 范围与边界

- 范围：`WorkspaceSettingsTaskPanel` / `WorkspaceSettingsTaskPanelActions` / 独立 `TaskArchiveSettingsModal`。
- 后端：既有 `PATCH /api/projects/workspaces/tenant_id/{tenant}/{workspace}/` 的 `task_archive_tier`（taskProjectService），本增量不改 API。
- 非目标：不实现冷数据物理归档任务；只开放已落库的档位选择。

## 约束与风险

- 打开模态为纯 UI：`Anti-Replay-OK: ui-only`。
- 保存仍须 `createClickGuard` + `Idempotency-Key`（既有）。
- 请求失败须 `showRequestError` 且错误节点带 `data-traceId`。
- `WorkspaceSettingsTaskPanel.vue` 已超 500 行：存档模态必须抽到独立组件，禁止继续堆进该文件。

## 业务意图 → 事件对照

| 业务意图 | 事件名 | MQ/契约 | 发布点 | 例外理由 |
|---------|--------|---------|--------|----------|
| 打开套餐/存档档位模态 | — | — | taskFE 本地状态 | 纯前端 UI，无服务端状态变更 |
| 保存工作空间任务存档档位 | — | — | taskProjectService PATCH workspace | 既有工作空间更新，无新领域事件；档位仅为工作空间属性 |

## 验收标准

1. 点击「套餐设置」出现标题含「套餐设置 · 任务存档时间」的模态，不抛错、不空点。
2. 工作空间行展示「存档：{档位文案}」，不再写「该功能暂未开放」。
3. 保存发出 PATCH，body 含 `task_archive_tier`，头含 `Idempotency-Key`。
4. 保存失败时错误 UI 带 `data-traceId`（有则）。

## 实施计划

1. 抽出档位选项工具与 `TaskArchiveSettingsModal`。
2. 去掉 `isTaskArchiveFeatureEnabled` 阻断；点击打开模态。
3. 组件测覆盖打开/展示档位/保存 PATCH。

## 变更记录

- 2026-08-27：初版（goal-mode：套餐设置点击空操作/报错）
