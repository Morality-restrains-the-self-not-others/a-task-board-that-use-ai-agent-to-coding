# Plan: 超管退款审批并入订单查看

- **Date:** 2026-08-19
- **Design:** `docs/superpowers/specs/2026-08-19-system-admin-refund-into-order-records-design.md`
- **Pipeline notes (goal-mode auto):**
  - **Role-Permission:** 无新端点；仍依赖既有 system-admin 超管闸门；侧栏入口合并不改变授权面。
  - **Worktrees:** SKIP
  - **Value Stream:** 超管「审批退款」步骤入口从独立页改为订单页 Tab；下游 API/事件不变。
  - **NFR:** 纯 FE；路径分片键 N/A（管理页无租户分片）；幂等 L0（无新写路径）；L2 UI。
  - **DDD:** 无新领域事件（纯查询/既有写路径入口搬迁）；书面例外：FE 信息架构。
  - **ArchiMate:** 无服务组件增删，跳过新视图。

## Tasks

- [x] T1 抽取 `SystemAdminOrderListPanel` / `SystemAdminRefundPanel`
- [x] T2 `SystemAdminOrderRecords` Tab 壳 + `?tab=refund`
- [x] T3 侧栏合并文案；移除独立退款项
- [x] T4 旧路由 redirect
- [x] T5 vitest（Tab + 策略二次确认）+ Playwright 路径更新
- [x] T6 意图/设计文档同步
