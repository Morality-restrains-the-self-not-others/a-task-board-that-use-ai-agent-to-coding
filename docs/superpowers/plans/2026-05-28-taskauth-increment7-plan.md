# taskAuth Increment 7 — Plan

- [x] **7.1** `delegate_send_verification_code` + Go handler + Django internal
- [x] **7.2** `taskauth_unavailable_response()` 守卫各 auth action
- [x] **7.3** `drop_shared_auth_tables.sh` + `verify_auth_tables_dropped.sh`
- [x] **7.4** value-stream + bridge pytest
- [x] **7.5** 审查 + 回归测试

验证:

```bash
cd taskAuth/src && go test ./...
cd task2app/Saas_project && pytest tests/test_taskauth_*.py accounts/view_test/UserViewSet_*_test.py -q
taskAuth/scripts/verify_auth_tables_dropped.sh  # drop 后
```
