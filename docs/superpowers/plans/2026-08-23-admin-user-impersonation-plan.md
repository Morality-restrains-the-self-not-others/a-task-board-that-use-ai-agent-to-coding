# 实施计划：管理员模拟用户登录

- **Date:** 2026-08-23
- **Goal:** `/system-admin/users/` 编辑模态「以该用户身份登录」，权限码 `user:impersonate`
- **ADR:** ADR-0037
- **设计:** `docs/superpowers/specs/2026-08-23-admin-user-impersonation-design.md`

## Task 1: Domain（已完成）

- [x] `taskAuth/domain/impersonation.go` + `_test.go`
- [x] `dataMigrate/taskAuth/036_impersonation_session.sql`

## Task 2: Store + Token 解析

- [ ] `impersonation_store.go`：签发 `imp_` token、open session 查询、结束会话
- [ ] `resolveTokenUserIDWithIP`：`imp_` 前缀查模拟表，返回 **target**，跳过 IP binding
- [ ] 回归：`go test ./src -run 'Impersonation|ValidateStart'`

## Task 3: HTTP handlers（TDD）

- [ ] `POST /api/system-admin/users/{id}/impersonate/` — `RequirePlatformPerm(user:impersonate)`，**不**套 `requireSuperuser`
- [ ] `POST /api/auth/impersonation/stop/`、`GET /api/auth/impersonation/status/`
- [ ] 错误：缺 key 400 / 无权限 403 / 自己或嵌套 409 / 不可登录 422 / 不存在 404
- [ ] 幂等：同 actor+key 返回同一 open session
- [ ] 测试：`taskAuth/src/handlers_impersonation_test.go`

## Task 4: 事件 + OpenAPI

- [ ] `eventTopicMap` + `publishUserImpersonationStarted/Stopped`
- [ ] `taskEvents/config` `EventTopic`
- [ ] `openapi.yaml` 三 path

## Task 5: Gateway

- [ ] `routes.yaml` `taskauth-impersonation` priority 853, `auth_mode: token`
- [ ] `routes-to-apisix.py` 透传 `X-Impersonator-Id`
- [ ] CORS `Idempotency-Key`
- [ ] `bash taskGateway/run.sh routes-apply`

## Task 6: Forward-auth

- [ ] 缓存条目 `impersonatorID`；`X-Impersonator-Id`

## Task 7: Frontend

- [ ] `useImpersonateUser.js` + 测：按钮守卫、Idempotency-Key、失败 `data-traceId`
- [ ] `SystemAdminUsers.vue` 编辑表单按钮 `v-if="hasPlatformPerm('user:impersonate')"`
- [ ] `ImpersonationBanner.vue` Navbar 横幅 + 退出

## Task 8: 验证

```bash
cd taskAuth && go test ./domain ./src -count=1 -timeout 120s -run 'Impersonation|ValidateStart|EventTopic'
python3 taskGateway/scripts/test_forward_auth_impersonator_header.py
# taskFE vitest useImpersonateUser.test.js
```
