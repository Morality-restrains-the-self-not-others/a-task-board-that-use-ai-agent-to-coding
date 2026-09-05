# NFR Clarification: 任务级运行中机器/容器计数

> 价值流：`docs/superpowers/plans/2026-08-14-task-level-running-comment-server-count-value-stream.md`  
> 默认等级：L2（0-auto-flow）；非支付/鉴权新面，安全维持既有工作区门禁。

## 路径分片键强制审视

| 路径 | 分片 ID | 判定 | 可伸缩性 | 动作 |
|------|---------|------|----------|------|
| GET `.../workspace-runtime-indicators/tenant_id/{t}/workspace_id/{w}` | tenant + workspace | tenant 合适（基数高、与库隔离对齐、稳定） | L1 | 扫描已按 tenant+workspace；计数列避免二次语义扫描 |
| GET `.../server-runtime-status/...?task_id&comment_id` | tenant + workspace + task + comment | tenant 分片；comment 隔离实例 | L2 | **禁止**回退任务级 instance |
| 任务详情 `.../task-detail/{taskId}/` | tenant + workspace + task | tenant 合适 | L1 | 壳只渲染 count |
| `recomputeTaskRunningCounts` SQL | 同库 WHERE tenant+workspace+task | 非跨服务 | L0 | 同任务评论数 ≪ 千；升级触发：单任务评论 >1 万再评估 |
| 逻辑事件 `TaskRunningCountsRecomputed` | task_id（进程内） | 不适配跨分片投递 | L0 | 证据豁免：同库投影，不发 Kafka |

**Hard Gate：通过。** 可伸缩性进入类别表。

## 类别定级

| 类别 | 等级 | 说明 |
|------|------|------|
| 安全 | L2 | 复用 `ensureTenantMember`；强制 comment_id；无新公网写接口 |
| 完整性 | L2 | 同事务重算两列；谓词与 snapshot 一致 |
| 可用性 | L2 | 重算失败打日志，不阻断启动成功响应 |
| 性能 | L2 | 重算 O(评论行/任务)；indicators 单次扫描 |
| 可伸缩性 | L1 | 路径带 tenant；计数列小整数 L0 |
| 可观测性 | L2 | `event=task_running_counts_updated` + task_id + machines + containers；禁 instance 当任务级权威 |

## 质量场景

1. **Starting 不计**：评论 Starting → 机器 0 容器 0。  
2. **Running 无 URL**：机器 1 容器 0。  
3. **无 comment persist**：任务级 instance 仍空。  
4. **仅任务级旧 instance**：不 heal；runtime-status 按空评论行。  
5. **跨租户**：他租户 GET indicators → 403。

## 领域模型影响

- 任务级行是 **投影**，不是运行聚合根。  
- 评论 CSC 为运行一致性边界。  
- 计数更新进程内，不引入跨上下文消息。

## 权衡

- 不做 Kafka、不做独立计数表、不做存量 heal。
