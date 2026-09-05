# 测试意图：容器执行步骤 Kafka → SSE → 落库 → 前端 hydrate

## 测试目标

覆盖「容器 PUSH → Kafka SSE_MESSAGE → SSE 直播 + 消费者落库 → 前端先拉 DB」成功路径与关键负例。

## 测试分层

| 层 | 范围 |
|---|---|
| 单元 | Cloud inbound 组 SSE 载荷并发布；persist 过滤 phase；GET 从行还原 steps |
| 单元 | taskEvents persist handler 跳过非 job-stream / chunk；调用 Cloud internal |
| 单元 | 容器 `publishJobStreamEventToSaas` POST 路径与 body |
| 单元 | 前端 watcher 在 endpoint 未就绪时仍 refresh；poller 不再启动 interval |
| 单元 | Gateway `maybeStartJobStream` 默认不启动 poll |

## 用例矩阵

| ID | 场景 | 期望 |
|---|---|---|
| T1 | job-stream-push 缺 job_id | 400，不发布 Kafka |
| T2 | job-stream-push 合法 step | 发布 SSE_MESSAGE，`status=container_job_stream`，含 seq/comment_id；HTTP 200 |
| T3 | Kafka publish 失败 | HTTP 502 |
| T4 | persist chunk | DispatchSuccess，不调 persist API |
| T5 | persist step | 调用 internal persist，幂等同 seq |
| T6 | GET log 有落库 step | 200，`steps.steps[].step_number` 对齐，无需 Gateway |
| T7 | 容器 recordJobEvent(step) | POST `.../job-stream-push/` |
| T8 | 前端 job 选中且 endpoint=false | 仍调用 refreshZTreeExecutionLog |
| T9 | activeJobExecLogPoller.sync | 不 setInterval |
| T10 | 意图发生时事件被投递 | T2 断言 publish 被调用（SSE_MESSAGE） |

## 数据与环境

- Cloud：`setupCloudTestDB` + `023` 迁移
- Events persist：注入 Persist 函数，不连真实 Kafka
- 容器：本地 http mock SaaS

## 通过标准

上表 T1–T10 全绿；`conf/value-stream.yaml` 步骤指向对应测试文件。
