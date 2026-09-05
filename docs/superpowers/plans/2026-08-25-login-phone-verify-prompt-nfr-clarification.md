# NFR 澄清：登录后未验证手机号弹窗引导

- 日期：2026-08-25
- 级别：认证 UX 相关项 L3（安全文案/PII），其余 L2
- 关联价值流：2026-08-25-login-phone-verify-prompt-value-stream.md

## 路径分片键强制审视

| 路径 | 分片 ID | 判定 | 动作 |
|------|---------|------|------|
| POST `/api/auth/`（存量） | 无租户键；按 user 登录 | L0：认证入口全局、不可按 tenant 分片 | 不改路由；升级触发：登录 QPS 需独立分片时再评估 user_id 哈希 |
| GET `/api/accounts/users/profile/`（存量） | 隐式当前 user_id | 合适：本人资料 | 不改 |
| POST bind-phone（存量） | 当前 user_id | 合适 | 不改 |
| 前端 `/auth/login/` | 无 | L0：公开登录页 | — |
| 前端 `/profile/#rg=profile.phone_binding` | 当前用户资料 | L0：单用户页 | 禁止拼他人 user id |

可伸缩性：全部路径 L0（单用户会话 UX，无列表扫描）。

## 幂等性强制审视

| 路径 | 副作用 | 重复触发 | 业务重复边界 | 幂等键 | 重放语义 |
|------|--------|----------|--------------|--------|----------|
| 弹窗展示 | 无（纯 UI） | 刷新/再登录 | — | L0 | 每次登录未绑定再提示是产品语义 |
| 「去验证」导航 | 无写 | 双击 | — | 前端 alert 关闭后跳转一次 | Anti-Replay-OK: 整页 location.href 或真实 a[href] |
| 业务页阻断层 | 无写 | 刷新 | — | L0 | 未验证每次进入业务页再拦 |
| bind-phone（存量，非本期） | 写 login_method | 重复提交 | 同一 user + 同一 E.164 | 既有 SMS+绑定接口 | 不改 |

无新写 API。用户点击走 `modalService.confirm` 关闭后再 `location.href`，不发新 POST。

## 其他 NFR

| 类别 | 级别 | 说明 |
|------|------|------|
| 安全 | L3 | 不在日志打印手机号；模拟登录/员工跳过；登录当下未知身份 fail-open，已判定未验证 fail-closed |
| 可用性 | L2 | 未验证必须先绑定；/me/ 失败 fail-open 不锁死整站 |
| 性能 | L2 | 密码登录用响应内 login_methods；门禁复用 GET /me/ |
| 可观测性 | L2 | `[login-phone-verify]` 前缀 + action/reason，无 PII |
| 可访问性 | L2 | 不可关闭叉的 alert + 阻断层真实 a[href] |

## 领域模型影响

- 引入前端值对象「手机已验证」：从 login_methods 或 has_phone 投影，不是新持久化实体。
- 不新增聚合/事件。
