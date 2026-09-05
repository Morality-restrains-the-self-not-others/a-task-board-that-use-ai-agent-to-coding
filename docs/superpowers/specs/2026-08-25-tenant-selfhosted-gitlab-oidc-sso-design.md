# 设计文档：租户自建 GitLab — Git OAuth 绑定 + 平台 OIDC SSO 登录

**日期：** 2026-08-25  
**状态：** 已批准（2026-08-25 总体设计审批 approve）  
**作者：** cursor  
**迭代名：** tenant-selfhosted-gitlab-oidc-sso  
**页面入口：** `/tenant/:tenant/settings/gitlab-connection/`  
**触发来源：** 用户确认「两边都要」——既要 Git 授权绑定，也要自建 GitLab Web 支持平台账号（taskAuth OIDC）登录

---

## 0. 架构现状理解（口头确认）

根据当前架构设计稿（**v109 ✅ current**）：

- 共有企业景观 / 应用集成等视图；GitLab 相关基线来自更早版本（v30 租户 GitLab OAuth 连接、ADR-0014 多区域、ADR-0016 平台 GitLab SSO-only）。
- **业务层：** 租户管理员配置代码托管；成员绑定 Git 身份；平台用户经 taskAuth 登录。
- **应用层：** `taskFE`（gitlab-connection / Git 网站授权）、`taskGitOauth`（租户 Application 连接 + OAuth 换票）、`taskAuth`（OIDC IdP）、`gitService`（平台 GitLab OmniAuth RP）、`taskTenantService`（成员解析）。
- **技术层：** 平台 GitLab Omnibus + `apply_auth_policy.sh`；公网 APISIX 已反代 `/api/oidc/*` 与 `/.well-known/openid-configuration`。
- **上次 current：** v109（系统管理租户详情）；积压 target 含 v107 / v106 / v105。

📋 **架构版本历史（与本需求相关）：**

- v30 ✅ — 租户自建 GitLab **OAuth Application** 连接（Git 网站授权）
- ADR-0016 ✅ — **仅平台 gitService** 禁止自行注册、Web 仅 taskAuth SSO
- v105 🎯 — 测试角色 + 平台 GitLab 区域 `access_mode`（OIDC 闸门 `gitlab-git-service*`）

本次若批准「自建 GitLab 作为 OIDC RP」，将在 v109 基础上新增 **v110 target**（应用集成 + 企业景观）。纯文档说明现有 OAuth 绑定 **不单独升版本**。

---

## 1. 先把两条协议拆开（当前产品已覆盖其中一条）

同一设置页上有两种完全不同的「登录 / 授权」，名称都带 OAuth/OIDC，容易混：

| | **链路 A（已交付）Git 网站授权** | **链路 B（本次拟新增）平台账号登录自建 GitLab** |
|---|---|---|
| 谁是授权服务器 | **租户的 GitLab** | **平台 taskAuth** |
| 谁是客户端 / RP | 平台 `taskGitOauth` | **租户的 GitLab**（OmniAuth `openid_connect`） |
| 用户感知 | 在 daydaymoney「Git 网站授权」点绑定 | 打开自建 GitLab 登录页，点「taskAuth SSO」 |
| 目的 | 拿到 GitLab access_token，clone/push | 用平台账号进入 GitLab Web |
| 设置页表单 | Base URL + Application ID/Secret + Redirect URI | **今天没有** |
| 协议 | GitLab **OAuth2** `/oauth/authorize`（scope 不含 openid） | **OIDC** `/api/oidc/authorize` |
| 平台能否改对方 `gitlab.rb` | 否（管理员在 GitLab UI 建 Application） | 否（管理员在自建 GitLab 配 OmniAuth） |

**结论：** 在 gitlab-connection 里填 Application ID/Secret，**不能**让自建 GitLab 变成「像内建 GitLab 一样用平台 OIDC 登录」。那是链路 B，必须另发 OIDC client，并在**对方** GitLab 上配置 OmniAuth。

ADR-0016 **不适用于** 客户自建实例：我们不能、也不应强制关掉他们的注册/账密。

