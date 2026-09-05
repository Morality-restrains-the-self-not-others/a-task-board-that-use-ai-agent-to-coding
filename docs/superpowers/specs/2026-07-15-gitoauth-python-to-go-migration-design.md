# 设计文档：gitOauth Python → Go 全量迁移

**日期：** 2026-07-15  
**状态：** 已采纳（goal-mode / 0-auto-flow 自动决策，跳过用户闸门）  
**作者：** claude  
**迭代名：** gitoauth-go-migration  
**架构版本：** v29 target  

---

## 1. 目标与成功标准（SMART）

| 标准 | 可验收判据 |
|------|------------|
| **S** 新建 Go 服务承接全部 gitOauth HTTP 契约 | `taskGitOauth` 实现 `/api/health/`、browser OAuth（GitHub/GitLab start/start-from-gateway/callback）、service_provider 动态 callback、全部 internal 换票/摘要/删除/审计 API |
| **M** 契约兼容 | 路径、方法、状态码、JSON 字段与现 Python 服务一致；既有 Fernet 密文可解密；消费者（taskCredential/Project/Cloud、Django bind 桥）无需改协议 |
| **A** 可运行切流 | runAll `git-oauth` 进程改为 Go；端口仍 **8002**；Gateway upstream 不变 |
| **R** 符合 Go-first / 单表所有权 | 表 owner 仍为 `git-oauth`；仅 `taskGitOauth` 直连 `db/git-oauth/` |
| **T** Python 清理完成 | 删除 `gitOauth/` Django 树（或降为文档/归档说明）；CI/runAll/语言推断不再依赖 Python gitOauth；相关测例迁至 Go |

**明确推翻：** `2026-07-05-task2app-api-go-split` 将「git-oauth 迁 Go」列为不拆项；本次以元规则 `20_go_service_first_apis.md` + 用户显式目标为准，将该 Non-Goal 废止并写入架构 v29。

---

## 2. 方案对比与决策

| 方案 | 描述 | 决策 |
|------|------|------|
| A. 扩展 taskAuth | 身份与 Git OAuth App 凭据混边界 | ❌ 与既有 taskAuth/gitOauth 边界文档冲突 |
| B. 扩展 taskCredentialService | 凭证消费与 OAuth 协议状态机混放 | ❌ credential 已是消费方；扩大耦合 |
| **C. 新建 `taskGitOauth`** | 独立限界上下文，端口 8002 直替 | ✅ **采纳** |
| D. 目录内原地把 `gitOauth/` 换成 Go | 路径不变但破坏嵌套仓/历史 | ⚠️ 采用 C，runAll `working_dir` 指向新目录 |

**迁移策略：** Strangler 一次性切流（同端口、同库、同路径契约）→ 验证 → 删除 Python。不做长期双写（表所有权禁止双服务直连）。

---

## 3. 目标拓扑

```
Browser / Vue → taskGateway → taskGitOauth(:8002)
                                      ├─ SQLite db/git-oauth/
                                      ├─ GitHub / GitLab OAuth
                                      └─ POST saas-backend …/internal/bind/
Go 消费者（Credential/Project/Cloud）→ taskGitOauth internal APIs
```

---

## 4. 数据与加密

- **表（沿用）：** `api_gitoauthappusercredential`、`api_gitoauthappaccesstokenuseaudit`、`api_gitoauthtaskcredentialaudit`
- **加密：** Fernet；`key = urlsafe_b64(sha256(SECRET_KEY))`，与 Python 互操作
- **Session：** 切流后使用 Go 签发的 `gitoauth_sessionid`（signed cookie）；切流瞬间进行中的 OAuth 需重开（可接受）
- **Bridge JWT：** 继续验证 task2app 签发的 HS256 start JWT（aud/iss/typ 不变）

---

## 5. 配置与运维

- 继续读 `conf/auth/git-oauth/`（`load_merged` 等价实现）
- runAll：`working_dir: taskGitOauth`，`./run.sh`，health `http://…:8002/api/health/`
- OpenAPI：`/api/schema/` 或等价 swagger 入口保持 Gateway 文档可达

---

## 6. 领域事件

| 意图 | 事件 | MQ |
|------|------|-----|
| OAuth 绑定成功 | `GIT_OAUTH_CREDENTIAL_BIND_ACTIVATED` | 可选投递（L2）；首期可结构化日志 + 进程内领域事件，与现 Python 行为对齐（Python 亦未强制 Kafka） |
| OAuth 绑定失败 | `GIT_OAUTH_CREDENTIAL_BIND_FAILED` | 同上 |
| access 换发 | `GIT_OAUTH_ACCESS_ISSUED`（审计表已是真源） | 审计表为准；不强制 MQ |

书面例外：纯查询（health、summary、user-ids）无事件。

---

## 7. 架构交付物

| 格式 | 路径 |
|------|------|
| PlantUML | `docs/architecture/v29-application-integration-20260715-1736-claude.puml` |
| ArchiMate | `docs/architecture/v29-application-integration-20260715-1736-claude.archimate` |
| Mermaid | `docs/architecture/v29-application-integration-20260715-1736-claude.mermaid.md` |

---

## 8. 风险与缓解

| 风险 | 缓解 |
|------|------|
| Fernet 不兼容导致无法换票 | 单测：Python 加密 → Go 解密往返 |
| Session 格式切换打断进行中 OAuth | 运维窗口短暂；前端错误码 `bad_state` 可重试 |
| 浏览器回调细节差异 | 契约测 + 迁移 Playwright 冒烟 |
| 删除 Python 后遗漏脚本 | runAll/CI/文档全文检索 `gitOauth` 清理清单 |

---

## 9. 非目标

- 不迁 Django accounts 的 authorize_url 桥 / connection 视图（仍 HTTP 调 gitOauth）
- 不合并 taskAuth OIDC
- 不改变公网 URI 前缀
