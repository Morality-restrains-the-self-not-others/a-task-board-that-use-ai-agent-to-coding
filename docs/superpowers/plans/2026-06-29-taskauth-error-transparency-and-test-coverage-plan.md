# 实施计划: taskAuth 错误透明度 + 测试覆盖提升

> 输入:
> - 设计: `docs/superpowers/specs/2026-06-29-taskauth-error-transparency-and-test-coverage-design.md`
> - 价值流: `docs/superpowers/plans/2026-06-29-taskauth-error-transparency-and-test-coverage-value-stream.md`
> - NFR: `docs/superpowers/plans/2026-06-29-taskauth-error-transparency-and-test-coverage-nfr-clarification.md`
> - DDD: `docs/superpowers/plans/2026-06-29-taskauth-error-transparency-and-test-coverage-ddd.md`

## 任务清单

### T1: taskAuth phone upsert 错误日志

- [ ] 在 `taskAuth/src/auth_profile_internal.go` 的 `handleUpsertPhoneLoginMethod` catch-all 分支（L237）添加 `log.Printf("[taskAuth] upsert phone failed for user %s: %v", userID, err)` 于 `writeJSON` 之前
- 文件: `taskAuth/src/auth_profile_internal.go`
- 验证: `cd taskAuth && go build ./... && go test ./src -run TestPhoneTakenAndUpsertLoginMethod -count=1`

### T2: taskAuth username upsert 错误日志

- [ ] 在 `taskAuth/src/auth_profile_internal.go` 的 `handleUpsertUsernameLoginMethod` catch-all 分支（L102）添加 `log.Printf("[taskAuth] upsert username failed for user %s: %v", userID, err)` 于 `writeJSON` 之前
- 文件: `taskAuth/src/auth_profile_internal.go`
- 验证: `cd taskAuth && go build ./...`

### T3: Django upsert 日志增强

- [ ] 在 `task2app/Saas_project/accounts/taskauth_bridge/login_methods_resolver.py` 的 `upsert_phone_login_method()` 5xx 分支（L156-157）添加 `logger.warning("taskAuth returned %s for phone upsert user=%s", status, uid)`
- [ ] 在 `task2app/Saas_project/accounts/taskauth_bridge/login_methods_resolver.py` 的 `upsert_username_login_method()` 5xx 分支添加同等日志
- 文件: `task2app/Saas_project/accounts/taskauth_bridge/login_methods_resolver.py`
- 验证: `cd task2app/Saas_project && source activate_env.sh && python -c "from accounts.taskauth_bridge.login_methods_resolver import upsert_phone_login_method, upsert_username_login_method; print('import OK')"`

### T4: username upsert 测试

- [ ] 在 `taskAuth/src/auth_phone_profile_internal_test.go` 新增 `TestUsernameTakenAndUpsertLoginMethod` 函数，覆盖：
  - 创建带 username 的用户 → `handleUsernameTaken` 返回 `taken: true`
  - 给无 username 的用户 patch username → 返回 200
  - patch 已存在的 username → 返回 409 Conflict
  - 删除 username（patch 空字符串）→ 返回 200
- 文件: `taskAuth/src/auth_phone_profile_internal_test.go`
- 验证: `cd taskAuth && go test ./src -run TestUsernameTakenAndUpsertLoginMethod -v -count=1`

### T5: 端到端验证

- [ ] 重建 taskAuth: `cd taskAuth && bash run.sh build && bash run.sh stop; sleep 1; bash run.sh start`
- [ ] 验证 phone upsert: `curl -X PATCH http://127.0.0.1:8003/api/internal/users/{uid}/phone-login-method/ ...` → 200
- [ ] 验证 username upsert: `curl -X PATCH http://127.0.0.1:8003/api/internal/users/{uid}/username-login-method/ ...` → 200
- [ ] 清理测试数据

## 执行顺序

```
T1 → T2 → T3 → T4 → T5
      (T1/T2/T3 无相互依赖，可并行)
```

## 风险

| 风险 | 概率 | 影响 | 缓解 |
|------|------|------|------|
| 日志格式不一致 | 低 | 低 | Go 侧遵循现有 `log.Printf("[taskAuth] ...")` 模式 |
| 测试依赖外部 DB | 低 | 中 | `setupTestAuthDB(t)` 创建临时 SQLite 实例 |
