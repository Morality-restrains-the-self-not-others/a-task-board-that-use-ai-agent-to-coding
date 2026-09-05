# 零额订单禁止开票 — 角色权限分析

设计：`docs/superpowers/specs/2026-08-29-zero-amount-invoice-block-design.md`

无新角色、无新端点。沿用 `billing:manage` / `IsPlatformStaff`。新增的是**资源状态条件**（金额 > 0），不是权限码。

| 改动点 | 主体 | 资源层级 | 操作 | 现有检查 | 是否缺失 | 建议 |
|--------|------|----------|------|----------|----------|------|
| POST .../orders/{oid}/invoice-applications/ | 租户管理员 billing:manage | Tenant/Order | write | RequirePerm + path tid + loadOrder(tid,oid) | ✅ 充分 | 零额 400 在鉴权之后；跨租户仍 404 |
| POST .../invoice-applications/{id}/approve/ | 平台员工 | System | write | IsPlatformStaff | ✅ | 零额 400，避免员工绕过租户拦截登记蓝票 |
| 订单详情「申请开票」按钮 | 能打开该订单的租户角色 | Tenant/Order | UI | 与订单 GET 相同 | ✅ | 仅展示控制，不替代后端 |

IDOR：拦截读取本租户订单行上的 `total_yuan_cents`，不引入按其它租户 ID 查询。日志不打买家抬头。
