# 实施计划 — 推荐绩效微信分账 Tab

- **日期**: 2026-08-23
- **设计**: `docs/superpowers/specs/2026-08-23-system-admin-referral-wechat-profit-sharing-tab-design.md`

## 切片

### Slice A — GET 按推荐人图列打标订单（taskBill）

- [x] 红：`receiver_user_id` 图过滤、无边不出现、不传参全局队列不变、禁 openid、非 staff 403
- [x] 绿：`listProfitSharingQueue` 增加 `referrer_user_id` JOIN；响应 `referred_user_id` + `receiver_registration_status`
- [x] OpenAPI `referrer_user_id`
- [x] 日志 `admin_profit_sharing_list_ok` 含 referrer 指纹长度而非原文 openid

### Slice B — POST refresh-wechat（taskBill）

- [x] 红：他人 id 400 且 QueryOrder 调用次数 0；FINISHED 不改 DB；缺 txn 行级错误
- [x] 绿：路由先于列表 subtree；staff + 图内 ids；注入 `profitSharingQueryOrderCall`
- [x] OpenAPI POST
- [x] warn 越权；info 刷新条数；错误人类可读

### Slice C — 抽屉 Tab（taskFE）

- [x] 红：默认不请求 profit-sharing；点 Tab 带 `referrer_user_id&status=all`；空态；同步 guard；失败 data-traceId
- [x] 绿：抽小组件避免抽屉 >500 行；`createClickGuard`；订单深链
- [x] Anti-Replay-OK 注释

### 事件

无。意图文档已书面例外。不新增 publish/consumer 任务。

### 验证

```
cd taskBill && go test ./src -count=1 -run 'ProfitSharing'
cd taskFE/app && npx vitest run src/components/SystemAdminReferralPerformanceDrawer.test.js src/components/system-admin/ReferralWechatProfitSharingTab.test.js
```
