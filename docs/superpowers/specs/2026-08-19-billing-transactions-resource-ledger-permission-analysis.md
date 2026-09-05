# 角色权限分析 — 交易记录资源流水账

- **Date:** 2026-08-19
- **Design:** `docs/superpowers/specs/2026-08-19-billing-transactions-resource-ledger-design.md`

## 结论

无新 endpoint、无新页面组、无新角色。沿用既有租户账单读权限。

| 改动点 | 主体 | 资源层级 | 操作 | 现有检查 | 是否缺失 | 建议 |
|--------|------|----------|------|----------|----------|------|
| GET `.../billing/transactions/` | 租户成员（页面 `billing.transactions`） | Tenant | read | 网关/PDP region `billing.transactions.main` + path tenant_id | ✅ 充分 | 不改鉴权 |
| GET `.../billing/transactions/list_filtered/` | 同上 | Tenant | read | 同上 | ✅ 充分 | 不改鉴权 |
| 列表 SQL `account_id = 当前租户账户` | 服务 | Tenant | read | `parseTenantID` + `getOrCreateBillingAccount` | ✅ 充分 | 回放查询必须同 `account_id`，禁止跨租户 |
| 前端表列 | 同上 | UI region `billing.transactions.main` | read | `showMenu('billing.transactions')` | ✅ 充分 | 不加新 region |

## 风险

- **IDOR**：回放任务帖剩余时若漏 `account_id`/`tenant_id` 会串户。实现必须绑定当前账户。
- **数据暴露**：瞬时余额本就属于该租户账单；不新增 PII 字段。

## 角色建模

不新增角色。
