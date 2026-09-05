# 推荐资格服务号：动态 scene 二维码 + 临时 ID 对账 + unionId 冲突提醒

- **Date:** 2026-08-26
- **Status:** accepted（goal-mode 自动采用）
- **Iteration:** referral-mp-dynamic-qr-ticket
- **Architecture:** v113 (based_on v112)
- **Supersedes (QR 部分):** v112 静态 `/img/jjf_qrcode.png`

## 运行时证据（无页面 data-traceId）

「尚未确认关注」是 follow-status **200 + `bound:false`** 的业务文案，错误节点不带 `data-traceId`。Loki `{job=~".+"}` 近 24h 无 `wechat_mp`（采集缺口）；以 `logs/task-auth/task-auth.log` 为准：

| 时间 (CST) | 证据 | 含义 |
|---|---|---|
| 10:27 | POST `/mp/callback/` → **503** `wechat_mp_callback_unconfigured` | 真实关注推送到达但 Token 未加载，事件丢弃 |
| 13:36 | GET 验签 403 后 200 | 配置生效，URL 验证成功 |
| 13:37 | POST `wechat_mp_subscribed` pending/bound | 测试夹具 openid，非本页用户 |
| 14:16 | GET callback 200 | 微信周期性验签 |
| 15:02 / 15:25 | GET follow-status 200 | 用户点「我已关注」；之后无真实 subscribe/SCAN POST |

根因（叠加）：(1) 早先 POST 被 503 丢掉且微信不重放；(2) 页面用服务号静态码，已关注用户再扫不会推 subscribe，且无 scene 时通常也没有 SCAN；(3) 回调只处理 Event=subscribe，忽略官方 SCAN。

官方依据：生成带参数的二维码文档 — 未关注 → subscribe + EventKey=qrscene_<scene>；已关注 → SCAN + EventKey=<scene>。

## 当前架构理解

- 基线 **v112 current**：taskAuth 验签服务号回调；wechat_identity.app_key=mp；auth_wechat_mp_subscribe_pending；GET follow-status；taskFE 静态码闸门。
- 微信服务号与网站应用经开放平台共享 unionId。
- 已有 WECHAT_IDENTITY_CONFLICT 与 findWechatUserByUnionID；关注路径尚未按「当前登录用户 vs unionId 已属他人」对账。

## 目标与成功标准

1. 推荐页闸门展示微信 QR_STR_SCENE 临时码（scene = 临时 ID），不再用静态 jjf_qrcode.png 作为关注凭证。
2. 未关注扫码 → subscribe；已关注扫码 → SCAN。两者都按临时 ID 命中票据，再取 unionId。
3. unionId 已绑定其他账号：不抢绑；票据记 conflict；用户点「我已关注」看到明确提醒（HTTP 失败时错误节点带 data-traceId；业务冲突返回稳定 conflict_code）。
4. unionId 未占用或即当前用户：写入 app_key=mp，bound=true。
5. 仍禁止关注建号；禁止前端 setInterval 轮询。

## 方案（采用）

登录用户签发 Snowflake 临时 ID → 微信带参临码 → 回调用 scene 对账 → unionId 冲突则记票提醒。

```
页面打开闸门
  POST /api/auth/wechat/mp/follow-qr/   (token + Idempotency-Key)
        复用未过期 pending 票，否则 Snowflake temp_id
        POST cgi-bin/qrcode/create
             QR_STR_SCENE, scene_str=temp_id, expire_seconds=86400
  { temp_id, qr_src: showqrcode?ticket=..., expires_at }

用户扫码
  未关注: Event=subscribe, EventKey=qrscene_{temp_id}
  已关注: Event=SCAN,      EventKey={temp_id}

POST /api/auth/wechat/mp/callback/  (public，验签)
  解析 scene → 查 auth_wechat_mp_follow_ticket
  unionId：XML 或 cgi-bin/user/info
  findWechatUserByUnionID(unionId)
    空 或 = ticket.user_id → upsert mp 别名；ticket.status=bound
                            发布 WECHAT_MP_SUBSCRIBED outcome=bound
    其他 user_id         → 不改 wechat_identity
                            ticket.status=conflict, conflict_code=unionid_bound_other
                            发布 WECHAT_IDENTITY_CONFLICT
                            发布 WECHAT_MP_SUBSCRIBED outcome=conflict

用户点「我已关注」（禁止轮询）
  GET follow-status
    先 claim 无 scene pending
    若仍未 bound：用未过期 pending 票 temp_id 对账粉丝 qr_scene_str
      （补偿微信未重放的 subscribe/SCAN POST）
    bound / ticket_status=conflict / pending / expired / has_unionid
```

