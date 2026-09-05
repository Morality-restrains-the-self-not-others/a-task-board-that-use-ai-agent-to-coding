# taskAuth Increment 4 — 实施计划

> 领域: `docs/superpowers/domain/2026-05-28-taskauth-increment4-domain.md`

## Slice 4.1: 链接重置

- [x] **Task 4.1.1** Go `db.go` — password reset token CRUD
- [x] **Task 4.1.2** Go `auth_password_reset.go` + routes
- [x] **Task 4.1.3** Django `post-password-reset-link` internal + delegate
- [x] **Task 4.1.4** pytest 回归 + Go 单测

## Slice 4.2: 验证码重置

- [x] **Task 4.2.1** Django internal send/verify code
- [x] **Task 4.2.2** Go handlers + delegate

## Slice 4.3: 桥接测试

- [x] **Task 4.3.1** `tests/test_taskauth_password_reset_bridge.py`
- [x] **Task 4.3.2** value-stream.yaml 字段已更新

## 验证

```bash
cd taskAuth && go test ./src/...
cd task2app/Saas_project && DJANGO_SETTINGS_MODULE=saas_project.settings_test \
  pytest accounts/view_test/UserViewSet_reset_password_test.py tests/test_taskauth_password_reset_bridge.py -q
cd valueStream/src && go test -run TestLoadProductionValueStream -count=1
```
