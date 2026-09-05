# 实施计划：推荐资格服务号动态 scene 码

## Slice A — 领域

- [x] `WechatMPIsFollowScanEvent` / `WechatMPSceneTempID` + 测
- [x] `047_wechat_mp_follow_ticket.sql`

## Slice B — 回调与票

- [x] SCAN/subscribe 命中票绑定
- [x] unionId 属他人 → conflict 不抢绑
- [x] 无 scene subscribe 回归 pending
- [x] POST follow-qr + 复用 pending 票
- [x] GET follow-status 扩展字段
- [x] handlers + OpenAPI
- [x] 冲突事件

## Slice C — 网关

- [x] routes.yaml follow-qr token
- [x] routes-apply 生成 apisix.yaml

## Slice D — 前端

- [x] Gate 进入时 POST follow-qr
- [x] conflict / expired 文案
- [x] 去掉 jjf_qrcode 凭证

## Slice E — 文档

- [x] intents / 架构 v113
