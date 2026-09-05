# 剩余问题修复 — 设计文档

## 问题清单

### 1. Vendor DB 记录缺失（原 SSO bridge 问题）

**症状**: 用户 `contact@daydaymoney.com` 点击「厂商门户（SSO）」提示未找到匹配厂商账号

**根因**: `db/ai-provider/ai-provider.sqlite3` 中 `marketplace_vendor` 表为空（0 条记录），而旧数据库 `Saas_Ai_Provider/db.sqlite3` 中的 Vendor 在 DB 路径迁移时未同步。

**修复**: 通过 Django shell 在 ai-provider DB 中创建 Vendor 记录。

### 2. taskAuth OIDC 多客户端支持

**现状**: taskAuth (`taskAuth/src/config.go`) 仅支持单个 bootstrap OIDC client：
```go
OidcBootstrapClientID     string
OidcBootstrapClientSecret string
OidcBootstrapRedirectURI  string
```

`seedOidcBootstrapClient()` 只创建这一个 client。DB 层 (`oidc_db.go`) 已支持多 client（`INSERT OR IGNORE`），仅为配置层限制。

**修复**: 将 `BootstrapClient*` 三个标量字段改为 `BootstrapClients []OidcBootstrapClient` 列表。

### 3. OIDC View 测试 + E2E

**现状**: views_oidc.py 无测试覆盖，E2E Playwright 测试未编写。

**修复**: 添加 view 层单元测试 + Playwright E2E 测试。

---

## 设计方案

### Fix 1: Vendor DB 创建

```bash
cd /tmp/ram-work/task2app/Saas_Ai_Provider
DJANGO_SETTINGS_MODULE=provider.settings python -c "
import django; django.setup()
from apps.marketplace.models import Vendor
import secrets
v = Vendor(email='contact@daydaymoney.com', company_name='测试厂商', contact_name='', is_active=True)
v.set_password(secrets.token_urlsafe(48))
v.save()
print(f'Vendor created: id={v.id}')
"
```

**影响**: 无代码变更，仅数据库操作。

### Fix 2: taskAuth Go 多客户端

#### 文件变更

| 文件 | 操作 | 说明 |
|------|------|------|
| `taskAuth/src/config.go` | 修改 | `BootstrapClient*` → `BootstrapClients []` |
| `taskAuth/src/oidc_bootstrap.go` | 修改 | 迭代列表创建多个 client |
| `taskAuth/src/config_test.go` | 新建 | 测试新旧配置格式兼容 |
| `conf/auth/task-auth/config.yaml` | 修改 | 改用 `bootstrapClients` 列表格式 |

#### config.go 变更

```go
// Before (single):
OidcBootstrapClientID     string
OidcBootstrapClientSecret string
OidcBootstrapRedirectURI  string

// After (list):
type OidcBootstrapClient struct {
    ClientID     string `yaml:"clientId"`
    ClientSecret string `yaml:"clientSecret"`
    RedirectURI  string `yaml:"redirectUri"`
}
OidcBootstrapClients []OidcBootstrapClient
```

配置兼容：若 YAML 中仍有旧格式（单 client），自动升级为列表（向后兼容）。

#### oidc_bootstrap.go 变更

```go
func seedOidcBootstrapClients() error {
    for _, c := range cfg.OidcBootstrapClients {
        redirectURIs := []string{c.RedirectURI}
        urisJSON, _ := json.Marshal(redirectURIs)
        if err := ensureOidcClient(c.ClientID, c.ClientSecret, c.ClientID, string(urisJSON)); err != nil {
            return err
        }
        log.Printf("[taskAuth] oidc bootstrap: client %s ready", c.ClientID)
    }
    return nil
}
```

#### config.yaml 变更

```yaml
oidc:
  signingKeyPath: ""
  accessTokenTTL: 3600
  idTokenTTL: 3600
  issuer: "${subdomains.gateway}"
  bootstrapClients:
    - clientId: "gitlab-git-service"
      clientSecret: "gsoidc-dev-secret-do-not-use-in-prod"
      redirectUri: "http://${subdomains.gitlab}/users/auth/openid_connect/callback"
    - clientId: "ai-provider"
      clientSecret: "aip-oidc-dev-secret"
      redirectUri: "http://${subdomains.provider}/api/auth/oidc/callback/"
  gitServicePublicBase: "http://${subdomains.gitlab}"
```

### Fix 3: View Tests + E2E

#### View 测试 (`tests/test_oidc_views.py`)

使用 Django `RequestFactory` + mock OIDC provider 测试：

| 测试场景 | 预期 |
|----------|------|
| `oidc_authorize` role=vendor → 302 到 taskAuth | 302, Location 含 `response_type=code` |
| `oidc_authorize` role=invalid → 400 | 400, detail 含 "无效的 role" |
| `oidc_authorize` 写入 session | session 含 oidc_auth 键 |
| `oidc_callback` state 不匹配 → 400 | 400, detail 含 "state 不匹配" |
| `oidc_callback` 缺少 code → 400 | 400 |
| `oidc_callback` 缺少 session → 400 | 400, "OIDC 会话已过期" |
| `oidc_callback` error from provider → 302 to /?error= | 302 |

#### E2E (Playwright)

```
test_oidc_login_button_visible → 确认 OIDC 按钮在页面上可见
test_oidc_authorize_redirect → 点击按钮 → 302 重定向到 taskAuth
```

---

## 实施顺序

```
Fix 1 (Vendor DB) → Fix 2 (taskAuth Go) → Fix 3 (View Tests) → E2E
```

Fix 1 可立即执行，无依赖。Fix 2 需要 taskAuth 重新编译。Fix 3 依赖 Fix 2 完成后 taskAuth 接受 ai-provider client。

## 总结清单

- Fix 1 (Vendor DB): 直接 DB 插入，无代码变更
- Fix 2 (taskAuth multi-client): config.go + oidc_bootstrap.go + config.yaml 共 3 文件
- Fix 3 (view tests + E2E): test_oidc_views.py + E2E playwright test
