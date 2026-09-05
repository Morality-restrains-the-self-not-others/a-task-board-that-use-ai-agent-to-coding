# 意图：ztree 层级节点在有 git 时应显示「提交」按钮

## 背景
- 2026-07-12：线上任务 `task_12949237300462721867`（`?relayToTrae=true`）。文件变动列表显示「N 个文件发生变动」，但 zTree 层级行不显示「提交」。
- 根因组合：
  1. 前端 `canSubmit` 仅在 `git_worktree_dirty===true` 时为真，干净工作区完全隐藏按钮（与容器 UI 不一致）。
  2. 容器 `gitWorktreeDirty` 只查主仓，次仓有未提交变更时 dirty 仍为 false。
  3. `layer_changes` 含 `git_layer_diff_only` 时被误判为可提交信号；已提交后仍可能干扰门控。
  4. 本地 feature 无 `@{u}` / 无同名 `origin/<branch>` 时 `no_upstream=true`、`ahead=null`，推送按钮也消失。

## 验收标准
1. 层有 git（`git_worktree_dirty !== null`）时，ztree 节点始终渲染「提交」（`data-testid="layer-ztree-submit-btn"`）；干净时禁用，title 含「暂无未提交变更」。
2. 多仓任一 dirty → `git_worktree_dirty===true`，提交可点。
3. 仅 `git_layer_diff_only` 的 layer_changes **不**经 impliesDirty 解锁提交。
4. 无上游时回退 `origin/HEAD`（或 master/main）计算 ahead，使可推送时「推送并创建PR」可见。
5. 当变动列表全为 `git_layer_diff_only`（工作区干净）时：ztree 旁文案为「N 个相对父层差异」（非「N 个文件变化」），提交按钮 title 说明「仅为相对父层差异、无需再提交」，避免用户误以为有未提交变更却无法点提交。

## 业务意图 → 事件对照

> 存量回填（自动）：对照 `.ai/08_prompt_management/01_intent_driven_development.md`。事件名若为启发式占位，可在后续迭代精修。

**无对应事件**：纯前端展示/交互或设计治理，无服务端业务状态变更意图。

| 业务意图 | 事件名（过去式） | 发布点 | 消费者/副作用 | 例外理由 |
|---------|----------------|--------|--------------|---------|
| 意图：ztree 层级节点在有 git 时应显示「提交」按钮 | — | — | — | 纯前端展示/交互或设计治理，无服务端业务状态变更意图 |
