# ADR-0029: 推荐分账资格可取消且操作必须带理由审计

- **Status:** accepted
- **Date:** 2026-08-22
- **Author:** cursor
- **Deciders:** goal-mode 自动采用

---

## Context

超管在「推荐码申请」页只能通过或拒绝待审批申请。已授予的分账资格无法人工收回；通过不记理由，拒绝理由可选；审计只靠结构化日志，无法在管理页按申请检索。分账属于资金路径，需要可追责记录。

## Decision

We will:

1. 在 `referral_code.status` 增加 `revoked`，由超管对活跃资格执行取消。
2. 通过 / 拒绝 / 取消均强制 `reason`（8–500 字），并写入 `referral_qualification_audit`（追加型，不更新历史行）。
3. 取消后调用 taskBill 将该推荐人边的 `commission_eligible` 置 0，并作废 **pending** 计提；不追回 **settled** 佣金。
4. 微信分账接收方删除为 best-effort，不作为资格取消的提交条件。

## Alternatives Considered

### Alternative 1: 把资格改成立即过期（status=expired）

- **Pros:** 不改枚举
- **Cons:** 与自然过期无法区分，审计语义混乱
- **Why rejected:** 运营需要区分「到期」与「人工取消」

### Alternative 2: 仅打日志、不建审计表

- **Pros:** 无 DDL
- **Cons:** Loki 不便于按申请 ID 给超管查阅
- **Why rejected:** 需求明确要求记录与理由以便审计

### Alternative 3: 取消时追回已结算佣金

- **Pros:** 资格与资金完全对齐
- **Cons:** 已打款微信分账回滚复杂，合规风险高
- **Why rejected:** 本增量只停未来分账；追回单独立项

## Consequences

### Positive

- 超管可停用违规推荐人的未来分成
- 每次资格变更可检索操作人与理由

### Negative / Trade-offs

- 已结算佣金在取消后仍保留
- 微信接收方删除失败时商户平台仍可能看到该接收方（后续分账仍被本地 `commission_eligible=0` 挡住）

### Mitigations

- 审计行记录 disable-eligibility 与 wechat delete 的结果摘要
- 取消后 status GET 不再补偿登记接收方

## References

- 设计：`docs/superpowers/specs/2026-08-22-referral-qualification-revoke-design.md`
- ADR-0025 多渠道码与 `commission_eligible` 快照
- 意图：`docs/intents/backend/task-referral.intent.md`
