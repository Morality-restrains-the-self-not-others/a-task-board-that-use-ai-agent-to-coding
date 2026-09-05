# 微信登录 openid/unionid 跨应用身份统一

## 意图

同一微信用户经不同微信应用（PC 网页扫码=网站应用、微信内置浏览器=公众号授权）登录时，
openid 不同但 unionid 相同（同一开放平台账号下）。目标：所有应用登录收敛到同一 taskAuth 账号。

- 新增 `wechat_identity` 表（(app_key, openid)→user_id、unionid→user_id 双映射）
- taskAuth 多应用配置模型（apps[]：web=qrconnect 扫码 / inapp=snsapi_userinfo 公众号授权）
- 登录匹配顺序：unionid 优先 → (app_key, openid) 兜底；openid 别名归属随 unionid 迁移
- 分裂防护：unionid 可得时做绑定转移（WechatIdentityLinked）；双射异常拒绝 + 告警（WechatIdentityConflict）
- 手机号/邮箱登录用户可**显式绑定微信**（登录态 bind 流程，state 带 bind 标记；绑定冲突 409 拒绝 + 引导；解绑校验剩余登录方式）
- **支付回调可靠性（登录/支付解耦）**：订单归属 = 下单会话 user_id；回调只确认支付成功并幂等入账，支付凭据（方式/appid/openid/unionid）仅记录不判定；pending 持久化 + 幂等 + PAYMENT_SUCCEEDED（修订 2026-08-05：移除身份核对，手机号登录+微信支付等场景不受影响）
- 不做用户业务数据自动合并（后续迭代，锚点手机号绑定）

## 验收

1. 同一用户分别经 web 扫码与 inapp 授权登录 → 同一 taskAuth user_id；wechat_identity 两行别名指向同 user
2. 首次经某应用登录且 unionid 缺失 → (app_key, openid) 建号；后续同应用重登命中同一账号（不分裂）
3. unionid 可得后，openid 别名绑定转移到 unionid 真源账号；auth_login_method openid 裸行同步收敛
4. 双射异常（(app,openid) 归属用户已有不同 unionid）→ 拒绝该 openid 登录 + WechatIdentityConflict
5. 旧 `/api/auth/wechat/callback/` 与无参 `/api/auth/wechat/login/` 行为不变（web 兼容）。若 APISIX 返回 `{"error":"服务暂时不可用，请稍后重试","trace_id":"..."}`，那是网关 502（taskAuth 未监听），不是 OAuth `code`/`state` 失败；禁止重放微信 `code`。见 ADR-0035（2026-08-29 callback `cf7b5ea2918884602b81fd80eae1efb1`）。
6. taskAuth 单测：匹配顺序 4 分支、绑定转移、冲突拒绝；029 迁移幂等（重跑无副作用）
7. conf：`conf/auth/task-auth/config.yaml` wechat.apps.web/inapp（旧顶层配置自动映射 web）

## 变更日期

2026-08-05

## 业务意图 → 事件对照

| 业务意图 | 事件名（过去式） | MQ类型/契约 | 发布点 | 消费者/副作用 | 例外理由 |
|---------|----------------|------------|--------|--------------|---------|
| 用户微信登录成功（任意应用） | USER_LOGGED_IN（payload +provider_app） | domain-event | handleWeChatCallback | taskEvents SSE | — |
| 微信新用户注册 | USER_CREATED | domain-event | findOrCreateWeChatUser | taskEvents SSE | — |
| 跨应用身份别名收敛（绑定转移） | WechatIdentityLinked 🆕 | domain-event | taskAuth 合并逻辑 | 审计日志消费者（可选） | — |
| 身份双射异常 / 绑定冲突 | WechatIdentityConflict 🆕 | domain-event | taskAuth 冲突检测 | 运维告警 | — |
| 用户绑定微信到既有账号 | WechatBound 🆕 | domain-event | 绑定回调成功分支 | 审计日志消费者（可选） | — |
| 用户解绑微信 | WechatUnbound 🆕 | domain-event | 解绑接口 | 审计日志消费者（可选） | — |
| 订单支付成功（回调入账） | PAYMENT_SUCCEEDED 🆕 | domain-event | markOrderPaid | 审计/对账/SSE | — |
| openid 别名 upsert（纯内部元数据） | — | — | — | — | 无跨边界副作用，taskAuth 内部状态 |
