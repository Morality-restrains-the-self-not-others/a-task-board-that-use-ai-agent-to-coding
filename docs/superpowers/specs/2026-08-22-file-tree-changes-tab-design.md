# 设计：任务详情项目文件树增加「文件变动」Tab

- **日期**: 2026-08-22
- **状态**: accepted（/goal 自动采用）
- **意图**: `docs/intents/frontend/task_detail_file_tree_changes_tab.intent.md`

## 问题

选中可写层后，项目文件树与层变动列表上下叠放，评论执行区过长；用户希望在「项目文件树（所有拉取仓库）」处用 Tab 查看变动列表。

## 约束

- 不新增 HTTP API、Kafka 事件、表结构
- 复用 `TaskDetailExecLayerChanges` 与 `TaskDetailProjectFileTree`
- 切 Tab 不得销毁树展开 / 文件预览 / 变动选中（`v-show`）
- 无架构组件/数据流变更 → **不更新** ArchiMate / `.puml`（见下文）
- 前端 Tab 无副作用：Anti-Replay-OK

## 方案（选定）

在 `TaskDetailTaskLayerAssociationPanel` 中，当 `selectedLayerGraphFileTreeLayerId` 非空时，用 `TaskDetailLayerFilesTabs` 包裹两个既有面板：

```
[项目文件树]  [文件变动 · N]
────────────────────────────────
active panel (v-show)
```

- 默认 Tab：`changes`
- 变动 Tab 计数：`selectedZTreeLayerChangesPanel.displayCount`
- 无变动载荷时仍显示变动 Tab，面板内保持 `TaskDetailExecLayerChanges` 的 `v-if`（无数据则不挂列表）
- 切换 `fileTreeLayerId` 时复位 `changes`
- Tab 视觉对齐 `CommentExecutionTabList`（`role="tab"` / `aria-selected` / sky 选中态）

### 拒绝的方案

| 方案 | 拒绝原因 |
|------|----------|
| 在文件树 `<details>` 内部再嵌套 Tab | 与 summary 折叠语义冲突 |
| 复制一份变动列表 | 提交/刷新/续拉状态会分叉 |
| 新 API 拉「文件树变动」 | 层 diff 已是 SSOT |

## 架构

**不更新架构视图。** 无新服务、无新 Rel_Flow、无数据所有权变化。基线保持 v98 current（v99 待分账为正交 target）。

## 🕸️ Code Review Graph 分析

`code-review-graph update --brief` 已执行。本次触及 Vue SFC，图增量 0 函数节点（前端模板不入 Go/Python 符号图）。设计基于源码：

- `TaskDetailTaskLayerAssociationPanel.vue` 顺序渲染 ztree → `TaskDetailExecLayerChanges` → `TaskDetailProjectFileTree`
- 变动载荷：`commentLayerPanelBind.layerChangesPanelFromSlot`
- 文件树刷新 nonce 与变动刷新已由既有 Playwright 覆盖

## 改动文件清单

- 新增 `taskFE/app/src/components/task-detail/TaskDetailLayerFilesTabs.vue`
- 修改 `TaskDetailTaskLayerAssociationPanel.vue`、`index.js`
- 测例与 Playwright 辅助 `openLayerFilesChangesTab`
- 意图 / 本设计 / 权限 / 价值流 / NFR / DDD 笔记 / 计划
