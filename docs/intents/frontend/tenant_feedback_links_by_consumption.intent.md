# 前端：意见与建议链接

- **状态:** accepted
- **日期**: 2026-08-30
- **设计:** `docs/superpowers/specs/2026-08-30-tenant-feedback-links-by-consumption-design.md`

## 用户故事

作为租户成员，我在控制台侧栏「意见与建议」下按组看到平台配置的链接，高消耗租户能看到更多组。作为超管，我在系统管理配置这些组与阈值。

## 验收标准

1. 侧栏一级「意见与建议」与「资源与订单」并列；子菜单按组名分段，其下为真实 `<a href>` 新标签打开。
2. 只渲染 GET 返回的组/链接，不在前端按消耗过滤。
3. 超管页可增删改组、多种资源阈值、链接；写操作 clickGuard + Idempotency-Key。
4. 实现时拆分 `Sidebar.vue`，避免超过 500 行。
5. 无 `feedback:view` 时不显示该一级菜单。

## 业务事件

配置写走服务端 `FEEDBACK_LINK_GROUP_*`。侧栏展示为纯查询，无对应 MQ 事件。
