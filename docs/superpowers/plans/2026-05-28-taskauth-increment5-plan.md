# taskAuth Increment 5 — 实施计划

## Slice 5.1: phone_register 原生 Go

- [x] **Task 5.1.1** Django internal `validate-phone-register` / `post-phone-register`
- [x] **Task 5.1.2** Go `auth_phone_register.go` + SQLite 写入
- [x] **Task 5.1.3** bridge `delegate_phone_register`

## Slice 5.2: domain 包提取

- [x] **Task 5.2.1** `taskAuth/domain/` — LoginMethodRef、PasswordCredential、ResetToken、Repository 接口

## Slice 5.3: E2E 保障

- [x] **Task 5.3.1** `test_taskauth_phone_register_bridge.py`
- [x] **Task 5.3.2** `test_taskauth_e2e` health（`TASKAUTH_E2E=1` 时运行）
- [x] **Task 5.3.3** value-stream phone-register 字段更新

## 验证

```bash
cd taskAuth && go test ./domain/... ./src/...
cd task2app/Saas_project && DJANGO_SETTINGS_MODULE=saas_project.settings_test \
  pytest accounts/view_test/UserViewSet_phone_register_test.py \
         tests/test_taskauth_phone_register_bridge.py \
         tests/test_taskauth_password_reset_bridge.py -q
# 全栈（需 runAll）:
# TASKAUTH_E2E=1 pytest tests/test_taskauth_phone_register_bridge.py::test_taskauth_health_when_e2e -q
```
