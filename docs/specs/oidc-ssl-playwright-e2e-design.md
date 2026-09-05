# Design: OIDC SSL Record Layer Failure — Playwright 验证 + 修复 + E2E 测试

**日期**: 2026-06-24
**状态**: 待批准

---

## 1. 问题描述

用户登录 flow:

1. 访问 `http://183.250.1.132:4000/auth/login/` → 输入邮箱密码登录
2. 点击「代码仓库」→ 弹出 `http://183.250.1.132:8012/users/sign_in` (GitLab)
3. 点击「taskAuth SSO」→ **报错**:

```
Could not authenticate you from OpenIDConnect because "Ssl connect returned=1 errno=0 peeraddr=183.250.1.132:8003 state=error: record layer failure"
```

### 根因

`openid_connect` Ruby gem (v2.3.1, 随 GitLab CE 19.0 打包) 在 OIDC Discovery 阶段:
1. `SWD.url_builder` 默认指向 `URI::HTTPS`
2. taskAuth OIDC issuer 是 `http://183.250.1.132:8003` (纯 HTTP)
3. Discovery URL 被错误构造成 `https://183.250.1.132:8003/.well-known/openid-configuration`
4. SSL 握手打在 HTTP 端口上 → "record layer failure"

### 已知修复 (memory: [[oidc-ssl-record-layer-failure-fix]])

```ruby
# /opt/gitlab/embedded/service/gitlab-rails/config/initializers/zzz_fix_oidc_http.rb
require "swd"
SWD.url_builder = URI::HTTP
```

**当前状态**: 初始器文件在 GitLab 容器中 **不存在** (已验证 `docker exec gitlab cat ...` 返回 FILE_NOT_FOUND)。

---

## 2. 解决方案

### 2.1 修复: 注入初始器到 GitLab 容器

创建脚本 `gitService/scripts/fix_oidc_ssl.sh`，将 `zzz_fix_oidc_http.rb` 注入 GitLab 容器后执行 `gitlab-ctl reconfigure`。

同时更新 `gitService/run.sh`，在 GitLab 启动后自动调用此修复脚本 (类似已有的 `sync_omniauth_oidc.sh` 调用)。

### 2.2 Playwright 端到端验证

创建 `gitService/playwright/tests/oidc-sso-login.playwright.test.js`:

- **Step 1**: 导航到 `http://183.250.1.132:4000/auth/login/`
- **Step 2**: 勾选隐私政策 + 服务协议，填写邮箱密码，点击登录
- **Step 3**: 点击「代码仓库」导航项
- **Step 4**: 在新弹出的 GitLab 页面 (`/users/sign_in`) 点击「taskAuth SSO」按钮
- **Step 5**: 验证成功回调 (URL 包含 `/users/auth/openid_connect/callback`，页面不含 "record layer failure")

### 2.3 Playwright 问题核验 (诊断模式)

创建 `gitService/playwright/tests/oidc-ssl-diagnostic.playwright.test.js`:

- 调用 `http://183.250.1.132:8003/.well-known/openid-configuration` 验证 OIDC discovery 可达
- 验证返回 JSON 包含 `authorization_endpoint`, `token_endpoint` 等字段
- 检查 GitLab 容器中 SWD.url_builder 的值

---

## 3. 文件变更清单

| 文件 | 操作 | 说明 |
|------|------|------|
| `gitService/scripts/fix_oidc_ssl.sh` | **新增** | 注入 `zzz_fix_oidc_http.rb` 到 GitLab 容器 + reconfigure |
| `gitService/run.sh` | **修改** | GitLab 启动后自动调用 `fix_oidc_ssl.sh` |
| `gitService/playwright/playwright.config.js` | **新增** | Playwright 配置 (headless, baseURL 指向 183.250.1.132:8012) |
| `gitService/playwright/tests/oidc-ssl-diagnostic.playwright.test.js` | **新增** | 诊断测试: 验证 OIDC discovery 可达性 |
| `gitService/playwright/tests/oidc-sso-login.playwright.test.js` | **新增** | E2E 测试: 完整 SSO 登录流程 |
| `gitService/playwright/tests/oidc-ssl-fix-verify.playwright.test.js` | **新增** | 修复验证: 确认修复后无 SSL 错误 |
| `gitService/playwright/package.json` | **新增** | Playwright 依赖声明 |
| `conf/value-stream.yaml` | **修改** | `oidc-ssl-protocol-fix` stream: `planned` → `active`, 更新 test_file |

