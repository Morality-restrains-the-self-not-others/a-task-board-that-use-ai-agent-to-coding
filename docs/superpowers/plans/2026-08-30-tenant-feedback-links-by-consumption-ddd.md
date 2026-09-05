# Domain Model: 意见与建议链接

> NFR: `docs/superpowers/plans/2026-08-30-tenant-feedback-links-by-consumption-nfr-clarification.md`

## Bounded Context

**Billing / 平台运营配置**（taskBill）。taskBill 现网为 `package main`，无独立 `domain/` 树。领域谓词与值对象落在 `taskBill/src/feedback_domain.go`（禁止 import database/sql / kafka），适配器在 `feedback_store.go` / `feedback_handlers.go`。

## Aggregates

**FeedbackLinkGroup**（根）

- id, name, sortOrder, enabled, updatedAt
- thresholds: set of (resourceKind, minQuantity) UNIQUE per kind
- links: list of (title, https URL, sortOrder, enabled)

整组保存替换子集合。未知 kind 拒绝。空阈值 = 对所有租户可见。

## Value Objects

- **ResourceKind**: 目录内登记的 kind 字符串
- **HttpsURL**: scheme 必须 `https`
- **ConsumptionSnapshot**: `map[kind]float64`；缺省 kind 视为 0
- **VisibilityRule**: `∀ t ∈ thresholds: snapshot[t.kind] >= t.minQuantity`

## Domain Service

`GroupVisible(group, snapshot) bool` — 纯函数，单测 T1–T5。

## Ports

- `FeedbackGroupStore`：LoadAll / Load / SaveReplace / Delete（基础设施实现）
- `TenantConsumptionReader`：按 tenant 投影四种 kind
- `EventPublisher`：现网 `publishEvent`（测试可替换）

## Domain Events

| 事件 | 触发 | key |
|------|------|-----|
| FEEDBACK_LINK_GROUP_CREATED | POST 提交 | group_id |
| FEEDBACK_LINK_GROUP_UPDATED | PUT 提交 | group_id |
| FEEDBACK_LINK_GROUP_DELETED | DELETE 提交 | group_id |

租户 GET 无事件。

## 消耗投影（非聚合）

| kind | 口径 |
|------|------|
| task_post | `SUM(quantity)-SUM(remaining)` on `billing_resource_grant` where resource_type=task_post |
| gitlab_traffic | `SUM(traffic_used_gb)` on `billing_tenant_gitlab_resource` |
| gitlab_disk | `SUM(disk_used_bytes)/GiB` |
| consumed_amount | `SUM(amount)` consumption txs for tenant account |

## Invariants

1. URL 非 https → 不入库
2. 租户 DTO 不得含 thresholds
3. disabled 组/链接不对租户出现
