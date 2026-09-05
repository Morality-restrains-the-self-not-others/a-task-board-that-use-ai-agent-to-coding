# Design Doc: taskAuth OIDC SSO SSL record layer failure 修复

**日期**: 2026-06-24
**状态**: 待审批
**作者**: AI Assistant

---

## 1. 问题概述

### 用户描述

登陆 `http://183.250.1.132:4000/auth/login/` → 点击"代码仓库" → 跳转 GitLab
`
/users/sign_in` → 点击 **taskAuth SSO** 登录 →
显示错误:

> Could not authenticate you from OpenIDConnect because "Ssl connect returned=1
> errno=0 peeraddr=183.250.1.132:8003 state=error: record layer failure".

### 受影响组件

| 组件 | 角色 | 端口 |
|------|------|------|
| taskAuth | OIDC Provider (Go, HTTP only) | 8003 |
| GitLab CE 19.0 | OIDC Relying Party (Ruby) | 8012 |
| taskGateway (APISIX) | TLS 网关 + OIDC 路由代理 | 18081 (HTTP) / 18444 (TLS) |

---

## 2. 根因分析

### 2.1 直接原因

GitLab 的 `omniauth_openid_connect` gem 尝试通过 **HTTPS** 协议连接
taskAuth 的 **HTTP** 端口 8003，SSL 握手失败。

### 2.2 完整调用链

```
浏览器点击 "taskAuth SSO"
  → GitLab POST /users/auth/openid_connect
  → omniauth_openid_connect.rb line 109: request_phase
  → config.discover! (用 issuer 获取 OIDC discovery)
  → OpenIDConnect::Discovery::Provider::Config.discover!(issuer)
  → Resource.new(uri)   ← BUG 在这里!
  → SWD.url_builder.build(...)  ← 强制使用 URI::HTTPS
  → Faraday 发起 HTTPS 请求
  → https://183.250.1.132:8003/.well-known/openid-configuration
  → taskAuth (HTTP only) 无法处理 SSL ClientHello
  → SSL record layer failure ❌
```

### 2.3 Bug 定位（Ruby 源码）

**文件**: `openid_connect-2.3.1/lib/openid_connect/discovery/provider/config/resource.rb`

```ruby
# BUG: initialize 丢弃了 URI scheme!
def initialize(uri)
  @host = uri.host
  @port = uri.port unless [80, 443].include?(uri.port)
  @path = File.join uri.path, '.well-known/openid-configuration'
  # 注意: @scheme 没有被保存!
  attr_missing!
end

# endpoint 使用 SWD.url_builder, 默认为 URI::HTTPS
def endpoint
  SWD.url_builder.build [nil, host, port, path, nil, nil]
end
```

**文件**: `swd-2.0.3/lib/swd.rb`

```ruby
# 默认 url_builder 是 HTTPS, 从不被 openid_connect 覆盖
def self.url_builder
  @@url_builder ||= URI::HTTPS   # ← 问题根源
end
```

### 2.4 验证

```bash
# 在 GitLab 容器内验证:
$ docker exec gitlab gitlab-rails runner "
require 'openid_connect'
uri = URI.parse('http://183.250.1.132:8003')
r = OpenIDConnect::Discovery::Provider::Config::Resource.new(uri)
puts r.endpoint
"
# 输出: https://183.250.1.132:8003/.well-known/openid-configuration
# 期望: http://183.250.1.132:8003/.well-known/openid-configuration
```

### 2.5 GitLab 日志证据

```json
{
  "severity": "ERROR",
  "time": "2026-06-24T13:56:54.959Z",
  "message": "(openid_connect) Authentication failure! SSL_connect returned=1
   errno=0 peeraddr=183.250.1.132:8003 state=error: record layer failure:
   OpenIDConnect::Discovery::DiscoveryFailed, SSL_connect returned=1 errno=0
   peeraddr=183.250.1.132:8003 state=error: record layer failure"
}
```

---

## 3. 解决方案

### 方案 A（推荐）⭐: 通过 taskGateway TLS 端口代理 OIDC 端点

**原理**: taskGateway (APISIX) 已有 TLS 证书且已路由所有 OIDC 路径。
将 OIDC issuer 改为 gateway 的 TLS 地址。

**变更点**:
1. 修改 `conf/auth/task-auth/config.yaml`:
   ```yaml
   oidc:
     issuer: "https://183.250.1.132:18444"
     bootstrapRedirectUri: "http://183.250.1.132:8012/users/auth/openid_connect/callback"
   ```

