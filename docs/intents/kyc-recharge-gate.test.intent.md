# KYC 充值门禁 — 测试意图

| # | 场景 | 预期 |
|---|------|------|
| T1 | taskAuth evaluate T0 + 金额>0 | `allowed=false`, `KYC_TIER_BLOCKED` |
| T2 | taskBill PayPal create 经 KYC mock 拒绝 | HTTP 拒绝 + reason_code |
| T3 | Django `GET /api/accounts/me/kyc/` 已登录 | 200 含 tier/limits（上游 mock） |
| T4 | 非超管访问 system_admin KYC | 403 |
| T5 | Kafka registry 含 KYC_* topics | 注册表断言通过 |
| T6 | 超管 KYC drawer 标题区等级说明 | 可见 `kyc-tier-help` 含 T0/T1/T2 文案（unit + playwright） |
