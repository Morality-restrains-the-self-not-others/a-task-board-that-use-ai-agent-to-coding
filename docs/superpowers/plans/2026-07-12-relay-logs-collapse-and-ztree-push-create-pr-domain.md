# Domain note: PushAndCreatePullRequest / RelayLogsDefaultCollapsed

- 日期: 2026-07-12
- Bounded Context: 任务可写层 Git 协作 + 任务详情直接启动观测

## 概念

| 概念 | 类型 | 说明 |
|------|------|------|
| PushAndCreatePullRequest | Application Service | 推送层提交并可选同步等待 PR |
| wait_for_pr | Command flag | 为真时响应含 github_pull_request |
| RelayStartupLogsExpanded | UI state | 默认 false（折叠） |

无新聚合根；沿用既有 Layer / GitRemote / ExternalAsyncJob。
