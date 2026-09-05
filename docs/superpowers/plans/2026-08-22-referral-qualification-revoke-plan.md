# 实施计划：推荐分账资格取消与操作审计

> Design / VS / NFR / DDD 见同主题 2026-08-22 文档。

## 任务清单

### T1 — DDL 与理由校验（Red→Green）
- [ ] `dataMigrate/taskReferral/008_referral_qualification_revoke.sql`：status 允许 revoked；建 `referral_qualification_audit`（utf8mb4）
- [ ] 测试：`validateReviewReason` 空/短/超长失败，8 字通过
- [ ] 测试夹具建审计表

### T2 — 领域：revoke + 审计写入
- [ ] 测试：pending 不能 revoke；approved 未过期可 revoke；重复 revoke 200 already_revoked
- [ ] 测试：approve/reject 无 reason 失败；有 reason 写入审计
- [ ] 实现 `insertQualificationAudit` / `revokeReferralQualification` / approve&reject 签名加 reason
- [ ] 发布 `REFERRAL_QUALIFICATION_REVOKED`（结构化日志）

### T3 — HTTP + 网关
- [ ] `POST .../revoke/`、`GET .../audit/`；approve/reject 读 reason + Idempotency-Key
- [ ] handlers_test：非超管 403；revoke 200；audit 列表
- [ ] `taskGateway/routes/routes.yaml` 已有 `applications/*`，确认覆盖新子路径
- [ ] `db/api_route_ownership.yaml` 注明 system-admin referral owner=task-referral

### T4 — taskBill 停未来分账
- [ ] 测试：disable 将边置 0 且 pending→voided，settled 不变；重复调用仍 200
- [ ] `POST /api/internal/taskbill/referral/disable-eligibility/`
- [ ] taskReferral revoke 成功后调用该 API（失败只记日志）

### T5 — 前端
- [ ] 测试：approved+is_active 显示「取消资格」而非单独「—」
- [ ] 测试：通过/拒绝/取消弹出理由且短理由不发请求；提交带 Idempotency-Key
- [ ] 测试：审计抽屉请求 GET audit
- [ ] `useReferralApplications.js` + Panel；`createClickGuard`

### T6 — 意图 / 价值流图 / 列表字段
- [ ] 更新 `docs/intents/backend/task-referral.intent.md` 与 test-intent
- [ ] `docs/flows/value-stream-test-integration.wsd` 增加取消资格测试点
- [ ] 列表返回 `is_active` `can_revoke` `status_display` `last_action_reason`

### T7 — 事件对照
- [ ] 意图表增加 REFERRAL_QUALIFICATION_REVOKED
- [ ] 日志字段：app_id, operator, reason_len, trace_id

## 验证命令

```bash
cd taskReferral && go test ./src -count=1 -timeout 60s
cd taskBill && go test ./src -count=1 -run 'DisableEligibility|ReferralAccrue' -timeout 60s
cd taskFE/app && npx vitest run src/components/SystemAdminReferralApplicationsPanel.test.js src/composables/useReferralApplications.revoke.test.js
```
