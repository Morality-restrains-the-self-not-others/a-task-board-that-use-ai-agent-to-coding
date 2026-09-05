# 注册邀请码 — 实施计划

设计：`docs/superpowers/specs/2026-07-22-registration-invite-code-design.md`

## 爆炸半径与测试缺口（CRG）

- CLI graph sparse；缺口：须补 taskAuth 单测 T1–T9；网关路由登记；OpenAPI。
- 影响文件：taskAuth migrate/handlers/register、taskGateway routes、Vue SystemAdmin/Profile/Login/Register、api_route_ownership、intents。

## 任务清单

### Backend taskAuth
- [x] migration `008_registration_invite.sql`
- [x] policy/code repository + Shanghai day helper
- [x] handlers: public policy, admin policy/relations, user apply/list
- [x] hook email/phone register redeem
- [x] events map + publish helpers
- [x] OpenAPI paths
- [x] unit tests T1–T9（T4/T7/T9 以领域逻辑覆盖；并发核销依赖 UPDATE WHERE unused）
- [x] trust `X-User-Id` in resolve helper for token routes

### Gateway / ownership
- [x] `taskGateway/routes/routes.yaml` 新路由
- [x] `db/api_route_ownership.yaml` 前缀 → taskAuth

### Frontend task2app
- [x] SystemAdmin 邀请码区块
- [x] `RegistrationInvitePanel.vue` + UserProfile 挂载
- [x] Register + Login 字段与 sessionStorage
- [x] 错误展示 `data-traceId`

### Docs / ship
- [x] intents 已写
- [x] value-stream / NFR / plan / review
- [x] PR：taskAuth#4 / taskGateway#7 / task2app#49 / docs#39 / db#12 / ram-work#11
