# 邮件邀请退订（Unsubscribe）— 设计文档

- **日期**: 2026-08-29
- **状态**: accepted（/goal 自动采用）
- **迭代**: email-invite-unsubscribe
- **作者**: cursor

## 背景

用户收到平台发出的**邀请邮件**后无法拒绝后续同类邮件。产品要求：

1. 邀请邮件中提供 **退订** 按钮/链接。
2. 点击后该邮箱进入 **退订列表**。
3. 此后任何人再对该邮箱发起**邮件邀请**时：**不发送邮件**，邀请链接仍生成，并提示邀请人「请手动复制邀请链接给对方」。

## 成功标准

1. 租户邀请邮件（`invitation`）与平台注册邀请邮件（`email_registration_invite`）HTML/纯文本均含退订链接；SMTP 带 RFC 8058 `List-Unsubscribe` + `List-Unsubscribe-Post: List-Unsubscribe=One-Click`。
2. 点击链接（GET 落地页确认 / POST one-click）将规范化邮箱写入 `auth_email_unsubscription`（taskAuth 唯一 owner）。
3. 前端 `/auth/unsubscribe/` 展示确认结果（无需登录）。
4. 租户 `POST .../members/invite/`（`invite_method=email`）与超管 `POST /api/system-admin/email-invitations/`（含 resend）：若目标邮箱已退订 → HTTP 201/200，`email_skipped=true`，`code=email_unsubscribed`，返回 `invite_token` + `invitation_url`，文案「该邮箱已退订邮件邀请，请手动复制邀请链接给对方」；**不**调用 SMTP。
5. taskFE「邀请人」邮箱方式和系统管理邀请弹窗：出现可复制链接的提示，不假装邮件已发出。
6. 验证码 / 密码重置 / 激活邮件 **不** 加退订、**不** 被退订列表拦截。
7. 领域事件 `EMAIL_UNSUBSCRIBED` 在成功写入退订表后发布。

## 方案（选定）

**Owner**：taskAuth 表 `auth_email_unsubscription`（`auth_` 前缀）。他服务经内部 API 查询，禁止直连。

**Token**：HMAC-SHA256，格式 `v1.{base64url(email)}.{base64url(mac)}`，密钥 `conf/auth/task-auth/config.yaml` 的 `unsubscribeHmacSecret`，空则回退 `SSOJwtSecret`，带域分隔前缀 `email-unsub-v1|`。无新密钥字面量。

**公开 API**（无登录）：

| 方法 | 路径 | 行为 |
|------|------|------|
| GET | `/api/public/email-unsubscribe/?token=` | 校验 token → upsert 退订 → 302 `/auth/unsubscribe/?ok=1` |
| POST | `/api/public/email-unsubscribe/` | body `token=` 或 RFC 8058 `List-Unsubscribe=One-Click` + query token；JSON `{ok:true}` |

**内部 API**（`X-TaskAuth-Internal-Secret`）：

| 方法 | 路径 | 行为 |
|------|------|------|
| GET | `/api/internal/email-unsubscription/?email=` | `{unsubscribed:bool, unsubscribe_url:string}` |

**邀请跳过邮件**：taskAuth 超管邀请在 `publishEmailSent` 前查询本表；taskTenantService 在发 `INVITATION_CREATED` 前调内部 API；taskEvents 投递邀请邮件再查一次（纵深）。已退订：`delivery_status=skipped_unsubscribed`，不 SMTP。

**退订范围**：仅拦截 **邀请类** 邮件。不提供本期重新订阅。

## 拒绝方案

| 方案 | 原因 |
|------|------|
| 退订表放 taskTenant | 超管注册邀请也在 taskAuth；邮箱身份属 auth |
| 无链接、只靠 List-Unsubscribe 头 | 部分客户端不展示头；产品要求按钮 |
| 退订即拒绝创建邀请 | 产品要求仍可复制链接 |
| 拦截验证码/重置邮件 | 安全邮件不可退订 |

## 🕸️ Code Review Graph 分析

`code-review-graph update --brief`：增量无邀请符号命中。代码理解回退 Grep：`handleCreateEmailInvitation` / `handleInvite` / `LocalDelivery.handleInvitationCreated` / 模板 `invitation.html` + `email_registration_invite.html` / `PeopleInvite.vue` / `SystemAdminEmailInvitationsSection.vue`。MCP `codegraph_explore` 不可用。

## 🏛️ 架构变更影响

**需要更新架构文件（v116）**：新表、公开退订 API、内部查询、事件 `EMAIL_UNSUBSCRIBED`。

| 格式 | 路径 |
|------|------|
| PlantUML | `docs/architecture/v116-application-integration-20260829-0940-cursor.puml` |
| | `docs/architecture/v116-enterprise-landscape-20260829-0940-cursor.puml` |
| ArchiMate 增量 | 同名 `.diff.archimate` |
| ArchiMate 全量 | 同名 `.full.archimate` |
| Mermaid | 同名 `.mermaid.md` |

## 业务意图 → 事件对照

| 业务意图 | 事件名 | Topic | 发布点 | 消费者 |
|---------|--------|-------|--------|--------|
| 用户退订邀请邮件 | EMAIL_UNSUBSCRIBED | email-unsubscribed | taskAuth 公开退订 handler | taskEvents observability（无自动再订阅） |
| 创建邀请（未退订） | INVITATION_CREATED / EMAIL_SENT | 既有 | 不变 | 既有发送 + 嵌入 unsubscribe_url |
| 创建邀请（已退订） | INVITATION_CREATED（payload `email_skipped`） | invitation-created | taskTenant | 消费者跳过 SMTP |

## Python 新增接口

不触发（全部 Go）。

## 冷热 / 分片

`auth_email_unsubscription` 为邮箱集合（非时间流水）。年增量远低于 100 万。主键 Snowflake，`UNIQUE(email)`。无需分区。
