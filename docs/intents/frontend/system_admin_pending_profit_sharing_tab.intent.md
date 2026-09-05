# 功能意图：管理端待分账 Tab 与列表

## 意图

`/system-admin/order-records/` Tab 栏在「订单记录」「退款审批」旁增加「待分账」，展示待处理分账订单队列。

## 角色

- 平台员工：可见
- 租户：无此页

## 行为

1. `?tab=profit-sharing` 展示待分账面板；点击 Tab 写入该 query。
2. 默认筛选「待处理」（open）；可切 pending/processing/failed/finished/全部。
3. 表格：订单号（链到订单 Tab）、租户、接收方、**AppID**、**OpenID**（空值「—」）、订单金额、分账金额、状态、最早分账时间、失败原因（机器码如 `qualification_revoked` 展示中文「推荐资格已撤销」；`fail_trace_id` 非空时失败原因单元格挂 `data-traceId`；**长文案在单元格内换行完整可见，禁止 `truncate`，原生 `title` 只作备份**）、微信订单号、微信分账单号、操作。
4. 空列表文案「暂无待分账订单」。加载失败展示错误并带 `data-traceId`。
5. 待分账/失败行可「分账」：打开表格上方缘由表单（自动滚入视口并聚焦；打开期间按钮不 disabled；仅提交中 `disabled` + `aria-busy`），填写缘由（8–500 字）后确认才 POST `/{id}/share/`，`createClickGuard` + `Idempotency-Key`。
6. 展示 AppID / OpenID（与推荐绩效抽屉同一列表字段）。Anti-Replay-OK：筛选/刷新/分页只读；分账确认走 Idempotency-Key。

## 非目标

- 不改退款 Tab

## 业务意图 → 事件对照

管理端出站分账，无新内部事件。

## 变更记录

- 2026-08-22：第三 Tab 待分账列表
- 2026-08-24：失败原因机器码本地化为中文（qualification_revoked → 推荐资格已撤销）
- 2026-08-25：微信订单号/分账单号列；超管带审计缘由发起分账
- 2026-08-26：失败原因列挂载 `fail_trace_id` 为 `data-traceId`
- 2026-08-26：点「分账」不再立刻 disabled；缘由表单置于表格上方并滚入视口
- 2026-08-26：表格增加 AppID / OpenID 列
- 2026-08-26：失败原因列去掉 truncate，长微信拒单文案在单元格内换行完整可见
