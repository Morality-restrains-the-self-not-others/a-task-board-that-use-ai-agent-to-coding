# auto_run @ 评论 PR 回填 — 实施计划

## 爆炸半径与测试缺口

- TTS：`triggerTaskAutoRun`、评论插入、`notifyContainerAgentPending`
- onlineServiceJS：kickoff、createJob 字段、delivery → complete
- 测试缺口：无端到端 PR 回填单测（本次补 Node + Go）

## 任务

- [x] P1 TTS `ensureAutoRunAtComment` + 单测（T1/T2）
- [x] P2 `notifyContainerAgentPending` 支持 `source`
- [x] P3 kickoff 分流 + createJob 持久化 mount ids（T3/T4）
- [x] P4 `extractPrUrlsFromPushResult` + `backfillAutoRunPrToAgentComment`（T5/T6）
- [x] P5 意图/测试意图更新
- [x] P6 Review + Ship PR
