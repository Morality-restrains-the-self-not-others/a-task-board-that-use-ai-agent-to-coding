# 设计：系统管理用户页增加「租户」Tab

- **日期**: 2026-08-25
- **页面**: `/system-admin/users/`
- **状态**: accepted（goal-mode 自动采用）
- **架构变更**: 否（扩展既有 taskTenantService 管理查询，无新服务/协议/存储）

## Context

超管用户页 Tab 仅有「活跃用户 / 已归档 / 推荐码申请」，无法从同一入口浏览全部租户公司。运营目前只能在「赠送资源」「订单记录」的下拉（`tenant-options`，上限 80）里搜公司。需要在用户页再加一栏「租户」，作为平台租户目录。

## Decision

1. 在 `data-testid="system-admin-users-tabs"` 增加第四个 Tab **「租户」**，位于「已归档」与「推荐码申请」之间。
2. 深链 `?tab=tenants`；切换时写入 `route.query.tab`（与推荐码申请 Tab 同模式）。
3. **新只读 API** `GET /api/system-admin/accounts/admin/tenants/`（taskTenantService，平台员工）：
   - Query：`search`（公司名 / 公司 ID / 创建者邮箱或手机，语义对齐 tenant-options）、`limit`（默认 50，最大 100）、`offset`。
   - 响应：`{ items: [{id, name, creator_id, email, phone, created_at, updated_at}], total, limit, offset }`。
   - 鉴权与 tenant-options 相同：`X-Gateway-Auth-Verified=1` + `authz.IsPlatformStaff`。
   - 创建者联系方式 fail-open：taskAuth 失败仍返回公司行，`email`/`phone` 为空串。
   - **不改** `tenant-options` 数组契约（下拉继续用）。
4. 前端抽出 `SystemAdminTenantsPanel`：搜索、表格、分页、错误 `data-traceId`。列：ID、名称、创建者 ID、邮箱、手机、创建时间。只读；刷新按钮 `Anti-Replay-OK: read-refresh`。
5. 切到租户/推荐码申请 Tab 时隐藏用户搜索条与用户列表；邮箱邀请列表与推荐码 Tab 一样隐藏。
6. `SystemAdminUsers.vue` 已超 500 行：Tab 条抽到 `SystemAdminUsersTabs.vue`，避免继续膨胀。

## Alternatives Considered

| 方案 | 拒绝原因 |
|------|----------|
| 复用 tenant-options 数组（上限 80）当目录 | 无法翻页，租户超过 80 时目录残缺 |
| 给 tenant-options 加分页并改响应为对象 | 破坏 GrantPoints / 订单记录下拉（Hyrum） |
| 独立 `/system-admin/tenants/` 路由页 | 用户明确要求加在现有 Tab 栏 |
| 行内「进入租户」切换公司上下文 | 超出本次「加一栏」范围，需模拟登录/公司切换，记 OPT |

## Consequences

- 空搜索走 SQL `COUNT` + `LIMIT/OFFSET`；带搜索时与 tenant-options 相同合并创建者命中，再内存切片（合并上限 500）。升级触发见 NFR。
- 网关须登记新 URI，否则落入 spa-catch-all → 502。
- 纯查询，无领域事件。

## 契约

```
GET /api/system-admin/accounts/admin/tenants/?search=&limit=50&offset=0
Authorization: 平台员工 Cookie / Bearer（经网关 forward-auth）

200 {
  "items": [
    {
      "id": "c1",
      "name": "Test Co",
      "creator_id": "admin1",
      "email": "owner@example.com",
      "phone": "13800138000",
      "created_at": "...",
      "updated_at": "..."
    }
  ],
  "total": 1,
  "limit": 50,
  "offset": 0
}
401 未鉴权 / 403 非平台员工
```
