# [运行时] 评论级克隆失败只有红字、没有「手动重试」

## 基本信息

- 版本：1.0.0
- 创建日期：2026-08-17
- 最后修改：2026-08-17
- 维护者：Trae AI 团队

## 现象

任务详情评论执行细节「项目克隆」失败行可见红字，例如：

`【项目克隆】(1/1) 失败 ram-work: git exit 128: Cloning into '/app/onlineProject_state/layers/…/ram-work'…`

`data-testid="comment-execution-clone-progress-message"` 存在，但同级没有 `comment-execution-clone-progress-retry`（「手动重试」）。

复现：冷打开已失败的任务详情（无实时 SSE live map），仅 binding 启动日志。

## 根因

1. `shouldShowCloneManualRetry` 要求 `looksLikeGitRepoRef(row.repoUrl || row.key)`。
2. 引导失败进度文案只写目录名 `ram-work`，不写 git URL；binding 日志截断后更不可能带 URL。
3. 冷打开没有 SSE `repo_url` 键，`parseCommentCloneProgressFromLogs` 只能得到 `key=ram-work`，按钮被隐藏。

同类：`35_bootstrap_clone_fail_no_repo_name.md`（任务级摘要）、`37_nested_repo_clone_fail_no_reclone.md`（子仓列表）。本条是评论级进度条冷打开路径。

## 解决方案

- 前端：`collectCloneRepoCatalog` + `enrichCloneProgressRowsWithCatalog`，用关联项目 `git_repos` / `git_repo_entries`、引导克隆日志 `━━` 段、失败文案内嵌 URL 回填 `repoUrl`（子仓附 `parentRepoUrl`/`cloneAlias`）。
- 容器：`formatBootstrapCloneRepoFailureMessage` 在仓库名后写入 git URL，供以后冷打开直接解析。
- 单测：`commentCloneProgressRepoCatalog.test.js` T24–T28；组件测 ram-work 失败行出现「手动重试」。

## 预防

- 失败态可行动作不得只绑「SSE 实时键是 git URL」；冷打开日志必须能还原重试身份。
- 进度文案点名仓库时同时带可 clone 的 URL，不要只写 basename。

## 验证

```bash
cd taskFE/app && npx vitest run \
  src/utils/commentCloneProgressRepoCatalog.test.js \
  src/utils/commentCloneProgressFromLogs.test.js \
  src/components/task-detail/TaskDetailCommentCloneProgress.test.js
cd trae-agent/onlineServiceJS && node --test src/bootstrap.cloneFailureFooter.test.mjs
```

公网：`cd taskFE/app && npm run build` 后硬刷新任务详情；失败行应出现「手动重试」。新克隆需重建 onlineServiceJS 镜像后才会在日志里内嵌 URL。

后续：按钮出现后点击仍可能显示「容器未启动」（任务级 endpoint 短路），见 `97_comment_clone_retry_endpoint_false_start.md`。重试成功后自动任务不续跑，见 `98_comment_clone_retry_no_autorun_resume.md`。
