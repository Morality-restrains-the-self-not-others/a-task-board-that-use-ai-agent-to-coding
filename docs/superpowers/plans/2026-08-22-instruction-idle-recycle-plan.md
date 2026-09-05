# 实施计划 — 指令闲置回收

> 设计 + 价值流 + NFR + DDD。TDD：每任务先测后实现。

## Increment 1 — task-detail 分钟数

- [ ] Cloud: `038_instruction_idle.sql` 加 `instruction_idle_since`（CSC）与 CPA `sts_release_role_arn`
- [ ] Cloud: GET `/api/internal/cloud/workspace-machine-policy/` + 单测
- [ ] Credential: PolicyFetcher HTTP（Proxy=nil）+ FetchTaskDetail 字段单测 T1/T2
- [ ] handlers 响应 `idle_recycle_minutes` + `instruction_idle`
- [ ] OpenAPI internal + 容器 skill.md
- [ ] conf `task_cloud_service_base`（同节点 8018）

## Increment 2 — Mark/Clear + 抢占

- [ ] Cloud heartbeat 解析 `instruction_idle`；Mark/Clear + 事件
- [ ] OSJS `instructionIdle.mjs`：createJob 先 interrupt 全部 running/pending 并 Clear
- [ ] heartbeat 带 instruction_idle 标志

## Increment 3 — L1 + L2 释放

- [ ] recycle 扫描 instruction_idle_since，即使 server_url 非空也 `releaseMachineForTerminal`（T8）
- [ ] 保留卸载 idle 路径（T9）
- [ ] OSJS 倒计时到期 `postRequestMachineRelease({ reason: 'instruction_idle' })`
- [ ] EventTopic 登记新事件类型

## Increment 4 — 交付门闩 + STS 可选

- [ ] finalizeJobCloseSideEffects：交付失败 `idleEligible=false`（T4）
- [ ] 无 Role 则无 machine_release_sts（T10）
- [ ] session policy 单测；L3 仅 L1 失败且 idleEligible

## 事件契约任务

- [ ] `CONTAINER_INSTRUCTION_IDLE_MARKED` / `_CLEARED` publish + topic map
- [ ] 释放仍 `CLOUD_SERVER_STOPPED`
- [ ] 更新 `docs/intents/backend/cloud/instruction_idle_auto_recycle.intent.md`

## 验证命令

```bash
cd taskCloudService && go test ./src -count=1 -timeout 120s -run 'Policy|Idle|Recycle|Heartbeat'
cd taskCredentialService && go test ./application ./interfaces -count=1 -timeout 60s -run TaskDetail
cd trae-agent/onlineServiceJS && node --test src/instructionIdle.test.mjs src/jobsRuntimeCloseSideEffects.test.mjs
```
