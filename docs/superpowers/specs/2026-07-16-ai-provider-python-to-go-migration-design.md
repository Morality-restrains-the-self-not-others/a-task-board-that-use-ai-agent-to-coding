# 设计文档：ai-provider Python → Go 全量迁移

**日期：** 2026-07-16  
**状态：** 已采纳（goal-mode / 0-auto-flow 自动决策，跳过用户闸门）  
**作者：** claude  
**迭代名：** ai-provider-go-migration  
**架构版本：** v32 target  

---

## 1. 目标与成功标准（SMART）

| 标准 | 可验收判据 |
|------|------------|
| **S** 新建 Go 服务承接全部 Saas_Ai_Provider HTTP 契约 | `taskAiProvider` 实现 `/api/health/`、OIDC/SSO、vendor/admin CRUD、public catalog、cloud proxy、SPA 托管 |
| **M** 契约兼容 | 路径、方法、状态码、JSON 字段与现 Django 一致；JWT `iss=saas-ai-provider` / `typ=vendor\|staff`；Snowflake ID 字符串传输；消费者（taskCloudService public client、主站 SSO）无需改协议 |
| **A** 可运行切流 | runAll `ai-provider` 进程改为 Go；端口仍 **8010**；`conf/ai/ai-provider` 继续；Gateway upstream 不变 |
| **R** 符合 Go-first / 单表所有权 | 表 owner 仍为 `ai-provider`；仅 `taskAiProvider` 直连 `db/ai-provider/` |
| **T** Python 清理完成 | 删除 `task2app/Saas_Ai_Provider` Django 树（前端迁入 Go 服务目录）；CI/runAll/语言推断不再依赖 Python ai-provider；关键测例迁至 Go / 保留 Playwright 指向 :8010 |

**明确推翻：** `2026-07-05-task2app-api-go-split` 将「ai-provider 迁 Go」列为 Non-Goal；本次以元规则 `20_go_service_first_apis.md` + 用户显式目标为准，废止该 Non-Goal 并写入架构 v32。

---

## 2. 方案对比与决策

| 方案 | 描述 | 决策 |
|------|------|------|
| A. 并入 taskCloudService | 市场域与云资源域混边界 | ❌ 扩大耦合；ai-provider 已是独立库 owner |
| B. 并入 taskAuth | 认证与镜像市场混放 | ❌ 仅 OIDC RP 依赖，业务主体是 marketplace |
| **C. 新建 `taskAiProvider`** | 独立限界上下文，端口 8010 直替 | ✅ **采纳** |
| D. 长期双跑 Django+Go | 违反单表所有权 | ❌ |

**迁移策略：** Strangler 一次性切流（同端口、同库、同路径契约）→ 验证 → 删除 Python。不做长期双写。

**前端：** Vue SPA 从 `Saas_Ai_Provider/frontend` 迁至 `taskAiProvider/frontend`；Go 托管 `dist`；dev 仍可 Vite proxy → :8010。

---

## 3. 目标拓扑

```
Browser / Vue SPA → (:8010 直连或 Gateway) → taskAiProvider(:8010)
                                              ├─ SQLite db/ai-provider/
                                              ├─ OIDC RP → taskAuth
                                              ├─ SSO bridge JWT ← saas-backend
                                              ├─ Proxy → taskCloudService(:8018)
                                              └─ OCI registry (manifest resolve)
taskCloudService → GET /api/public/* (catalog / runtime)
```

---

## 4. 数据与鉴权

### 4.1 表（沿用，不改 schema）

`marketplace_vendor`、`marketplace_platformstaff`、`marketplace_containerimagegroup`、`marketplace_vendorcontainerimage`、`marketplace_userdatatemplate`、`marketplace_vendorcloudserverimage`、`marketplace_containercloudserverassociation`、`marketplace_containerimagereviewhistory`、`marketplace_vendorcloudserverimageuserdata`

Django 系统表（`django_session` 等）切流后仅 OIDC PKCE 态改用 Go signed cookie / 内存+SQLite session 表或复用 `django_session` 行格式；切流瞬间进行中的 OIDC 需重开（可接受）。

### 4.2 JWT / SSO / OIDC

