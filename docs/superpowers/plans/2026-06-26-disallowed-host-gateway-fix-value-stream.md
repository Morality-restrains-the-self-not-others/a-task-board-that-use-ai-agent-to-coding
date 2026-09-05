# Value Stream: gitOauth DisallowedHost 网关 Host 头校验修复

> Derived from design: `docs/designs/disallowed-host-gateway-fix.md`

## Value Summary

经网关代理到 gitOauth 的 OAuth 请求不再因 `DisallowedHost` 而 400 失败 — gitOauth ALLOWED_HOSTS 动态包含网关公网地址。

## Related Value Streams

- **`task-gateway`**: 扩展 — 本修复补齐 task-gateway 代理请求在 gitOauth 端的 Host 校验放行，与 gateway-trust-headers-auth 步骤形成互补（一个在网关门校验 token，一个在后端放行 Host）
- **`create-project-oauth-validation-loop`**: 修复 — 本修复消除该流中 `start-from-gateway` 请求的 400 报错
- **`gitlab-oauth-scope-failfast-governance`**: 无直接依赖，同属 gitOauth 服务配置层面

## End-to-End Flow

[浏览器发起 OAuth 授权经网关] → [APISIX 代理请求携带网关 Host 头] → [gitOauth CommonMiddleware 校验 Host] → [ALLOWED_HOSTS 白名单包含网关公网 IP] → [请求正常进入视图处理] → [用户获得 OAuth 授权 URL]

## Value Increments

### Increment 1: 网关公网 Host 加入 ALLOWED_HOSTS (Thin Slice — 唯一增量)

**Value to user:** 通过网关发起的 OAuth 授权请求不再 400 DisallowedHost
**Scope:** 
- `port_config.py` 新增 `load_gateway_public_host()` 读取 `conf/gateway/task-gateway/config.yaml` 的 `publicBase` hostname
- `settings.py` ALLOWED_HOSTS 列表追加该 hostname
**Depends on:** nothing

**Verification:**
- 单元测试：验证 `load_gateway_public_host()` 正确解析 `publicBase. hostname`
- 集成测试：验证 ALLOWED_HOSTS 列表包含 `183.250.1.132`
- 手工验证：浏览器通过网关 18081 访问 OAuth start 接口，返回 200 而非 400
