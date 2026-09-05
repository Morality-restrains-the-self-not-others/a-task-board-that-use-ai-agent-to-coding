# 功能意图：推荐绩效抽屉「微信分账」Tab

## 意图

系统管理员在 `/system-admin/users/` 推荐绩效抽屉中，于「推荐用户支付明细」旁切换 Tab，按「推荐人→被推荐人→打标订单」查看微信分账结果，并可同步当前页微信状态。

## 角色

系统管理员。

## 行为

1. 「支付明细 | 微信分账」两个 Tab；默认支付明细。
2. 微信 Tab 调用 `GET /api/system-admin/profit-sharing/?referrer_user_id={抽屉userId}&status=all`。
3. 表格：被推荐人 ID、**AppID**、**OpenID**（空值「—」）、订单号（链到订单记录）、金额、本地状态、最早可分账时间、失败原因（机器码如 `qualification_revoked` 展示中文「推荐资格已撤销」；失败原因单元格在 `fail_trace_id` 非空时挂 `data-traceId`，无则省略属性；**长文案在单元格内换行完整可见，禁止 `truncate` / `max-w-[8rem]`，原生 `title` 只作备份不可替代页内全文**）、微信订单号、微信分账单号、**微信分账单状态**（微信支付 QueryOrder `state`，不是用户是否绑定微信；`尚未提交微信` 展示「未向微信发起分账」并悬停说明冻结期内尚未 POST `/v3/profitsharing/orders`；`PROCESSING`→「微信处理中」；`FINISHED`→「微信已分账完成」）、操作（待分账/失败可「分账」）。
4. 「同步微信状态」对当前页 POST `refresh-wechat`；同步门闩 + `aria-busy`；失败 `data-traceId`。
5. 「分账」打开表格上方缘由表单（自动滚入视口并聚焦；打开期间按钮不 disabled，文案改为「取消」；仅提交中 `disabled` + `aria-busy`），填写缘由（8–500 字）后确认才 POST `/{id}/share/`，`createClickGuard` + `Idempotency-Key`；失败 `data-traceId`。
6. 展示 AppID / OpenID（来自列表 `app_id` / `openid`，须为同一 `wechat_identity` 支付/mp 投影成对；空值「—」）。空列表文案「暂无微信分账记录」。
7. Anti-Replay-OK：Tab 切换与列表 GET 为只读；同步按钮有 `createClickGuard`；分账确认走 Idempotency-Key。

## 非目标

- 不按被推荐人行钻取
- 不在本 Tab 发起分账回退

## 业务意图 → 事件对照

纯前端展示 + 管理端出站分账（无新内部事件）。

## 变更记录

- 2026-08-23：推荐绩效抽屉增加微信分账 Tab
- 2026-08-24：失败原因机器码本地化为中文（与待分账队列共用 `profitSharingFailReasonLabel`）
- 2026-08-25：增加微信订单号/分账单号列；超管可带审计缘由发起分账
- 2026-08-26：失败原因列挂载持久化 `fail_trace_id` 为 `data-traceId`（元规则 24；对应出站失败当次请求，非列表 GET）
- 2026-08-26：点「分账」不再立刻 disabled；缘由表单置于表格上方并滚入视口，确认后才出站
- 2026-08-26：表格增加 AppID / OpenID 列（超管对照「appid 与 openid 不匹配」）
- 2026-08-26：列值须与 `wechat_identity` 支付/mp 行一致（后端成对取接收方，前端原样渲染）
- 2026-08-26：失败原因列去掉 truncate，长微信拒单文案在单元格内换行完整可见
