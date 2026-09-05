# DDD：推荐分账资格取消

> NFR: `docs/superpowers/plans/2026-08-22-referral-qualification-revoke-nfr-clarification.md`

本增量落在既有 taskReferral 主包（与现网 `approveReferralApplication` 同层），不新建独立 hexagonal 目录，以免与服务现状分叉。以下为领域契约，实现用普通函数满足测试。

## 限界上下文

| 上下文 | 服务 | 职责 |
|--------|------|------|
| ReferralQualification | taskReferral | 申请生命周期、审计、发布领域事件 |
| ReferralCommission | taskBill | 边资格快照、计提/作废、微信分账 |

## 聚合

**ReferralQualification**（根：`referral_code.id`）

- 实体：Application（status, expires_at, reviewed_by, personal_intro）
- 实体（子）：AuditEntry（action, reason, operator_id, at）追加
- 不变量：
  - 仅 `pending` 可 approve/reject
  - 仅 `approved` 且未过期可 revoke
  - 每次状态迁移必须有 8–500 字 reason 并追加 AuditEntry
  - 已 revoked/expired 不能再 revoke

## 值对象

- `ReviewReason`：trim 后 rune 数 8–500
- `QualificationStatus`：pending | approved | rejected | expired | revoked

## 领域服务

- `RevokeQualification(appID, operator, reason, idemKey)`  
  成功后发布 `REFERRAL_QUALIFICATION_REVOKED`，调用 Commission 端口。
- `RecordDecision`：approve/reject 共用审计写入

## 端口

```
CommissionEligibilityPort.Disable(referrerUserID, reason) error
AuditRepository.Insert(entry) error
AuditRepository.GetByIdempotencyKey(key) (*entry, error)
AuditRepository.ListByApplication(appID, limit, offset) ([]entry, total, error)
```

## 领域事件

| 事件 | 载荷（禁止 intro 正文） | Key |
|------|-------------------------|-----|
| REFERRAL_APPROVED | app_id, user_id, reviewed_by, reason_len, expires_at | application_id |
| REFERRAL_REJECTED | app_id, user_id, reviewed_by, reason_len | application_id |
| REFERRAL_QUALIFICATION_REVOKED | app_id, user_id, operator_id, reason_len | application_id |

幂等键与业务重复边界同为 **application_id + action 成功一次**；HTTP `Idempotency-Key` 映射到审计唯一键。

## 跨上下文

Revoke 提交本地聚合后调用 Disable：边 `commission_eligible=0`，pending accrual → voided。Commission 上下文不拥有资格状态。
