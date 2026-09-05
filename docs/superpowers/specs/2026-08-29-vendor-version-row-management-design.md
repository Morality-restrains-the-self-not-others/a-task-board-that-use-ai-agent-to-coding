# 厂商门户镜像版本行管理手段

- **Date:** 2026-08-29
- **Status:** accepted (goal-mode 自动采用)
- **Author:** cursor
- **Page:** https://provider.daydaymoney.com/ 「AI 容器镜像市场」厂商门户镜像组版本行

## 问题分析

用户高亮 `div.version-row`（已上架 `private_x86_64-latest`，组内激活为另一版本 `x86_64-latest`），可见文案只有 ID / 版本 / 接口 / 状态 / 仓库 URL，**没有删除等管理按钮**。

根因（代码现状，非猜测）：

1. **布局**：`.version-row` 为 5 列 grid（`9.5rem 4rem 7.5rem minmax(0,1fr) auto`，见 `.ai/09_failure_experience/02_runtime_errors/26_provider_list_columns_misaligned.md`），但行内有 6 个子节点：id、version、接口版本、status、**url**、**actions**。`version-url` 占用本应给操作列的 `auto` 轨，操作钮折到隐式第二行第 1 列（9.5rem），插件/用户第一行看不到「删除/撤回/设为激活」。
2. **后端撤回缺口**：UI 对 `approved` 显示「撤回」，但 `handleVendorContainerAction` 的 `withdraw` 只调用 `WithdrawPending()`（仅 `pending_review→draft`）。已上架点击撤回会 400「仅待审核可撤回」。下架能力只在管理员 `unpublish`。
3. **删除展示过窄**：前端删除仅 `draft|rejected`；领域 `CanDelete()` 同样如此。HTTP `DELETE` **未调用** `CanDelete()`，与文档/UI 不一致。激活版本若被直删会搞乱公开目录。

无 `data-traceId`；CRG `update --brief` 成功（0 风险）。当前浏览器会话未登录该厂商，无法在公网页复现该组数据，以源码与选择器为准。

## 🕸️ Code Review Graph 分析

- CRG available；增量 update 3 files / 0 nodes。分析基于 `VendorImageGroupsTab.vue` → `useVendorPortalImageGroups.js` → `/api/vendor/container-images/` 调用链。

## 选定方案

不新增 HTTP path。补齐既有 `POST .../withdraw/` 与 `DELETE .../{id}/` 的领域语义，并修 grid，使**每一行第一轨都有操作列**。

| 状态 | 区域明细 | 编辑 | 提交审核 | 设为激活 | 下架/撤回审核 | 删除 |
|------|----------|------|----------|----------|----------------|------|
| draft / rejected | ✓ | ✓ | ✓ | | | ✓ |
| pending_review | ✓ | | | | 撤回审核 | |
| approved 非激活 | ✓ | | | ✓ | 下架 | ✓ |
| approved 且激活 | ✓ | | | | 下架（同时取消激活） | 禁止（须先下架） |

- URL 单独占第二行（`grid-column: 1 / -1`），不再与操作列争轨。
- 写操作：自定义确认弹窗（禁止 `window.confirm`）+ `createClickGuard` + `Idempotency-Key`。
- 领域：`VendorWithdraw`；`Unpublish` 将 `IsActive=false`；`CanDelete` 允许非激活的 draft/rejected/approved；handler 强制 `DeleteGuard`。
- 事件：`ContainerImageReviewWithdrawn` / `ContainerImageUnpublished` / `ContainerImageDeleted`（LogEventBus）。

## 业务意图 → 事件对照

| 业务意图 | 事件名（过去式） | 发布点 | 消费者 | 例外理由 |
|---------|----------------|--------|--------|---------|
| 厂商撤回待审核 | ContainerImageReviewWithdrawn | vendor withdraw | 无（目录不再待审） | LogEventBus，无 Kafka |
| 厂商下架已上架 | ContainerImageUnpublished | vendor withdraw | 公开目录不再展示该版本 | 同上 |
| 厂商删除版本 | ContainerImageDeleted | vendor DELETE | 组内版本列表减少 | 同上 |
| 查看/展开区域明细 | — | — | — | 纯前端 UI |

## 🐍 Python 新增接口

not_applicable：扩展既有 Go `taskAiProvider` 路径，无 Python endpoint。

## 🏛️ 架构变更影响

无需新版本 ArchiMate：无新服务、无新 API path、无数据所有权变更。仅修正既有 Application_Interface 行为与 UI 栅格。

## Value Stream 影响

增量挂在「镜像市场」域：厂商管理已上架/草稿版本的下架与删除。见 `*-value-stream.md`。
