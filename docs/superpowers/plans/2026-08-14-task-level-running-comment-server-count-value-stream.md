# 价值流：任务级只标记运行中机器数与容器数

> 设计：`docs/superpowers/specs/2026-08-14-task-level-running-comment-server-count-design.md`

## Related Value Streams

- `comment-binding-recover-after-start-success`（`cloud-integration` 步骤）：启动成功回填评论 binding。本流**扩展**：启动/停机后重算任务两计数，不再写任务级 instance。
- 工作区机器摘要 / `workspace-runtime-indicators`：本流改读路径——忽略模板行 instance/status，布尔随两计数。
- 不冲突：不撤销评论级 CSC；删除 heal 兼容路径。

## 价值阶段

| 阶段 | 类型 | 说明 |
|------|------|------|
| 评论 CSC 写运行态 | Essential | persist / last_runtime_status / URL 只写评论行 |
| 任务级两计数重算 | Core | 模板行标记 machines/containers |
| 看板/任务壳读 N/M | Core | 用户看到数量而非一份 Starting/Running |
| DDL 清空任务级旧运行态 | Essential | 不兼容存量 |

## 端到端流

Trigger：评论机器进入/离开运行中（启动回填、Describe、停机、释放、登记 URL）  
→ 只 UPDATE 该评论 CSC  
→ `recomputeTaskRunningCounts` 写模板行两列  
→ indicators / 任务壳读 N、M  
Delivery：看板环/点随计数；任务壳「N 台机器 / M 个容器运行中」

## 增量切片

1. **I1 计数投影** — DDL + 重算 + 禁止任务级写 instance/status + 删 heal  
2. **I2 读路径** — snapshot/indicators 忽略模板行；JSON 带两计数  
3. **I3 前端** — 归一两计数；任务壳展示 N/M  

配置：`conf/value-stream.yaml` 流 `task-running-comment-server-count`
