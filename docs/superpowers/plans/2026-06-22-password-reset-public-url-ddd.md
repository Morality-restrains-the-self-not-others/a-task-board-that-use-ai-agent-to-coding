# DDD 领域建模: 密码重置邮件域名可配置化

> 输入:
> - 设计: `docs/superpowers/specs/2026-06-22-password-reset-public-url-design.md`
> - 价值流: `docs/superpowers/plans/2026-06-22-password-reset-public-url-value-stream.md`
> - NFR: `docs/superpowers/plans/2026-06-22-password-reset-public-url-nfr-clarification.md`

## 跳过声明

纯配置修复——无新实体、值对象、聚合、仓储或领域事件。变更仅涉及配置加载层（`settings_manager.py`），为现有 `get_frontend_domain()` 方法增加回退逻辑。

已有领域概念不变:
- Bounded Context: `用户与认证` (user-auth)
- Entity: `User`, `LoginMethod` (不变)
- 流程: 密码重置邮件 URL 构造 → `settings_manager.get_frontend_domain()` (行为变更)
