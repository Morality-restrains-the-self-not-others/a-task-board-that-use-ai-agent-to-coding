# 意图：Fork 后新任务落在进度第一列

## 背景与目标

工作面板 / 任务详情点击 `button#task-fork-btn`（可见文本 Fork）会复制源任务属性并 `POST .../todos/` 创建派生任务。当前实现把源任务的 `progress_column_id` 原样写入新任务，导致已推进到第二列及以后的任务被 Fork 后仍停在同一进度列。

目标：Fork 后的新任务进度状态必须处于工作区进度系统的**第一列**（与工作面板新建任务默认列一致），源任务进度不变。

## 范围与边界

- 范围内：`taskDetailEditing.forkTask` 的创建 payload；任务详情传入的 `progressStatusOptions`（工作区 progress-system 列，API 已按 `order_num` 排序）。
- 范围外：不改进度系统配置、不改看板拖拽、不改源任务 PATCH；不新增 API。

## 约束与风险

- 进度列归属 taskProjectService；taskTaskService 只存 `progress_column_id` 字符串。Fork 客户端必须显式写入第一列 ID，不能依赖看板「空列回落到第一列」的展示兜底。
- 进度选项尚未加载或为空时：不得回退为源任务列；省略 `progress_column_id`（后端存空，看板展示仍归第一列）。
- 权限与创建任务一致，不新增角色。

## 验收标准

1. 源任务在非第一列时 Fork：POST body `progress_column_id` 等于 `progressStatusOptions[0].id`，且不等于源任务当前列。
2. `progressStatusOptions` 为空时：POST body **不**携带源任务的 `progress_column_id`。
3. 源任务自身 `progress_column_id` 不变。
4. 新任务在工作面板看板出现在进度纵轴第一列。

## 业务意图 → 事件对照

| 业务意图 | 事件名（过去式） | MQ类型/契约 | 发布点 | 消费者/副作用 | 例外理由 |
|---------|----------------|-----------|--------|--------------|---------|
| Fork 新任务落在进度第一列 | — | — | — | — | 纯前端创建 payload 约束；服务端仍走既有 `POST todos` → `TASK_CREATED` |

## 实施计划

1. 单测：源任务非第一列时 Fork payload 使用第一列 ID。
2. `forkTask` 使用 `progressStatusOptions` 第一列，禁止复制 `task.progress_column_id`。
3. Playwright Fork 回归：mock 两列进度系统，断言 POST 第一列。

## 变更记录

- 2026-08-18：页面元素调整——Fork 后任务须处于进度状态第一列。