| 机制 | 契约（保持） |
|------|----------------|
| 本服务 JWT | HS256，`iss=saas-ai-provider`，`typ=vendor\|staff`，Bearer |
| SSO Bridge | `iss=task2app-sso`，`aud=saas-ai-provider`，secret=`ssoJwtSecret` |
| OIDC RP | client `ai-provider` → `{oidcRpIssuer}/api/oidc/*`；PKCE；callback 签本服务 JWT |
| 本地 login/register | 继续 **403**（SSO-only） |

### 4.3 ID 传输

进程间/JSON：`id` / `*_id` 一律 **string**（见 `11_id_field_string_transit.md`）。

---

## 5. 配置与运维

- 继续读 `conf/ai/ai-provider/`（`config.yaml` + `django.yaml` sso 秘密）
- runAll：`working_dir: taskAiProvider`，`build_command: ./build.sh`，`start_command: ./bin/taskAiProvider`，去掉 `DJANGO_SETTINGS_MODULE`
- OpenAPI：`/api/schema/` + `/api/swagger/`；Gateway `/gateway/openapi/aiProvider.json`（若已有则改 rewrite 目标）
- 日志：`tracelog` + 入站/出站请求日志（对齐现 Django `incoming_requests` / `outgoing_requests` 语义）
- listen：`0.0.0.0:8010`

---

## 6. 领域事件

| 意图 | 事件 | MQ |
|------|------|-----|
| Vendor OIDC 登录成功 | `VendorLoggedInViaOidc` | 首期结构化日志（对齐现 LoggingEventPublisher） |
| Staff OIDC 登录成功 | `StaffLoggedInViaOidc` | 同上 |
| OIDC 登录失败 | `OidcLoginFailed` | 同上 |
| 镜像提交审核 | `ContainerImageSubmitted` | 结构化日志；后续可接 Kafka |
| 镜像审批通过/驳回 | `ContainerImageApproved` / `Rejected` | 同上 |

书面例外：纯查询（health、public catalog、list GET）无事件。

**废弃：** `container-image-hash-calculation` MQ consumer（字段已删，死代码不迁）。

---

## 7. 架构交付物

| 格式 | 路径 |
|------|------|
| PlantUML | `docs/architecture/v32-application-integration-20260716-0013-claude.puml` |
| ArchiMate | `docs/architecture/v32-application-integration-20260716-0013-claude.archimate` |
| Mermaid | `docs/architecture/v32-application-integration-20260716-0013-claude.mermaid.md` |

---

## 8. 风险与缓解

| 风险 | 缓解 |
|------|------|
| DRF ViewSet 细节差异（分页/过滤/嵌套） | 契约测对关键 serializer 字段；Playwright django8010 套件回归 |
| OIDC session 切换打断进行中登录 | 运维窗口短暂；前端可重开 authorize |
| Proxy 路径漏转 | 显式注册与 Python 相同的 4 条 proxy |
| 删除 Python 后遗漏脚本 | 全文检索 `Saas_Ai_Provider` / `provider.settings` 清理清单 |
| SPA 静态路径 | Go FileServer + SPA fallback，对齐 Django catch-all |

---

## 9. 非目标

- 不改 marketplace 业务语义 / 审批状态机
- 不合并进 taskCloudService
- 不一次引入 Kafka 真投递（可后续）
- 不迁移 Django Admin（`/admin/django/` 废弃；运营用 SPA AdminPortal）

---

## 10. 清理清单（Python 删除前）

- [ ] `conf/runAll.yaml` ai-provider → taskAiProvider
- [ ] `db/table_ownership.yaml` code_roots → `taskAiProvider`
- [ ] `db/api_route_ownership.yaml` 更新 process / routes
- [ ] `task2app/run.sh` 去掉嵌入式 Saas_Ai_Provider 启动（或改为调用 Go）
- [ ] `db/ai-provider/migrate.sh` 不再 `manage.py migrate`（Go 确认 schema 已存在即可 exit 0）
- [ ] 删除 `task2app/Saas_Ai_Provider`（保留 README 指向 `taskAiProvider`）
- [ ] 更新 `.ai` / skills 中 Django ai-provider 表述
- [ ] Playwright 基地址仍 :8010，路径不变
