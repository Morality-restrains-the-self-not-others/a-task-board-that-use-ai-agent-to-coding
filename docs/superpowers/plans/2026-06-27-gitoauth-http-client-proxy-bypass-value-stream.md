# Value Stream: gitOauth HTTP Client Proxy Bypass Fix

> Derived from design: `docs/specs/oauth-gitlab-token-exchange-fix-design.md`
> **Type:** Infrastructure fix (gitOauth outbound HTTP reliability) — no new business value stream.

## Value Summary

用户在 GitLab OAuth 授权回调后 Token 交换不再因宿主 SOCKS 代理劫持而失败，授权流程可端到端完成。

## Related Value Streams

- **gitoauth-binding-state-persistence** (`2026-05-26-gitoauth-bind-failure-persistence-value-stream.md`): extension — 本修复硬化的 HTTP client 使 callback → exchange → bind 链路不再受宿主机 `ALL_PROXY` 干扰
- **project-detail-repo-oauth-row-action** (`2026-05-26-project-detail-oauth-button-value-stream.md`): extension — OAuth start → callback → token exchange 全链路受益于 proxy bypass
- **gitlab-oauth-scope-failfast-governance** (`2026-05-26-gitlab-oauth-scope-failfast-value-stream.md`): extension — GitLab OAuth 闭环（scope → start → exchange）的 exchange 阶段可靠性提升
- **fix-health-check-proxy-bypass** (`2026-06-22-fix-health-check-proxy-bypass-value-stream.md`): sibling — 同模式 proxy bypass fix（runAll health check），gitOauth 的出站 HTTP 同样被 SOCKS 代理劫持

## End-to-End Flow

```
[用户点击 OAuth 授权] → [GitLab 回调到 gitOauth] → [exchange_authorization_code_for_tokens()]
                                                              ↓
                                              POST http://183.250.1.132:8012/oauth/token
                                                              ↓
                                              [trust_env=False → 直连 GitLab]  ← fix
                                                              ↓
                                              [GitLab 返回 tokens] → [bind 主站] → [用户看到 "授权成功"]
```

## Value Increments

### Increment 1: Proxy-Free gitOauth HTTP Client (Only Increment)
**Value to user:** GitLab OAuth 授权回调后的 token 交换不再因宿主机 `ALL_PROXY` / `HTTP_PROXY` 环境变量而失败。
**Scope:** 创建 `gitOauth/api/http_client.py`（thread-local `requests.Session` with `trust_env=False`），替换所有 `requests.post/get` 直接调用。
**Depends on:** nothing.
**Test file:** `gitOauth/api/test_http_client.py` (new) + `gitOauth/api/tests.py` (existing, verify no regression).

## YAML Config

**New value stream entry in `conf/value-stream.yaml`:**

```yaml
- name: gitoauth-http-client-proxy-bypass
  domain: 云平台与资源
  description: gitOauth 出站 HTTP 调用绕过宿主机代理，防止 SOCKS 代理劫持导致 OAuth token 交换失败
  steps:
  - name: gitoauth-http-client-trust-env-false
    status: active
    test_file: ../../gitOauth/api/test_http_client.py
    fields:
    - name: git-oauth.runtime.http_session_trust_env
      description: HTTP session 的 trust_env 必须为 False，防止读取 ALL_PROXY/HTTP_PROXY 环境变量
    - name: git-oauth.api_githubappusercredential.bind_status
      description: proxy bypass 后 bind_status 可正常从 pending 转为 active
```
