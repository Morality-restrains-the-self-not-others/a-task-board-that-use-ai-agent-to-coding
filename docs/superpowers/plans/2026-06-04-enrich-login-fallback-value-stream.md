# 价值流：enrich-login 兜底修复

**设计**：`docs/superpowers/specs/2026-06-04-enrich-login-fallback-design.md`

## Related Value Streams

修改既有 `user-auth` → `login`：enrich-login 响应契约与条款幂等，非 greenfield。

## 增量

### Inc-1（薄切片，必交付）

邮箱/密码 → taskAuth → enrich-login 成功返回真实 `companies` + `redirect_url`。

- Django：`record_login_policy_consents` 已同意则跳过 body 校验
- taskAuth：enrich 4xx 透传，去掉 200 假成功

### Inc-2（可选）

前端提交前校验条款 id 非空；Playwright 登录 `/projects/` 回归。

## 价值流步骤

| 步骤 | 变更 |
|------|------|
| login | enrich-login 契约、测试扩展 |
| email-register | 无代码变更；登录幂等依赖注册 consent |
