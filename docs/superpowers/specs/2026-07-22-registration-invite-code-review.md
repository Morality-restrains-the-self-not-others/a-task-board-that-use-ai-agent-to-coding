# 注册邀请码 — Code Review

- **日期**: 2026-07-22
- **对照计划**: `docs/superpowers/plans/2026-07-22-registration-invite-code-plan.md`

## 结论

**通过（可开 PR）** — 无阻断级问题；重要项已在实现中覆盖。

## 检查项

| 项 | 结果 |
|---|---|
| Go-first / 无新 Django 公网 API | ✅ |
| 单表所有权 task-auth.db | ✅ |
| 注册开启时强制 invite_code + 核销 | ✅ |
| Admin superuser 门禁 | ✅ |
| 网关路由优先级高于 django system-admin | ✅ |
| OpenAPI 更新 | ✅ |
| 前端 data-traceId 错误展示 | ✅（panel/field） |
| 单测 T1/T2/T3/T5/T6/T8 | ✅ `go test ./...` |
| Intent→Event publish 点 | ✅ POLICY/ISSUED/REDEEMED |
| Log Audit | ✅ `[taskAuth]` 关键路径有日志 |

## CRG 风险

- graph_status: sparse；`detect-changes --brief` 风险分 0（图未索引 taskAuth）
- 人工风险：配额并发、X-User-Id 直连伪造（端口不公网）

## 残留（非阻断）

- Login.vue 仍为遗留超大文件（本次净减少行数）
- Kafka 不可用时事件 best-effort（与 EMAIL_SENT 同类）
