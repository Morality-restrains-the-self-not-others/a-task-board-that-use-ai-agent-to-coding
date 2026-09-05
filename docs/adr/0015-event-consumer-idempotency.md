# ADR-0015: 领域事件消费采用 at-least-once + 幂等处理

- **Status:** accepted
- **Date:** 2026-08-18
- **Author:** Trae AI
- **Deciders:** 工程团队

---

## Context

本仓库领域事件经 Kafka 投递，由 `taskEvents` intent worker 消费。Kafka 消费语义是 **at-least-once**：生产者重试、消费者超时重投、rebalance、以及「已处理但尚未 Ack」的崩溃，都会让同一条业务意图被 handler 看到两次。

若把「broker 只投一次」当成正确性前提，会出现重复扣费、重复停机、重复发信。若用过粗的租户/用户字段当去重键，又会把后续独立意图静默吞掉（失败经验 16：`company_id`；失败经验 64：`user_id`）。

需要在「Kafka 事务 exactly-once」与「业务层幂等」之间做架构选择，并规定唯一消费入口。

## Decision

We will treat event consumption as **at-least-once delivery + idempotent dispatch = effectively-once**:

1. All Kafka consumers run only in `taskEvents`, through `eventbin.RunIntent` → `IdempotentDispatchService`.
2. Idempotency keys match the **business duplicate boundary** (`event_id` / request id / natural business key). We will not default to `company_id` / `tenant_id` / `user_id`.
3. Duplicate delivery is a successful no-op (Ack) with a `warn` `idempotency skip` log (key fingerprint only).
4. We will **not** rely on Kafka transactions / EOS as the correctness mechanism.
5. Process-local `MemoryStore` is the default fast path; funds/quota/cloud (NFR ≥ L3) must still have a DB unique constraint on the owner service.

## Alternatives Considered

### Alternative 1: Kafka exactly-once semantics (EOS / transactions)

- **Pros:** Broker-level once; less application code
- **Cons:** Operational complexity; does not cover producer-side duplicate *business* publishes with different offsets; poor fit for HTTP fan-out already committed in owner services
- **Why rejected:** Effectively-once at the business key is the actual requirement; EOS would not fix republished events with a new offset

### Alternative 2: Offset commit as the only de-duplication

- **Pros:** Simple
- **Cons:** Crash between side effect and commit replays the message; rebalance can redeliver
- **Why rejected:** Offset is a transport cursor, not a business idempotency key

### Alternative 3: Ad-hoc consumers inside each business service

- **Pros:** Fewer hops
- **Cons:** Each service reimplements consume/retry/DLT/idempotency; violates single consume path and poll-loop isolation
- **Why rejected:** Shared runner already exists; duplication has caused silent skips

## Consequences

### Positive

- Duplicate Kafka deliveries cannot double-apply side effects when the key is correct
- One review checklist for all new intents (`RunIntent` + keyFn + replay tests)
- Silent skip becomes observable (`idempotency skip` + fingerprint)

### Negative / Trade-offs

- `MemoryStore` is lost on process restart; a replay after restart may run the handler again
- Dedicated key functions are required for events that always carry tenant/user fields
- L3 money/resource paths need an extra unique constraint in the owner DB

### Mitigations

- Handlers should be naturally idempotent (state transfer, not `increment()`)
- L3 paths persist idempotency in the owner service (unique key / idempotency table)
- CI `check_event_consumer_idempotency.py` blocks Reader bypass, nil keyFn, and coarse generic field order
- NFR 元规则 48 still requires documenting the key before DDD

## References

- `.ai/01_project_constraints/54_event_consumer_idempotency.md`
- `.ai/01_project_constraints/53_nfr_idempotency.md`
- `.cursor/rules/dead-letter-topic.mdc`
- `.ai/09_failure_experience/02_runtime_errors/16_stop_server_idempotency_company_id_skip.md`
- `.ai/09_failure_experience/02_runtime_errors/64_task_created_idempotency_collapses_on_user_id.md`
- [ADR-0011: 业务服务禁止进程内轮询](0011-no-service-internal-poll-loop.md)
