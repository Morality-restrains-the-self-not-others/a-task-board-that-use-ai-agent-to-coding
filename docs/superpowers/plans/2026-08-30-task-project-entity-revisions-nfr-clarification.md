# NFR 澄清：任务与项目内容历史版本

- 日期：2026-08-30
- 设计：`docs/superpowers/specs/2026-08-30-task-project-entity-revisions-design.md`
- 价值流：`docs/superpowers/plans/2026-08-30-task-project-entity-revisions-value-stream.md`
- 默认级别：L2 Standard；资金路径不适用

## 路径分片键强制审视

| 路径 | 分片 ID | 是否合适 | 级别 | 动作 |
|------|---------|----------|------|------|
| GET/写入 task revisions（path tenant_id + workspace_id + taskId） | tenant_id + task_id | 是：与表所有权/查询对齐；列表必带 task_id | L2 | 禁止无 task_id 的租户级扫全表 |
| GET/写入 project revisions（path tenant_id + projectId） | tenant_id + project_id | 是 | L2 | 禁止无 project_id 扫描 |
| Kafka TASK_REVISION_RECORDED key=task_id | task_id | 是（实体级；观察消费） | L2 | 不用 user_id |
| Kafka PROJECT_REVISION_RECORDED key=project_id | project_id | 是 | L2 | 不用 tenant_id 作幂等键 |
| FE `/tenant/:tid/workspace/:wid/task/:taskId` 历史面板 | 路由含 tenant/workspace/task | 是 | L0 路由 | 不新增无 ID 路由 |

升级触发：单实体 revisions > 1e6 行或跨任务检索需求 → 再评估归档/二级索引。

## 幂等性强制审视

| 路径 | 副作用 | 重复触发源 | 业务重复边界 | 幂等键 | 重放语义 | 级别 |
|------|--------|------------|--------------|--------|----------|------|
| 创建任务 + insert v1 | 写表 + 事件 | 双击/重试 POST | 一个 task 的初始快照 | 任务创建 Idempotency-Key；revision 行随 task_id 唯一 v1 | 创建已存在则不二次 insert v1 | L2 |
| PATCH 内容变更 + insert vN | 写表 + 事件 | 重复 PATCH 同文案 | 同一 task 同一内容状态 | 内容相等则 0 行；事件 revision_id | 同标题正文不新增版 | L2 |
| GET list/detail | 无 | — | — | — | — | L0 纯查询 |
| 消费 1_observe | 仅日志 | at-least-once Kafka | 一次 revision 观察 | revision_id | 第二次 Ack 空操作 | L2 |
| 用户点击打开面板 | 无写 | — | — | Anti-Replay-OK GET | — | L0 |

禁止用 `tenant_id` / `user_id` 作消费幂等键。

## 类别支撑程度

| 类别 | 级别 | 说明 |
|------|------|------|
| 可伸缩性 | L2 | 月分区；查询必带 entity_id |
| 数据一致性 | L2 | 同事务 fail-closed；事件在 commit 后 |
| 安全 | L2 | 与实体相同 ACL；IDOR 404 |
| 性能 | L2 | 列表 limit≤100；打开面板才拉 |
| 可观测性 | L2 | 结构化 event + trace_id |
| 可用性 | L2 | 插入失败则保存失败，避免静默丢历史 |

## 质量场景

1. **刺激**：成员改标题并保存。**响应**：热表与新 revision 同事务可见；事件带 revision_id。
2. **刺激**：无权限 GET。**响应**：403，不泄漏其它租户行。
3. **刺激**：Kafka 重放同一 revision_id。**响应**：无第二条业务副作用。

## 领域模型影响

- Aggregate：Task / Project 根；Revision 是实体内不可变集合，不跨聚合写
- 分区键 created_at 在 PK 中
- 事件不含正文以控制总线负载