2. 修改 `taskAuth/src/oidc_handlers.go` 的 `issuerURL()`:
   - 无需修改（`cfg.OidcIssuer` 配置优先）

3. 重新编译并重启 taskAuth。

**优点**:
- 不需要修改 GitLab 容器或 Ruby gem
- 利用现有 APISIX TLS 基础设施
- OIDC 通信全程加密（生产就绪）
- issuer 对浏览器和 Docker 容器内 GitLab 均可达

**缺点**:
- 增加一层网关跳转（延迟可忽略）
- 需要网关容器保持运行

**APISIX 已有 OIDC 路由确认**:
```yaml
# taskGateway/apisix/apisix.yaml 已包含:
- id: oidc-discovery    uri: /.well-known/openid-configuration
- id: oidc-jwks        uri: /api/oidc/jwks
- id: oidc-authorize   uri: /api/oidc/authorize
- id: oidc-token       uri: /api/oidc/token
- id: oidc-userinfo    uri: /api/oidc/userinfo
```

### 方案 B: GitLab 容器 Monkey-Patch `SWD.url_builder`

**原理**: 在 GitLab 加载 OIDC gem 之前设置 `SWD.url_builder = URI::HTTP`。

**实现方式**: 通过 GitLab docker-compose 添加 Rails initializer 文件。

**步骤**:
1. 创建 initializer 文件 `gitService/gitlab_home/config/initializers/zzz_fix_oidc_http.rb`:
   ```ruby
   # Fix: openid_connect gem incorrectly forces HTTPS on HTTP issuers
   require 'swd'
   SWD.url_builder = URI::HTTP
   ```

2. 确保 docker-compose 挂载此目录 (已挂载 `/etc/gitlab`):
   ```yaml
   volumes:
     - './gitlab_home/config:/etc/gitlab'  # 已存在
   ```

3. 但注意: GitLab 的 Rails initializers 位于容器内
   `/opt/gitlab/embedded/service/gitlab-rails/config/initializers/`，
   而不是 `/etc/gitlab/`。这需要额外挂载或使用 `gitlab.rb` 配置。

**优点**:
- 无需修改 taskAuth 或架构
- 改动最小

**缺点**:
- Monkey-patch 脆弱，Gem 升级可能破坏
- 所有 OIDC 通信走 HTTP（不安全）
- GitLab 版本升级后需重新验证

### 方案 C: 禁用 OIDC Discovery，手动配置 Endpoint

**原理**: 在 GitLab OIDC 配置中设置 `discovery: false` 并手动指定所有端点 URL。

**变更点** (修改 `gitService/docker-compose.yml` 或 `gitlab.rb`):
```ruby
gitlab_rails['omniauth_providers'] = [
  {
    name: 'openid_connect',
    label: 'taskAuth SSO',
    args: {
      name: 'openid_connect',
      scope: ['openid', 'profile', 'email'],
      response_type: 'code',
      issuer: 'http://183.250.1.132:8003',
      discovery: false,   # ← 禁用 discovery
      client_auth_method: 'basic',
      uid_field: 'sub',
      client_options: {
        identifier: 'gitlab-git-service',
        secret: 'gsoidc-dev-secret-do-not-use-in-prod',
        authorization_endpoint: 'http://183.250.1.132:8003/api/oidc/authorize',
        token_endpoint:         'http://183.250.1.132:8003/api/oidc/token',
        userinfo_endpoint:      'http://183.250.1.132:8003/api/oidc/userinfo',
        jwks_uri:               'http://183.250.1.132:8003/api/oidc/jwks',
        redirect_uri: "http://183.250.1.132:8012/users/auth/openid_connect/callback"
      }
    }
  }
]
```

**优点**:
- 完全绕过 buggy discovery 代码路径
- 不需要 patch gem

**缺点**:
- 手动维护端点 URL
- 端点变更需要同步更新
- 仍是 HTTP 明文通信

---

## 4. 推荐决策

**分两阶段修复**:

### 即时修复（方案 B+）: GitLab Rails Initializer

当前 gateway TLS 端口 (18444) 需额外排查（APISIX standalone SSL 配置问题），
因此采用更直接的 Ruby monkey-patch 方式立即修复：

1. 创建 GitLab Rails initializer 文件，在 OIDC gem 加载后设置 `SWD.url_builder = URI::HTTP`
2. 通过 docker exec 注入到 GitLab 容器
3. 重启 GitLab (gitlab-ctl restart)

这绕过了 `openid_connect` gem 的 scheme 丢弃 bug。

### 长期优化（方案 A）: Gateway TLS 代理

