# 价值流 — 超管在用户页浏览租户目录

- **日期**: 2026-08-25
- **增量**: 用户页「租户」Tab + 分页列表

## Related Value Streams

- `2026-08-23-system-admin-users-tenant-company-column-value-stream.md`：用户行上的所属公司投影。本增量是**公司实体目录**，不是用户列。
- tenant-options 下拉：赠送/订单选租户。本增量**扩展**为可翻页目录，不替换下拉。

## 当前价值流（增量）

1. 超管打开 `/system-admin/users/`，点击「租户」或打开 `?tab=tenants`。
2. 前端 `GET /api/system-admin/accounts/admin/tenants/?limit=50&offset=0[&search=]`。
3. 网关 forward-auth → taskTenantService：平台员工校验 → 查 `tenant_company` → best-effort 补创建者 email/phone。
4. 面板渲染表格；可搜索、翻页、刷新。

## 测试点

| ID | 步骤 | 断言 |
|----|------|------|
| TP-TN-1 | Tab 栏 | 含「租户」，位于已归档与推荐码申请之间 |
| TP-TN-2 | 点击租户 Tab | 挂载 tenants panel，不请求用户列表 |
| TP-TN-3 | `?tab=tenants` | 直接进入租户 Tab |
| TP-TN-4 | 有公司 | 表格显示 id/name |
| TP-TN-5 | 空列表 | 「暂无租户」 |
| TP-TN-6 | 未鉴权 GET | 401 |
| TP-TN-7 | 非平台员工 | 403 |
| TP-TN-8 | 分页 | offset 第二页不与第一页重复；total 正确 |
| TP-TN-9 | 加载失败 | 错误节点带 data-traceId |

无新 MQ 事件（只读查询）。