---

## 2. 链路 A — 自建 GitLab 如何配才能 OAuth 连通（已支持，补操作说明）

成员绑定走：`taskFE` Git 网站授权 → `taskGitOauth` → `{base_url}/oauth/authorize`。  
`provider_key` = `gitlab:tenant-{company_id}`；Redirect URI 只读，形如：

`https://www.daydaymoney.com/api/accounts/tenant-{company_id}/oauth/callback/`

### 2.1 租户管理员在自建 GitLab 上的操作

1. GitLab 版本：CE/EE 均可；需能创建 **OAuth Application**（Admin Area → Applications，或实例级 Applications）。
2. **New application：**
   - Name：例如 `Daydaymoney Git 授权`
   - **Redirect URI：** 原样粘贴设置页只读框（必须精确匹配，含租户 id 段）
   - **Confidential：** 是
   - **Scopes（必勾，与平台默认一致）：** `api`、`read_user`、`read_repository`、`write_repository`  
     - **不要**指望勾 `openid` 来完成链路 A；平台换票不走 GitLab OIDC discovery。
   - Trusted（可选）：勾选后成员少一次 GitLab 授权确认页。
3. 保存后把 **Application ID** / **Secret** 填回设置页，连同 **GitLab Base URL**（无尾斜杠，须浏览器与平台出站都能访问）。
4. 成员到「Git 网站授权」绑定该租户条目。

### 2.2 常见失败

| 现象 | 原因 |
|------|------|
| redirect_uri mismatch | GitLab Application 白名单与设置页不一致（含 http/https、尾斜杠、旧共享 `/tenant-gitlab/` 回调） |
| 授权后无法 clone | scope 缺 `write_repository` / `api` |
| 平台换票失败 | GitLab 对公网不可达，或自签证书不被平台信任 |
| 内网 GitLab | 浏览器能开、平台服务器不能出站 → 换票失败 |

链路 A **无新接口、无架构变更**。本次交付应在设置页补一段上述操作说明（文案），避免再被理解成 OIDC SSO。

---

## 3. 链路 B — 目标与成功标准（SMART）

| 标准 | 可验收判据 |
|------|------------|
| **S** 租户管理员可为本租户自建 GitLab **签发** taskAuth OIDC client，并复制 OmniAuth 片段 | 设置页「平台账号登录自建 GitLab」可启用/轮换/停用 |
| **M** 每租户最多 1 个 SSO client；`client_id` 稳定 | `gitlab-tenant-{company_id}` UNIQUE；重复启用 = 更新 redirect_uri，轮换才换 secret |
| **A** 仅本租户**在籍成员**能走完 authorize | 非成员 `access_denied`；平台 `gitlab-git-service*` 区域闸门不受影响 |
| **R** Go-first；OIDC 表仍归 taskAuth | 新 CRUD 在 `taskAuth`；零新增 Python path |
| **T** 对方 GitLab 配好 OmniAuth 后，成员可用平台会话登录 GitLab Web | Playwright/手册：成员 SSO 成功；外人失败 |

**非目标：**

- 平台代写 / 代重启客户 `gitlab.rb`
- 对客户实例执行 ADR-0016（关闭注册/账密）
- 每租户多个自建 GitLab SSO
- 用自建 GitLab 当 IdP 登录 daydaymoney.com（反向联邦，本次不做）
- 把租户 client 做成 `gitlab-git-service-*`（会误入区域 `access_mode` / tester 闸门）

---

## 4. 方案对比与决策

| 方案 | 描述 | 决策 |
|------|------|------|
| A. 文档-only：让客户自己找超管加 bootstrapClients | 无自助、conf 膨胀、secret 进仓库 | ❌ |
| B. 复用 `gitlab-git-service` + 多 redirect_uri | 所有租户共用一个 client；redirect 面过大；区域闸门无法区分 | ❌ |
| C. client_id = `gitlab-git-service-tenant-{id}` | 命中 `oidc_region_gate` 前缀，被当成平台区域实例 | ❌ 禁止 |
| **D. 独立 client `gitlab-tenant-{company_id}` + 成员闸门 + 设置页签发** | 数据所有权在 taskAuth；GitLab 侧手工 OmniAuth | ✅ **采纳** |

