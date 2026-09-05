# zTree「合并到目标分支」设计

- **日期**: 2026-07-12
- **作者**: claude (goal-mode 自动采纳)
- **状态**: approved (goal-mode 跳过用户门)

## 问题 / 意图

任务详情页可写层 zTree 节点已有「提交」「推送」，缺少把当前工作区提交**本地合并进合并目标分支**（`branch_strategy.merge_target_branch_name`）的一键能力。容器 `onlineServiceJS` 亦无 `git/merge`。

## 成功标准

1. 有 git 的可写层节点展示「合并到目标分支」按钮；工作区脏时禁用并提示先提交；**当前已在合并目标分支时不展示该按钮**。
2. 点击后经 SaaS/网关转发至容器 `POST …/git/merge`，将当前 HEAD 合并进 `target_branch`（合并目标）。
3. 冲突时 abort 并返回可读错误；成功后刷新层图。
4. 意图/测试意图与 skill.md、单元测试同步。

## 方案（已采纳）

沿用「推送」按钮模式垂直切片：

| 层 | 改动 |
|----|------|
| 前端 | `canMerge` + 按钮 + `layer-merge` 事件 + `onLayerGraphLayerMerge` |
| Django | `container-layer-git-merge` forward + `MIGRATED_OUTBOUND_ACTIONS` |
| taskContainerGateway | L0 registry 条目 → 容器 `/git/merge` |
| onlineServiceJS | `POST /api/layers/:id/git/merge` |

### 合并语义

- **源**: 当前 HEAD（或 `source_ref`）
- **目标**: 请求体 `target_branch`（前端填 `merge_target_branch_name`）
- **步骤**: 工作区须干净 → 解析/检出目标分支（本地或 `origin/<target>`）→ `git merge <source>` → 成功留在目标分支；冲突则 `merge --abort` 并切回源分支，HTTP 409
- **多仓**: MVP 仅主工作区（与基础 `git/commit`/`git/push` 一致）

### 不做

- 不替代推送后自动 PR；不自动 push 合并结果
- 不实现 `GET …/git/branches`

## 架构变更影响

- 🟡 [MODIFIED] onlineServiceJS — 新增 `git/merge`
- 🟡 [MODIFIED] taskContainerGateway L0 — `container-layer-git-merge`
- 🟡 [MODIFIED] Django cloud compute forward
- **说明**: 属既有组件接口扩展；不新增服务节点。完整 ArchiMate 三件套延后至独立架构迭代（本迭代以可运行证据交付）。

## Domain 概念（轻量）

- Bounded Context: 任务可写层 Git 协作
- 操作: MergeToTargetBranch（应用服务，非新聚合）
- 输入: layer_id, target_branch, source_ref?
- 事件: 无跨上下文事件（同步 HTTP）

## 价值流影响

- 影响域: 任务协作 / 可写层 Git
- 新步骤: ztree-merge-to-target
- 测试: 前端 vitest + onlineServiceJS 单测 + gateway registry

## Python 新接口

- Django 仅 **转发** 既有模式 endpoint（与 commit 同构），无新业务逻辑落 Python；归属 Go 网关 L0 + Node 容器。不触发 Python 专项审批加严。
