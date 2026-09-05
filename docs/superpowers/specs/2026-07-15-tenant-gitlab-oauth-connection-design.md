# 设计文档：租户级自建 GitLab OAuth 连接

**日期：** 2026-07-15  
**状态：** 已采纳（goal-mode / 0-auto-flow 自动决策，跳过用户闸门）  
**作者：** claude  
**迭代名：** tenant-gitlab-oauth-connection  
**架构版本：** v30 target  
**页面入口：** `/tenant/:tenant/settings/gitlab-connection/`（侧栏「设置」→「GitLab 连接」）

---

## 1. 目标与成功标准（SMART）

| 标准 | 可验收判据 |
|------|------------|
| **S** 租户管理员可登记本公司自建 GitLab OAuth App | 设置页可 PUT/GET/DELETE 唯一连接（base_url、client_id、client_secret） |
| **M** 每租户最多 1 条 | `company_id` UNIQUE；重复 PUT 为更新 |
| **A** 成员可授权绑定 | 「Git 网站授权」providers 出现本租户条目；OAuth 走 taskGitOauth；refresh 落入既有表 |
| **R** Go-first + 单表所有权 | CRUD + 表归 `taskGitOauth` / `db/git-oauth`；零新增 Python 公网 path |
| **T** 删除级联 | DELETE 后该 `provider_key` 下用户绑定清除；事件入 Kafka（有 broker 时） |

**产品决策（已锁定）：** 租户级；入口 A（独立「GitLab 连接」菜单）；每租户最多 1 个实例。

**相对头脑风暴草案的修订：** 表与 CRUD **不**放 `taskCredentialService`，改放 **`taskGitOauth`**（v29 已迁 Go；密钥与换票同进程，避免跨服务传 secret）。

---

## 2. 方案对比与决策

| 方案 | 描述 | 决策 |
|------|------|------|
| A. CredentialService 存表 + GitOauth 远程 resolve | 密钥跨服务 | ❌ 扩大攻击面与延迟 |
| B. taskCloudService 仿云平台授权 | 云与 Git 混域 | ❌ |
| **C. taskGitOauth 存表 + CRUD** | 运行时本地 resolve | ✅ **采纳** |
| D. 仅运维 YAML | 无产品自助 | ❌ 不满足需求 |

**Redirect URI：** 按租户隔离（嵌入 `tenant-{company_id}`）  
`{public_gateway}/api/accounts/tenant-{company_id}/oauth/callback/`  
旧共享回调 `/api/accounts/tenant-gitlab/oauth/callback/` 仍保留兼容；GET/PUT 与 authorize/换票一律使用租户级 URI。

**provider_key：** `gitlab:tenant-{company_id}`

---

## 3. 目标拓扑

```
Admin Vue (settings/gitlab-connection)
  → GW → taskGitOauth CRUD (/api/tenant/{tid}/gitlab-oauth-connection/)
       → SQLite tenant_gitlab_oauth_connections (Fernet secret)

Member Vue (git-site-oauth)
  → Django GET providers（合并 YAML + 当前租户连接）
  → Django app/start → taskGitOauth gitlab oauth start
       → Resolve：YAML 或 tenant DB 行
       → 自建 GitLab authorize → callback → api_gitoauthappusercredential
```

---

## 4. 数据模型

### `tenant_gitlab_oauth_connections`（git-oauth DB）

| 列 | 类型 | 说明 |
|----|------|------|
| id | TEXT PK | snowflake/uuid |
| company_id | TEXT UNIQUE NOT NULL | 租户 |
| base_url | TEXT NOT NULL | GitLab origin（无尾斜杠） |
| client_id | TEXT NOT NULL | |
| client_secret_enc | TEXT NOT NULL | Fernet |
| remark | TEXT | 可选展示名 |
| redirect_uri | TEXT NOT NULL | 系统生成只读副本 |
| scope | TEXT | 默认 `read_repository write_repository api read_user` |
| active | INTEGER DEFAULT 1 | |
| created_at / updated_at | TEXT | |

Owner：`git-oauth`（`taskGitOauth` 唯一直连）。登记 `db/table_ownership.yaml`。

用户 refresh 仍用 `api_gitoauthappusercredential.provider` = `gitlab:tenant-{company_id}`。

