# NFR 澄清：特权角色一号多账号绑定

- **日期:** 2026-08-28
- **级别:** 认证域 L3；其余 L2
- **价值流:** `docs/superpowers/plans/2026-08-28-privileged-phone-multi-bind-value-stream.md`

## 路径分片键强制审视

| 路径 | 分片 ID | 适配性 | 可伸缩性 | 动作 |
|------|---------|--------|----------|------|
| POST `/api/accounts/users/profile/bind-phone/` | `user_id`（token） | 用户级绑定，非租户分片 | L0 | 手机占用按全局 (cc,national) 计数；年增量远低于百万；升级触发：单号查询 P95&gt;50ms 再考虑号段索引 |
| POST `/api/accounts/users/profile/replace-phone/` | 同上 | 同上 | L0 | 同上 |
| POST `/api/accounts/users/bind_phone/` | 同上 | 同上 | L0 | 同上 |
| POST `/api/internal/login-methods/phone-taken/` | `exclude_user_id` | 内部只读占用 | L0 | 理由：内部低 QPS |
| PATCH `/api/internal/users/id/{user_id}/phone-login-method/` | `user_id` | 用户主键合适 | L0 | 无纠正 |
| POST `/api/accounts/login/`（phone+password） | 无租户键 | 登录全局标识查找 | L0 | 既有路径；共享后最多扫 5 行 password verify |
| POST send_password_reset_code / reset_password_with_code | 无租户键 | 同上 | L0 | 共享时直接拒绝，不扫改多行 |
| POST phone_register | 无租户键 | 公开注册全局唯一 | L0 | 不放宽 |

升级触发：单号活跃绑定需要 >5（产品变更）或 login_method 全表扫描成为瓶颈。

## 幂等性强制审视

| 路径 | 副作用 | 重复触发源 | 业务重复边界 | 幂等键 | 重放语义 |
|------|--------|------------|--------------|--------|----------|
| bind-phone | 写 login_method | 双击 / 重试 | (user_id, cc, national) 活跃 phone 行 | 前端 `Idempotency-Key`（既有 clickGuard）+ 服务端「已是自己的号 → upsert 同一行」 | 同用户重复绑定 200，不新增第二行 |
| replace-phone | 同上 | 同上 | 同上 | 同上 | 同上 |
| reclaim | 作废他人+写入自己 | 重复 reclaim | 该号全部他人绑定作废 | 短信 code 单次消费（既有） | 验证码已用 → 400，不二次作废 |
| internal upsert | 写 login_method | 内部重试 | 同 bind | 无 HTTP 幂等键；DB 同人单行 phone | 重放更新同一行 |
| login | 发 token / 登录历史 | 双击 | 一次成功登录一条 history | 既有 token getOrCreate | 重复登录复用 token |
| send reset code | 发短信 | 双击 | 共享号无副作用（拒绝） | 既有发送限流 | ≥2 绑定不发短信 |
| phone_register | 建用户 | 双击 | 号码全局独占 | 既有占用检查 | 第二次 400 |

资金/云资源：无。认证写路径 L3：验证码消费与占用检查顺序保持「先判定占用再 consume」（taken 不耗码）；limit 同样不耗码。

## 类别支撑程度

| 类别 | 级别 | 说明 |
|------|------|------|
| 安全 | L3 | fail-closed 消歧；不泄露 user_id 列表；不静默共享客户号 |
| 数据一致性 | L3 | 活跃绑定计数与角色在同一请求内读；无跨服务双写 |
| 性能 | L2 | 每号最多 5 次 bcrypt |
| 可观测性 | L2 | `phone_bound` / `phone_bind_rejected` / `login rejected` |
| 可伸缩性 | L0 | 见上表 |
| 容错 | L2 | DB 错误 500；策略拒绝 4xx |

## 质量场景

1. **刺激:** 第 6 个 tester 绑定同一号。**响应:** 409 limit，验证码仍可用，库中仍 5 行。
2. **刺激:** 两 staff 不同密码用同一号登录。**响应:** 各自进入自己的账号。
3. **刺激:** 共享号请求短信重置。**响应:** 400 ambiguous，无短信供应商调用。

## 领域模型影响

- 引入值对象「共享额度」与领域服务 `EvaluatePhoneShareBind`，聚合根仍是 User + LoginMethod（不新建表）。
- 查找端口从 `FindOneByPhone` 扩展为 `ListLiveByPhone`（最多 5+余量防御）。
