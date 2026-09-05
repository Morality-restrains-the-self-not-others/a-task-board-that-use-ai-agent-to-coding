# NFR 澄清 — 推荐资格个人介绍

- **日期:** 2026-08-20
- **价值流:** `docs/superpowers/plans/2026-08-20-referral-personal-intro-value-stream.md`
- **默认等级:** L2；安全按 L2（UGC 长度+转义）；非资金扣款故不对申请写路径强制 L3 资金级

## 路径分片键审视

| 路径 | 已携带 ID | 分片键判定 | 动作 |
|------|-----------|------------|------|
| POST `/api/accounts/users/referral-codes/apply/` | 会话 `user_id`（网关注入） | 合适：申请行按 user 归属；`personal_intro` 非分片键 | 无 |
| GET `/api/accounts/users/referral-codes/status/` | 会话 `user_id` | 合适 | 无 |
| GET `/api/system-admin/referral/applications/` | 无租户键 | L0 系统级超管扫表；年增量远低于分片阈值 | 维持现状，不按 tenant 拆 |
| POST approve/reject `{id}` | application `id` | 合适（实体键） | 无 |
| FE `/profile/referral/` | 登录用户 | 合适 | 无 |

无「缺分片键却声称无可伸缩性」的新公开路径。超管列表是存量系统扫，本次不扩大。

## 幂等性审视

| 路径 | 副作用 | 重复触发源 | 业务重复边界 | 幂等键 | 重放语义 | 等级 |
|------|--------|------------|--------------|--------|----------|------|
| POST apply | INSERT `referral_code` | 双击、超时重试 | 同一用户同时仅一条 pending/approved | 前端 `Idempotency-Key`（流量）+ 服务端 pending/active 业务互斥 | 首次 200；重复 pending → 400 `already_pending`（与现状一致，非资金双写） | L2 |
| GET status / GET list | 无 | — | — | — | L0 纯查询 | L0 |
| POST approve/reject | 状态转移 | 双击 | 该 `app_id` 从 pending 离开 | 行状态 `pending` 条件更新 | 非 pending → 400 `not_pending` | L1 自然幂等 |

禁止用 `user_id` 作为「永远只能申请一次」的消费幂等键：过期/冷却后必须能再申请（新行 + 新介绍）。

前端：`createClickGuard` + 同一次 `run` 内回传同一 `Idempotency-Key`。本期服务端不建独立幂等表（与现网 apply 一致）；名额控制仍由人工审批完成。

## 类别定级

| 类别 | 级别 | 说明 |
|------|------|------|
| 可伸缩性 | L2 | 列级 TEXT，无新表爆炸 |
| 数据一致性 | L2 | 与申请 INSERT 同语句 |
| 安全 | L2 | 长度上限、本人/超管读、Vue 文本插值、日志禁正文 |
| 可用性 | L2 | 非法介绍 400，不落行 |
| 性能 | L2 | 列表多一列 TEXT，分页已有 |
| 可观测性 | L2 | `personal_intro_len` + 既有 apply 失败日志 |

## 质量场景

1. 刺激：20 字介绍申请。响应：200 pending/approved，库中原文一致。
2. 刺激：19 字或空。响应：400 `invalid_intro`，无新行。
3. 刺激：501 字。响应：400。
4. 刺激：pending 时再 POST。响应：400 `already_pending`。
5. 刺激：超管列表。响应：条目含 `personal_intro`。

## 领域模型影响

`ReferralCode` 申请聚合增加不可变属性 `PersonalIntro`（创建后一期不改）。值对象：规范化后的介绍字符串。
