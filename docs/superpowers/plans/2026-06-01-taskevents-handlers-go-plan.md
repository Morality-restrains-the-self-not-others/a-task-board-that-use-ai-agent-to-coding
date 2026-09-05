# taskEvents Handler Go 化 — 实施计划

## G0 骨架

- [x] `config/event_registry.go` + `LoadEvent(slug)`
- [x] `eventbin/run.go` 单事件启动器
- [x] `bin/_template/` 脚手架
- [x] `run.sh` 支持 `build|start|stop|status {event_slug}`
- [x] `port_config.json` 8020+ 事件消费者键
- [x] `bin/README.md`

## G1 低垂果实

- [x] `internal/handlers/billing` + `bin/billing_transaction_created`
- [x] `internal/handlers/sse` + `bin/sse_message`
- [x] 单元测试 + `go test ./...`

## G1′ notifications 拆分

- [x] `bin/email_sent`, `bin/invitation_created`, `bin/user_activated`
- [x] 复用 `notifications.LocalDelivery` + `filter.SingleEvent`

## G2（已完成）

- [x] `internal/repository/saas` + `internal/publish` + `bin/user_created`
- [x] `bin/company_created`（deliverable → progress → workspace 链 + WORKSPACE_CREATED）

## G3（已完成）

- [x] `bin/workspace_created` — 管理员 + 交付物/进度体系设置
- [x] `bin/project_updated` — 审计日志（对齐 Python TODO）
- [x] `bin/task_completed` — 审计日志
- [x] `bin/ai_assistant_reply_completed` — 写入 `projects_todo_ai_comment.assistant_response`

## G4（已完成 — 见已知缺口）

- [x] `bin/cloud_server_stopped` — 审计日志
- [x] `bin/cloud_platform_authorization_created` — STS GetCallerIdentity + IAM 关联 + SSE
- [x] `bin/cloud_server_started` — 事件状态 + ECS RunInstances（最小字段）+ 配置落库 + SSE
- [x] `bin/cloud_server_start_auto` — 链式 CLOUD_SERVER_STARTED（**auto_create_* 未实现**，需预填 vpc/sg/vswitch）

## G4.5 E2E 验证（已完成）

- [x] runbook + `g45_verify.sh` 门禁
- [x] v4 18 intent 二进制 + Go integration（注册链 + company fan-out）
- [x] runAll：`domain-events-intents`

## G5（已完成）

- [x] 删除 `core/kafka/handlers/`、域级 `cmd/*`、Django dispatch 路由
- [x] runAll 移除 `domain-events` / `domain-events-bin`