**redirect_uri（系统生成，只读）：** `{base_url}/users/auth/openid_connect/callback`  
（GitLab OmniAuth OpenID Connect 固定路径；`base_url` 与链路 A 同一 Origin。）

**issuer：** 公网 `${subdomains.gateway}`（与现网 `oidc.issuer` 一致），例如 `https://api.daydaymoney.com`。  
租户 SSO 片段使用 `{issuer}/api/oidc/{tenantId}/*`（ADR-0044），`discovery: false`，避免全局 well-known 覆盖租户路径。平台 RP 仍走全局 `/api/oidc/*`。

---

## 5. 目标拓扑

```
Admin Vue (gitlab-connection)
  ├─ 链路 A（现有）PUT GitLab Application → taskGitOauth
  └─ 链路 B（新增）PUT 启用 SSO → taskAuth
        → auth_oidc_client (gitlab-tenant-{tid}, secret hash)
        → 返回一次性 secret + OmniAuth 片段

Member 打开自建 GitLab 登录
  → GitLab OmniAuth → GET {issuer}/api/oidc/{tid}/authorize
       → taskAuth：path tenant 与 client 所属租户一致 + redirect_uri 白名单 + 成员闸门
       → 登录/同意 → code
  → GitLab 服务端 POST {issuer}/api/oidc/{tid}/token + GET .../jwks
       → 必须能出站访问公网 issuer（纯内网无出站则无法 SSO）
```

---

## 6. 数据模型

Owner：**taskAuth** / `auth_oidc_client`（不把 OIDC secret 写入 `taskGitOauth` 表）。

### 6.1 `auth_oidc_client` 扩展（dataMigrate/taskAuth）

| 列 | 说明 |
|----|------|
| `managed_by` | 现有 `bootstrap` \| `admin`；**新增 `tenant`**。bootstrap seed **不得** UPDATE/覆盖 `tenant` 行 |
| `owner_company_id` | 可空；租户 SSO 行必填；平台 gitlab-git-service / chrome-extension 为空 |
| `purpose` | 建议 `tenant_gitlab_sso`（便于闸门与审计，避免只靠 client_id 前缀） |

`client_id` 稳定：`gitlab-tenant-{company_id}`。  
secret 仅明文哈希入库；GET 永不回传明文。

冷热：OIDC client 为配置型小表（每租户 1 行），年增量 ≪ 10 万，**无需分区**。

### 6.2 与链路 A 的耦合

- **独立开关：** 可只配 Application、只开 SSO、或两者都开。
- 启用 SSO **需要已保存的 `base_url`**（否则无法计算 callback）。若尚未保存连接，SSO 区块提示先填 Base URL 并保存（可先只填 URL，Application 稍后补——PUT 连接 API 今日要求 client_id/secret，见下「缺口」）。

**缺口与处理：** 现有 PUT 连接强制 Application 三件套。SSO-only 时允许「仅 base_url」会改变链路 A 校验。  

**决策（推荐）：** 不放宽 PUT。启用 SSO 的请求 **body 自带 `base_url`**（须与已保存连接一致；若未保存连接则 SSO API 返回 400，文案：请先保存自建 GitLab Base URL）。若产品坚持「未配 Application 也能 SSO」，则另开 `PATCH` 只写 `base_url`——默认 **不** 做，避免半残连接行。

---

## 7. API（全部 Go）

### 7.1 公网 — taskAuth

| 方法 | 路径 | 权限 |
|------|------|------|
| GET | `/api/tenant/{tid}/gitlab-oidc-sso/` | 租户成员（无 secret） |
| PUT | `/api/tenant/{tid}/gitlab-oidc-sso/` | **租户管理员**；body `{ "base_url" }`；首次/轮换返回 `client_secret` |
| POST | `/api/tenant/{tid}/gitlab-oidc-sso/rotate/` | 管理员；新 secret 只出现在本次响应 |
| DELETE | `/api/tenant/{tid}/gitlab-oidc-sso/` | 管理员；废止 client（后续 authorize 失败） |

