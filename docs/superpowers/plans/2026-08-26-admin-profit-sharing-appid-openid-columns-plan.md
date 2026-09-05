# 实施计划 — 管理端分账表 AppID / OpenID

## Task 1：意图文档

- [x] 更新 backend/frontend 意图：staff 列表含 `app_id`/`openid`；推荐人自助仍禁止
- [x] 更新 `docs/flows/value-stream-test-integration.wsd` VS-PSQ-6

## Task 2：后端（红 → 绿）

- [x] `handlers_admin_list_profit_sharing_test.go`：staff JSON 含 `app_id`/`openid`，不含 `referrer_openid` 键
- [x] `handlers_admin_list_profit_sharing_referrer_test.go`：有接收方时返回登记 appid+openid 成对
- [x] 回退测：台账 openid 空则用 receiver.openid
- [x] 混用回归：快照为 web openid、接收方为 mp 对时返回 mp 对
- [x] `resolveProfitSharingReceiverOpenid` 优先接收方，覆盖过期快照
- [x] `listProfitSharingQueue` LEFT JOIN + SELECT/Scan
- [x] 推荐人自助测仍禁止泄漏（不改实现）

## Task 3：前端（红 → 绿）

- [ ] `ReferralWechatProfitSharingTab`：AppID / OpenID 列 + 单测
- [ ] `SystemAdminProfitSharingPanel`：同列 + colspan/过滤空单元格 + 单测

## Task 4：验证

- [ ] Go / Vue 相关单测绿；gofmt
