# Test Intent: 用户推荐码资格申请与审批管理

对应功能意图：`task-referral.intent.md`

## 单元 / 集成测试

### 推荐码申请 (referral_code_test.go)

| ID | 场景 | 期望 |
|----|------|------|
| T1 | approval 模式申请 | status=pending，含 message |
| T2 | open 模式申请 | status=approved，referral_code 为不透明码（非 `u{userId}`），有效期 180 天；调用 ensure 微信分账接收方 |
| T3 | 重复申请（pending） | 400 `already_pending` |
| T4 | 已有活跃码再申请 | 400 `already_active`（含剩余天数提示） |
| T5 | 拒绝冷却期内申请 | 400 cooldown（含剩余天数提示） |
| T5b | 空/短/超长个人介绍 | 400 `invalid_intro`，不落行 |
| T5c | 合法介绍申请 | pending 且 `personal_intro` 入库；status/list 回传原文 |
| T5d | 未勾选身份绑定授权 | 400 `identity_bind_consent_required`，不落行 |
| T5e | 勾选后申请 | `identity_bind_consented_at` 非空 |

### 推荐码状态查询 (referral_code_test.go)

| ID | 场景 | 期望 |
|----|------|------|
| T6 | 无申请记录 | can_apply=true，application_status=null，**access_code 非空且非 userId 派生** |
| T7 | pending 状态 | application_status=pending，can_apply=false，**access_code 仍为同一不透明码** |
| T8 | approved 未过期 | has_active_code=true，referral_code 与 access_code 相同；调用 ensure 微信分账接收方 |
| T8b | 无活跃资格 | status 不调用 ensure |
| T9 | approved 已过期 | has_active_code=false，can_apply=true，**access_code 仍非空** |

### 审批/拒绝 (referral_code_test.go)

| ID | 场景 | 期望 |
|----|------|------|
| T10 | 审批通过 | status=approved，发放 referral_code；调用 ensure 微信分账接收方；无理由 400；写入审计 |
| T11 | 拒绝申请 | status=rejected，含拒绝原因，标记冷却；无理由 400 |
| T50 | 取消活跃资格 | status=revoked，可立即再申请；重复 revoke 200 already_revoked |
| T51 | pending 取消 | 400 not_revocable |
| T52 | 审计列表 | GET audit 返回操作人/理由 |

### 过期扫描 (referral_code_test.go)

| ID | 场景 | 期望 |
|----|------|------|
| T12 | 过期码自动标记 | 过期 approved→expired；未过期保持 approved；pending 不受影响 |

### 申请列表 (referral_code_test.go)

| ID | 场景 | 期望 |
|----|------|------|
| T13 | 按状态筛选 | pending 筛选返回正确数量和条目 |
| T14 | 全部列表 | status 为空时返回全部 |
| T53 | 列表带比例 | 即使配置表写入 12%，每行 `referral_rate_display=5%` 且 `profit_sharing_ratio_display=5%` |
| T54 | 推荐资格管理 POST 比例 | POST `referral_rate_percent=18` 被忽略，响应仍为 5%；不因 4 或 31 返回 400 |
| T55 | 申请表列 | `SystemAdminReferralApplicationsPanel` 表头含「分成比例」一列，行内展示接口回传值 |

### 推荐统计 (referral_stats_test.go)

| ID | 场景 | 期望 |
|----|------|------|
| T15 | 无数据 | referral_count=0，monthly_earnings 为空 |
| T15b | 推荐比例 | 库写入 12% 或微信返回 12% 时 stats/status/performance 仍展示 5% |
| T16 | 有推荐关系 | referral_count 正确 |
| T17 | 有月度收益 | 按月聚合消费/佣金点数→元，按月份 DESC 排序 |
| T18 | billDB 不可用 | 返回 "billDB unavailable" 错误 |
| T18b | 按 channel_code / from / to 过滤 | 人数与分账只含匹配渠道与时间窗 |
| T18c | 分渠道汇总 | channels[] 含各渠道人数与佣金 |

### 管理员推荐绩效 (referral_performance_test.go)

| ID | 场景 | 期望 |
|----|------|------|
| T37 | 被推荐人 `resource_purchase` | `referred_users` 计 1 笔 / 0.55 元；`admin_grant` 不计 |
| T38 | `resource_purchase.user_id` 为空 | 经同账户反查仍计入该被推荐人 |
| T39 | 打开被推荐人（无向下推荐） | `as_referred` 含推荐人 ID 与自身实付 |
| T40 | 错误地把 `transaction_type=user_recharge_wechat` 当支付 | 不计（须 `transaction_type=recharge` + `points_source_type`） |
| T41 | 推荐比例 | 配置表写入 12% 时 `commission_rate_display=5%` |
| T48 | 比例只来自微信接口 | mock 微信 200 `max_ratio=1500` 展示 15%；直连 400 则 503、source=unavailable，不得展示 conf 30% 或本地 5% |
| T49 | 失败不回退 5% | conf 越界或 query 失败：内部 API 503、source=unavailable、打款比例 0、referral/前端不得显示 5% |

### 多渠道码 (referral_share_code_test.go / referral_channel_test.go)