无 scene 的裸 subscribe（搜公众号关注）仍走 v112 pending 表，作为旁路，不作为推荐页主路径。

运行时（2026-08-26 17:20 CST）：用户已扫 scene 码（粉丝 `qr_scene_str`=票 `880376598551883776`，`subscribe_time`=17:20:04），但 `logs/task-auth.log` 无对应 POST callback；MP `user/info` 无 unionid（服务号未返回 unionId）。对账键是 temp_id / `qr_scene_str`，不是 unionId。

### 拒绝的方案

| 方案 | 拒绝原因 |
|---|---|
| 继续静态码 + 仅补 SCAN | 静态码无 scene，已关注再扫常无事件，无法对账当前登录用户 |
| 永久码 QR_LIMIT_STR_SCENE | 10 万上限，不适合每用户每次申请 |
| 整表拉粉匹配 unionId | 慢、限额、无临时 ID，无法做当前页用户对账 |
| 冲突时把 mp 改绑到当前用户 | 抢身份；与既有 WECHAT_IDENTITY_CONFLICT 语义相反 |
| 前端轮询 follow-status | 元规则 51 |
| 关注即建号 | 分裂账号 |

## API

| 方法 | 路径 | 鉴权 | 说明 |
|---|---|---|---|
| POST | `/api/auth/wechat/mp/follow-qr/` | token | 签发/复用票据并调微信创码。Idempotency-Key 必填。 |
| GET | `/api/auth/wechat/mp/follow-status/` | token | 扩展响应见下 |
| GET/POST | `/api/auth/wechat/mp/callback/` | public | 增加 SCAN；EventKey 解析 qrscene_ 前缀 |

### POST follow-qr 响应

```json
{
  "temp_id": "8803",
  "qr_src": "https://mp.weixin.qq.com/cgi-bin/showqrcode?ticket=TICKET",
  "expires_at": "2026-08-27T15:50:00+08:00"
}
```

同一用户存在未过期且 status=pending 的票 → 直接返回，不再打微信。微信 qrcode/create 失败 → 503，错误节点须有 data-traceId；禁止回退静态图。

### GET follow-status 响应

```json
{
  "bound": false,
  "has_unionid": true,
  "ticket_status": "conflict",
  "conflict_code": "unionid_bound_other",
  "message": "该微信已绑定其他账号。请用已绑定该微信的账号登录后再扫码。"
}
```

ticket_status: none | pending | bound | conflict | expired。

禁止把他人 user_id / unionId / openid 返回给前端。冲突文案只说明已绑定其他账号。

错误 JSON 仍 `{ error, detail }`；HTTP 失败时前端错误 DOM 写 data-traceId。

## 数据

新表 `auth_wechat_mp_follow_ticket`（taskAuth，前缀 auth_）：

| 列 | 说明 |
|---|---|
| id BIGINT PK | Snowflake = scene_str = temp_id |
| user_id BIGINT | 申请二维码的登录用户 |
| status VARCHAR(32) | pending / bound / conflict |
| wechat_ticket VARCHAR(512) | 微信 ticket（可空，仅展示） |
| expire_at DATETIME | 与微信码有效期对齐 |
| conflict_owner_user_id BIGINT NULL | 冲突时对方账号，仅服务端/审计 |
| mp_openid / unionid | 回调写入，默认空串 |
| created_at / updated_at | |

- 伸缩：每用户每次闸门一行；年增量远低于百万。TTL：过期 pending 由既有/后续 timer worker 清理（禁止 taskAuth 进程内 ticker）。热路径按 id 点查。
- 冲突不写 mp 行。
- 保留 auth_wechat_mp_subscribe_pending 给无 scene 关注。

## 领域规则

1. WechatMPIsFollowScanEvent：subscribe 或 SCAN（大小写不敏感）。
2. WechatMPSceneTempID(eventKey)：去掉 qrscene_ 前缀后的 scene；空则非本票路径。
3. 命中票后以票的 user_id 为绑定目标，不再依赖当前用户必须先有 web unionId 才能锁定（冲突检测以 unionId 全局占用为准）。
4. findWechatUserByUnionID 得到的 user_id 不等于 ticket.user_id → 冲突，不 upsert。
5. 同一 mp openid 已挂在其他 user → 同样冲突（uk_app_openid）。
6. 关注仍不创建 auth_user。

## 事件

