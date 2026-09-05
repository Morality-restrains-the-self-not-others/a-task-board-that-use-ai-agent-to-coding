# Value Stream: taskAuth OIDC Issuer Docker 网络可达性修复

> Derived from design: `docs/specs/taskauth-oidc-issuer-fix-design.md`

## Value Summary

用户在任何机器上通过外部 IP 访问 GitLab 并点击 taskAuth SSO 登录时，OIDC 认证流程正常完成——不再因 issuer URL `127.0.0.1` 在 Docker 容器内不可达而报 `connection refused`。

## Related Value Streams

- **`gitlab-oauth-scope-failfast-governance`**: sibling fix — 同一个 `gitlab-oauth-app-bootstrap` 步骤确保 OAuth Application 存在；本修复确保 OIDC issuer URL 在 Docker 网络中可达。两者互补：前者管理 Application 生命周期，后者管理网络可达性。
- **`runall-startup-race-eaddrinuse-fix`**: independent — 同为基础设施修复，但无直接依赖关系。
- **`user-auth` > `runall-task-auth-orchestration`**: extension — taskAuth OIDC 配置变更后，健康检查仍应通过（issuer 从 `0.0.0.0` 变为可路由地址）。

## End-to-End Flow

[用户浏览器访问 GitLab] → [点击 taskAuth SSO] → [GitLab 发现 OIDC Provider] → [issuer URL 容器内可达 ✅] → [浏览器重定向到 taskAuth 授权页] → [用户授权] → [回调 redirect_uri 可达 ✅] → [GitLab 交换 Token] → [用户登录成功]

## Value Stage Classification

- **Core value** — GitLab 容器内能连接到宿主机 taskAuth OIDC Provider（issuer 可达）
- **Essential support** — taskAuth 发现的 issuer 值与 GitLab 配置的 issuer 一致（OIDC 协议要求）
- **Essential support** — redirect_uri 对用户浏览器可达（回调不丢失）
- **Enhancement** — issuerURL() fallback 增强（GatewayPublicBase host 提取）
- **Future** — 无需

## Value Increments

### Increment 1: OIDC Issuer URL Docker 可达 (Thin Slice — the whole fix)

**Value to user:** taskAuth SSO 登录在 Docker 部署环境下正常工作——点击即可完成 OIDC 认证流程。

**Scope:**
1. `conf/auth/task-auth/config.yaml` — `oidc.issuer` 显式设置，避免 fallback 到 `0.0.0.0`
2. `gitService/docker-compose.yml` — `GITLAB_OIDC_ISSUER` 默认值从 `127.0.0.1` 改为 `${GITLAB_EXTERNAL_HOST}:8003`；`redirect_uri` 参数化
3. `taskAuth/src/oidc_handlers.go` — `issuerURL()` fallback 增强：当 `GatewayPublicBase` 可用时，提取其 host 并拼接 taskAuth 自身端口
4. `gitService/run.sh` — 导出 `GITLAB_OIDC_ISSUER` 环境变量

**Depends on:** nothing (配置修复，无上游依赖)

**Test verification:** `docker exec gitlab curl http://${GITLAB_EXTERNAL_HOST}:8003/.well-known/openid-configuration` → 200；完整 SSO 登录流程手动验证

## Impacted Existing Streams

| Stream | Impact |
|--------|--------|
| `user-auth` > `runall-task-auth-orchestration` | taskAuth 健康检查应通过；issuer 配置项从隐式 fallback 变为显式值 |
| `gitlab-oauth-scope-failfast-governance` > `gitlab-oauth-app-bootstrap` | OAuth Application 自愈不变；OIDC issuer URL 变化不影响 scope 同步 |

## Field Changes

| Field | Change |
|-------|--------|
| `task-auth.runtime.oidc_issuer` | **新增**: OIDC issuer URL 从不可路由 (`0.0.0.0`) 修正为外部可达地址 |
| `git-service.runtime.oidc_issuer_env` | **新增**: `GITLAB_OIDC_ISSUER` 环境变量注入 Docker 容器 (`${GITLAB_EXTERNAL_HOST}:8003`) |