GET 响应含：`configured`、`client_id`、`issuer`、`redirect_uri`、`omniauth_snippet`（无 secret）、`discovery_url`。

Swagger：taskAuth OpenAPI 同步。网关：APISIX 增加上述前缀 → taskAuth（forward-auth）。

### 7.2 Authorize 闸门（修改现有 `handleOidcAuthorize`）

在现有 redirect_uri 校验之后：

1. 若 `purpose=tenant_gitlab_sso` 或 `client_id` 匹配 `gitlab-tenant-{id}`：
   - 解析 `owner_company_id`
   - 当前用户须为该租户在籍成员（复用 taskTenantService `GET /api/internal/tenant/members/resolve`，与 taskGitOauth `ensureTenantMember` 同源；fail-closed）
   - 非成员：`error=access_denied`（不得签发 code）
2. **禁止** 对 `gitlab-tenant-*` 套用 `oidc_region_gate`（前缀不是 `gitlab-git-service`）

幂等：纯授权码发放，重复点击走既有 code 一次性消费；无资金副作用。NFR：L2 — 幂等键为 OIDC `code` 一次性。

---

## 8. 自建 GitLab OmniAuth 设置（客户侧，平台只给片段）

前提：GitLab **Omnibus / 官方 Helm** 且版本带 `omniauth-openid-connect`（GitLab 13.4+ 常见；以对方实例为准）。管理员需能改 `gitlab.rb` 并 `gitlab-ctl reconfigure`。

推荐片段（设置页一键复制；值由 API 填入）：

```ruby
gitlab_rails['omniauth_enabled'] = true
gitlab_rails['omniauth_allow_single_sign_on'] = ['openid_connect']
gitlab_rails['omniauth_block_auto_created_users'] = false
gitlab_rails['omniauth_auto_link_user'] = ['openid_connect']
gitlab_rails['omniauth_providers'] = [
  {
    name: 'openid_connect',
    label: 'Daydaymoney SSO',
    args: {
      name: 'openid_connect',
      scope: ['openid', 'profile', 'email'],
      response_type: 'code',
      issuer: '<ISSUER>',
      discovery: false,
      client_auth_method: 'basic',
      uid_field: 'sub',
      client_options: {
        identifier: '<CLIENT_ID>',
        secret: '<CLIENT_SECRET>',
        redirect_uri: '<BASE_URL>/users/auth/openid_connect/callback',
        authorization_endpoint: '<ISSUER>/api/oidc/<TENANT_ID>/authorize',
        token_endpoint: '<ISSUER>/api/oidc/<TENANT_ID>/token',
        userinfo_endpoint: '<ISSUER>/api/oidc/<TENANT_ID>/userinfo',
        jwks_uri: '<ISSUER>/api/oidc/<TENANT_ID>/jwks'
      }
    }
  }
]
```

**强制约束（写入文案）：**

1. GitLab **进程**必须能 HTTPS 访问 issuer（token + JWKS）。纯内网、无出站 → SSO 会在换票阶段失败。
2. `redirect_uri` 必须与平台登记完全一致（含 scheme http 或 https、主机、无多余路径）。
3. **不要**把 `client_id` 配成平台内建的 `gitlab-git-service*`。
4. 平台 **不** 要求关闭对方密码登录；若对方同时开账密，身份可能双轨（由租户自负）。
5. 成员闸门在 **taskAuth**；即使对方 `block_auto_created_users=false`，非本租户平台用户也拿不到 code。
6. HTTP（非 TLS）自建 GitLab：**允许签发**；`redirect_uri` 与 Path A `base_url` 同 scheme（须与 GitLab `external_url` 一致）。签发时 slog warn `tenant_gitlab_oidc_sso_http_redirect`。GitLab 进程仍须 HTTPS 访问平台 issuer（token + JWKS）。

