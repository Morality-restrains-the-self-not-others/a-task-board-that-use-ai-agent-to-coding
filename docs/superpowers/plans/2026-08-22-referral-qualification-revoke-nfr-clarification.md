# NFR 澄清：推荐分账资格取消与操作审计

> Design: `docs/superpowers/specs/2026-08-22-referral-qualification-revoke-design.md`  
> Value stream: `docs/superpowers/plans/2026-08-22-referral-qualification-revoke-value-stream.md`

资金/资格写路径默认 **L3**。

## 路径分片键强制审视

| 路径 | 分片 ID | 适配性 | 可伸缩性 | 动作 |
|------|---------|--------|----------|------|
| `POST /api/system-admin/referral/applications/{id}/approve/` | `application_id` | 平台级资格，非租户分片；基数小 | **L0** | 理由：超管操作 QPS 极低；升级触发：申请表 >100 万行再按 user_id 分片 |
| `POST .../reject/` | `application_id` | 同上 | L0 | 同上 |
| `POST .../revoke/` | `application_id` + 副作用 `referrer_user_id` | 资格按 user 唯一活跃；边表按 referred_user_id | L1 纠正 | disable-eligibility 必须带 `referrer_user_id`（已是边定位键），禁止全表扫未带推荐人 |
| `GET .../applications/` | 无租户键；`status` 过滤 | 平台超管列表 | L0 | 已有 limit/offset；升级触发：单次扫描 >1 万则加 covering index |
| `GET .../audit/` | `application_id` | 审计按申请聚集 | L0 | `INDEX (application_id, created_at)` |
| `/system-admin/users/?tab=referral-apps` | 无 | 单页超管 | L0 | — |
| `POST /api/internal/taskbill/referral/disable-eligibility/` | `referrer_user_id` | 边/计提均有 referrer 列，适合作为更新键 | L1 通过 | WHERE referrer_user_id=? |
| `REFERRAL_QUALIFICATION_REVOKED` 日志 | `user_id`/`app_id` | 首期非 Kafka 分区 | L0 | 待 Kafka 时用 `application_id` 作 key，禁止只用 tenant_id |

## 幂等性强制审视

| 路径 | 副作用 | 重复触发源 | 业务重复边界 | 幂等键 | 重放语义 | 键粒度 |
|------|--------|------------|--------------|--------|----------|--------|
| GET applications / GET audit | 无 | — | — | L0 | 只读 | — |
| POST approve | 改 status、写审计、ensure 微信接收方 | 双击、网络重试 | 同一申请从 pending→approved 一次 | `Idempotency-Key`；成功后 status≠pending 再调用 400 | 同 key 返回首次 200；不同 key 且已批准 → 400 not pending | 与申请 ID 同粒度，禁止 user_id |
| POST reject | 改 status、写审计 | 同上 | pending→rejected 一次 | 同上 | 同上 | 同上 |
| POST revoke | 改 status、写审计、disable-eligibility、void pending、删接收方 | 双击、重试 | 同一申请活跃资格取消一次 | `Idempotency-Key`；二次 revoke 无 key 冲突时 200 `already_revoked` 空操作 | 同 key 重放首次结果；已 revoked 不再 void 第二次 | `application_id`，禁止 user_id |
| POST disable-eligibility | 边置 0、void pending | revoke 重试 | 该 referrer 的边与 pending 计提 | `referrer_user_id` 状态机（已 0 / 已 void 仍 200） | 重复调用无额外破坏 | 与推荐人边界同粒度 |
| 前端点击 | 触发写 | 双击 | 同一次意图 | `createClickGuard` + 同一 `Idempotency-Key` | 进行中 disabled | — |

资金路径：**L3**（资格关闭 + 停计提）。

## 支撑程度

| 类别 | 等级 | 说明 |
|------|------|------|
| 数据一致性 | L3 | 资格与边资格最终一致：先提交本地 revoked 再调账单；账单失败记审计但仍保持 revoked（停本地新资格），提供补偿重试日志 |
| 容错 | L2 | 微信 delete 失败不回滚；账单失败 warn + trace |
| 安全 | L3 | superuser + 内部 secret；理由长度限制防日志炸弹 |
| 可观测性 | L2 | 结构化日志 event=`referral_qualification_revoked`，含 app_id/operator/trace_id，不含 intro 正文 |
| 可伸缩性 | L0 | 见路径表 |
| 性能 | L1 | 单申请更新 + 按 referrer 批量 update，无 N+1 UI |

## 质量场景

1. **双击取消**：100ms 内两次点击只产生 1 条审计、1 次边更新。
2. **取消后消费**：新 consumption 不计提；此前 settled 行不变。
3. **无理由提交**：400 `invalid_reason`，status 不变。
4. **非超管**：403。

## 领域模型影响

- 聚合：`ReferralQualification`（`referral_code` + 其审计）以 `application_id` 为一致边界。
- 跨上下文：Commission（taskBill）通过端口 DisableEligibility，不共享库。
- 幂等键持久化在审计表 `idempotency_key`。
