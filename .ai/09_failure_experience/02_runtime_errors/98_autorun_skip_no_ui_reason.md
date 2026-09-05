# [运行时] 自动运行软跳过启服后详情页只显示「未启动」

## 基本信息

- 版本：1.0.0
- 创建日期：2026-08-17
- 最后修改：2026-08-17
- 维护者：Trae AI 团队

## 现象

任务详情评论区曾在零评论时挂 `comment-execution-details-fallback-wrap`（「执行细节 / 串行（无前序）/ 当前执行」），服务器启动状态为 **「未启动」**，且「暂无评论」。该伪执行细节已于 2026-08-17 移除；跳过原因改由评论区上方 `TaskDetailAutoRunSkipBanner` 展示。

任务辅助信息「是否自动运行」为 **是**，创建时间与任务创建成功接近，用户预期应自动启动服务器。

复现：`task_877108648822730752`（fork from `task_877083769524219904`，`auto_run=true`）。

## 根因（日志证据）

Loki `{job="task-task-service"} |= "877108648822730752"`：

```
[taskTaskService] auto_run start skipped task_id=task_877108648822730752
reason=无法获取子 Git 仓库列表：Git 授权刷新超时（无法连接 OAuth 端点）。公开仓库可匿名探测；私有仓库请检查网络后重试，或重新完成 Git 网站绑定。
```

对应 `{job="task-project-service"}` nested-git-repos 探测耗时约 **10s**（匿名 Contents 失败后 OAuth refresh 超时）。

软跳过符合 `autorun_skip_on_git_inaccessible`（fail-closed），但：

1. **Fork 路径未弹窗**：工作面板创建走 `alertAutoRunStartSkippedIfNeeded`，`forkTask` 未调用 → 用户看不到跳过原因。
2. **跳过原因未持久化**：仅写在 create/update 响应字段，GET 任务详情不含 `auto_run_start_skip_reason` → 冷打开只见「未启动」。
3. **无评论**：跳过发生在 `ensureAutoRunAtComment` 之前，故无「【自动运行】」评论、无 CSC binding、无启动日志。

## 解决方案

- `dataMigrate/taskTaskService/012_auto_run_start_skip_reason.sql`：落库 `auto_run_start_skip_reason`；GET `taskJSON` 回传 `auto_run_start_skipped` + reason。
- create/update 跳过时 `persistAutoRunStartSkipReason`；真正 schedule 前 `clearAutoRunStartSkipReason`。
- `forkTask` 调用 `alertAutoRunStartSkippedIfNeeded`。
- 任务详情页 `TaskDetailAutoRunSkipBanner`：展示原因 +「强制重新启动」（`force_auto_run`）。
- **存量兜底**：落库前已跳过的任务须在 **评论 feed 拉取完成** 后，若仍无任何评论才显示琥珀色横幅与强制重试。禁止用工作面板列表里的 `comments:[]` stub 误报「Git 探测失败」。

## 预防

- 任何 `auto_run=true` 创建入口（工作面板 / Fork / Chrome 插件）须共用 skip 弹窗。
- 软跳过字段须可冷打开回放；禁止只写响应不落库。
- 评论空态「未启动」旁须能解释「为何没启服」或提供强制重试。

## 验证

```bash
# 需 dataMigrate 012 已应用（9999 初始化）
cd taskTaskService/src && go test -count=1 -run 'TestCreateTaskAutoRunSkipsStartWhenNestedGitNeedsAuth' .
cd taskFE/app && npx vitest run \
  src/components/task-detail/TaskDetailAutoRunSkipBanner.test.js \
  src/utils/autoRunGateHints.test.js
```

公网：Fork/创建 `auto_run` 且 Git 探测失败应弹「自动运行未启服」；刷新任务详情应见琥珀色横幅与「强制重新启动」。存量空评论任务亦可见横幅。