---

## 9. 前端

`WorkspaceSettingsGitlabConnection.vue` 自建区块下增加第三节：

- 标题：「用平台账号登录自建 GitLab」
- 说明：与上方 Application **不是**同一件事；小白向步骤写明值放入 `/etc/gitlab/gitlab.rb` 的 `identifier` / `issuer` / `secret`，不是填在本页
- 启用 / 轮换密钥 / 停用（`createClickGuard` + `Idempotency-Key`）
- 只读：issuer、client_id、redirect_uri、snippet
- secret：仅启用/轮换成功后的一次性展示 + 复制
- 链路 A 操作说明（§2）以可折叠帮助插入现有 Application 表单

---

## 10. 权限

| 操作 | 角色 |
|------|------|
| 启用/轮换/停用 SSO client | 租户管理员（`company:manage` / ensureTenantAdmin 同源） |
| GET 配置（无 secret） | 租户成员 |
| 走 OIDC authorize 登录对方 GitLab | 该 `owner_company_id` 在籍成员；超管不自动放行，除非也是成员 |

---

## 11. 业务意图 → 事件对照

| 业务意图 | 事件名（过去式） | 发布点 | 消费者/副作用 | 例外理由 |
|---------|----------------|--------|--------------|---------|
| 管理员启用/更新租户 GitLab OIDC SSO | `TenantGitLabOidcSsoEnabled` | taskAuth PUT | Kafka + 审计；无自动改对方 GitLab | — |
| 管理员轮换 secret | `TenantGitLabOidcSsoSecretRotated` | taskAuth POST rotate | Kafka + 审计 | — |
| 管理员停用 | `TenantGitLabOidcSsoDisabled` | taskAuth DELETE | Kafka；authorize 立即失败 | — |
| 成员 SSO 登录对方 GitLab | — | OIDC authorize | 既有授权码；**不**另发领域事件 | 与平台 gitService SSO 一致：认证协议路径，不改变平台业务事实 |
| GET 配置 | — | — | — | 纯查询 |
| 链路 A 保存 Application | 已有 `TenantGitLabOAuthConnectionUpserted` | taskGitOauth | 不变 | — |

主题：`conf/domain-events/tenant_gitlab_oidc_sso_enabled` 等（与既有 tenant gitlab oauth 事件同目录惯例）。

---

## 12. 领域概念（轻量，供 /6-ddd）

- **BC：** 身份（taskAuth OIDC）∩ 租户集成（自建 GitLab RP）
- **Aggregate：** `OidcClient`（root: client_id）；租户 SSO 是带 `owner_company_id` 的特化
- **政策：** `TenantMembershipGate`（authorize 时）
- **既有：** `TenantGitLabOAuthConnection`（链路 A，taskGitOauth）保持独立聚合

---

## 13. 价值流影响

仓库根无 `value-stream.yaml`（本次 skipped）。逻辑影响：

- 扩展「GitLab 连接」设置流：增加 SSO 签发步骤
- 扩展「成员使用 GitLab」：除 Git 授权外，可选 Web SSO
- 测试：Go 单测（签发、成员闸门、区域闸门互不干扰）；设置页文案 + 启用按钮；无新 Python 测

---

## 14. 🕸️ Code Review Graph 分析

| 项 | 内容 |
|----|------|
| 图状态 | CRG unavailable：仓库根 `.code-review-graph/graph.db` 不存在 |
| 关键发现 | 未查询 |
| 决策影响 | 以源码静态阅读为准：`handleOidcAuthorize`、`oidc_region_gate.go`、`tenant_connection_handlers.go`、`WorkspaceSettingsGitlabConnection.vue` |
| skip 理由 | `unavailable` — 缺图目录 |

---

## 15. 🐍 Python 新增接口清单与 Go 替代评估

**不触发。** 无新 Django/Flask path。OIDC CRUD 与闸门均在 Go `taskAuth`；链路 A 仍在 Go `taskGitOauth`。

`python_api_approval: not_applicable`（2026-08-25）

---