| 业务意图 | 事件 | 发布点 | 消费者 | 幂等键 |
|---|---|---|---|---|
| 关注/扫码锁定成功 | WECHAT_MP_SUBSCRIBED outcome=bound | process ticket | 审计 | mp_openid |
| 票冲突（unionId 属他人） | WECHAT_IDENTITY_CONFLICT | 同上 | 已有 wechat-identity-conflict 告警 | owner_user_id + openid |
| 票冲突（产品结果） | WECHAT_MP_SUBSCRIBED outcome=conflict | 同上 | 审计 | temp_id |
| 无 scene 未命中用户 | WECHAT_MP_SUBSCRIBED outcome=pending | v112 旁路 | 审计 | mp_openid |
| 签发二维码 | 无 MQ | 只读写本聚合票据 | — | 例外：不跨服务副作用 |

## 前端

- ReferralServiceAccountFollowGate：qrSrc 改为 follow-qr 返回值；进入闸门时一次 POST（同步门闩 + Idempotency-Key，禁止 refresh 连打微信）。
- 「我已关注」仍只在点击时 GET follow-status。
- ticket_status=conflict：红字用服务端 message（data-testid=referral-mp-follow-error）。
- expired：提示刷新页面重新取码。
- 静态 jjf_qrcode.png 不再作为闸门凭证（资源可留仓但不引用）。

## 网关

- POST /api/auth/wechat/mp/follow-qr/：auth_mode token，priority 与 follow-status 同档。
- callback 仍 public。

## 可观测性

回调与 follow-status 必须打结构化日志（小写 level）：event、ticket_status、bound、has_unionid、conflict_code、openid/unionid 指纹、trace_id。忽略的非关注事件打 wechat_mp_event_ignored + event 名。

## Code Review Graph 分析

CRG / codegraph MCP 本会话不可用（CRG unavailable: no codegraph MCP; no .codegraph index）。设计基于 auth_wechat_mp.go 回调、findWechatUserByUnionID、publishWechatIdentityConflict、APISIX taskauth-wechat-mp-*。

## Python 新接口

全部落 Go taskAuth。Python 门禁 not_applicable。

## 价值流影响

- 流：用户推荐资格申请（关注闸门步）。
- 字段：taskAuth.auth_wechat_mp_follow_ticket.*（新）；wechat_identity 写入条件增加票 user_id 与 unionId 占用一致。
- 测试：auth_wechat_mp_test.go、domain scene/SCAN、FE gate 动态码 + conflict 文案。

## 权限影响分析

见 `docs/superpowers/specs/2026-08-26-referral-mp-dynamic-qr-ticket-permission-analysis.md`。结论：绿灯。follow-qr / follow-status 仅会话用户；callback public 验签；冲突响应禁止泄露他人身份。

## 架构变更影响

- **迭代版本**: v113 🎯 target
- **迭代名称**: referral-mp-dynamic-qr-ticket
- **作者**: cursor
- **设计日期**: 2026-08-26 15:50
- **新增文件**（每个视图四类，缺一不可）:
  - `docs/architecture/v113-application-integration-20260826-1550-cursor.puml`
  - `docs/architecture/v113-enterprise-landscape-20260826-1550-cursor.puml`
  - `docs/architecture/v113-application-integration-20260826-1550-cursor.diff.archimate`（增量变迁：v112→v113）
  - `docs/architecture/v113-enterprise-landscape-20260826-1550-cursor.diff.archimate`
  - `docs/architecture/v113-application-integration-20260826-1550-cursor.full.archimate`（全量拓扑）
  - `docs/architecture/v113-enterprise-landscape-20260826-1550-cursor.full.archimate`
  - `docs/architecture/v113-application-integration-20260826-1550-cursor.mermaid.md`
  - `docs/architecture/v113-enterprise-landscape-20260826-1550-cursor.mermaid.md`
- **已有文件（未修改）**: `docs/architecture/v112-*-20260826-0950-cursor.*` (current)
- **变更明细**: 🟢 动态码 + 票据表；🟡 回调 SCAN / 冲突 / follow-status；废弃静态码作为关注凭证

### .archimate 架构变迁要点

| 文件 | 内容 |
|------|------|
| **`.diff.archimate`** | Plateau v112 → Gap（静态码无 scene）→ WP → Plateau v113；目标拓扑含 follow-qr / SCAN / 冲突不抢绑 |
| **`.full.archimate`** | 与 diff 同切片全量（本增量局部视图，对齐 v112 交付口径） |

Archi `--loadModel`：四个 `.archimate` 均出现 `Loaded model:`。

## 领域概念清单（供 /6-ddd）

- 限界上下文: 认证 / 微信身份（taskAuth）
- 聚合: FollowTicket（temp_id 根）、WechatIdentity（unionId / app openid）
- 事件: WECHAT_MP_SUBSCRIBED、WECHAT_IDENTITY_CONFLICT
