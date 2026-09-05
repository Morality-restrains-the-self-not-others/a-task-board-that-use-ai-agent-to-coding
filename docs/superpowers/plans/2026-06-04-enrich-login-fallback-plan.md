# 实施计划：enrich-login 兜底修复

## Task 1 — Django 条款幂等服务

- [x] 新增 `accounts/login_policy_consent.py`：`record_login_policy_consents`
- [x] `taskauth_internal_views._record_login_policies` 委托该服务
- [x] 测试：`test_enrich_login_with_existing_consent_skips_body_ids`

## Task 2 — taskAuth 透传

- [x] 修改 `auth_login.go` handleLogin
- [x] 测试：`TestHandleLoginEnrichFailureReturnsStatus`

## Task 3 — 验证

- [x] `pytest` 新用例
- [x] `go test ./taskAuth/src -run TestHandleLoginEnrichFailure`
