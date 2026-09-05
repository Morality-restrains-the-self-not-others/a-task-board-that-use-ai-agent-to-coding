# DDD 领域建模: Fix Password Reset Datetime Parsing

> 输入:
> - 设计文档: `docs/superpowers/specs/2026-06-22-fix-password-reset-datetime-parsing.md`
> - 价值流文档: `docs/superpowers/plans/2026-06-22-fix-password-reset-datetime-parsing-value-stream.md`
> - NFR 澄清文档: `docs/superpowers/plans/2026-06-22-fix-password-reset-datetime-parsing-nfr-clarification.md`
>
> 输出使用者: `/6-plans-实施计划`

## 跳过声明

**此修复不涉及领域模型变更。** 理由：

1. **纯基础设施修复** — `parseDateTime()` 是一个纯工具函数，处理 datetime 字符串解析
2. **无新业务概念** — 不引入新的实体、值对象、聚合或领域事件
3. **现有领域模型不变** — `LoginMethod` 实体、`accounts_login_method` 表、token 生命周期均保持不变
4. **NFR 确认无模型影响** — NFR 澄清文档的「领域模型影响」表格标注为「无」

## 现有领域模型参考

此修复操作的限界上下文保持不变：

```
用户认证上下文 (user-auth)
├── 聚合根: User (accounts_user)
├── 实体: LoginMethod (accounts_login_method)
│   ├── password_reset_token: string
│   ├── password_reset_token_expires_at: datetime
│   ├── activation_token: string
│   └── activation_token_expires_at: datetime
└── 领域服务: (unchanged)
```

`isPasswordResetTokenValid()` 和 `isActivationTokenValid()` 是 `LoginMethod` 实体的行为方法，修复仅改变其内部 datetime 解析逻辑，接口契约未变。

## 自检

- [x] 无新领域文件需要生成
- [x] 现有领域模型未被修改
- [x] skip 声明理由明确