## 16. NFR 摘要（完整表留给 /5-nfr）

| 路径 | 分片键 | 幂等 |
|------|--------|------|
| PUT/DELETE `/api/tenant/{tid}/gitlab-oidc-sso/` | `tid`（租户）适配 | L3：client_id 自然键；Idempotency-Key 防双击建两 secret |
| GET `/api/oidc/authorize`（tenant client） | 用户会话；闸门按 company_id | L1：code 一次性 |
| GitLab token 出站 | 无平台写 | 客户网络依赖 |

安全：secret 轮换后旧 secret 立即失效；日志禁止打 secret / 授权码。

---

## 17. 🏛️ 架构变更影响

- **迭代版本**: v110 🎯 target
- **迭代名称**: tenant-selfhosted-gitlab-oidc-sso
- **作者**: cursor
- **设计日期**: 2026-08-25 20:26
- **ADR**: [ADR-0043](../../adr/0043-tenant-selfhosted-gitlab-oidc-sso.md)
- **新增文件**（每个视图四类伴生格式，**缺一不可**）:
  - 🆕 `docs/architecture/v110-enterprise-landscape-20260825-2026-cursor.puml`
  - 🆕 `docs/architecture/v110-application-integration-20260825-2026-cursor.puml`
  - 🆕 `docs/architecture/v110-enterprise-landscape-20260825-2026-cursor.diff.archimate`（增量变迁：v109→v110 + Plateau/Gap/WP 链）
  - 🆕 `docs/architecture/v110-application-integration-20260825-2026-cursor.diff.archimate`
  - 🆕 `docs/architecture/v110-enterprise-landscape-20260825-2026-cursor.full.archimate`（全量拓扑）
  - 🆕 `docs/architecture/v110-application-integration-20260825-2026-cursor.full.archimate`
  - 🆕 伴生 `.mermaid.md`（每个视图）
- **已有文件（未修改）**:
  - `docs/architecture/v109-*-1930-cursor.puml`（current）
- **变更明细**: 🟢 自建 GitLab OmniAuth RP + gitlab-oidc-sso API + SSO 事件 / 🟡 taskAuth authorize 成员闸门与 oidc client 列 / 🟡 taskFE 设置页 / 🟡 APISIX

### .archimate 架构变迁要点

| 文件 | 内容 |
|------|------|
| **`.diff.archimate`** | 增量模型 — Plateau v108 + Plateau v109 + Gap + WorkPackage + 本次 🟢/🟡 元素；视图含 `sourceConnection` |
| **`.full.archimate`** | 全量模型 — 链路 A（taskGitOauth）+ 链路 B（taskAuth OIDC）+ 平台 gitService SSO 对照 |

---

## 18. 风险与缓解

| 风险 | 缓解 |
|------|------|
| 任意平台用户 SSO 进客户 GitLab | **强制**成员闸门 fail-closed |
| 客户配错成平台 gitlab-git-service client | 文案禁止；签发 API 只给 `gitlab-tenant-*` |
| 客户 GitLab 无出站 | 设置页明确网络前提；不承诺内网隔离实例能 SSO |
| `managed_by=admin` 语义被租户行污染 | 新枚举 `tenant`，bootstrap 跳过 |
| 发现端点在部分边缘未反代 | snippet 含显式 `/api/oidc/*` 回退 |

---

## 18.1 角色权限（Step 2）

详见 `docs/superpowers/specs/2026-08-25-tenant-selfhosted-gitlab-oidc-sso-permission-analysis.md`。

- 管理 API：`settings.gitlab.main` + GET 成员 / 写 租户管理员
- Authorize：仅该 tid 在籍成员；超管不旁路
- 跨租户一律 403；secret 仅签发/轮换响应一次

## 19. 建议实施切片（批准后 /4-value-stream）

1. dataMigrate + authorize 闸门单测（红绿）
2. taskAuth CRUD + 事件 + OpenAPI + 网关
3. 设置页 UI + 链路 A 帮助文案
4. 手册：对方 OmniAuth；可选一台可达的自建 CE 冒烟
