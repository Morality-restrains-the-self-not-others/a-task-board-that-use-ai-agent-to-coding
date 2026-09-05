# NFR：终态按评论 CSC 释放

- 日期：2026-08-14
- 设计：`docs/superpowers/specs/2026-08-14-terminal-release-comment-csc-design.md`
- 价值流：`docs/superpowers/plans/2026-08-14-terminal-release-comment-csc-value-stream.md`
- 默认等级：L2（非支付/鉴权域）

## 路径分片键强制审视

| 路径 | 分片 ID | 适配？ | 动作 |
|------|---------|--------|------|
| GET list-by-task?tenant_id=&workspace_id=&task_id= | tenant_id | 是（租户隔离、与 CSC 查询一致） | 强制 query 带 tenant_id+task_id |
| Kafka TASK_STATUS_CHANGED key=task_id | tenant 在 payload | 键为 task_id（与现网一致） | 不改分区键；查询必带 tenant |
| Kafka CLOUD_SERVER_STOPPED key=task:comment | tenant 在 payload | 评论级释放，避免互盖 | 新 key 格式 |

可伸缩性：**L1**（按 tenant 查询；单任务评论数预期 ≪ 100）。升级触发：单任务评论 CSC > 500 再分页。

## 类别定级

| 类别 | 级 | 说明 |
|------|----|------|
| 可伸缩性 | L1 | 见上 |
| 一致性 | L2 | 停机事件最终一致；部分成功依赖幂等 |
| 可用性 | L2 | list 失败 DispatchRetryable；无资源不重试 |
| 安全 | L2 | internal secret；边界字段 |
| 可观测性 | L2 | stage=no_running_resource_skip / terminal_release_comment_csc |
| 性能 | L1 | 单次 SELECT 按 task_id |

## 质量场景

1. 刺激：终态 + 计数 0 → 响应：<100ms success，无 DLT。
2. 刺激：两评论 Running → 响应：两条停机事件，各带 comment_id。
3. 刺激：list 5xx → 响应：DispatchRetryable。

## 领域模型影响

- 释放对象是 CommentCloudServerConfig 集合，不是 Task 级单例。
- 无新聚合事务；逐评论发布事件。
