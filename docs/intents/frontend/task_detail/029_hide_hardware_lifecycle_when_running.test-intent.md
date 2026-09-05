# 测试意图：硬件卡仅启动配置、不关联运行态

## 对应意图

`029_hide_hardware_lifecycle_when_running.intent.md`

## 测试目标

任务详情硬件卡在任何运行态下都不出现启停跳转；运行态只在评论执行细节。

## 测试分层

| 层 | 位置 |
|----|------|
| 单元 | `ServerConfigHardwarePanel` 壳 / `hardwarePanelSplit.test.js` |
| Playwright | 任务详情 mock：硬件卡无按钮；评论运行区仍可停止 |

## 用例矩阵

| ID | 测点 | 类型 | 位置 |
|----|------|------|------|
| T1 | 壳层不再挂载 `HardwareServerControls` | 单元 | `hardwarePanelSplit.test.js` |
| T2 | mount 后无 `#start-server-btn` / `start-server-disabled-reason`（含 mock 运行中） | 单元 | HardwarePanel mount |
| T3 | 评论执行细节仍有停止 / 跳转 | Playwright | `TaskDetail.server-lifecycle-stop-vm` 改锚 |
| T4 | 原 `#start-server-btn` 启动 E2E 改走评论 `@镜像` | Playwright | `TaskDetail.start-vm-*` |
| T5 | 临时配置 POST comments 带 `server_run_template` | 单元 | `taskDetailFetchFns` / composer |
| T6 | 事件带模版时消费者不再读项目模版 | 单元 | taskEvents handler |
| T7 | 无模版字段时仍走项目模版 | 单元 | 同上 |
| T8 | 临时配置不齐仍 POST 评论、body 无残缺 `server_run_template` | 单元 | `taskDetailFetchFns.imageMention.test.js` |

## 数据与环境

Mock 运行中与已停止两种任务详情。不打真实云。

## 通过标准

- T1–T2 单测绿
- T3–T4 mock Playwright 绿或明确改锚
- 「无对应事件」行不要求 MQ 断言
