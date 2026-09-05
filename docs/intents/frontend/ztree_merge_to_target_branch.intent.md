# zTree「合并到目标分支」


## 业务意图 → 事件对照

> 存量回填（自动）：对照 `.ai/08_prompt_management/01_intent_driven_development.md`。事件名若为启发式占位，可在后续迭代精修。

**无对应事件**：纯前端展示/交互或设计治理，无服务端业务状态变更意图。

| 业务意图 | 事件名（过去式） | 发布点 | 消费者/副作用 | 例外理由 |
|---------|----------------|--------|--------------|---------|
| zTree「合并到目标分支」 | — | — | — | 纯前端展示/交互或设计治理，无服务端业务状态变更意图 |
## 变更记录
- 2026-07-12：首次记录。任务详情可写层 zTree 增加「合并到目标分支」，调用容器本地 merge。
- 2026-07-12：若层 `git_remote.current_branch` 已等于 `merge_target_branch_name`，不展示「合并到目标分支」按钮。

## 意图
1. 有 git 的层/任务行展示按钮「合并到目标分支」；**当前检出分支已是合并目标分支时不展示**。
2. 工作区有未提交变更时按钮禁用，title 提示先提交。
3. 点击后 POST `…/cloud/compute/container-layer-git-merge/`，body 含 `layer_id`、`target_branch`（来自 `merge_target_branch_name`）、`container_page_url`。
4. 成功提示并刷新层图；失败 alert 错误信息。

## 非目标
- 不自动推送合并结果；不创建 PR。
