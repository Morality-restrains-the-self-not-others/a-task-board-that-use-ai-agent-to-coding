# ADR-0051: 任务与项目主要属性不可变历史版本

- **Status:** accepted
- **Date:** 2026-08-30
- **Author:** cursor
- **Deciders:** goal-mode auto-adopt

---

## Context

任务帖（`task_tasks`）与项目（`project_entries`）的标题/名称与正文会随协作改写，热表只保留最新值。用户需要在主要属性变更时留下可查阅记录。进度列、完成态、镜像、仓库等运行态不在「内容历史」范围内。

约束：单服务数据所有权、DDL 仅 `dataMigrate/`、时间累积表须冷热分离、写意图须投递领域事件、新增接口落 Go。

## Decision

We will persist an **append-only snapshot** of versioned fields in owner-service tables (`task_revision`, `project_revision`), written in the **same transaction** as the entity create/update when those fields actually change. Version 1 is recorded on create. Readers use paginated GET APIs with the same authorization as viewing the live entity. Domain events `TASK_REVISION_RECORDED` / `PROJECT_REVISION_RECORDED` carry identifiers only (not full body). MVP does not restore a snapshot onto the live row.

Versioned fields:

- Task: `title`, `description`
- Project: `name`, `description`

## Alternatives Considered

### Alternative 1: Event-sourcing the live row from a log

- **Pros:** 单一事实来源
- **Cons:** 热路径读放大；与现有 CRUD 热表冲突
- **Why rejected:** 现网列表/详情依赖热表；历史是旁路查阅

### Alternative 2: Audit log of diffs only

- **Pros:** 存储更小
- **Cons:** 查阅需还原整篇；易漏字段
- **Why rejected:** 「查阅」需要完整当时正文

### Alternative 3: Git-like content store / 对象存储

- **Pros:** 大文本友好
- **Cons:** 跨服务、运维复杂度过高
- **Why rejected:** 标题+描述体量适合 InnoDB + 分区

### Alternative 4: Fail-open insert (like login history)

- **Pros:** 不阻断保存
- **Cons:** 静默丢失正是本需求要消灭的缺口
- **Why rejected:** 历史是功能本身，不是旁路审计

## Consequences

### Positive

- 授权用户可按时间线查看标题/正文
- 所有权清晰，不跨库写
- 分区 + Snowflake 满足冷热与分片预备

### Negative / Trade-offs

- 存量无「变更前」快照
- 同事务使修订插入失败会回滚业务更新
- Kafka 消费者仅为观察，不驱动二次写

### Mitigations

- UI 说明存量从下次内容变更开始记
- 插入失败打 error 日志 + trace_id，便于修库后重试保存
- payload 不含正文，降低总线与日志泄露面

## References

- [设计文档](../superpowers/specs/2026-08-30-task-project-entity-revisions-design.md)
- [ADR-0001 DDL in dataMigrate](0001-ddl-must-reside-in-datamigrate.md)
- [ADR-0015 事件消费幂等](0015-event-consumer-idempotency.md)
