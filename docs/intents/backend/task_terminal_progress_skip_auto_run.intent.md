# task_terminal_progress_skip_auto_run（功能意图 · 后端）

## 背景与目标

任务详情页「进度状态」下拉（`#task-progress-status`）PATCH 只改 `progress_column_id`。若任务已开启 `auto_run`，更新路径仍会走 `validateAutoRunPrerequisites`。镜像已删除时返回 400「自动运行不可用：无法获取已安装镜像信息」，用户无法把进度改成**已完成**或**已取消**。

终态关闭任务不需要启动机器，因此不得再校验自动运行前置条件。

## 范围与边界

- **范围内**：`taskTaskService` 更新任务（PATCH/PUT todos）进入终态（进度列「已完成/已取消」或 `completed` 变为 true）
- **范围内**：工作面板拖拽改进度（同一更新接口）
- **范围外**：创建任务 `auto_run=true` 仍须门禁
- **范围外**：排队自动运行 / `force_auto_run` 真正启机路径仍须门禁
- **范围外**：改到「待处理 / 进行中」等非终态时，若 `auto_run` 仍为 true，继续校验自动运行

## 约束与风险

- 终态仍发布既有 `TASK_STATUS_CHANGED`（下游可关停机器）
- 不因终态把 `auto_run` 标志改成 false
- 终态不得 `scheduleTaskAutoRun`

## 验收标准

1. `auto_run=true` 且已安装镜像查找失败时，PATCH 进度到「已完成」→ 200，任务列更新成功
2. 同上，PATCH 到「已取消」→ 200
3. 同上，PATCH 到「进行中」→ 400，`code=AUTO_RUN_RUNTIME_ENV_REQUIRED`
4. 终态成功路径不调用 start-vm
5. 终态成功路径仍发布 `TASK_STATUS_CHANGED`

## 实施计划

1. 进入终态判定提前到 auto_run 门禁之前
2. `autoRun && !enteringTerminal` 才 `validateAutoRunPrerequisites`
3. `shouldTriggerAutoRun` 增加 `&& !enteringTerminal`

## 业务意图 → 事件对照

| 业务意图 | 事件名（过去式） | MQ类型/契约 | 发布点 | 消费者/副作用 | 例外理由 |
|---------|----------------|------------|--------|--------------|---------|
| 进度改为已完成/已取消 | TASK_STATUS_CHANGED | Kafka `task-status-changed` | `handleUpdateTask` → `publishTaskStatusChangedFn` | taskEvents 关停/结算 | 既有事件，不新增 |
| 终态跳过 auto_run 门禁 | — | — | — | 不启机 | 应用策略门禁；无新领域事实 |

## 变更记录

| 日期 | 变更 | 原因 |
|------|------|------|
| 2026-08-20 | 初稿 | 详情页改完成/取消被已删镜像的 auto_run 门禁挡住 |
