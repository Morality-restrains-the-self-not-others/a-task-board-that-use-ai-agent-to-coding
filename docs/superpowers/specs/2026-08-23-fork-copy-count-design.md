# 创建设计：Fork 确认弹窗副本数量

**日期**: 2026-08-23  
**状态**: 已采用（goal-mode 自动选型，跳过用户确认）  
**范围**: 任务详情页 Fork 确认模态 `button#fork-confirm-no-auto-run-btn` 所在弹窗

## 🕸️ Code Review Graph 分析

- CRG `update --brief` 已执行（增量 2 files / risk 0.00）。
- 既有调用链：`TaskDetailPageHeader.openForkConfirm` → `ForkAutoRunConfirmModal` → `onForkConfirm` → `forkTask` → `POST /api/tasks/todos/tenant_id/{tid}/workspace_id/{wid}/`（`fork_from` + `auto_run`）。
- 服务端 `taskCreateDedupKey`：有 `Idempotency-Key` 时按键去重；否则 `fork_from+owner+title` 10s 窗口合并为 1 条。批量派生必须为每份副本使用不同 Idempotency-Key，否则会被短窗折叠。

## 目标

用户在 Fork 确认弹窗选择一次要制作的副本数量：默认 1，最小 1，最大 99。两个派生动作（自动运行 / 仅派生）都使用该数量。

## 方案对比与决策

| 方案 | 说明 | 取舍 |
|------|------|------|
| A. 前端循环 N 次既有 POST，每份独立 `Idempotency-Key=${batch}:${i}` | 不改 API；配额/auto_run 沿用单次创建；部分失败可保留已成功副本 | **采用** |
| B. 后端新增 `fork_count` 一次创建 N 条 | 单 RTT，事务边界清晰 | 拒绝：改动 `handleCreateTask` 配额/启服/事件路径，超时与部分失败语义复杂 |
| C. 标题加后缀绕过 fork 短窗 | 无 Idempotency-Key 也能连 POST | 拒绝：污染标题，且仍是 N 次请求 |

**架构影响**: 无新服务、无新 HTTP 路由、无新领域事件（每份副本仍走既有 `TASK_CREATED`）→ **不更新** ArchiMate 三件套。

## 交互

1. 打开 Fork 模态时数量复位为 1。
2. 数量选择器：`input[type=number]`，id=`fork-copy-count-input`，范围 1–99；失焦与确认时 clamp。
3. 数量 > 1 时提示将创建 N 个副本；若随后选自动运行，文案说明会为每个副本尝试启动云资源。
4. 「不自动运行，仅派生」/「自动运行并派生」均带上当前数量。
5. 进行中禁用动作，文案 `派生中 i/N…`（N=1 时仍为「派生中…」）。
6. N=1：保持现网，新标签打开该任务。
7. N>1：只打开**第一份**副本的新标签，避免 99 个弹窗；`onForked` 刷新看板。
8. 第 1 份失败：模态保持打开。第 k 份失败（k>1）：提示已成功数，关闭模态并打开第一份。

## 落点

| 文件 | 职责 |
|------|------|
| `taskFE/app/src/utils/forkCopyCount.js` | `clampForkCopyCount`（1–99） |
| `ForkAutoRunConfirmModal.vue` | 数量选择 UI，confirm payload `{ autoRun, copyCount }` |
| `TaskDetailPageHeader.vue` | 传递 copyCount；`createClickGuard` 生成 batch key |
| `taskDetailEditing.js` `forkTask` | 循环 POST + `${batchKey}:${i}` |
| `useTaskDetail.js` | 透传 `copyCount` / `batchIdempotencyKey` / `onForkProgress` |

## 角色权限（摘要）

不新增权限点。每份副本仍走工作区任务创建 + 任务帖配额 + auto_run 门禁。

## NFR（L2；写路径幂等 L3）

见 `docs/superpowers/plans/2026-08-23-fork-copy-count-nfr-clarification.md`。
