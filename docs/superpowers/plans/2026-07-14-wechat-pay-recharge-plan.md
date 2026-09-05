# 实施计划: 微信支付 Native 充值

> 输入: `docs/superpowers/specs/2026-07-14-wechat-pay-recharge-design.md`
> 价值流: `docs/superpowers/plans/2026-07-14-wechat-pay-recharge-value-stream.md`
> NFR: `docs/superpowers/plans/2026-07-14-wechat-pay-recharge-nfr-clarification.md`
> 权限: `docs/superpowers/specs/2026-07-14-wechat-pay-recharge-permission-analysis.md`

---

## 任务清单

### Phase 0: 配置与脚手架 (P0)

- [ ] **T0.1** 新增微信支付配置目录与样例
  - 文件: `conf/billing/wechatPay/config.yaml.example`、`pub_key.pem.example`（占位说明）
  - 内容: `mode`, `mch_id`, `app_id`, `pub_key_id`, `notify_url` 字段说明
  - 验证: `test -f conf/billing/wechatPay/config.yaml.example`

- [ ] **T0.2** 配置加载（Go）
  - 文件: `taskBill/src/wechatpay_config.go`（新建）
  - 逻辑: 缺 `merchant_private_key.pem` → `mode=mock`；live 校验必填项
  - 验证: `go test taskBill/src/ -run TestWechatPayConfig -v`

- [ ] **T0.3** 登记积分来源类型
  - 文件: 文档 + 前端/后端常量（`user_recharge_wechat`）
  - 验证: grep 全仓一致

### Phase 1: taskBill — mock 与入账 (P0)

- [ ] **T1.1** mock-complete internal API
  - 文件: `taskBill/src/wechatpay_mock.go`、`handlers.go`
  - 路由: `POST /api/internal/taskbill/wechat/mock-complete/`
  - 约束: `X-Internal-Secret` + `mode=mock`
  - 验证: `go test taskBill/src/ -run TestWechatMockComplete -v`

- [ ] **T1.2** 幂等入账 wiring
  - 文件: `taskBill/src/wechatpay_credit.go`
  - txn: `wechat:{out_trade_no}`；`points_source_type=user_recharge_wechat`
  - 验证: 重复调用返回 `duplicate: true`

- [ ] **T1.3** pending 订单缓存
  - 文件: `taskBill/src/wechatpay_pending.go`
  - Key: `wechat_recharge_pending:{out_trade_no}`；TTL 15min
  - 验证: 单测 create pending → mock complete → 清理

### Phase 2: taskBill — live 预下单与回调 (P1)

- [ ] **T2.1** 引入 wechatpay-go SDK
  - 文件: `taskBill/go.mod`
  - 依赖: `github.com/wechatpay-apiv3/wechatpay-go`
  - Cipher: `WithWechatPayPublicKeyAuthCipher`
  - 验证: `go mod tidy && go build ./...`

- [ ] **T2.2** Native 预下单
  - 文件: `taskBill/src/wechatpay_native.go`
  - 返回: `code_url`, `out_trade_no`, `expires_at`
  - mock 模式: 返回固定占位 `code_url`
  - 验证: `go test -run TestWechatNativePrepay`

- [ ] **T2.3** 回调 notify handler
  - 文件: `taskBill/src/wechatpay_notify.go`
  - 路由: `POST /api/billing/wechat/notify/`
  - 逻辑: 验签 → 解密 → 校验金额 → credit
  - 验证: 假签名拒绝 + 真签名 fixture（沙箱）

- [ ] **T2.4** 状态查询
  - 文件: `taskBill/src/wechatpay_status.go`
  - 路由: delegate `recharge_wechat_status`
  - 验证: pending/success/failed 分支单测

- [ ] **T2.5** handlers 路由挂载
  - 文件: `taskBill/src/handlers.go`
  - 新增: notify、wechat delegate cases（与 paypal 平行）
  - 验证: 本地 curl health + route smoke

### Phase 3: Django billing_bridge — 薄编排 (P1)

- [ ] **T3.1** `RechargeWechatCreateView`
  - 文件: `billing_bridge/recharge_views.py`
  - 门禁: SMS + tenant（复制 PayPal create 模式）
  - 验证: `pytest tests/test_billing_recharge_validation.py -k wechat_create -v`

