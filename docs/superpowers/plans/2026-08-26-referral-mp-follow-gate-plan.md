# 实施计划：推荐资格服务号关注闸门

## Slice A — 领域与回调解析（taskAuth）

- [x] `domain/wechat_mp_subscribe.go` + 测：签名串、subscribe 判定、禁止建号
- [x] `auth_wechat_mp.go`：XML/加密解析、验签、user/info 注入点
- [x] `046_wechat_mp_subscribe_pending.sql`

## Slice B — 绑定与 HTTP

- [x] subscribe：unionid 命中 upsert mp；未命中 pending
- [x] GET follow-status 消费 pending
- [ ] 登录/绑定微信成功后顺带 Claim pending（延期 OPT-20260826-004，避免撑破 `auth_wechat.go`）
- [x] OpenAPI + handlers 注册
- [x] `WECHAT_MP_SUBSCRIBED`

## Slice C — 申请闸门（taskReferral）

- [x] status 返回 `service_account_bound`（authDB 查 `wechat_identity` app_key=mp）
- [x] apply 未绑定 → `service_account_not_followed`

## Slice D — 前端

- [x] `ReferralServiceAccountFollowGate.vue` + 测
- [x] `UserReferral.vue` 套闸门；「我已关注」调 follow-status
- [x] QR hash query

## Slice E — 网关与配置

- [x] routes.yaml 两条路由（callback public 高优先级；follow-status token）
- [x] `routes-to-apisix.py` 生成
- [x] `wechat.apps.mp` 配置骨架

## Slice F — 文档与架构 v112

- [x] intents + INDEX
- [x] architecture 四件套 + VERSION_HISTORY
