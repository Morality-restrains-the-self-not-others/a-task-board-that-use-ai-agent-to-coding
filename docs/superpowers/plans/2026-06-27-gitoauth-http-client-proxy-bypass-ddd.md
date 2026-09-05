# DDD 领域建模: gitOauth HTTP Client Proxy Bypass Fix

> 输入:
> - 设计文档: `docs/specs/oauth-gitlab-token-exchange-fix-design.md`
> - 价值流文档: `docs/superpowers/plans/2026-06-27-gitoauth-http-client-proxy-bypass-value-stream.md`
> - NFR 澄清: `docs/superpowers/plans/2026-06-27-gitoauth-http-client-proxy-bypass-nfr-clarification.md`

## 跳过声明

**DDD 领域建模不适用于本次修复。** 理由：

1. 本次修复是**纯基础设施层变更** — 创建 `gitOauth/api/http_client.py`（`trust_env=False` 的 thread-local `requests.Session`），替换 `requests.post/get` 直接调用
2. 不引入新的业务概念（无新实体、值对象、聚合、领域服务、领域事件）
3. 不修改现有领域模型
4. 现有领域概念（`GitOAuthAppUserCredential`、`OauthCredentialBinding`、`OauthProviderRoute` 等）保持不变

## 受影响的基础设施文件

| 文件 | 变更类型 | 说明 |
|------|---------|------|
| `gitOauth/api/http_client.py` | **新增** | 提供 `trust_env=False` 的 thread-local session |
| `gitOauth/api/gitlab_tokens.py` | 修改 | 替换 `requests.post/get` → session 调用 |
| `gitOauth/api/github_tokens.py` | 修改 | 同上 |
| `gitOauth/api/gitlab_browser_views.py` | 修改 | 替换 `requests.post` (bind API call) |
| `gitOauth/api/github_browser_views.py` | 修改 | 替换 `requests.post` (bind API call) |

## 自检

- [x] 无新领域概念，DDD 步骤正确跳过
- [x] 跳过理由已声明（纯基础设施变更）
- [x] 现有领域模型不受影响
