# 设计文档：镜像市场管理（SSO）Connection Refused 修复

> 状态: 待审批 | 日期: 2026-06-29 | 作者: AI Assistant

---

## 1. 问题描述

账号 `author@example.com` 登录 `http://183.250.1.132:4000/system-admin/` 后，点击侧边栏「镜像市场管理（SSO）」，浏览器跳转到 `http://183.250.1.132:8010/admin#sso_bridge=<jwt>` 后显示 **Connection Refused**。

## 2. Playwright 核验结果

| 步骤 | 操作 | 结果 |
|------|------|------|
| 1 | 导航至 system-admin | ✅ 重定向到 `/auth/login/` |
| 2 | 填写邮箱/密码 + 勾选隐私协议 → 登录 | ✅ 登录成功，到达 `/system-admin/` |
| 3 | 找到「镜像市场管理（SSO）」链接 | ✅ href=`http://183.250.1.132:18081/accounts/sso/ai-provider/admin/` |
| 4 | 点击链接（新标签页打开） | ❌ 最终 URL: `chrome-error://chromewebdata/` |
| 5 | 直接访问 `http://183.250.1.132:8010/admin` | ❌ `net::ERR_CONNECTION_REFUSED` |
| 6 | 访问 `http://127.0.0.1:8010/admin` | ✅ 连接成功（返回 404，SPA 路由问题另案处理） |

**结论**：SSO 重定向链正常（登录 → 签发 JWT → 302 跳转），问题出在跳转目标端口 8010 无法从外部 IP 访问。

## 3. 根因分析

### 3.1 配置链追踪

```
SSO 重定向 URL 来源:
  conf/base.yaml:34
    local.provider: ${PROVIDER_ADDR:-183.250.1.132:8010}
  → 解析为: subdomains.provider = "183.250.1.132:8010"

  conf/core/django/config.yaml:4
    containerRegisterAdminSSO: http://${subdomains.provider}
  → SSO 视图读此值构造 302 Location:
    http://183.250.1.132:8010/admin#sso_bridge=<jwt>

服务绑定地址来源:
  conf/ai/ai-provider/config.yaml:2
    host: 127.0.0.1
  → port_config_loader.py:77-80
    bind_addr = "127.0.0.1" (localhost only)
  → Django runserver 绑定: 127.0.0.1:8010
```

### 3.2 核心矛盾

```
SSO 跳转目标:  183.250.1.132:8010  (外部可达地址)
服务实际监听:  127.0.0.1:8010      (仅本地回环)
              ↑ 浏览器连接外部 IP → 被拒绝 ↑
```

### 3.3 影响范围

- 所有 `DEPLOY_MODE=local` 环境
- 浏览器在**同机**时也无法通过外部 IP 访问 localhost-only 服务
- 浏览器在**异机**时更无法访问
- 影响入口：管理员 SSO（`/admin`）和厂商 SSO（`/`）

### 3.4 附加发现：配置路径 Bug

`port_config_loader.py` 和 `run.sh` 中配置文件路径为 `conf/ai-provider/config.yaml`，但实际路径为 `conf/ai/ai-provider/config.yaml`（runAll `conf_app: ai/ai-provider` 保持一致）。导致配置从未被读取，服务始终使用默认值 (`127.0.0.1:8010, isDev=false`)。

### 3.5 附加发现：ALLOWED_HOSTS 模板未解析

配置中 `allowedHost: http://${subdomains.provider}` 的 `${subdomains.provider}` 模板变量被 `port_config_loader.py` 的裸 YAML 读取器原样保留，未被解析为实际 IP。导致 ALLOWED_HOSTS 中缺少外部 IP。

## 4. 解决方案

### 方案 A：修改绑定地址为 0.0.0.0（推荐 ⭐）

**改动**：`conf/ai/ai-provider/config.yaml` `host: 127.0.0.1` → `host: 0.0.0.0`

**原理**：`port_config_loader.py` 将 host 直接作为 bind_addr（非 localhost 时），设为 `0.0.0.0` 后服务监听所有网络接口。

**优点**：
- 一行改动，影响最小
- 同时兼容 localhost 和外部 IP 访问
- 与项目中其他服务一致（Django 主站 `host: 127.0.0.1` 但通过 gateway 代理；ai-provider 是直连模式，需要直接可达）

**缺点**：
- 开发环境稍微暴露（本地开发可接受）

### 方案 B：修改 SSO 跳转地址为 127.0.0.1

**改动**：`conf/base.yaml` `local.provider: 127.0.0.1:8010`

**优点**：不改服务绑定

**缺点**：
- 仅浏览器与服务同机时可用
- 异机访问彻底不可用
- 与其他服务配置不一致（其他 local 地址均使用外部 IP）

