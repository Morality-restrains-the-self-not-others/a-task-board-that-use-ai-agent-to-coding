# DDD — 用户列表分账资格（只读投影）

- **日期**: 2026-08-23
- **NFR**: `docs/superpowers/plans/2026-08-23-system-admin-users-profit-sharing-qualification-column-nfr-clarification.md`

## 限界上下文

- **Identity (taskAuth)**：拥有 `auth_user`；编排超管用户列表。
- **Referral (taskReferral)**：拥有 `referral_code`（资格 SSOT）。

跨上下文只走内部 HTTP 端口，不共享表。

## 模型

- **值对象** `ProfitSharingQualification`：`user_id` + `active bool`（当前是否具备分账资格）。
- **领域服务（既有）** `referrerHasActiveQualification`：approved 且未过期。
- **应用服务** `lookupActiveQualificationsBatch`：同一判定的 IN 查询。
- **不新增聚合、不新增领域事件**。

## 业务意图 → 事件

纯查询。例外见意图文档。

## 端口

```
ReferralQualificationLookup.BatchByUserIDs(ctx, userIDs) -> (map[userID]bool, ok)
```

`ok=false` 表示下游失败（投影为 JSON `null`）。实现：HTTP 适配器，超时 5s。
