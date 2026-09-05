# Intent: 用户推荐码资格申请与审批管理

## 背景

平台需要推荐码机制来控制用户注册来源和推荐关系。用户可申请推荐资格，管理员可审批并管理推荐策略。

## 需求（2026-07-25）

1. 用户可查看推荐码状态、申请推荐资格
2. 管理员可审批/拒绝申请、**取消已授予的分账资格**；每次通过/拒绝/取消必须填写理由并写入 `referral_qualification_audit`；审批列表展示申请人个人介绍与最近操作理由
3. 推荐码有效期 180 天，过期自动标记
4. 拒绝后有 7 天冷却期
5. 推荐关系统计（推荐人数、月度收益）
6. **分享用 accessCode 与推荐资格解耦**：已登录用户始终拥有**至少一条**不透明 `access_code`（默认渠道「默认」，`referral_share_code` 持久化，**禁止**用 `userId` 派生）；可再为不同渠道创建独立码与链接；`has_active_code` 只控制收益分成，不控制发码。绑边快照 `commission_eligible`：无资格仍计推荐人数。**点数计提与微信分账均以支付/下单时刻现查资格为准**（ADR-0033），不以绑边快照冻结；先推荐后获资仍可从已推荐用户的后续下单获得分成。
7. **申请须填写个人介绍**（20–500 字）：分成名额有限，介绍随申请行持久化，供超管审批参考
8. **申请须明示并勾选同意**：系统将绑定账号标识（微信 OpenID）与用户身份信息并提交微信支付做一致性校验（添加分账接收方）。未勾选 → 400 `identity_bind_consent_required`；同意时间写入 `identity_bind_consented_at`

## 契约

### 用户端点

| 项 | 值 |
|---|---|
| 状态查询 | `GET /api/accounts/users/referral-codes/status/` |
| 申请推荐 | `POST /api/accounts/users/referral-codes/apply/` |
| 申请体 | JSON `{ "personal_intro": "<20–500 字>", "identity_bind_consent": true }`，缺/短/超长介绍 → 400 `invalid_intro`；未勾选身份绑定授权 → 400 `identity_bind_consent_required` |
| 状态/列表字段 | `personal_intro` 回传申请时提交的介绍 |
| 推荐统计 | `GET /api/referral/stats/user_id/{userId}/?channel_code=&from=&to=`。`referral_rate_display` 为 **固定 5%**，与点数计提同源。**禁止**把微信 `max_ratio` 或库内配置显示成政策比例。 |
| 渠道列表 | `GET /api/referral/channels/` |
| 创建渠道 | `POST /api/referral/channels/` `{ "name": "1–32 字" }` + `Idempotency-Key` |
| 禁用渠道 | `POST /api/referral/channels/code/{code}/disable/` |
| 认证 | X-User-Id header（网关注入） |
| 成功 | 200 + JSON body |
| 未认证 | 401 |
| 方法错误 | 405 |
| 状态字段 `access_code` | 已认证即返回不透明分享码，与资格无关，不得为 `u{userId}` |
| 状态字段 `referral_code` | 仅 `has_active_code=true` 时返回（与 `access_code` 同一条码） |
| 状态字段 `wechat_receiver_status` | 仅有活跃资格时：`registered` / `pending_openid` / `skipped_not_live` / `failed`。本地资格 ≠ 微信商户平台接收方；开通与 status GET 均幂等调用 taskBill `POST /api/internal/taskbill/profit-sharing/receivers/ensure/`（`PERSONAL_OPENID` + `DISTRIBUTOR`，openid 来自 taskAuth `wechat_identity`，appid 用该身份所属应用）。失败不阻断资格开通。 |

### 管理员端点

| 项 | 值 |
|---|---|
| 申请列表 | `GET /api/system-admin/referral/applications/?status=&limit=&offset=` |
| 审批通过 | `POST /api/system-admin/referral/applications/{id}/approve/` `{ "reason" }` + `Idempotency-Key` |
| 拒绝申请 | `POST /api/system-admin/referral/applications/{id}/reject/` `{ "reason" }` + `Idempotency-Key` |
| 取消资格 | `POST /api/system-admin/referral/applications/{id}/revoke/` `{ "reason" }` + `Idempotency-Key`（仅活跃 approved） |
| 操作审计 | `GET /api/system-admin/referral/applications/{id}/audit/?limit=&offset=` |
| 策略管理 | `GET/PUT /api/system-admin/referral/policy/` |
| 佣金配置 | `GET/POST /api/system-admin/referral/config/`（`billing_referral_config`，taskBill 所有权）：`settle_delay_days`（≥8）。**分成比例固定 5%**，不从配置读取、POST 忽略 `referral_rate_percent`。申请列表每行回传 `referral_rate_display=5%`。微信打款金额 = `min(5%, 微信商户 max_ratio)`；微信比例不可用则打款 0。点数计提用同一固定 5%。 |
| 推荐绩效 | `GET /api/system-admin/users/{uid}/referral-performance/`：`referred_users` 为向下推荐人实付（`recharge`+`user_recharge_*` 或 `resource_purchase`，不含 `admin_grant`；缺 `user_id` 按同账户反查）；被推荐人无向下推荐时回填 `as_referred`；`commission.commission_rate_display` 为配置的推荐比例（与计提同源），失败返回空串（前端展示「—」） |
| 权限 | 需 superuser |
| 非超管 | 403 |

