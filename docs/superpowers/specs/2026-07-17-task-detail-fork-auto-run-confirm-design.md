# 创建设计：任务详情 Fork 确认是否自动运行

**日期**: 2026-07-17  
**状态**: 已采用（goal-mode 自动选型，跳过用户确认）  
**范围**: 任务详情页 `/tenant/:tid/workspace/:wid/task-detail/:taskId/` Fork 按钮

## 目标

点击 Fork 时先让用户确认新任务是否 `auto_run`，再调用既有创建任务接口完成派生。

## 方案对比与决策

| 方案 | 说明 | 取舍 |
|------|------|------|
| A. 独立确认模态 + 两个派生动作 | 「自动运行并派生」「不自动运行，仅派生」+ 取消 | **采用**：语义清晰，符合「确认是否自动运行」 |
| B. `window.confirm` | 原生确认 | 拒绝：前端规范禁止浏览器原生弹窗 |
| C. 复用创建任务大弹窗 | Fork 打开 CreateTaskModal | 拒绝：改动面大，偏离「复制当前任务属性」的轻量派生 |

**架构影响**: 无新服务、无新 HTTP 契约（沿用 `auto_run` / `fork_from`）→ **不更新** ArchiMate 三件套。

## 交互

1. Fork → 打开 `ForkAutoRunConfirmModal`。
2. 取消 / 点遮罩 → 关闭，不请求。
3. 「不自动运行，仅派生」→ `auto_run: false`。
4. 「自动运行并派生」→ `auto_run: true`，并尽量写入 `client_public_ip`。
5. 请求进行中禁用动作按钮；成功后关模态并新标签打开新任务。

## 落点

| 文件 | 职责 |
|------|------|
| `components/task-detail/ForkAutoRunConfirmModal.vue` | 确认 UI |
| `composables/taskDetail/taskDetailEditing.js` | `forkTask` 接收 `autoRun` 并写入 payload |
| `composables/useTaskDetail.js` / `views/TaskDetail.vue` | 打开模态 → 确认后派生 |
| `taskDetailEditing.test.js` | T3–T5 |

## 角色权限（摘要）

不新增权限点；派生仍要求用户具备工作区任务创建能力（与现网 Fork 一致）。

## NFR（L2）

- 可用性：确认文案中文清晰；派生中禁用重复提交。
- 安全：不新增密钥/鉴权面；auto_run 门禁仍在 taskTaskService。