待 gateway TLS 修复后，将 issuer 改为 `https://183.250.1.132:18444`，
实现 OIDC 通信加密。

---

## 5. 领域概念清单 (Domain Concept Inventory)

### Bounded Contexts
- **Auth Context** (taskAuth) — OIDC Provider, 用户认证
- **Git Service Context** (GitLab) — OIDC Relying Party, 代码仓库
- **Gateway Context** (taskGateway/APISIX) — TLS 终结, 路由代理

### Key Entities
- `OidcClient` (oidc_client 表) — OIDC 注册客户端 (gitlab-git-service)
- `User` (accounts_user 表) — 用户主体
- `AuthorizationCode` (oidc_authorization 表) — 授权码

### Domain Events
- `OIDC.AuthorizationRequested` — 用户发起 SSO
- `OIDC.TokenExchanged` — 授权码兑换 token
- `OIDC.DiscoveryFailed` — Discovery 失败 (当前 bug)

---

## 6. 价值流影响分析

### 受影响流

| Stream | 影响 |
|--------|------|
| `user-auth` (用户与认证) | OIDC issuer URL 变更，所有 OIDC 端点通过 gateway 代理 |
| `gitlab-oauth-scope-failfast-governance` (GitLab OAuth scope 治理) | 间接影响: OIDC 是 SSO 基础通道，修复后 GitLab OAuth scope 治理才能正常工作 |

### 新增流需求

需要新增 `oidc-provider-reachability` 价值流步骤，验证:
- GitLab 容器内可通过 HTTPS 访问 OIDC discovery
- OIDC token endpoint 可达性
- ID token 验证链完整

### Fields Impact

| Field | 变更 |
|-------|------|
| `conf/auth/task-auth.oidc.issuer` | `http://183.250.1.132:8003` → `https://183.250.1.132:18444` |

---

## 7. 实施步骤

### Phase 1: 即时修复 (方案 B+)

```bash
# 1. 创建 GitLab Rails initializer
docker exec gitlab bash -c 'cat > /opt/gitlab/embedded/service/gitlab-rails/config/initializers/zzz_fix_oidc_http.rb << "RUBY"
# Fix: openid_connect-2.3.1 discovery Resource discards URI scheme,
# and SWD.url_builder defaults to URI::HTTPS, forcing HTTP issuers to HTTPS.
# This breaks OIDC discovery when the provider (taskAuth) only serves HTTP.
require "swd"
SWD.url_builder = URI::HTTP
RUBY'

# 2. 验证 initializer 已创建
docker exec gitlab cat /opt/gitlab/embedded/service/gitlab-rails/config/initializers/zzz_fix_oidc_http.rb

# 3. 重启 GitLab 加载新的 initializer
docker exec gitlab gitlab-ctl restart

# 4. 验证修复
docker exec gitlab gitlab-rails runner "
require 'openid_connect'
uri = URI.parse('http://183.250.1.132:8003')
r = OpenIDConnect::Discovery::Provider::Config::Resource.new(uri)
puts 'Endpoint: ' + r.endpoint.to_s
puts r.endpoint.to_s.include?('http://') ? '✅ FIXED' : '❌ STILL BROKEN'
"
```

### Phase 2: 长期优化 (方案 A)

1. 修复 gateway APISIX SSL 配置（standalone YAML 模式下需 `ssls` 配置）
2. 修改 `conf/auth/task-auth/config.yaml` 的 `oidc.issuer` 为 `https://183.250.1.132:18444`
3. 移除 GitLab initializer monkey-patch
4. 回归测试

---

## 📎 附录

### A. 诊断脚本

`docs/specs/oidc-ssl-debug/reproduce_ssl_error.py` — Playwright 自动化复现脚本，
包含环境诊断、SSO 流程模拟、GitLab 日志收集。

### B. 相关文件

| 文件 | 说明 |
|------|------|
| `taskAuth/src/oidc_handlers.go` | OIDC Provider 实现 |
| `taskAuth/src/config.go` | taskAuth 配置加载 |
| `conf/auth/task-auth/config.yaml` | taskAuth OIDC 配置 |
| `gitService/docker-compose.yml` | GitLab 容器定义 + OIDC 客户端配置 |
| `taskGateway/apisix/apisix.yaml` | APISIX 路由 (OIDC 路径已配置) |
| `taskGateway/docker-compose.yml` | Gateway TLS 端口映射 |

### C. 相关 Memory

- [[taskauth-oidc-issuer-docker-reachability]] — OIDC Issuer Docker 网络可达性
- [[gitlab-oauth-bootstrap-fix]] — GitLab OAuth Application bootstrap 自愈
