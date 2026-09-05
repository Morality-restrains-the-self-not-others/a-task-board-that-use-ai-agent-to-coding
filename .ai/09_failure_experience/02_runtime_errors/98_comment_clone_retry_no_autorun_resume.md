# [运行时] 克隆失败手动重试成功后自动任务未继续

## 基本信息

- 版本：1.0.0
- 创建日期：2026-08-24
- 最后修改：2026-08-24
- 维护者：Trae AI 团队

## 现象

任务详情评论执行细节「项目克隆」失败（如 git exit 128），点「手动重试」后克隆成功，层图 ztree 仍只有引导克隆完成节点，自动执行的命令（auto_run / at_mention 首指令）没有被唤起。

复现：`task_879738818045964288` 引导克隆 (1/1) 失败 somanyad，手动重试成功。

无 `data-traceId`（重试成功路径不展示错误节点）。

## 根因

1. 单仓失败不 abort bootstrap，随后 `runPostBootstrapAgentKickoff` 仍调用 `createJob`。
2. `assertReposLayoutReadyForJobs` 在无 `.git` 时抛「请先完成克隆仓库」，kickoff 失败且不写 first-job 标记。
3. `POST /api/repos/reclone` 成功只同步 Git 身份，不补跑 kickoff。凭证恢复路径已有补跑，reclone 遗漏。

同类：`96_comment_clone_fail_no_manual_retry.md`（按钮缺失）、`97_comment_clone_retry_endpoint_false_start.md`（点击短路）。本条是重试成功后任务不续跑。

## 解决方案

- 引导已尝试克隆且无 git：`AGENT_KICKOFF_DEFERRED`，不建任务。
- reclone 成功：`runRecloneSuccessSideEffects` → `AGENT_KICKOFF_RESUME` + 与凭证恢复相同的 kickoff。
- 单测：`postBootstrapAgentKickoff.test.mjs` T1–T8。

## 预防

- 克隆失败可恢复路径必须与凭证恢复一样补跑 Agent 首指令。
- 失败态「可手动重试」不仅是再 git clone，还包含被中断自动任务的复原。

## 验证

```bash
cd trae-agent/onlineServiceJS && node --test src/postBootstrapAgentKickoff.test.mjs src/runtimeEventLog.test.mjs
```

存量容器须重建镜像（`DOCKER_PUSH=1 ./buildDocker.sh`）后才会带上该逻辑。