---

## 4. Playwright 测试设计

### 4.1 oidc-ssl-diagnostic (诊断)

```
Test: OIDC Discovery 端点可达
  GET http://183.250.1.132:8003/.well-known/openid-configuration
  → 验证 200 + JSON 包含 issuer/token_endpoint

Test: GitLab 容器内 SWD.url_builder 检查
  docker exec gitlab gitlab-rails runner "puts SWD.url_builder"
  → 修复前应为 URI::HTTPS, 修复后应为 URI::HTTP
```

### 4.2 oidc-sso-login (E2E)

```
Test: 完整 taskAuth SSO 登录流程
  browser.newPage()
  → page.goto('http://183.250.1.132:4000/auth/login/')
  → 勾选 checkbox[隐私政策, 服务协议]
  → fill email, password
  → click 登录按钮
  → waitForNavigation
  → click '代码仓库' link/button
  → waitForEvent('popup') → gitlabPage
  → gitlabPage.click('taskAuth SSO')
  → 验证成功 (无 "record layer failure" 错误文本)
```

### 4.3 oidc-ssl-fix-verify (核心验证)

```
Test: 修复后 OIDC 回调不出现 SSL 错误
  → 完整 SSO flow
  → 断言: page 不含 "record layer failure"
  → 断言: page 不含 "Could not authenticate"
  → 断言: 最终在 GitLab dashboard 或 callback 页面
```

---

## 5. 价值流影响分析

### 受影响的价值流

| Stream | 当前状态 | 变更 |
|--------|---------|------|
| `oidc-ssl-protocol-fix` / `swd-url-builder-http-fix` | **planned** | → **active** |
| `taskauth-oidc-issuer-docker-reachability` | active | 无变更 (依赖关系: issuer 可达性是本修复的前置条件) |

### 字段影响

| 字段 | 变更 |
|------|------|
| `git-service.runtime.oidc_protocol_fix` | 新增: SWD.url_builder 状态标记 |
| `git-service.runtime.oidc_ssl_initializer_present` | 新增: `zzz_fix_oidc_http.rb` 是否已注入 |

### 测试影响

| 测试文件 | 变更 |
|----------|------|
| `gitService/scripts/test_sync_omniauth_oidc.sh` | 已存在于 value-stream.yaml; 保留 |
| `gitService/playwright/tests/oidc-sso-login.playwright.test.js` | **新增** |
| `gitService/playwright/tests/oidc-ssl-fix-verify.playwright.test.js` | **新增** |
| `gitService/playwright/tests/oidc-ssl-diagnostic.playwright.test.js` | **新增** |

---

## 6. 领域概念清单 (为 /5-ddd 准备)

### Bounded Contexts

- **认证上下文 (Auth Context)** — taskAuth OIDC Provider, GitLab OmniAuth Consumer
- **基础设施上下文 (Infra Context)** — GitLab 容器生命周期、配置注入

### Key Entities

- **OIDC Client** (taskAuth): `oidc_clients` 表, bootstrap client `gitlab-git-service`
- **GitLab Application** (gitService): OmniAuth provider 配置

### Domain Events

- `GitLabReconfigured` — reconfigure 完成后 OmniAuth 已加载
- `OIDCFixApplied` — `zzz_fix_oidc_http.rb` 已注入

---

## 7. 实施顺序

1. **创建 Playwright 诊断测试** — 先写失败的测试 (red)
2. **创建修复脚本** `fix_oidc_ssl.sh`
3. **运行修复** — 注入 initializer + reconfigure
4. **验证修复** — 运行诊断测试 (green)
5. **创建 E2E 测试** — 完整 SSO 登录流程
6. **集成到 run.sh** — 自动化修复流程
7. **更新 value-stream.yaml** — 标记 `active`

---

## 8. 风险与注意事项

- **容器重建**: GitLab 容器 `docker compose down && up` 会丢失 initializer，需在 `run.sh` 中自动重新注入
- **幂等性**: `fix_oidc_ssl.sh` 必须幂等 — 检查文件是否已存在再决定是否 reconfigure
- **reconfigure 耗时**: `gitlab-ctl reconfigure` 可能需要 1-3 分钟，Playwright 测试需足够超时
- **网络**: Playwright 测试需要能访问 `183.250.1.132` 的各端口 (4000, 8003, 8012)
