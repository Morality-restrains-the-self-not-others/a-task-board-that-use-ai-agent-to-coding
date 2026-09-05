# 设计：推荐分账资格取消与操作审计

- **Date:** 2026-08-22
- **Status:** accepted（goal-mode 自动采用）
- **Architecture:** v95
- **ADR:** ADR-0029
- **Page:** `/system-admin/users/?tab=referral-apps`

## 问题

系统管理员「推荐码申请列表」第 9 列「操作」仅对 `pending` 提供「通过 / 拒绝」；已通过（含活跃分账资格）行渲染为「—」，无法取消分账资格。通过操作不要求理由；拒绝理由可选；无独立审计表。资金路径缺少可检索的操作人+理由记录。

## 决策

1. **取消资格**：对 `status=approved` 且未过期的申请，操作列展示「取消资格」。调用  
   `POST /api/system-admin/referral/applications/{id}/revoke/`  
   将资格置为 `revoked`（新状态，区别于自然过期 `expired` 与审批拒绝 `rejected`）。
2. **理由强制**：通过 / 拒绝 / 取消均要求 `reason`（8–500 字）。前端弹窗确认；服务端校验。
3. **审计 SSOT**：表 `referral_qualification_audit`（owner: taskReferral）。每条操作追加一行：`application_id`、`user_id`、`action`（`approve`/`reject`/`revoke`/`auto_approve`）、`reason`、`operator_id`、`idempotency_key`、`trace_id`、`created_at`。列表「拒绝原因」列改为「操作理由」，展示最近一次理由。
4. **审计查阅**：`GET /api/system-admin/referral/applications/{id}/audit/?limit=&offset=` 分页返回该申请的操作历史。行内「审计」打开只读列表。
5. **取消的资金副作用**（不追回已结算）：
   - 本地资格失效 → `referrerHasActiveQualification=false`，新绑边 `commission_eligible=0`
   - taskBill `POST /api/internal/taskbill/referral/disable-eligibility/`：该推荐人全部边 `commission_eligible=0`；`pending` 计提改 `voided`（`void_reason=qualification_revoked`）
   - 已 `settled` 佣金与已发起的微信分账单不回滚
   - 微信删除接收方 best-effort，失败不回滚资格取消
6. **再申请**：`revoked` 无 7 天冷却（冷却仅 `rejected`）。分享码保留。
7. **幂等**：同一 `Idempotency-Key` 重放返回首次结果；同一申请重复 revoke 返回 200 `already_revoked` 且不再写第二笔副作用。
8. **权限**：仅 superuser；非超管 403。
9. **不新建服务**：扩展 taskReferral / taskBill / taskFE。

## 契约

### 管理员

| 方法 | 路径 | Body | 成功 |
|------|------|------|------|
| POST | `/api/system-admin/referral/applications/{id}/approve/` | `{ "reason": "..." }` + `Idempotency-Key` | 200 `{ status, expires_at, ... }` |
| POST | `/api/system-admin/referral/applications/{id}/reject/` | `{ "reason": "..." }` + `Idempotency-Key` | 200 `{ status }` |
| POST | `/api/system-admin/referral/applications/{id}/revoke/` | `{ "reason": "..." }` + `Idempotency-Key` | 200 `{ status: "revoked" }` |
| GET | `/api/system-admin/referral/applications/{id}/audit/` | query `limit`/`offset` | 200 `{ items, total }` |

错误：`reason` 不合规则 400 `invalid_reason`；非 pending 上 approve/reject → 400；非活跃 approved 上 revoke → 400 `not_revocable`。

列表项新增：`is_active`、`can_revoke`、`status_display`、`last_action_reason`。筛选增加 `revoked`。

### 内部（taskBill）

`POST /api/internal/taskbill/referral/disable-eligibility/`  
`{ "referrer_user_id": "...", "reason": "qualification_revoked" }`  
需 `X-TaskBill-Internal-Secret`。幂等：边已 0、pending 已 void 时仍 200。

## 前端

- 待审批：通过 / 拒绝 → 理由弹窗（必填）→ 提交
- 已通过且活跃：取消资格 → 理由弹窗 → 提交；不再显示单独的「—」
- 任意行：审计按钮查看历史
- `createClickGuard` + `Idempotency-Key`；错误带 `data-traceId`

## 事件

| 意图 | 事件 | 发布点 |
|------|------|--------|
| 审批通过 | `REFERRAL_APPROVED` | `approveReferralApplication`（补 `reason`） |
| 拒绝 | `REFERRAL_REJECTED` | 已有 |
| 取消资格 | `REFERRAL_QUALIFICATION_REVOKED` | `revokeReferralQualification` |
| 审计落库 | 同行结构化日志 | 审计表写入后 |

首期仍为结构化日志（与现网审批事件同档），待 Kafka 不阻塞本增量。

## 非目标

- 不追回已结算佣金 / 已成功微信分账
- 不改用户侧申请表单
- 不把分享码作废
- 不引入新角色

## Schema 评估（冷热 / 前缀）

| 表 | 伸缩 | 年增量 | 策略 |
|----|------|--------|------|
| `referral_qualification_audit` | 时间累积 | ≪ 10 万 | 全量热表保留（合规审计不删）；`created_at` 索引；暂不分区 |
| `referral_code.status` | 无 | — | 枚举扩 `revoked` |

前缀 `referral_` 符合 taskReferral。

## 🕸️ Code Review Graph 分析

- Step 0：`code-review-graph update --brief` 成功（6 文件增量，与本主题无关）。
- 现状调用链：`SystemAdminReferralApplicationsPanel` → `useReferralApplications` → `POST .../approve|reject` → `handleAdminReferral*` → `approve/rejectReferralApplication` → `referral_code`；资格门闩 `referrerHasActiveQualification` → 绑边 `commission_eligible` → taskBill 计提/分账。
- MCP `codegraph_explore` 本环境不可用，设计基于源码阅读。

## 验收

- 活跃资格行操作列出现「取消资格」，提交必填理由后状态为已取消，用户 `has_active_code=false`
- 通过/拒绝无理由被拒；有理由则审计表可查
- 取消后新消费不再计提；已结算佣金不变
- 审计列表含操作人、时间、理由
