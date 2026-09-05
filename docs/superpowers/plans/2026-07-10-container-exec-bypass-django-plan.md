# 实施计划：容器执行热路径零 Django（Phase A）

- **日期**: 2026-07-10
- **设计**: `docs/superpowers/specs/2026-07-10-container-exec-bypass-django-design.md`

## 现状校准

源码已接线（`auth_cloud_clients.go` / `authorizeContainerRequest`），但 **运行中二进制仍为 12:38 旧版**，日志持续 `django_validate` 502。本计划以「验证 + 收口 + 切流」为主。

## 任务清单

- [x] T1 跑通 taskAuth `container_gateway_validate` 单测
- [x] T2 跑通 taskContainerGateway 热路径相关单测（handlers / observability）
- [x] T3 删除热路径死代码 `djangoValidateSession` / `djangoResolveTarget`（保留 `djangoPost` 供冷路径）
- [x] T4 重编译并重启 `task-auth` / `task-cloud-service` / `task-container-gateway`
- [x] T5 运行时验收：新请求日志出现 `auth_validate`/`cloud_resolve`，无新的 `django_validate`
- [x] T6 更新设计文档状态为 Phase A shipped（冷路径 B–D 待续）
- [x] T7 Review 通过
- [x] T8 Phase B：job-stream → Kafka SSE（删除 Django publish）
- [x] T9 Phase C：git-push prepare/complete → taskCloudService
- [x] T10 Phase D：mock-run → tcg → go_run_container + gateway 路由
- [x] T11 taskAuth sessionid → django_session 解析
- [x] T12 删除未使用 `djangoPost`
- [x] T13 auth-context → Cloud；删除 djangoGet
- [x] T14 identity_id prepare 实现
- [x] T15 mock-run installed_image_id + env-defaults
- [x] T16 open-runtime-session → Cloud（tcg + taskEvents）
- [x] T17 删除 Django 已迁 tcg internal 路由

## 验证命令

```bash
cd taskAuth && go test ./src/ -count=1 -run 'ContainerGateway|Validate'
cd taskContainerGateway && go test ./src/ -count=1 -run 'Observability|JobEdit|Authorize|HotPath|GitPush'
# 重启后
rg 'auth_validate|django_validate' logs/task-container-gateway.log | tail
```
