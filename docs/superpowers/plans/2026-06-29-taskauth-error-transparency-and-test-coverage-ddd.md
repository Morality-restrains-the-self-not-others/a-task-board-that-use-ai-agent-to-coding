# DDD 领域建模: taskAuth 错误透明度 + 测试覆盖提升

> 输入:
> - 设计文档: `docs/superpowers/specs/2026-06-29-taskauth-error-transparency-and-test-coverage-design.md`
> - NFR 澄清: `docs/superpowers/plans/2026-06-29-taskauth-error-transparency-and-test-coverage-nfr-clarification.md`
>
> 输出使用者: `/6-plans-实施计划`

## 跳过声明

本次改进属于**纯可观测性增强 + 测试覆盖补充**，不引入新业务概念、新实体、新聚合或新领域事件。根据 DDD 步骤跳过条件「价值流增量不涉及新的业务概念」，完整领域建模可跳过。

## 现有领域模型参考

以下为本次改进涉及的已有领域概念（来自 `taskAuth/src/`），**无需修改**：

### 限界上下文: auth（用户与认证）

| 概念 | 类型 | 文件 | 状态 |
|------|------|------|------|
| `LoginMethodRow` | Entity | `taskAuth/src/db.go` | 已有 — 登录方式(邮箱/手机/用户名) |
| `upsertPhoneLoginMethod` | Domain Service | `taskAuth/src/auth_profile_internal.go` | 已有 — 幂等手机绑定 |
| `upsertUsernameLoginMethod` | Domain Service | `taskAuth/src/auth_profile_internal.go` | 已有 — 幂等用户名绑定 |
| `errPhoneTaken` | Domain Error | `taskAuth/src/auth_profile_internal.go` | 已有 — 手机号已被占用 |
| `errUsernameTaken` | Domain Error | `taskAuth/src/auth_profile_internal.go` | 已有 — 用户名已被占用 |

### 无新增领域概念

本次改进全部为已有代码的增量修改：
- 错误日志 → 在已有 handler 中添加 `log.Printf`
- 测试覆盖 → 对已有 `upsertUsernameLoginMethod` 补充测试
- Django 日志 → 对已有 `login_methods_resolver.py` 补充 warning 上下文

## 自检

- [x] 跳过理由已明确（不涉及新业务概念）
- [x] 已有领域模型已引用确认
- [x] 无新增领域文件