---

## 5. API

### 公网（Gateway → taskGitOauth）

| 方法 | 路径 | 权限 |
|------|------|------|
| GET | `/api/tenant/{tid}/gitlab-oauth-connection/` | 租户成员（secret 脱敏） |
| PUT | `/api/tenant/{tid}/gitlab-oauth-connection/` | **租户管理员** |
| DELETE | `/api/tenant/{tid}/gitlab-oauth-connection/` | **租户管理员** |

PUT body：`{ "base_url", "client_id", "client_secret", "remark?" }`  
GET 响应含 `redirect_uri`、`provider_key`、`service_provider`、`configured`；无明文 secret。

### Catalog

扩展既有 `GET /api/accounts/git-oauth/providers/`：若请求带租户上下文（`X-Tenant-Id` 或 query `company_id`），Django 调 taskGitOauth internal `GET .../tenant-gitlab-connection?company_id=` 合并一条虚拟 provider。  
**无新 Django 公网 path**（仅改现有 view 响应）。

### OAuth

复用 `gitlab/oauth/start` + `/api/accounts/{service_provider}/oauth/callback/`；当 `service_provider` 匹配 `tenant-{id}` 时从 DB 加载 `ProviderConfig`。

---

## 6. 业务意图 → 事件对照

| 业务意图 | 事件名 | 发布点 | 消费者/副作用 |
|----------|--------|--------|---------------|
| 管理员保存连接 | `TenantGitLabOAuthConnectionUpserted` | taskGitOauth PUT | Kafka + 审计日志 |
| 管理员删除连接 | `TenantGitLabOAuthConnectionDeleted` | taskGitOauth DELETE | 级联删用户 credential；Kafka |
| 成员 OAuth 绑定成功 | 复用既有 bind 审计/事件策略 | callback | 与 v29 一致 |
| GET 连接 / providers | — | — | 纯查询，无事件 |

主题文件：`conf/domain-events/tenant_gitlab_oauth_connection_upserted`、`tenant_gitlab_oauth_connection_deleted`。

---

## 7. 前端

1. `Sidebar.vue`：设置下增加「GitLab 连接」  
2. 新页 `WorkspaceSettingsGitlabConnection.vue`：仿云平台授权表单（单条）  
3. `router.js`：`/tenant/:tenant/settings/gitlab-connection/`  
4. `UserGitSiteOAuthSettings.vue`：providers 请求附带当前 `tenantId`（若有）

---

## 8. 权限

| 操作 | 角色 |
|------|------|
| PUT/DELETE 连接 | tenant_admin（`ensureTenantAdmin` 同源：Django resolve-user-member） |
| GET 连接（脱敏） | tenant_member |
| 成员 OAuth 绑定 | 登录用户本人；仅当本租户已配置且 active |

---

## 9. 领域概念（轻量）

- **BC：** Git OAuth / Tenant Integration  
- **Aggregate：** `TenantGitLabOAuthConnection`（root: company_id）  
- **VO：** `GitLabBaseUrl`、`OAuthClientId`、`ProviderKey`  
- **既有：** `OauthCredentialBinding`（user × provider_key）

---

## 10. 价值流影响（摘要）

- 扩展 Git OAuth 绑定流：前置「租户配置 App」  
- 测试：Go 单测 CRUD/resolve；Playwright 设置页 + providers 出现租户项  

---

## 11. 架构交付物

| 格式 | 路径 |
|------|------|
| PlantUML | `docs/architecture/v30-application-integration-20260715-2014-claude.puml` |
| ArchiMate | `docs/architecture/v30-application-integration-20260715-2014-claude.archimate` |
| Mermaid | `docs/architecture/v30-application-integration-20260715-2014-claude.mermaid.md` |

---

## 12. 非目标

- 每租户多个自建 GitLab  
- 用户级自填 OAuth App  
- 迁移平台 YAML providers 到 DB  
- 自建 GitLab 上代用户创建 Application（仍由管理员在 GitLab UI 手工创建）

---

## 13. 🐍 Python 新增接口

**不触发。** 无新增 Python 公网 path；仅扩展既有 providers catalog 响应；全部新 CRUD 在 Go `taskGitOauth`。
