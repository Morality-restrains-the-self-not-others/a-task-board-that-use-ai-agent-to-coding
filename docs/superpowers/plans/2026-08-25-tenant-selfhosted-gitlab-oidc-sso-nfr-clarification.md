# NFR 澄清：租户自建 GitLab 平台 OIDC SSO

- **日期**: 2026-08-25
- **价值流**: `docs/superpowers/plans/2026-08-25-tenant-selfhosted-gitlab-oidc-sso-value-stream.md`
- **设计**: `docs/superpowers/specs/2026-08-25-tenant-selfhosted-gitlab-oidc-sso-design.md`

默认管理面 L2；**认证/密钥路径抬到 L3**。

## 路径分片键强制审视

| 路径 | 分片 ID | 适配？ | 可伸缩性 | 说明 |
|------|---------|--------|----------|------|
| `GET/PUT/DELETE /api/tenant/{tid}/gitlab-oidc-sso/` | `{tid}`=`company_id` | 是 | L2 | 一行一租户；查询必带 owner_company_id |
| `POST /api/tenant/{tid}/gitlab-oidc-sso/rotate/` | `{tid}` | 是 | L2 | 同上 |
| `GET /api/oidc/{tid}/authorize` | `{tid}`=`company_id` | 是 | L3 | 租户 SSO 协议面；path 与 client owner 必须一致 |
| `POST /api/oidc/{tid}/token` | `{tid}` | 是 | L3 | 同上 |
| `GET/POST /api/oidc/{tid}/userinfo` | `{tid}` | 是 | L3 | aud → owner_company_id 与 path 对齐 |
| `GET /api/oidc/{tid}/jwks` | `{tid}` | 预留 | L1 | URL 可分片；密钥集暂与全局相同 |
| `GET /api/oidc/authorize?client_id=gitlab-tenant-{tid}` | client_id 内嵌 tid | 是 | L3 | 全局路径保留兼容；新片段不再指向此 URL |
| FE `/tenant/:tenant/settings/gitlab-connection/` | `:tenant` | 是 | L1 | 低频设置页 |
| Kafka `TenantGitLabOidcSso*` key=`company_id` | company_id | 是 | L1 | 审计事件，无自动消费者 |
| `GET /.well-known/openid-configuration` | 无 | — | **L0** | 全局发现文档；单实例可承受。升级触发：公网发现 QPS 持续 > 单节点 P95 目标时再 CDN/边缘缓存 |

禁止跨租户扫描 `auth_oidc_client`；签发路径用稳定 `client_id` 主键查找。

## 幂等性强制审视

| 路径 | 副作用 | 级别 | 重复边界 | 幂等键 | 重放语义 |
|------|--------|------|----------|--------|----------|
| GET SSO | 无 | L0 | — | — | 纯查询 |
| PUT 签发 | 写 client + 事件 | **L3** | 同一 `company_id` 一个 SSO client | `client_id` UNIQUE + `Idempotency-Key` | 已存在：200 `configured:true` **不**再发 secret；并发 UNIQUE 冲突读已有行 |
| POST rotate | 换 hash + 事件 | **L3** | 同一 tid + 同一次按钮意图 | `Idempotency-Key` 表 `(company_id, rotate, key)` UNIQUE | 同键：200 无 secret；新键：换密并只回一次 plaintext |
| DELETE 吊销 | 删行 + 事件 | **L3** | 同一 tid 关闭 SSO | `Idempotency-Key` + 行不存在 | 已删：204 空操作 |
| OIDC authorize | 写授权码 | L3（沿用现网） | 每次登录新 code | 授权码 UNIQUE | 非成员不写码 |
| OIDC token | 消费 code | L3（沿用） | 单 code 单用 | `used=1` 原子 UPDATE | 重放失败 |
| 领域事件消费 | 无本增量消费者 | L0 | — | — | 审计-only；禁止为 DLT 注册自动消费者 |

资金/配额路径不适用。密钥泄露面：rotate 必须 L3，禁止双击产生两个有效 secret 同时展示。

## 其他质量属性

| 属性 | 程度 | 决策 |
|------|------|------|
| 可用性 | L2 | tenant 成员解析失败 → authorize **拒绝**（安全优先于可用性） |
| 保密 | L3 | secret 哈希存储；日志无 secret |
| 一致性 | L3 | client 行与 Path A base_url 校验同步于 PUT |
| 可观测 | L2 | WARN 拒绝（client_id、user_id、tid、trace_id）；INFO 签发/轮换/吊销 |

## 领域模型影响

- 聚合：`TenantGitLabOidcClient`（根 = `client_id` / `owner_company_id` 一对一）
- 不引入跨租户集合
- 幂等表从属于该聚合的应用服务，非独立 BC