- [ ] **T3.2** `RechargeWechatStatusView`
  - 文件: `billing_bridge/recharge_views.py`
  - 约束: 订单归属当前用户
  - 验证: `pytest -k wechat_status -v`

- [ ] **T3.3** `recharge_phone_status` 扩展
  - 文件: `billing_bridge/recharge_views.py`
  - 字段: `wechat_enabled`, `wechat_mode`
  - 验证: 现有 phone_status 测试仍绿

- [ ] **T3.4** URL 注册
  - 文件: `billing_bridge/urls.py`
  - 路径: `recharge_wechat_create/`, `recharge_wechat_status/`
  - 验证: `python manage.py show_urls | grep wechat`

- [ ] **T3.5** internal delegate handlers（若沿用 Django 回环）
  - 文件: `billing_bridge/internal_views.py`（或等价 delegate 模块）
  - 验证: taskBill delegate 端到端 smoke

### Phase 4: 前端 (P1)

- [ ] **T4.1** 支付方式切换 UI
  - 文件: `front_project/app/src/views/BillingRecharge.vue`
  - 展示: PayPal | 微信支付 Tab/Radio
  - 验证: 本地手动 — wechat_enabled 时可见

- [ ] **T4.2** 微信 QR 展示
  - 文件: `BillingRecharge.vue` + QR 组件（复用现有或 qrcode lib）
  - 数据: `code_url`
  - 验证: mock create 后 QR 渲染

- [ ] **T4.3** 轮询状态
  - 文件: `BillingRecharge.vue`
  - 间隔: 2s；超时 120s；文案分级
  - 验证: mock-complete 后 success 展示

- [ ] **T4.4** mock 开发按钮（可选）
  - 条件: `wechat_mode === 'mock'` && 非生产
  - 验证: 仅 dev 构建可见

### Phase 5: 测试 (P1)

- [ ] **T5.1** Django 单测
  - 文件: `tests/test_billing_recharge_validation.py`
  - 用例: SMS 拒绝、跨用户 status、wechat_enabled 字段
  - 验证: `pytest tests/test_billing_recharge_validation.py -k wechat -v`

- [ ] **T5.2** Go 单测套件
  - 文件: `taskBill/src/wechatpay_*_test.go`
  - 覆盖: 验签失败、幂等、金额 mismatch
  - 验证: `go test ./... -count=1`

- [ ] **T5.3** Playwright E2E
  - 文件: `playwright/front_project/tests/BillingRecharge.wechat-mock.playwright.test.js`
  - 流: 选微信 → create → mock-complete API stub → 成功
  - 验证: `npx playwright test BillingRecharge.wechat-mock`

### Phase 6: Swagger / Ownership / 架构 (P2)

- [ ] **T6.1** api_route_ownership 登记
  - 文件: `db/api_route_ownership.yaml`
  - 条目: wechat create/status/notify
  - 验证: `python db/scripts/ci/check_django_new_api_routes.py`

- [ ] **T6.2** taskBill OpenAPI
  - 文件: `taskBill/openapi.yaml`（或项目约定路径）
  - 验证: Swagger UI 可渲染 notify/mock-complete

- [ ] **T6.3** 架构 v22 target 制品
  - 文件:
    - `docs/architecture/v22-application-integration-20260714-wechat-pay-claude.puml`
    - 同名 `.archimate`、`.mermaid.md`
    - `docs/architecture/VERSION_HISTORY.md` 更新
  - 验证: Archi CLI load + 视图连线自检

- [ ] **T6.4** value-stream.yaml 登记
  - 文件: `conf/value-stream.yaml`
  - 流 id: `wechat-pay-recharge`
  - 验证: YAML lint

### Phase 7: 文档与意图同步 (P2)

- [ ] **T7.1** 意图文档与索引
  - 文件: `docs/intents/wechat-pay-recharge.intent.md`、`.test.intent.md`
  - 验证: 文内验收标准可执行

- [ ] **T7.2** dev-commit / 变更记录（实施完成后）
  - 记录: 2026-07-14 设计包落地

---

## 依赖顺序

```
T0.* → T1.* → T2.* ∥ T3.* → T4.* → T5.* → T6.* → T7.*
```

## 验收门槛（合并前）

1. mock 模式 E2E 绿
2. Go 验签失败 + 幂等单测绿
3. Django SMS 门禁单测绿
4. 无密钥进日志（grep 审查）
5. `api_route_ownership.yaml` 已登记
6. v22 架构 target 三类文件存在
