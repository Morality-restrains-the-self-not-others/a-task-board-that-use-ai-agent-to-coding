# 测试意图：超管用户列表按列过滤

| ID | 场景 | 期望 |
|----|------|------|
| T1 | `email` contains | 只返回 identifier 含该串的邮箱用户 |
| T2 | `role=superuser` | 只返回 is_superuser=1 |
| T3 | `login_method=phone` | 只返回有手机登录方式的用户 |
| T4 | `date_joined_from` | 不含该日之前注册的用户 |
| T5 | `is_active=false` | 只返回已禁用 |
| T6 | `id` contains | 命中 ID 子串 |
| T7 | `phone` contains | 命中手机标识 |
| T8 | `q` + `email` AND | 须同时满足 |
| T9 | 非超管 | 403 |
| T10 | `has_profit_sharing=true` | 只返回有资格用户（mock 下游） |
| T11 | `tenant_company` | 公司名 contains（mock 下游） |
| T12 | `referrer` | 推荐人展示名 contains（mock 下游） |
