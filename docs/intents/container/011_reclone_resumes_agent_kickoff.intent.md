# 意图：克隆手动重试成功后恢复被中断的自动任务

- **日期**: 2026-08-24
- **状态**: 已实现
- **页面**: 任务详情「云端开发」评论执行细节：项目克隆失败行 + 「手动重试」+ 层图 ztree

## 背景与目标

引导克隆失败时（例如 git exit 128），容器仍完成 `BOOTSTRAP_COMPLETE` 并保持业务端点就绪，失败仓可 `POST /api/repos/reclone` 手动重试。但 `createJob` 在无 `.git` 时拒绝建任务（「请先完成克隆仓库」），bootstrap 后的 Agent kickoff（at_mention / auto_run 首指令）因此失败；reclone 成功路径只同步 Git 身份，**不再唤起**被中断的自动执行命令。

目标：克隆失败视为自动任务被中断；手动重试成功视为复原，必须继续此前应执行的首指令。

## 范围与边界

- **范围内**：onlineServiceJS bootstrap kickoff 门闩；`POST /api/repos/reclone` 成功后补跑与凭证恢复相同的 kickoff；结构化运行时事件。
- **范围外**：前端「手动重试」按钮本身（已有 F-034）；reclone HTTPS 规范化（C-002）；空工作区 `POST /api/repos/clone` 新建层后的 kickoff（记 OPT）。

## 约束与风险

- kickoff 仍走既有幂等标记（`auto_run_first_job.json` / at_mention marker）：部分仓已成功并已建任务时，补跑不得重复创建。
- reclone 已 202 后台执行；kickoff 失败不得把克隆结果改成失败，只记日志。
- 引导仍不因单仓失败 abort（既有策略）。

## 验收标准

- [x] 引导已尝试克隆且层内无 git 时：不调用 `createJob`，发出 `AGENT_KICKOFF_DEFERRED`
- [x] 引导克隆成功（有 git）时：行为与现网一致，立即 kickoff
- [x] `POST /api/repos/reclone` 成功后：调用与凭证恢复相同的 kickoff（`AGENT_KICKOFF_RESUME` reason=reclone）
- [x] reclone 失败：不 kickoff
- [x] 已有 first-job 标记时补跑不重复建任务

## 业务意图 → 事件对照

| 业务意图 | 事件名（过去式） | MQ类型/契约 | 发布点 | 消费者/副作用 | 例外理由 |
|---------|----------------|------------|--------|--------------|---------|
| 克隆失败推迟 Agent 首指令 | AgentKickoffDeferred | AGENT_KICKOFF_DEFERRED | onlineServiceJS `maybeRunPostBootstrapAgentKickoff` | Loki/runtime-event；等待 reclone | 证据豁免：容器内 runtime-event HTTP，非 Kafka |
| 重试克隆成功后恢复首指令 | AgentKickoffResumed | AGENT_KICKOFF_RESUME | onlineServiceJS `resumeAgentKickoffAfterCloneReady` | 创建 trae job / 层图任务 | 证据豁免：容器内 runtime-event HTTP，非 Kafka |

## 实施计划

1. 抽出 `shouldDeferAgentKickoff` / `maybeRunPostBootstrapAgentKickoff` / `resumeAgentKickoffAfterCloneReady`
2. `server.mjs` listen 后 kickoff 改走 defer 门闩
3. reclone 成功副作用改为身份同步 + kickoff 恢复
4. 凭证恢复路径复用同一 resume 函数