### 策略模式

| 模式 | 行为 |
|------|------|
| `approval` | 申请后为 pending，需管理员审批 |
| `open` | 申请即自动 approved，`referral_code` 为该用户已有的不透明分享码 |

### 推荐统计

| 项 | 值 |
|---|---|
| referral_count | 推荐人数（`billing_referral_edge`，可按 `channel_code`/`bound_at` 过滤） |
| referral_rate_display | **固定 5%**，与点数计提同源。stats / status / performance 均走 `configuredReferralRateDisplay()`。**禁止**读 `billing_referral_config` 或微信 `max_ratio` 冒充政策比例。 |
| monthly_earnings | 按月聚合的 consumption/commission 点数（折算为元；可按渠道与 `consumed_at` 过滤） |
| channels | 分渠道人数与分账汇总 |

## 业务意图 → 事件对照

| 业务意图 | 事件名（过去式） | MQ类型/契约 | 发布点 | 消费者/副作用 | 例外理由 |
|---------|----------------|------------|--------|--------------|---------|
| 用户提交申请 | REFERRAL_APPLICATION_SUBMITTED | 日志（待 Kafka） | handleReferralCodeApply | 审计（`personal_intro_len`，禁止正文） | 证据豁免：与审批事件同档，首期仅结构化日志 |
| 管理员审批通过 | REFERRAL_APPROVED | 日志（待 Kafka） | approveReferralApplication | `referral_qualification_audit` + 审计日志 | 证据豁免：首期仅结构化日志 |
| 管理员拒绝申请 | REFERRAL_REJECTED | 日志（待 Kafka） | rejectReferralApplication | `referral_qualification_audit` + 审计日志 | 证据豁免：首期仅结构化日志 |
| 管理员取消分账资格 | REFERRAL_QUALIFICATION_REVOKED | 日志（待 Kafka） | revokeReferralQualification | 审计表 + taskBill disable-eligibility（边置 0、void pending） | 证据豁免：与审批事件同档 |
| 超管查看固定分成比例 | — | — | — | — | 纯只读；比例不可配置 |
| 超管保存入账天数 | — | — | handleAdminReferralConfig | 仅更新 settle_delay_days；忽略比例字段 | 无新 MQ；比例不再可变 |
| 过期自动标记 | — | — | expireReferralCodes (1h cron) | status=expired | 无领域事件 |
| 状态查询 / 统计 / 策略查询 | — | — | — | 有活跃资格时 status GET 补偿登记微信分账接收方 | 资格查询有副作用（幂等 ensure），失败不阻断展示 |
| 登记微信分账接收方 | — | HTTP 内部 | open 申请 / 审批通过 / 有资格 status GET | taskReferral → taskBill ensure → 微信 `POST /v3/profitsharing/receivers/add` | 商户平台「管理分账接收方」只在该 API 成功后出现 |
| 注册绑推荐边 | — | HTTP 内部 | taskAuth **新建**注册成功 → taskReferral bind-from-code → taskBill sync-edge（须带 `X-TaskBill-Internal-Secret`）。已有手机号 `phone_register` 400 不绑边 | billing_referral_edge（channel_code + commission_eligible 快照）+ 合格时历史消费回填计提 | 同步绑边以免统计页延迟；**微信分账打标不读该快照**，见 billing_paytime_referral_profit_sharing_flag |
| 支付时刻现查资格 | — | HTTP 内部 | taskBill Native 预下单 / markOrderForProfitSharing → `GET /api/internal/referral/qualification/active/` | 只读 `referral_code` 是否 approved 且未过期 | 无新 MQ；ADR-0033 |
| 创建推荐渠道 | REFERRAL_CHANNEL_CREATED | 日志（待 Kafka） | handleCreateReferralChannel | 审计（channel_name，禁止完整码刷屏） | 证据豁免：与申请事件同档 |
| 禁用推荐渠道 | REFERRAL_CHANNEL_DISABLED | 日志（待 Kafka） | handleDisableReferralChannel | 审计 | 证据豁免：首期仅结构化日志 |

