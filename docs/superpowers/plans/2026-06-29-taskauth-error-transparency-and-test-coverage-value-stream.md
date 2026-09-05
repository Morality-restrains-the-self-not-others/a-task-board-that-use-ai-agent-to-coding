# Value Stream: taskAuth 错误可观测性 + 测试覆盖提升

> 派生自设计: `docs/superpowers/specs/2026-06-29-taskauth-error-transparency-and-test-coverage-design.md`

## 价值摘要

运维人员可通过 taskAuth 日志快速定位登录方式绑定失败的具体数据库错误；开发者在重构 username upsert 逻辑时有测试安全网。

## 相关价值流

- **user-auth** (conf/value-stream.yaml): 修改 — 在现有 `user-auth` 流中新增 `username-upsert` 步骤（测试覆盖）。本次不改变现有步骤的字段归属或 test_file。

## 端到端流程

```
[taskAuth upsert 失败] → [log.Printf 记录具体错误] → [运维 grep 日志定位根因]
```

```
[开发者修改 username upsert] → [go test 运行 TestUsernameTakenAndUpsertLoginMethod] → [CI 门禁捕获回归]
```

## 价值增量

### Increment 1: 错误日志透明化 (Thin Slice)

**用户价值**: 运维从 "db error" 无法定位问题 → 日志包含具体错误信息和用户 ID

**范围**:
- `taskAuth/src/auth_profile_internal.go`: `handleUpsertPhoneLoginMethod` L237 和 `handleUpsertUsernameLoginMethod` L102 添加 `log.Printf`
- `task2app/Saas_project/accounts/taskauth_bridge/login_methods_resolver.py`: 5xx 响应分支增强 warning 日志

**依赖**: 无 — 纯粹加日志，零行为变更

**测试**: 无需新增测试（日志不改变行为）

### Increment 2: username upsert 测试覆盖

**用户价值**: 开发者重构 username upsert 时有测试保护

**范围**:
- `taskAuth/src/auth_phone_profile_internal_test.go`: 新增 `TestUsernameTakenAndUpsertLoginMethod` 测试函数

**依赖**: Increment 1（无强依赖，但建议先完成日志改造再写测试以验证日志输出）

**测试**: Go test `TestUsernameTakenAndUpsertLoginMethod` 覆盖 4 个场景

## YAML 更新

在 `conf/value-stream.yaml` 的 `user-auth` 流中新增 `username-upsert` 步骤（状态: `planned`，Go 测试暂不接入 valueStream pytest runner）。
