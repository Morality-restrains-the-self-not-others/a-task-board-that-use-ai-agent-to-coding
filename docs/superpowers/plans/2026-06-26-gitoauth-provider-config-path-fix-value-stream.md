# Value Stream: gitOauth Provider 配置加载路径修复

> Derived from design: `.claude/skills/1-brainstorming-设计文档/design.md`

## Value Summary

gitOauth 服务启动时能正确从 `conf/auth/git-oauth/providers/` 加载所有 GitLab/GitHub OAuth 提供者配置，使 CreateProject 等页面的 OAuth 授权入口不再因配置空而返回 503 `bad_state`。

## Related Value Streams

- **[gitoauth-gitlab-provider-config-compat](2026-05-26-gitoauth-gitlab-provider-config-compat-value-stream.md)**: extension — 该流修复了 provider 配置的归一化与命中逻辑（list/dict 双结构兼容），本流修复配置加载的**上游路径**，使归一化逻辑能接收到实际数据。
- **[gitoauth-binding-state-persistence](2026-05-26-gitoauth-bind-failure-persistence-value-stream.md)**: downstream consumer — OAuth 回调依赖 provider 配置中的 `client_id`/`client_secret` 进行 token 交换。
- **[project-detail-repo-oauth-row-action](2026-05-26-project-detail-oauth-button-value-stream.md)**: downstream consumer — OAuth start 入口依赖 provider 配置路由到正确的 GitLab 实例。

## End-to-End Flow

gitOauth 服务启动 → Django settings 初始化 → `load_merged()` 扫描 `conf/auth/git-oauth/providers/*.yaml`
→ `normalize_provider_configs()` 扁平化配置 → `GITOAUTH_PROVIDER_CONFIGS` 填充
→ `get_provider_configs("gitlab")` 返回完整列表 → `resolve_provider_config("synology-gitlab")` 命中
→ OAuth start 请求到达 → `_resolve_start_authorize_context` 成功 → 返回 `authorize_url`
→ 用户跳转 GitLab 授权页 → 授权成功回调 → token 交换 → 绑定完成

## Affected Existing Streams

| Stream | Impact |
|--------|--------|
| `gitoauth-gitlab-provider-config-compat` | 上游修复 — 该流的归一化逻辑依赖本流的加载路径 |
| `gitoauth-binding-state-persistence` | 间接受益 — callback 中的 `resolve_provider_config` 恢复工作 |
| `project-detail-repo-oauth-row-action` | 直接受益 — OAuth start 入口不再 503 |

## New Stream Needed?

不需要新增业务流。此修复属于现有 `gitoauth-gitlab-provider-config-compat` 流的上游补丁，可在该流下追加一个 `planned` 步骤。但考虑到独立可测性，建议新增一个轻量级流。

## Value Increments

### Increment 1: 修正提供者配置加载路径（Thin Slice）
**Value to user:** gitOauth 启动后所有 OAuth 提供者配置可用，OAuth start 请求不再返回 503。
**Scope:** 修改 `port_config.py:53` 将 `conf/git-oauth/providers` 改为 `conf/auth/git-oauth/providers`。
**Depends on:** nothing

### Increment 2: 路径回退兼容（Essential Support）
**Value to user:** 路径变更后已有部署不会因目录迁移再次失败。
**Scope:** `load_merged()` 中主路径不存在时回退到旧路径。
**Depends on:** Increment 1

### Increment 3: 可观测性增强（Enhancement）
**Value to user:** DEBUG 模式下 OAuth 启动失败携带真实错误原因，加速排查。
**Scope:** `_gateway_start_error` 在 DEBUG=True 时附加内部错误码。
**Depends on:** Increment 1

## Fields Impact

本次修复不引入新的持久化字段。相关引用字段（已有）：

- `git-oauth.api_githubappusercredential.provider` — provider 维度隔离
- `git-oauth.api_githubappusercredential.bind_status` — 授权绑定状态

## Test Impact

- 新增 `test_port_config.py` 单测：验证 `load_merged()` 从正确路径加载提供者配置
- 新增 `test_port_config.py` 单测：验证回退路径兼容性
- 无需 Playwright 回归（路径修复不影响前端行为）

## Status Changes

无现有流状态变更。建议新增轻型流 `gitoauth-provider-config-loading`（status: active）。

## Cross-Stream Dependencies

- `gitoauth-provider-config-loading` → `gitoauth-gitlab-provider-config-compat`
  （前者保证配置加载，后者消费配置做归一化与路由）
- `gitoauth-provider-config-loading` → `gitoauth-binding-state-persistence`
  （配置可用是 OAuth callback token 交换的前置条件）
