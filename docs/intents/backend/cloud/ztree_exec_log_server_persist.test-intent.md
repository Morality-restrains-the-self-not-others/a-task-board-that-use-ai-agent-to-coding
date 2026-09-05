# 测试意图：ztree 执行日志服务端持久化

## 测试目标

覆盖层图快照 UPSERT、容器关闭后 GET hydrate；明确 **不** 持久化克隆日志。

## 测试分层

| 层 | 范围 |
|---|---|
| 单元 | Cloud snapshot UPSERT / GET |
| 单元 | FE：`containerEndpointRegistered=false` 仍请求 layer-graph 与 job-execution-log 并渲染 |
| 意图事件 | layer-graph-push 成功路径发布 LayerGraphSnapshotPersisted |

## 用例矩阵

| ID | 场景 | 期望 |
|---|---|---|
| T1 | layer-graph-push 缺 layers/jobs | 400，不写快照 |
| T2 | 合法 push | UPSERT 一行；发布 LayerGraphSnapshotPersisted；SSE 仍发 |
| T3 | 同 comment 再 push | 行数仍 1，graph_json 为后者 |
| T4 | GET layer-graph 无容器 | 200，body 来自快照 |
| T5 | GET 缺 workspace 或 task | 400 |
| T6 | FE endpoint=false | 仍 GET 层图与 job 步骤；克隆区允许空 |
| T7 | 无 clone-log 写库路径 | 不存在 exec-stream-push / cloud_exec_stream_segment |
| T8 | 意图发生时事件被投递 | T2 断言 publish |

## 数据与环境

- Cloud：`setupCloudTestDB` + 快照 DDL
- FE：vitest mock fetch

## 通过标准

T1–T8 全绿。
