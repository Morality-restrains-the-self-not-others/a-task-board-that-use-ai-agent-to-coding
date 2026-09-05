# [运行时] 已授权父仓 + 关闭自动克隆子仓库，自动运行仍显示「授权异常无法启动」

- **日期**：2026-08-18
- **页面**：项目详情 → 是否允许自动运行 / Git 仓库
- **项目示例**：`proj_-2304947540687519745`（`https://gitlab-tencent-sh-1.daydaymoney.com/example-user/ram-work`）
- **data-testid**：`project-auto-run-git-gate-hint`

## 现象

1. 父仓徽章「已授权」
2. 「自动克隆子仓库」未勾选
3. 自动运行展示红字：`存在 Git 仓库授权异常，自动运行无法启动；请先完成授权后再启用`，无法启用

## 根因

`resolveAutoRunGitAuthBlock` 把 **子仓** OAuth `token_error` 与父仓一起聚合。ram-work 的 `.gitmodules` 指向大量 `git@github.com:task2money/…` 子仓；GitHub 子仓授权异常（或 refresh 失败）会阻断自动运行，即使：

- 容器只会克隆已授权的 GitLab 父仓
- `auto_clone_nested_repos=false` 时 credential enrich 已跳过子仓（`nested_repos_enrich.go`）

关闭自动克隆时，子仓列表仍用于展示（「不影响上方发现列表」），但不得作为自动运行门禁输入。

## 与相关条目区分

- [49_github_dot_git_misclassified_as_gitlab.md](./49_github_dot_git_misclassified_as_gitlab.md)：nested 列表走错 provider。
- [88_nested_git_unbound_when_oauth_refresh_timeout.md](./88_nested_git_unbound_when_oauth_refresh_timeout.md)：子仓列表文案误报未绑定。
- [100_autorun_skip_github_oauth_refresh_no_retry.md](./100_autorun_skip_github_oauth_refresh_no_retry.md)：任务启服软跳过（nested-git-repos 探测父仓），不是项目详情门禁。
- 本条：**项目详情 Git 门禁误把子仓 OAuth 算进自动运行**，且未尊重自动克隆开关。

## 修复

1. `resolveAutoRunGitAuthBlock` 增加 `nestedTokenStatuses` + `autoCloneNestedRepos`：关闭时忽略子仓 token / nestedError。
2. `ProjectDetailGitReposSection` 父仓与子仓状态分列后再送入门禁。
3. 父仓 `token_error` 仍阻断；开启自动克隆时行为不变。

## 验收

```bash
cd taskFE/app && npx vitest run \
  src/utils/projectDetailAutoRunGitGate.test.js \
  src/components/ProjectDetailGitReposSection.nested-oauth.test.js
```

硬刷新项目详情：父仓已授权且自动克隆关闭时，自动运行可启用，不再出现 `project-auto-run-git-gate-hint`。
