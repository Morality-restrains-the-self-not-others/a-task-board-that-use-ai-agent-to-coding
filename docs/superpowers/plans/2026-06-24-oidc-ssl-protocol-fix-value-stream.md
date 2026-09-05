# Value Stream: OIDC SSL Protocol Fix — openid_connect gem HTTP→HTTPS bug

> Derived from design: `docs/specs/oidc-ssl-debug/design.md`
> **Status: ACTIVE** (fixed 2026-06-24 with Playwright E2E verification)
> Extended by: `docs/superpowers/plans/2026-06-24-oidc-ssl-playwright-e2e-value-stream.md`

## Value Summary

用户点击 GitLab 的 taskAuth SSO 按钮后，OIDC 认证流程正常完成——不再因 Ruby `openid_connect` gem 强制将 HTTP issuer 升级为 HTTPS 而导致 SSL record layer failure。

## Related Value Streams

- **`taskauth-oidc-issuer-docker-reachability`**: **extension** — 前置修复解决了 issuer URL 从 Docker 容器内的可达性问题（`127.0.0.1` → `183.250.1.132`）。本修复在其基础上解决协议层问题：issuer 已可达但 Ruby gem 丢弃了 URI scheme (`http://` → `https://`)，导致 SSL 握手失败。两者串联构成完整的 OIDC SSO 通道。
- **`gitlab-oauth-scope-failfast-governance`**: dependent sibling — OIDC SSO 是 GitLab OAuth scope 治理的前置条件；OIDC 不通则 OAuth 授权流程无法启动。

## End-to-End Flow

```
[用户浏览器访问 GitLab] → [点击 taskAuth SSO]
  → [GitLab omniauth_openid_connect 发起 OIDC]
  → [OpenIDConnect::Discovery::Provider::Config.discover!(issuer)]
  → [Resource.new(uri) — BUG: 丢弃 scheme]
  → [SWD.url_builder.build → URI::HTTPS 默认]
  → [Faraday 发起 HTTPS 请求到 HTTP 端口 8003]
  → [SSL record layer failure ❌]

修复后:
  → [SWD.url_builder = URI::HTTP ← GitLab Rails initializer]
  → [Faraday 发起 HTTP 请求]
  → [taskAuth 返回 discovery 文档]
  → [authorize → 用户授权 → token exchange]
  → [用户登录成功 ✅]
```

## Value Stage Classification

- **Core value** — GitLab 内 OIDC discovery 使用正确协议 (HTTP)，SSO 登录成功
- **Essential support** — GitLab Rails initializer 持久化（容器重建后自动注入）
- **Future** — Gateway TLS 修复后切换 issuer 为 `https://183.250.1.132:18444`

## Value Increments

### Increment 1: SWD.url_builder HTTP Protocol Fix (Thin Slice — the whole fix)

**Value to user:** taskAuth SSO 登录在 Docker 部署下正常工作——点击即可完成 OIDC 认证流程，不再报 SSL 错误。

**Scope:**
1. GitLab 容器内创建 Rails initializer `zzz_fix_oidc_http.rb`，设置 `SWD.url_builder = URI::HTTP`
2. 重启 GitLab 加载 initializer
3. 验证 OIDC Discovery 使用 HTTP 协议（非 HTTPS）

**Depends on:** `taskauth-oidc-issuer-docker-reachability` (issuer 已可达)

**Test verification:**
```bash
docker exec gitlab gitlab-rails runner "
require 'openid_connect'
uri = URI.parse('http://183.250.1.132:8003')
r = OpenIDConnect::Discovery::Provider::Config::Resource.new(uri)
puts r.endpoint  # must output http://... not https://...
"
```

### Increment 2: Initializer 持久化 (Future/Next)

**Value to user:** GitLab 容器重建后自动恢复 OIDC protocol fix。

**Scope:**
1. 将 initializer 文件写入 `gitService/gitlab_home/` 目录（已挂载到容器 `/etc/gitlab`）
2. 在 `gitService/run.sh` 的启动流程中添加 initializer 同步步骤（如 `sync_omniauth_oidc.sh` 模式）
3. 或者在 `docker-compose.yml` 中通过额外 volume 挂载 initializers 目录

**Depends on:** Increment 1

## Impacted Existing Streams

| Stream | Impact |
|--------|--------|
| `taskauth-oidc-issuer-docker-reachability` | **extension** — 在前置修复基础上追加 protocol 修复；原 stream 的 issuer reachability 已验证通过 |
| `gitlab-oauth-scope-failfast-governance` | **unblocked** — OIDC SSO protocol 修复后，GitLab OAuth scope 治理的 SSO 前置条件满足 |
| `user-auth` (用户与认证) | 无直接变更 — OIDC issuer 配置不变，仅 Ruby 客户端侧行为修正 |

## Field Changes

| Field | Change |
|-------|--------|
| `task-auth.runtime.oidc_issuer` | 不变 — 保持 `http://183.250.1.132:8003` |
| `git-service.runtime.oidc_protocol_fix` | **新增** — `SWD.url_builder = URI::HTTP` initializer 注入到 GitLab Rails |
