# [运行时] 子 Git 仓库列表误报「未检测到可用授权」（GitHub OAuth refresh 超时）

- **日期**：2026-08-10
- **页面**：任务详情 → 关联项目 → 子 Git 仓库（`data-testid="task-nested-repos-clone-error"`）
- **项目示例**：`proj_-2965951621741476751`（`https://github.com/ruandao/somanyad`，公开仓）
- **traceId**：`9bbb77e0-e27d-4dfd-88c1-dfb75753d862`

## 现象

1. 任务详情红字：`无法获取子 Git 仓库列表：未检测到可用授权。请先在个人资料完成 Git 网站绑定…`
2. 用户侧已完成 GitHub 绑定（`user-app-connection` 200）
3. 请求耗时约 **15s** 后才返回业务错误（HTTP 仍 200）

## 日志时间线

| 时间 | 服务 | 事件 |
|------|------|------|
| 22:25:58 | task-auth | forward-auth 200 |
| 22:25:58 | task-project-service | nested-git-repos parent=`https://github.com/ruandao/somanyad` |
| 22:26:14 | task-git-oauth | `POST /api/internal/github/oauth/access-for-user/` **502**（15252ms） |
| 22:26:14 | task-git-oauth | refresh 失败：`Post "https://github.com/login/oauth/access_token": http2: timeout awaiting response headers` |
| 22:26:13 | task-project-service | nested-git-repos http_request 200 duration_ms=15003 |

## 根因

1. **出站分化**：本机 `api.github.com` 可达（~2s），但 `github.com/login/oauth/access_token` **超时不可达**。
2. **强制先换票**：`listNestedGitRepos` 在 token 为空时直接返回 `nestedNeedsAuthMessage`，且丢弃 `fetchGitAccessToken` 的 `tokenErr`。
3. **文案误导**：refresh 超时 / 502 被折叠成「未检测到可用授权 / 请先绑定」，掩盖真实网络问题。
4. **公开仓本可匿名**：父仓 `private=false` 且无 `.gitmodules` 时，匿名 Contents API 即可得出空列表，无需 OAuth。

## 与相关条目区分

- [49_github_dot_git_misclassified_as_gitlab.md](./49_github_dot_git_misclassified_as_gitlab.md)：provider 误判走 GitLab API。
- [87_github_oauth_exchange_failed_dead_proxy.md](./87_github_oauth_exchange_failed_dead_proxy.md)：换票路径死代理 / 超长超时。
- 本条：**refresh 端点不可达 + nested-git 错误映射错误**，公开仓被误报未绑定。

## 修复（taskProjectService）

1. GitHub nested-git：**先匿名**探测 `.gitmodules`，成功即返回（跳过 OAuth refresh）。
2. 匿名失败后再取 token；token 失败时用 `nestedAuthFailureMessage(tokenErr)` 区分 unbound / timeout / 502。
3. 举一反三：`listGitHubBranches` / `resolveGitHubCommit` 同样支持匿名公开仓回退；`fetchGitHub*` 统一走 `githubAPIBase`。
4. taskGitOauth：日志 `uid=%d` → `%s`（user_id 为 string，避免 `%!d(string=…)`）。

## 验收

```bash
curl -sS "http://127.0.0.1:8016/api/internal/nested-git-repos/?user_id=<uid>&repo_url=https://github.com/ruandao/somanyad" \
  | python3 -c 'import sys,json;d=json.load(sys.stdin); assert d.get("error")==""'
cd taskProjectService && go test ./src/ -count=1 \
  -run 'ListNestedGitReposGitHubAnonymous|ListNestedGitReposGitHubPrivate|ListGitHubBranchesAnonymous|NestedAuthFailure'
```

页面硬刷新后：公开父仓无 `.gitmodules` 时子仓区为空列表，不再出现「未检测到可用授权」红字。
