# 测试意图：zTree「推送并创建PR」与 PR 按钮

- 对应功能意图：`ztree_push_and_create_pr.intent.md`
- 日期：2026-07-12（2026-07-13 更新：GitLab MR）

## 单元

| 场景 | 期望 |
|------|------|
| 按钮文案 | `canPush` 节点显示「推送并创建PR」 |
| 请求体 | `onLayerGraphLayerPush` 发送 `wait_for_pr: true` |
| 成功不自动打开 | 响应含 `html_url` 时 **不**调用 `window.open` |
| 写入快照 | 成功后 `git_remote.pr_html_url` 等于该 URL |
| PR 按钮可见 | `pr_html_url` 存在时节点 `canOpenPr===true`，渲染 `layer-ztree-pr-btn` |
| 点击打开 | PR 为 `<a href>`，`href` 等于审查页 URL，`target=_blank` |
| 刷新保留 | `ahead===0` 刷新后仍保留 `pr_html_url`；容器 `rememberLayerPrHtmlUrl` 持久化 |
| 事件冒泡 | 嵌套节点 `layer-submit-and-push` / 根节点 `layer-submit-and-merge` 可到达 TaskDetail |
| Django wait_for_pr | `wait_for_pr=true` 时响应含解析后的 `github_pull_request`（含 html_url） |
| Django 默认 | 无 `wait_for_pr` 时主响应不含 `github_pull_request`（异步） |
| GitLab 容器 MR | `createGitlabMergeRequest` 成功时返回 `web_url` |
| GitLab SaaS 兜底 | 仅 GitLab 多仓推送时不再 `no_github_repo`，返回 MR `html_url` |
| GitLab 容器已创建 | follow-up 从 `github_oauth_multirepo.repos[].pr` 提取 `html_url` |
| push 失败芯片 | `git_remote.last_push_error` 时渲染 `layer-ztree-push-error-label`；缺 token 等通用失败 `pushErrorLabel==='push 失败'`；`BINDING_MISSING` /「缺少绑定」为 `pushErrorLabel==='未绑定 Git 授权'`；GitHub `Permission to … denied` 等写权限拒绝为 `pushErrorLabel==='push 无权限'`；与「提交并创建PR」同时可见 |
| 复制失败信息 | 有错误芯片时渲染 `layer-ztree-push-error-copy`（文案「复制」）；点击写入 `pushErrorTitle` + 可选 `traceId:` 行；成功后文案「已复制」；无错误时不渲染 |
| 失败 trace | 有 `last_push_error_trace_id` 时芯片带 `data-traceId` |
| 成功清除 | `rememberLayerPrHtmlUrl` / 推送成功后 snapshot 不再含 `last_push_error` |

## Playwright（回归）

| 场景 | 期望 |
|------|------|
| GitLab mock 推送 | 响应含 gitlab MR `html_url` 后可见 `layer-ztree-pr-btn`；锚点 `href` 为该 URL |
| 既有 ztree push-ahead | 按钮文案更新后仍可点击推送路径 |
