# 邮件邀请退订 — 实施计划

- **日期**: 2026-08-29
- **设计/NFR/DDD**: 同主题 2026-08-29 文档

## Tasks

- [x] **T1 Domain** `taskAuth/domain/email_unsubscription.go` + `_test.go`：Normalize、Sign/Parse、幂等语义
- [x] **T2 DDL** `dataMigrate/taskAuth/049_email_unsubscription.sql` utf8mb4 + UNIQUE(email)
- [x] **T3 Public+Internal handlers** + 事件 EMAIL_UNSUBSCRIBED；路由 handlers.go；conf HMAC
- [x] **T4 Gateway** `routes/routes.yaml` public unsubscribe；`routes-apply`；`db/api_route_ownership.yaml`
- [x] **T5 Super-admin invite skip** `auth_email_invite.go` + tests
- [x] **T6 Tenant invite skip** `invite_handlers.go` 调内部 API + tests（fake checker）
- [x] **T7 Templates + taskEvents** unsubscribe_url、List-Unsubscribe、skip SMTP
- [x] **T8 FE** `UnsubscribeConfirm.vue` + publicRoutes；PeopleInvite + SystemAdmin 复制提示；削行数
- [x] **T9 Observability** slog `email_unsubscribed` / `invite_email_skipped`（无完整 token/PII 邮箱可打规范化）
- [x] **T10 Intent/gateway catalog** 同步 `docs/architecture/service-url-api-catalog.json` 若有生成脚本则跑生成
