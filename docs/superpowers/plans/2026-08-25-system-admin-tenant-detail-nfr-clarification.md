# NFR 澄清：系统管理租户详情

- **日期**: 2026-08-25
- **默认档**: L2；资金只读展示不升 L3 写路径

## 类别

| 类别 | 档 | 说明 |
|------|----|------|
| 可用性 | L2 | 三块并行，单块失败其余仍展示 |
| 性能 | L2 | 每块分页/limit≤50；禁止 N+1 |
| 安全 | L2 | staff only；错误不泄露内部 SQL |
| 可观测性 | L2 | 结构化日志 event + tenant_id + trace |
| 可伸缩性 | 见表 | |

## 路径分片键强制审视

| 路径 | 分片 ID | 是否合适 | 可伸缩性 | 动作/理由 |
|------|---------|----------|----------|-----------|
| `/system-admin/tenants/:tenantId/` | tenantId | 是（租户） | L1 路径已带键 | 无 |
| `GET .../tenants/{id}/` | id=tenant | 是 | L1 | 点查 PK |
| `GET .../tenant-quotas/tenant_id/{id}/` | tenant_id | 是 | L1 | 配额按租户 |
| `GET .../tenant-workspaces/tenant_id/{id}/` | tenant_id | 是 | L1 | `WHERE company_id=?` |
| `GET .../orders/?tenant_id=` | tenant_id query | 是 | L1 | 沿用 `listTenantOrders` |
| `/system-admin/users/?tab=tenants` | 无 | — | L0 | 平台目录；升级触发：租户>1e5 再分区目录 |

## 幂等性强制审视

| 路径 | 副作用 | 档 | 理由 |
|------|--------|----|------|
| 上述全部 GET | 无 | L0 | 纯查询；无写库/事件/外部写 |
| 名称 `<a href>` | 无 | L0 | 浏览器导航 |

## 领域模型影响

无新聚合。读既有 Company、Quota 视图、Workspace、ResourceOrder。无新领域事件。
