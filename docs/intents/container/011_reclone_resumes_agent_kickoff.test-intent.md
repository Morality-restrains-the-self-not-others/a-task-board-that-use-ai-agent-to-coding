# 测试意图：克隆手动重试成功后恢复被中断的自动任务

对应：`011_reclone_resumes_agent_kickoff.intent.md`

## 测试目标

证明：克隆失败会推迟 Agent 首指令；手动 reclone 成功会恢复同一 kickoff，且幂等。

## 测试分层

- 单元：`trae-agent/onlineServiceJS/src/postBootstrapAgentKickoff.test.mjs`
- 单元：`trae-agent/onlineServiceJS/src/runtimeEventLog.test.mjs`（事件名 allowlist）
- 接线扫描：同文件断言 `server.mjs` / `routesReposClone.mjs` 调用新符号

## 用例矩阵

| ID | 场景 | 期望 |
|----|------|------|
| T1 | `cloneAttempted=true` 且 `hasGit=false` | `shouldDeferAgentKickoff` 为 true；`maybeRun` 不调用 `createJobFn`，`deferred=true` |
| T2 | `cloneAttempted=true` 且 `hasGit=true` | 不推迟，调用 `createJobFn` |
| T3 | 未尝试克隆（无关联仓库） | 不因本门闩推迟 |
| T4 | `resumeAgentKickoffAfterCloneReady` + auto_run 详情 | 调用 `createJobFn`，发出 resume |
| T5 | `runRecloneSuccessSideEffects` 身份失败仍 kickoff | `kickoff` 仍被调用 |
| T6 | `runRecloneSuccessSideEffects` kickoff 抛错 | 不向外抛，记录 error |
| T7 | 已有 auto_run first-job 标记 | resume 不重复 `createJobFn` |
| T8 | 源码接线 | `server.mjs` 含 `maybeRunPostBootstrapAgentKickoff`；`routesReposClone.mjs` 含 `runRecloneSuccessSideEffects` |

## 数据与环境

- `ONLINE_PROJECT_STATE_ROOT` 临时目录；不启 HTTP；`createJobFn` 为内存 mock。

## 通过标准

`node --test src/postBootstrapAgentKickoff.test.mjs src/runtimeEventLog.test.mjs` 全绿。