## 变更记录

| 日期 | 变更 | 原因 |
|------|------|------|
| 2026-07-25 | 初始实现（Go 独立服务，端口由配置决定） | 推荐码系统从 Django 迁至 Go 微服务 |
| 2026-07-30 | 新增推荐统计端点 + billDB 支持 | OPT-049 Django→Go 迁移补全 |
| 2026-08-19 | 状态接口始终返回不透明 `access_code`（非 userId） | 无推荐资格时分享链接 accessCode 为空；禁止 `u{userId}` |
| 2026-08-20 | 申请必填个人介绍 20–500 字，落库并给超管展示 | 分成名额有限，审批需要申请人背景 |
| 2026-08-20 | 注册期 bind-from-code 接线 + 禁止解析 u{userId} | 推荐页人数/收益为 0：边从未写入 |
| 2026-08-20 | 多渠道码 + 分渠道/时段统计；绑边资格快照 | 不同投放渠道要独立链接与报表；无资格消费不分账 |
| 2026-08-21 | `phone_register` 已有手机号返回 400 且不绑边；taskReferral 经 conf-sync 读 taskBill `internalSecret` | 已注册用户再走注册页曾提示「注册成功」；sync-edge 无密钥则新用户边也写不进 |
| 2026-08-21 | 管理员推荐绩效支付口径与 taskBill 累计支付对齐；被推荐人回填 `as_referred` | 误查 `transaction_type IN (user_recharge_*)` 导致明细全 0；列表默认点开被推荐人显示「0 人」 |
| 2026-08-21 | `referral_rate_display` 从 taskBill 微信分账最大比例接口获取，失败回退 5% | 推荐页硬编码 5%，与商户平台分账管理比例脱节 |
| 2026-08-21 | 管理员推荐绩效 `commission_rate_display` 同源微信分账比例 | 超管抽屉「待入账/已入账佣金（5%）」须与商户平台分账管理比例一致 |
| 2026-08-21 | 推荐资格开通后幂等登记微信分账接收方；status GET 对存量有资格用户补偿 | 用户已获资格但商户平台「管理分账接收方」查不到：`addProfitSharingReceiver` 从未被调用，且走未签名 HTTP |
| 2026-08-22 | `referral_rate_display` 改为打款口径 `min(上限, 5%)`，与 `receivers.amount` 同源；不再展示文档默认上限 30% | 误用合作伙伴 `max_ratio`/conf 30% 作为收益分成，与微信后台真实设定和实际分账金额都不一致 |
| 2026-08-22 | 查询/配置失败禁止回退本地 5%；内部 API 503；前端缺省展示「—」 | 失败回退 5% 会把故障显示成真实分成比例 |
| 2026-08-22 | 申请须勾选同意绑定账号标识与用户身份信息，落库 `identity_bind_consented_at` | 微信支付添加分账接收方要求：传输身份信息与账号标识做一致性校验须已合法征得用户授权 |
| 2026-08-22 | 推荐码管理可设分账比例/推荐比例；申请列表展示；打款 `min(分账, 微信上限)`，计提用推荐比例 | 超管申请表看不到比例，且比例不在推荐码管理中配置 |
| 2026-08-22 | 管理员只设一个分成比例（5–30）；5%~30% 为范围展示；两列落库同值 | 双输入被理解为区间两端，与「一个政策比例」不符 |
| 2026-08-22 | 内部 GET qualification/active 供 taskBill 支付时刻现查；微信分账不以绑边 `commission_eligible` 为准 | 先推荐后获资时旧边快照为 0，导致后续订单无法分账（ADR-0033） |
| 2026-08-22 | 用户推荐页「当前的分账比例」改读 `billing_referral_config.referral_rate_percent`；stats 不再直接调微信 merchant-configs（直连 400 导致「—」） | 页面展示的是后台当前政策比例，不是微信查询上限 |
| 2026-08-25 | 推荐页/渠道/无资格提示改为下单时现查资格文案；需求 6 点数计提与分账同口径 | 先推荐后获资仍可从已推荐用户后续下单分成，旧文案写「绑边时不分账」会误导 |
| 2026-08-25 | 分成比例固定 5%，不从配置读取；POST 忽略比例字段 | 产品要求比例不可变 |
