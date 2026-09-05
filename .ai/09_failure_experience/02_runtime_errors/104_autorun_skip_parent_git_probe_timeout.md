# [运行时] 已授权 GitLab 父仓因探测超时被误报「探测失败」并跳过自动启服

## 基本信息

- 日期：2026-08-21
- 页面：任务详情 `data-testid="auto-run-start-skipped-banner"`
- 任务：`task_878541740905099264`
- 父仓：`https://gitlab-tencent-sh-1.daydaymoney.com/example-user/ram-work`（用户已有访问权限）
- 关联：失败经验 98（跳过原因落库）、100（OAuth refresh 超时）、102（关闭自动克隆时勿用子仓门禁）

## 现象

琥珀色横幅：

> 自动运行已开启，但未自动启动服务器无法验证父 Git 仓库访问：探测失败，已跳过自动启动服务器强制重新启动

任务辅助信息可见仓库 `…/ram-work→main`，项目侧父仓已授权。

## 根因（本地 logs/，Loki :3100 本会话不可达）

时间线（2026-08-21 15:31 CST）：

| 时间 | 服务 | 事件 |
|------|------|------|
| 15:31:13 | task-task-service | 分配 workspace_seq，开始同步 Git 探测（create POST 总耗时 16042ms） |
| 15:31:17 | task-git-oauth | `POST /api/internal/gitlab/oauth/access-for-user/` **200** duration_ms=3590（有票） |
| 15:31:28 | task-task-service | `parent-git probe transport error`：`Client.Timeout exceeded while awaiting headers`（`projectHTTP` **15s**） |
| 15:31:28 | task-task-service | `auto_run start skipped` + 笼统「探测失败」 |
| 15:31:37 | task-project-service | `POST …/validate-git-repos` **200** duration_ms=**24177** |

父仓 OAuth 与远程探测最终都成功；task 侧 15s 客户端先放弃，把运输超时映射成「无权限」。`probe_access=true` 会打 GitLab REST，延迟叠加 refresh 后轻易超过 15s。

## 修复

1. `auto_clone_nested_repos=false` 时父仓走 **token-only**（`probe_access=false`），与 OPT-20260818-047「token 探测」及 UI「已授权」一致。
2. Git 探测专用 `projectGitProbeHTTP` 超时 **40s**（nested-git 路径同样使用，避免 15s vs gitHTTPClient 25s 同类误杀）。
3. 超时 skip 文案含「超时」；非 200 带 HTTP 状态码。
4. 回归：`auto_run_parent_probe_test.go`。

## 验收

```bash
cd taskTaskService && go test ./src -count=1 -run 'TestProbeParentGitReposAccessUsesTokenOnly|TestProjectGitProbeTimeoutCoversObservedRemoteLatency|TestParentGitProbeFailureReasonTimeout|TestProbeGitAccessForAutoRun'
```

存量已跳过任务：详情页点「强制重新启动」（`force_auto_run`）。精准编译重启 `task-task-service` 后新建 auto_run 任务不应再因 15s 超时误跳过。