### 方案 C：通过 Gateway 代理 8010

**改动**：在 `routes.yaml` 添加 ai-provider 上游和路由，SSO 跳转改为 gateway 路径

**优点**：架构正确，统一鉴权

**缺点**：
- 改动范围大（gateway 路由 + SSO URL 配置 + CORS/域名处理）
- 当前 ai-provider 设计为独立直连，改为 gateway 代理需评估 cookie/域名影响

### 建议

**短期**：方案 A（改 bind 为 0.0.0.0），立即修复 Connection Refused。

**长期**：方案 C（gateway 代理），统一架构。当前先修复可用性。

## 5. 涉及文件

| 文件 | 改动 |
|------|------|
| `conf/ai/ai-provider/config.yaml:2` | `host: 127.0.0.1` → `host: 0.0.0.0` |
| `conf/ai/ai-provider/config.yaml` | 新增 `allowedExtendHosts: ['*']` (dev 模式) |
| `Saas_Ai_Provider/provider/port_config_loader.py` | 修复配置路径: `conf/ai-provider/` → `conf/ai/ai-provider/` |
| `Saas_Ai_Provider/run.sh` | 修复配置路径: `conf/ai-provider/` → `conf/ai/ai-provider/` |
| `Saas_Ai_Provider/provider/settings.py:41` | 新增: dev 模式 `ALLOWED_HOSTS = ["*"]` |
| 新增 E2E 测试 | `task2app/playwright/saas_ai_provider/tests/django8010-sso-connection-refused-fix.playwright.test.js` |

## 6. 端到端测试设计

### 测试场景

```
测试名称: SSO 桥接完整流程 — 主站管理员登录后跳转至镜像市场管理后台
前置条件: Saas_Ai_Provider 运行在 8010 端口
测试步骤:
  1. 导航至 system-admin 登录页
  2. 以 author@example.com 登录（若未登录）
  3. 点击「镜像市场管理（SSO）」链接
  4. 等待新标签页打开并完成重定向
  5. 验证: 最终页面不包含 Connection Refused 错误
  6. 验证: URL 包含 :8010/admin（成功到达目标服务）
  7. 验证: 页面能加载 SPA（出现 Vue 根元素或特定文案）
```

### 测试用例列表

1. **TC-01**: 无效 bridge token 被拒绝（已有测试，保留）
2. **TC-02**: 本地注册/密码登录返回 403 sso_only（已有测试，保留）
3. **TC-03**: **新增** — 超管登录后 SSO 跳转成功到达 admin 页面
4. **TC-04**: **新增** — 直接访问 8010 端口确认服务可达
5. **TC-05**: **新增** — SSO bridge 换票接口正常工作

## 7. 领域概念清单

| 类别 | 概念 |
|------|------|
| Bounded Context | `ai-provider`（镜像市场上下文）、`auth`（认证上下文） |
| Key Entities | `PlatformStaff`（管理员）、`User`（主站用户）、`Vendor`（厂商） |
| Candidate Aggregates | `PlatformStaff` 聚合根、`Vendor` 聚合根 |
| Domain Events | `StaffBridgeTokenIssued`（桥接 token 签发）、`SSOTokenExchanged`（token 换票完成） |

## 8. 价值流影响

无现有价值流受直接影响。此变更为 Bug 修复，涉及：
- **配置层**：`conf/ai/ai-provider/config.yaml` — host 绑定地址
- **新增测试**：`task2app/playwright/saas_ai_provider/tests/sso-bridge-connection-e2e.playwright.test.js`

---

## 附录：SSO 桥接时序图

```
用户浏览器                    主站 Django (8001/gw)         Saas_Ai_Provider (:8010)
    │                              │                              │
    │─ GET /system-admin/ ────────>│                              │
    │<─ 200 HTML (含SSO链接) ─────│                              │
    │                              │                              │
    │─ 点击「镜像市场管理（SSO）」  │                              │
    │─ GET /accounts/sso/ai-provider/admin/ ─>│                   │
    │                              │─ 验证登录态 + superuser      │
    │                              │─ issue_staff_bridge_token()  │
    │<─ 302 Location: :8010/admin#sso_bridge=<JWT> ─│             │
    │                              │                              │
    │─ GET :8010/admin#sso_bridge ─────────────────────────────>│
    │                                                             │
    │  ❌ CONNECTION REFUSED (127.0.0.1 only)                    │
    │                                                             │
    │  ✅ 修复后: bind=0.0.0.0                                   │
    │<─ 200 SPA index.html ─────────────────────────────────────│
    │─ POST /api/auth/sso/exchange/ {bridge} ──────────────────>│
    │<─ 200 {access, role: "staff"} ────────────────────────────│
    │                                                             │
    │  ✅ 管理员已登录镜像市场                                     │
```
