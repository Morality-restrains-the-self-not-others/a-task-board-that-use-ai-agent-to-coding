# 设计：超管退款审批并入订单查看

- **Date:** 2026-08-19
- **Status:** accepted（goal-mode 自动采纳）
- **Scope:** taskFE 系统管理导航与页面；**不改** taskBill API / 权限模型

## Context

超管侧存在两个相邻入口：

| 路径 | 页面 | 职责 |
|------|------|------|
| `/system-admin/order-records/` | 订单查看 | 租户/全局订单列表 |
| `/system-admin/refund-applications/` | 退款审批 | 策略开关 + 申请列表审批 |

二者同属「计费运营」，分开展示增加导航负担；目标是合并为单一入口。

## Decision

1. **单一导航入口**：侧栏仅保留「订单查看」（文案可改为「订单与退款」）；移除「退款审批」菜单项。
2. **页面内 Tab**：`SystemAdminOrderRecords` 提供 `orders` / `refund` 两个 Tab；退款能力以 `SystemAdminRefundPanel` 组件嵌入。
3. **旧路由 compulsory 重定向**：`/system-admin/refund-applications/` → `/system-admin/order-records/?tab=refund`（保留书签兼容）。
4. **API 不变**：继续使用既有 `/api/system-admin/refund-applications/` 与 `/api/system-admin/refund-policy/`。
5. **行数门禁**：订单列表与退款面板各自独立组件，外壳页 ≤500 行。
6. **无新服务/事件**：纯前端信息架构调整；不新增领域事件；不更新 ArchiMate 服务拓扑（应用层组件未增删）。

## Alternatives Considered

| 方案 | 拒绝原因 |
|------|----------|
| 订单展开行内嵌审批 | 全局 pending 列表与策略开关仍需独立区块；展开行无法替代列表审批 |
| 仅 iframe/嵌套整页 | 重复布局与双滚动条 |
| 删除旧路由无重定向 | 破坏书签与外部文档链接 |

## Consequences

- 正面：运营在同一页完成订单核对与退款审批；侧栏更短。
- 负面：页面信息量增加 → 用 Tab 隔离缓解。
- 废弃：独立 `SystemAdminRefundApplications` 视图文件删除；路由仅作 redirect。

## Acceptance

1. 侧栏无「退款审批」独立项；「订单查看」可进入并切换到退款 Tab。
2. 访问旧 `/system-admin/refund-applications/` 落到 `order-records?tab=refund`。
3. 退款策略开关二次确认、列表审批行为与合并前一致（既有 vitest/playwright 改指向新挂载点后仍绿）。
4. `SystemAdminOrderRecords.vue` 及拆出组件均 ≤500 行。
