# [运行时] 自动运行跳过：GitHub OAuth Refresh 无重试（私有仓 nested-git）

## 基本信息

- 日期：2026-08-17
- 页面：任务详情 `data-testid="auto-run-start-skipped-reason"`
- 任务：`task_877171547301769216`（父仓 `https://github.com/task2money/ram-work`，私有）
- 关联：失败经验 88（公开仓匿名优先）、90（dial fallback）、98（跳过原因落库横幅）

## 现象

琥珀色横幅：

> 无法获取子 Git 仓库列表：Git 授权刷新超时（无法连接 OAuth 端点）。公开仓库可匿名探测；私有仓库请检查网络后重试，或重新完成 Git 网站绑定。

Loki：

| 时间 | 服务 | 事件 |
|------|------|------|
| 20:46:33 | task-project-service | nested-git-repos parent=`https://github.com/task2money/ram-work` |
| 20:46:43 | task-git-oauth | access-for-user refresh 失败：`timeout awaiting response headers`（~8s） |
| 20:46:43 | task-task-service | `auto_run start skipped` + 上述文案 |

## 根因

1. 父仓匿名 `api.github.com` 返回 404（私有）→ 必须 OAuth refresh。
2. `github.com/login/oauth/access_token` 边缘 IP 偶发「TCP/TLS 通但 HTTP 挂起」。
3. `ExchangeGitHubCode` 已有直连重试 + dial 轮转；**`RefreshGitHubToken`（access-for-user 唯一路径）无同 client 重试**，一次挂起即 502。
4. `ResponseHeaderTimeout=8s` 占满 project 侧探测预算，无法再试下一 IP。

## 修复

1. `RefreshGitHubToken`：与 Exchange 对齐，无 proxy 时最多 3 次直连尝试；失败调用 `ReportGithubDialHTTPFailure` 轮转优先 IP；保留首错。
2. `ResponseHeaderTimeout` 8s → 4s（正常换票 <1s，给重试留预算）。
3. `taskProjectService` `gitHTTPClient.Timeout` 15s → 25s；conf `gitoauth_timeout_seconds` 文档值改为 25。
4. 回归：`TestRefreshGitHubTokenRetriesAfterHeaderTimeout`。
5. 行数：`oauth_clients.go` 拆出 `oauth_gitlab.go` / `oauth_resolve.go`。

## 验收

```bash
cd taskGitOauth && go test ./infrastructure/ -count=1 -run TestRefreshGitHubTokenRetriesAfterHeaderTimeout
# 精准重启 task-git-oauth + task-project-service 后：
curl -sS "http://127.0.0.1:8016/api/internal/nested-git-repos/?user_id=<uid>&repo_url=https://github.com/task2money/ram-work" \
  | python3 -c 'import sys,json;d=json.load(sys.stdin); assert d.get("error")==""'
```

存量已跳过任务：详情页点「强制重新启动」（`force_auto_run`）即可；根因修复后探测应通过。
