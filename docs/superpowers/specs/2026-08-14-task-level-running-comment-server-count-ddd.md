# DDD：任务级运行中计数投影

## Bounded Context

云资源（taskCloudService）— 评论级运行实例 + 任务级模板/投影。

## 值对象

- `RunningCounts`：Machines int、Containers int（≥0）
- `CommentRuntime`：CommentID + MachineStarted + ContainerReachable（由既有谓词分类后传入）

## 聚合

- **CommentCloudServer**（评论 CSC）：运行态权威（instance / last_runtime_status / URL）
- **TaskCloudServerTemplate**（`comment_id=''`）：平台模板 + `RunningCounts` 投影；**不是**运行实例

## 领域服务

- `ComputeRunningCounts(rows []CommentRuntime) RunningCounts` — 纯函数，只计 `comment_id!=''`

## 端口

- `TaskRunningCountStore.Write(tenant, workspace, task, RunningCounts)` — 基础设施 UPDATE 模板行
- 不新增 EventBus 端口：投影豁免 Kafka（见意图）

## 领域事件（逻辑）

| 事件 | 触发 | 投递 |
|------|------|------|
| TaskRunningCountsRecomputed | 评论运行态/URL 变更后重算 | 进程内写列 + 结构化日志；`publish_evidence_exempt` |

## 适配器

- MySQL：`recomputeTaskRunningCounts` 读评论行 → domain.Compute → UPDATE 模板行  
- 读：snapshot 忽略模板行 instance/status，用同一谓词或已存两列
