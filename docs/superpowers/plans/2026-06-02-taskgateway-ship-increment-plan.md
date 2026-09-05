# Plan: taskGateway Ship Increment

## Task 1: taskAuth phone internal API

- [x] `POST /api/internal/login-methods/phone-taken/`
- [x] `PATCH /api/internal/users/{id}/phone-login-method/`
- [x] Go tests

## Task 2: Django profile_replace_phone

- [x] `phone_taken_by_other` / `upsert_phone_login_method` in resolver
- [x] `profile_replace_phone` 读 resolver、写 upsert
- [x] `_phone_active_bound_to_other_user` taskauth 分支
- [x] pytest（现有 ORM 路径仍绿）

## Task 3: Gateway ship artifacts

- [x] `taskGateway/domain/route.py` auth_mode 四分法
- [x] `scripts/ci/smoke_taskgateway.sh`
- [x] 更新 review / value-stream

## 验证

```bash
cd taskAuth && go test ./src -run Phone -count=1
cd task2app/Saas_project && pytest tests/test_profile_phone_replace.py tests/test_profile_payload_login_methods_resolver.py -q
bash scripts/ci/check_taskgateway_routes.sh
bash scripts/ci/smoke_taskgateway.sh
python scripts/ci/check_ddd_bdd_compliance.py
```
