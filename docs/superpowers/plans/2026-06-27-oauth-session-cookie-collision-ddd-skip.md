# DDD 领域建模: OAuth Session Cookie 冲突修复 — 跳过声明

> 输入:
> - 设计文档: `docs/specs/oauth-redirect-loop-port-4000/design.md`
> - 价值流文档: `docs/superpowers/plans/2026-06-27-oauth-session-cookie-collision-value-stream.md`
> - NFR 澄清: `docs/superpowers/plans/2026-06-27-oauth-session-cookie-collision-nfr-clarification.md`

## 跳过理由

本次修复为**纯配置变更**：在 `gitOauth/config/settings.py` 中添加一行 `SESSION_COOKIE_NAME = 'gitoauth_sessionid'`。

- ❌ 不引入新的限界上下文
- ❌ 不引入新的实体、值对象或聚合
- ❌ 不引入新的领域服务或仓储接口
- ❌ 不引入新的领域事件
- ✅ 现有 OAuth 领域模型完全不变

符合 DDD 步骤跳过条件：「配置变更、或价值流增量不涉及新的业务概念」。

## 现有领域模型参考

本次修复涉及的现有领域概念（无需修改）：

| 概念 | 位置 | 说明 |
|------|------|------|
| OAuthFlow (隐式) | gitOauth session state | OAuth 流程临时上下文，不受 cookie 命名影响 |
| GitOAuthAppUserCredential | gitOauth/api/models.py | OAuth 凭据实体，不受影响 |
| OAuthProviderConfig | conf/auth/git-oauth/providers/ | Provider 路由配置，不受影响 |
