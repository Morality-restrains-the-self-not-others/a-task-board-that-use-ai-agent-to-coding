# 测试意图：指令完成后按工作空间闲置策略回收机器

## 测试目标

证明：task-detail 带策略分钟数；引导完成或交付成功才进入闲置；到期释放；新指令抢占；交付失败不拆机；无指令但已就绪超时仍回收。

## 测试分层

| 层 | 覆盖 |
|----|------|
| 单元 | Credential 组装 `idle_recycle_minutes`；Cloud 写/清 `instruction_idle_since`；recycle 扫描新条件；OSJS timer/interrupt/delivery gate |
| 契约 | `CONTAINER_INSTRUCTION_IDLE_MARKED` / `_CLEARED`；`CLOUD_SERVER_STOPPED` reason=`instruction_idle` |
| 不测本轮 | 真阿里云 AssumeRole；设置页 Playwright；跨评论并行机 |

## 用例矩阵

| # | 预置 | 动作 | 期望 |
|---|------|------|------|
| T1 | policy 30 分钟 | 容器 task-detail | `idle_recycle_minutes=30` |
| T2 | 无 policy 行 | task-detail | 默认 5（与 GET policy 一致） |
| T3 | policy 0 | 交付成功 | 不写 `instruction_idle_since`，不倒计时 |
| T4 | job 结束、交付失败 | 等待 N | 不 release、无 STS |
| T5 | 交付成功 | heartbeat `instruction_idle=true` | 列有值；事件 Marked |
| T6 | 闲置中新指令 | createJob/instruct | 旧 job interrupted；列清空；事件 Cleared |
| T7 | 闲置满 N，容器可达 | 倒计时到期 | `request-machine-release`；Stopped |
| T8 | 闲置满 N，容器已死 | timer recycle | 即使 `server_url` 非空也 stop |
| T9 | 卸载闲置（旧路径） | server_url 空 + idle_since | 仍回收 |
| T10 | 无 RAM Role | task-detail | 无 `machine_release_sts` |
| T11 | 有 Role、交付未成功 | 伪造到期 | 不调 DeleteInstance |
| T12 | BOOTSTRAP_COMPLETE | runtime-event | 写入 `instruction_idle_since`（列空时） |
| T13 | 无 instruction_idle_since、userdata 已过 N、无 job | recycle timer | 即使 server_url 非空也 stop |
| T14 | 无 since、userdata 已过 N、job running | recycle timer | 不释放 |
| T15 | 容器 BOOTSTRAP_COMPLETE | maybeStartIdleAfterJob idleEligible=true | 心跳 instruction_idle=true 并启动倒计时 |
| T16 | userdata 空、created_at 已过 N、无 job | recycle timer | 即使 server_url 非空也 stop |
| T17 | userdata 空、created_at CST 墙钟若当 UTC 会落在未来 | recycle timer | 按上海时区解释后过期则 stop |
| T18 | userdata 空、created_at 在窗口内 | recycle timer | 不释放 |

## 数据与环境

- 不连真 Kafka / 真 ECS；STS 用假 Role 与 mock AssumeRole。
- OSJS 测 `setTimeout` 用可注入时钟。

## 通过标准

- T1–T18 全绿；暂存相关单测 100% 通过。
