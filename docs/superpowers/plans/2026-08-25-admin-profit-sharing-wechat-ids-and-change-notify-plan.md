# 实施计划 — 超管微信分账与动账通知

日期：2026-08-25

## 切片

- [x] **I1 列表字段**  
  测试：GET 返回 `wechat_transaction_id` / `wechat_profit_sharing_id`，无 openid。  
  实现：`listProfitSharingQueue` SELECT `o.wechat_transaction_id`、`ps.wechat_profit_sharing_id`。  
  FE：两列。

- [x] **I2 超管分账**  
  测试：无 reason 400；非 staff 403；pending 调 `executeProfitSharing`；同 Idempotency-Key 二次不重复出站；审计行含 actor+reason。  
  实现：`069_profit_sharing_admin_action.sql`；`POST .../{id}/share/`；OpenAPI。  
  FE：就地缘由表单 + clickGuard。

- [x] **I3 动账通知**  
  测试：mock 明文按 `out_order_no` 更新；重复 notify.id 空操作；无 handler 时 FAIL。  
  实现：`070` inbox 表；`ParseNotifyRequest`；双 URL；APISIX change-notify。

## 文件

- `dataMigrate/taskBill/069_*.sql` `070_*.sql`
- `taskBill/src/profit_sharing_admin.go` `handlers_admin_*` `wechat_profit_sharing_notify.go` `openapi.yaml` `handlers.go`
- `taskFE/.../ReferralWechatProfitSharingTab.vue` `SystemAdminProfitSharingPanel.vue` + tests
- `taskGateway/routes/routes.yaml` `apisix/apisix.yaml`
- intents + OpenAPI

## 事件契约

无新 publish。入站微信 notify 不是内部 MQ。
