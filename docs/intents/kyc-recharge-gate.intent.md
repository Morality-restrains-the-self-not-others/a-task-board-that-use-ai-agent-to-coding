# KYC 身份等级与充值门禁

## 意图
平台对用户充值施加 KYC 身份等级限额；超管可查看/覆盖等级与 AML 提示；用户在充值页可见当前限额。

## 范围
- **taskAuth**：KYC profile / evaluate / admin-override / AML SSOT（internal API）
- **task2app**：Django 薄代理（`/api/accounts/me/kyc/`、`/api/system_admin/users/<id>/kyc/*`）；充值页 banner；系统管理用户 KYC drawer；Kafka topic 登记
- **taskBill**：微信/PayPal create 前调用 taskAuth evaluate；结构化拒绝码

## 非目标（MVP）
- 管理员 `credit-recharge` 不强制 KYC（见 OPT-20260722-024）
- 无独立 AML list GET（超管从 audit 推导 latest_aml）

## 业务意图 → 事件对照

| 业务意图 | 事件名（过去式） | MQ类型/契约 | 发布点 | 消费者/副作用 | 例外理由 |
|---------|----------------|------------|--------|--------------|---------|
| KYC 等级变更（含超管 override） | kyc-tier-changed | KYC_TIER_CHANGED | taskAuth `publishKycEvent` / kyc_policy | Kafka topic 登记 / 审计 | — |
| KYC 状态变更 | kyc-status-changed | KYC_STATUS_CHANGED | taskAuth `publishKycEvent` | Kafka topic 登记 / 审计 | — |
| 充值前 evaluate（只读判定） | — | — | — | — | 无对应事件：同步门禁查询，不产生新聚合跃迁 |
| 用户/超管查看 KYC profile/audit | — | — | — | — | 无对应事件：只读查询 |

## 验收
- 用户侧低等级充值被拒绝并返回 `KYC_*` reason
- 超管可打开 KYC drawer 查看 profile/audit
- 超管 KYC drawer 标题区展示 T0/T1/T2 各等级说明（`data-testid=kyc-tier-help`）
- 相关单测：`taskAuth` kyc_*、`taskBill` kyc_gate、`task2app` test_kyc_proxy_views、`SystemAdminUserKycDrawer.tier-help.unit.test.js`
