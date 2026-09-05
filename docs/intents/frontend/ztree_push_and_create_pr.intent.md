# 意图：zTree「推送并创建PR」后出现 PR 按钮

- 日期：2026-07-12
- 状态：implemented（2026-07-13：GitLab MR + PR 按钮）
- 相关页面：任务详情 zTree 可写层节点

## 1. 问题

层节点「推送」文案未表达会创建 PR；且 PR follow-up 异步化后主响应不含 `github_pull_request`，完成后无法进入 PR 审查页。自动 `window.open` 打断操作流，用户希望显式出现 **PR** 按钮后再跳转。

**2026-07-13 线上复现**（`task_12988690963090217865`，仓库 `gitlab.daydaymoney.com/.../somanyad`）：推送成功后未出现 PR 按钮、GitLab 亦无 Merge Request。根因：follow-up 仅支持 GitHub，GitLab 多仓推送返回 `skipped: no_github_repo`。

## 2. 行为约定

1. 当既有 `canPush` 为真（工作区干净且相对远端有未推送提交）时，按钮文案为 **「推送并创建PR」**（`data-testid="layer-ztree-push-btn"`）。
2. 点击后 POST `container-layer-git-push/`，请求体含 `wait_for_pr: true`。
3. 服务端在推送成功后同步执行 PR/MR follow-up，并等待外部 job 终态（有超时）；响应含 `github_pull_request`（字段名历史兼容；GitLab 时 `html_url` 为 MR `web_url`，可含 `provider: gitlab`）。
4. **GitHub**：创建 Pull Request；**GitLab**（含自建 `gitlab.daydaymoney.com`）：创建 Merge Request。容器 oauth 推送路径在 `pr_base_branch` 存在时同步创建；SaaS 对仅 GitLab 任务兜底创建。
5. 若返回 `html_url`：
   - **不**自动打开新标签；
   - 将 URL 写入层快照 `git_remote.pr_html_url`；
   - 容器侧 `rememberLayerPrHtmlUrl` 持久化到层目录，后续 `layerGitRemoteSnapshot` / 层图刷新带回；
   - zTree 同层节点出现可点击 **PR** 锚点（`data-testid="layer-ztree-pr-btn"`，`href` 为审查页，`target=_blank`）；
   - 用户点击 PR 链接打开审查页。
6. 层图刷新且 `ahead===0` 时，保留既有 `pr_html_url` / `last_pushed_count`（前端合并 + 容器持久化双保险）。
7. 无 `html_url` 时沿用既有 skipped / compare_url 提示与兜底。
8. **push / 自动运行交付失败**时，容器把原因写入层目录 `git_last_push_error.json`，层快照 `git_remote.last_push_error`（可选 `last_push_error_trace_id`）。zTree 同层节点在「提交并创建PR」旁显示红色错误芯片（`data-testid="layer-ztree-push-error-label"`，`title` 为完整原因；有 trace 时挂 `data-traceId`）：仓库写权限拒绝（GitHub `Permission to … denied` 等）时文案为 **push 无权限** 且 title 提示换有写权限的账号重新授权；**缺评论 Git 授权 / `BINDING_MISSING`** 时文案为 **未绑定 Git 授权**（见 `043_layer_oauth_gitlab_binding`）；其它失败仍为 **push 失败**。芯片旁提供 **复制** 按钮（`data-testid="layer-ztree-push-error-copy"`），点击将完整失败原因写入剪贴板（有 trace 时另起一行 `traceId: …`），成功后按钮文案短暂变为「已复制」。成功推送或记下 PR URL 后清除该错误。执行细节摘要「Git OAuth · 已绑定」在出现写权限拒绝时 overlay 为「Git OAuth · 无写权限」并提供换账号授权链接（见 `034_comment_level_repo_identity` T17b）。

## 3. 非目标

- 不新增独立「仅推送」按钮。
- 不改变 `canPush` 门控语义。
- 无 `wait_for_pr` 时仍保持异步 PR（兼容其他调用方）。
- 不新建站内 PR 审查路由；审查页即 GitHub PR / GitLab MR 外链。

## 4. 变更记录

| 日期 | 变更 |
|------|------|
| 2026-07-12 | 初版：文案 + wait_for_pr + 自动打开 html_url |
| 2026-07-13 | 成功态改为显示 PR 按钮，点击后再打开审查页；快照保留 pr_html_url |
| 2026-07-13 | 支持 GitLab Merge Request（容器 + SaaS 兜底），修复仅 GitLab 任务无 PR 按钮 |
| 2026-08-17 | 容器持久化 pr_html_url；PR 改为 `<a href>` 附着节点；修复 submit-and-push/merge 事件冒泡 |
| 2026-08-22 | 生成 html_url 后另建嵌套回复 + 合并状态/一键合并，见 `pr_reply_merge_status.intent.md` |
| 2026-08-29 | push 写权限拒绝时芯片为「push 无权限」；执行细节 OAuth overlay「无写权限」+换账号授权 |
| 2026-08-29 | push 失败芯片旁增加「复制」按钮，写入完整失败原因（含 traceId） |
| 2026-09-02 | `BINDING_MISSING` 芯片为「未绑定 Git 授权」；换票须识别 GitLab 评论站点级 L2（043） |

## 业务意图 → 事件对照

> 存量回填（自动）：对照 `.ai/08_prompt_management/01_intent_driven_development.md`。事件名若为启发式占位，可在后续迭代精修。

**无对应事件**：纯前端展示/交互或设计治理，无服务端业务状态变更意图。

| 业务意图 | 事件名（过去式） | 发布点 | 消费者/副作用 | 例外理由 |
|---------|----------------|--------|--------------|---------|
| 意图：zTree「推送并创建PR」后出现 PR 按钮 | — | — | — | 纯前端展示/交互或设计治理，无服务端业务状态变更意图 |
