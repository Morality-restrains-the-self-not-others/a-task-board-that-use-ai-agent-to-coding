# Value Stream: 推荐分账资格取消与操作审计

> Derived from design: `docs/superpowers/specs/2026-08-22-referral-qualification-revoke-design.md`

## Value Summary

系统超级管理员可以取消已授予的分账资格，并且每一次通过/拒绝/取消都留下带理由的审计记录，从而能停掉未来分成并事后追责。

## Related Value Streams

- **referral-channel-codes (v92)**：extension — 本增量在资格门闩与 `commission_eligible` 快照之上增加「取消」与审计。
- **task-referral 审批列表**：modification — 操作列从「仅 pending 通过/拒绝」扩展为「活跃资格可取消 + 全操作强制理由」。

## End-to-End Flow

超管打开推荐码申请 tab → 看到活跃资格行的「取消资格」（不再是「—」）→ 填写理由并确认 → 资格 `revoked`、边资格关闭、pending 计提作废 → 审计行可查 → 该用户未来消费不再分账。

## Value Increments

### Increment 1: 取消资格薄切片（Thin Slice）
**Value to user:** 活跃资格行可取消，状态变为已取消。
**Scope:** `revoked` 状态 + revoke API + 操作列按钮 + 强制理由。
**Business intents → events:** 取消资格 → `REFERRAL_QUALIFICATION_REVOKED`
**Depends on:** nothing

### Increment 2: 审计落库与查阅
**Value to user:** 通过/拒绝/取消均可按申请查看操作人、时间、理由。
**Scope:** `referral_qualification_audit` + GET audit + 列表「操作理由」+ 前端审计抽屉；approve/reject 也强制 reason。
**Business intents → events:** 审批通过/拒绝沿用现事件并补 reason；审计写入结构化日志
**Depends on:** Increment 1

### Increment 3: 停止未来分账
**Value to user:** 取消后新消费不再计提/分账；已结算不追回。
**Scope:** taskBill disable-eligibility；void pending；best-effort 删微信接收方。
**Business intents → events:** 无独立领域事件（副作用在 revoke 同一意图内）
**Depends on:** Increment 1
