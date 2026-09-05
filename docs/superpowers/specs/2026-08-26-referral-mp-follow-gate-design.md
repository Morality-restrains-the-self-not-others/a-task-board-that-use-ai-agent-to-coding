# 推荐资格申请：先关注服务号再填资料

- **Date:** 2026-08-26
- **Status:** accepted (goal-mode 自动采用)
- **Iteration:** referral-mp-follow-gate
- **Architecture:** v112 (based_on v111)

## 目标

已登录用户在 `/profile/referral/` 申请推荐资格时，必须先关注微信支付绑定的服务号。系统在收到关注事件后用 **unionId** 锁定已有用户并写入服务号 **openId**，然后才允许填写个人名称与简介提交申请。

## 成功标准

1. 未绑定服务号时，「推荐资格」卡展示服务号二维码（`/img/jjf_qrcode.png` + 内容 hash），不展示名称/简介表单。
2. 用户点击「我已关注」（用户触发，禁止后台轮询）后，若已绑定则展示原申请表。
3. 微信 `subscribe` 回调验签后：有 unionId 则绑定到已有 `wechat_identity` 用户的 `app_key=mp` 别名；未命中用户则写入 pending，不建新账号。
4. `POST .../referral-codes/apply/` 在未绑定服务号 openId 时返回 `service_account_not_followed`。
5. 分账 `PERSONAL_OPENID` 优先使用支付 AppID（`wx31273ca77c89dffe`）对应的 mp openId。

## 当前架构理解

- 基线 **v111 current**：taskAuth 持有 `wechat_identity`（unionid 真源 + 每应用 openid 别名）；登录应用 `web`=`wx625802b55b33608f`（网站应用）；支付 AppID 是服务号 `wx31273ca77c89dffe`。
- 尚无公众号服务器回调；关注事件无法写入 mp openid，分账接收方会 `NO_AUTH` / `pending_openid`。
- 申请表已要求 legal_name + intro + 身份绑定勾选。

## 方案（采用）

**关注事件 → unionId 锁定 → mp openid 别名；前端两步闸门。**

```
用户扫静态码关注服务号
        │
        ▼
微信 POST /api/auth/wechat/mp/callback/  (public，验签)
        │  Event=subscribe, FromUserName=mp_openid
        │  UnionID 来自 XML；缺失则 cgi-bin/user/info
        ▼
findWechatUserByUnionID
  ├─ 命中 → upsert wechat_identity(app_key=mp, app_id=支付AppID, openid, unionid)
  │         发布 WECHAT_MP_SUBSCRIBED（已绑定）
  └─ 未命中 → auth_wechat_mp_subscribe_pending(unionid PK)
               发布 WECHAT_MP_SUBSCRIBED（pending）
        │
用户点「我已关注」
        ▼
GET /api/auth/wechat/mp/follow-status/ (token)
  按当前用户 unionid 消费 pending 并绑定
        │
申请表出现 → POST apply（服务端再验 mp 别名）
```

### 拒绝的方案

| 方案 | 拒绝原因 |
|------|----------|
| 带 scene 的临时二维码 | 需求指定静态 `jjf_qrcode.png` |
| 关注即建新用户 | 会分裂账号；「锁定」= 绑定已有用户 |
| 前端 setInterval 轮询 | 违反元规则 51 |
| 用网站应用 openid 当分账接收方 | 支付 AppID 不是网站应用 |

## API

| 方法 | 路径 | 鉴权 | 说明 |
|------|------|------|------|
| GET | `/api/auth/wechat/mp/callback/` | public | 微信服务器 URL 验证，回显 `echostr` |
| POST | `/api/auth/wechat/mp/callback/` | public | 关注/取关事件；成功回 `success` |
| GET | `/api/auth/wechat/mp/follow-status/` | token | `{ bound: bool }`；顺带消费 pending |
| GET | `/api/accounts/users/referral-codes/status/` | token | 增补 `service_account_bound` |
| POST | `/api/accounts/users/referral-codes/apply/` | token | 未绑定 → 400 `service_account_not_followed` |

错误格式保持 `{ error, detail }`。

## 配置

`conf/auth/task-auth/config.yaml` `wechat.apps.mp`：

- `type: mp`（无 redirectUri）
- `appId`: 与 `conf/billing/wechatPay` 的 `appid` 相同
- `appSecret` / `token` / `encodingAESKey`：本地覆盖或环境变量 `WECHAT_MP_*`；token 空则回调 503

微信公众平台服务器 URL：`https://www.daydaymoney.com/api/auth/wechat/mp/callback/`

## 数据

- 复用 `wechat_identity`，`app_key='mp'`
- 新表 `auth_wechat_mp_subscribe_pending`：unionid PK、openid、app_id、created_at。年增量远低于百万，TTL 策略（绑定后删除；未匹配行可后续清理）

## 事件

`WECHAT_MP_SUBSCRIBED`：`user_id`（可空）、`app_key=mp`、`openid`、`unionid`、`outcome=bound|pending`。幂等键：`mp_openid`（同一关注重放不重复副作用）。无自动消费者（审计）。

## 前端

新组件 `ReferralServiceAccountFollowGate`：未绑定展示二维码 + 「我已关注」；已绑定渲染 slot（原申请表）。静态图 `/img/jjf_qrcode.png?h=<sha256-12>`。

## 🕸️ Code Review Graph 分析

CRG / codegraph MCP 本会话不可用（`CRG unavailable: no codegraph MCP tools`）。设计基于 `auth_wechat.go` 既有 unionid 匹配、`wechat_identity` upsert、`lookupWechatPayOpenID` 按 preferred_app_id 优先。

## Python 新接口

全部落 Go（taskAuth / taskReferral）。🐍 not_applicable。

## 架构交付物

- `docs/architecture/v112-*-20260826-0950-cursor.{puml,diff.archimate,full.archimate,mermaid.md}`