| ID | 场景 | 期望 |
|----|------|------|
| T32 | 默认渠道 + 第二渠道 | 两码不同；lookup 各自指向同一 user |
| T33 | 同名 POST 两次 | 仍 1 行，返回同一 code |
| T34 | 禁用后 lookup | 空；历史边仍可统计 |
| T35 | 无资格绑边 | sync-edge commission_eligible=false；人数仍计 |

### HTTP 端点 (handlers_test.go)

| ID | 场景 | 期望 |
|----|------|------|
| T19 | 健康检查 | 200，status=ok，service=taskReferral |
| T20 | 未认证访问 | 401 |
| T21 | 方法不允许 | 405 |
| T22 | 非超管访问管理员端点 | 403 |
| T23 | 超管访问管理员端点 | 200 |
| T24 | 管理员列表含状态筛选 | 筛选后仅返回匹配条目 |
| T25 | 无效申请 ID 审批 | 400 |
| T26 | 无效策略模式更新 | 400 |

### 注册绑边 (referral_bind_test.go)

| ID | 场景 | 期望 |
|----|------|------|
| T27 | 不透明码反查 | lookupShareCodeOwner 返回对应 user_id |
| T28 | 未知码 / `u{userId}` | 反查为空，不解析 userId |
| T29 | 自荐 | Bind 返回 skipped/self_referral |
| T30 | 空码 | skipped/empty_code，不调 sync-edge |
| T31 | HTTP bind-from-code | 调 tenant by-creator + taskBill sync-edge |
| T31b | sync-edge 密钥 | `BillInternalSecret` 非空时请求带 `X-TaskBill-Internal-Secret`；密钥来自本目录 `task-bill.yaml`（env 可覆盖） |
| T36 | 已有手机号 `phone_register`（含 `access_code`） | 400 `user_existed`「该手机号已被注册，请直接登录」；不调 bind-from-code；不改密 |

### 微信分账接收方登记 (referral_wechat_receiver_test.go / wechat_profit_sharing_receiver_test.go / auth_wechat_pay_openid_test.go)

| ID | 场景 | 期望 |
|----|------|------|
| T42 | 添加接收方 payload | PERSONAL_OPENID + DISTRIBUTOR，不含 name / custom_relation；appid 用身份所属应用 |
| T43 | 幂等 ensure | 同 user+openid+appid 只调微信 add 一次 |
| T44 | 无 openid | status=pending_openid，不调微信 |
| T45 | 微信未 live | status=skipped_not_live |
| T46 | 微信失败 | status=failed 落库 fail_reason |
| T47 | taskAuth 优先 preferred_app_id，否则任意已绑定身份 | 返回对应 openid，不向公开 API 暴露 |

### 支付时刻资格内部查询 (referral_qualification_internal_test.go)

| ID | 场景 | 期望 |
|----|------|------|
| T60 | GET `/api/internal/referral/qualification/active/?user_id=` | approved 未过期 `active=true`；revoked/无记录 `false`；缺 user_id 400 |

## 测试覆盖映射

| 测试文件 | 覆盖意图 |
|----------|---------|
| `referral_code_test.go` / `referral_personal_intro_test.go` / `referral_qualification_test.go` / `handlers_revoke_test.go` / `referral_identity_bind_consent_test.go` | T1–T14、T5b–T5e、T50–T52（申请/介绍/身份绑定授权/状态/审批/拒绝/取消/过期/列表/审计） |
| `taskBill/src/referral_disable_eligibility_test.go` | T50 副作用：边资格关闭、pending void、settled 保留 |
| `SystemAdminReferralApplicationsPanel.test.js` | 取消资格按钮、理由弹窗、Idempotency-Key、审计抽屉 |
| `referral_stats_test.go` | T15–T18、T15b（推荐统计；分成比例固定 5%，不读配置/微信） |
| `referral_performance_test.go` | T37–T41（管理员推荐绩效支付口径 / as_referred / 固定 5%） |
| `SystemAdminReferralPerformanceDrawer.test.js` | T41（抽屉佣金比例文案来自 `commission_rate_display`） |
| `handlers_test.go` | T19–T26（HTTP 端点/权限/策略） |
| `referral_share_code_test.go` / `referral_bind_test.go` / `referral_channel_test.go` / `config_bill_secret_test.go` | T27–T36（多渠道码、绑边跳过、资格快照、sync-edge 密钥） |
| `referral_wechat_receiver_test.go` | T8b / T2 / T10 / T42 接线（开通与 status 补偿登记） |
| `taskBill/src/wechat_profit_sharing_ratio_test.go` | T48 / T49（微信 max_ratio 查询失败 503 不回退 5%；政策比例另为固定 5%） |
| `taskFE/.../ReferralStatsPanel.test.js` / `UserReferral.accessCode.test.js` / `SystemAdminReferralPerformanceDrawer.test.js` | T49（前端缺省不替换成 5%） |
| `taskBill/src/wechat_profit_sharing_receiver_test.go` | T42–T46 |
| `taskAuth/src/auth_wechat_pay_openid_test.go` | T47 |
| `taskAuth/src/auth_phone_register_test.go` | T36（已有手机号 phone_register 400 不绑边） |
| `referral_qualification_internal_test.go` | T60 支付时刻现查：approved→active=true；revoked/缺参→false/400 |

## 关联

- 功能意图：`task-referral.intent.md`
- 相关服务：taskReferral（Go）、billDB（MySQL）
- 历史：原推荐码逻辑在 Django `taskAuth` 中，2026-07 迁至独立 Go 服务
