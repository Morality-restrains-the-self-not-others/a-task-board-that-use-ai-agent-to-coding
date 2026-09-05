# 测试意图：任务级只标记运行中机器数与容器数

## 测试目标

验证任务级不再存储服务器生命周期，只维护与评论 CSC 谓词一致的机器数、容器数。

## 测试分层

| 层 | 覆盖 |
|----|------|
| 单元 | 两列重算；persist 无 comment 不写任务级；last_runtime_status 按 comment/instance；snapshot 忽略模板行 |
| 前端单元 | indicators 解析两计数；任务壳不把任务级 Starting 盖到评论卡 |
| 意图→事件 | 投影路径登记豁免，不断言 Kafka |

## 用例矩阵

| # | 场景 | 期望 |
|---|------|------|
| T1 | 评论 Running、无 server_url | 机器 1、容器 0 |
| T2 | 再登记 server_url | 机器 1、容器 1 |
| T3 | 两评论均 Running 且可达 | 机器 2、容器 2 |
| T4 | 仅 Starting | 机器 0、容器 0 |
| T5 | Running → Released | 两计数减 1 |
| T6 | persist 无 comment_id | 任务级 instance 仍空 |
| T7 | setLastRuntimeStatus 只更新该评论 | 其它评论 status 不变 |
| T8 | indicators 模板行残留 instance | 不单独点亮；布尔随两计数 |
| T9 | runtime-status 无 comment | 400 |
| T10 | 任务级有 instance、评论行 instance 空 | **不 heal**；runtime-status 仍按评论行（空 instance → 创建中/未启动）；任务级字段可被 DDL 清空 |
| T11 | DDL 后任务级行 | `instance_id`/`last_runtime_status`/`public_ip`/`server_url` 为空；两计数仅来自评论行 |

## 数据与环境

- `setupCloudTestDB`；两行评论 CSC + 一行模板。
- 不依赖真云 Describe（`mock-` instance）。

## 通过标准

上表全绿；与设计 S1–S6 对齐。
