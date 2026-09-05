# 邮箱邀请系统增强设计

- **状态**: 🎯 设计阶段（待审批）
- **作者**: claude
- **日期**: 2026-08-03
- **页面**: https://www.daydaymoney.com/system-admin/users/

## 问题分析

当前 `auth_email_registration_invite` 表已有 `inviter_user_id` 字段记录邀请人，但：
1. **前端列表不展示邀请人姓名** — 只显示了邮箱、状态、投递状态、发送次数、过期时间
2. **无法记录邀请原因** — 管理员发起邀请的业务上下文丢失
3. **无法设置账号有效期** — 邀请过期(7天)和账号有效期是两个独立概念，目前混为一谈
4. **无法预设用户身份** — 邀请时不能指定被邀请人的系统角色（超管/员工/租户/普通用户）
5. **邀请表单信息不足** — 只有一个邮箱输入框

## 需求定义

| # | 需求 | 优先级 |
|---|------|--------|
| 1 | 邀请列表展示邀请人姓名（根据已有 `inviter_user_id` 查 `auth_user`） | P0 |
| 2 | 新增 `invite_reason` 字段，邀请时填写原因 | P1 |
| 3 | 新增 `account_expires_at` 字段，设定受邀账号的有效期（NULL=永久） | P1 |
| 4 | 新增 `assigned_role` 字段，邀请时预设用户身份 | P1 |
| 5 | 邀请弹窗增加角色选择、原因、账号有效期字段 | P1 |
| 6 | 权限控制：仅系统管理员（super admin）可邀请（**后端已实现**，前端无需额外改动） | P0 |

## 数据库变更

### DDL: `dataMigrate/taskAuth/019_email_invite_enhancement.sql`（新建）

```sql
-- 邮箱邀请增强：邀请原因、账号有效期、指定角色
-- 使用幂等存储过程 guarded_add_column 防止重复执行
DROP PROCEDURE IF EXISTS guarded_add_column_019;
CREATE PROCEDURE guarded_add_column_019(IN tbl VARCHAR(64), IN col VARCHAR(64), IN col_def TEXT)
BEGIN
  IF NOT EXISTS (
    SELECT 1 FROM information_schema.COLUMNS
    WHERE TABLE_SCHEMA = DATABASE() AND TABLE_NAME = tbl AND COLUMN_NAME = col
  ) THEN
    SET @ddl = CONCAT('ALTER TABLE `', tbl, '` ADD COLUMN `', col, '` ', col_def);
    PREPARE stmt FROM @ddl;
    EXECUTE stmt;
    DEALLOCATE PREPARE stmt;
  END IF;
END;

CALL guarded_add_column_019('auth_email_registration_invite', 'invite_reason',
  'varchar(500) DEFAULT NULL COMMENT ''邀请原因''');
CALL guarded_add_column_019('auth_email_registration_invite', 'account_expires_at',
  'datetime DEFAULT NULL COMMENT ''受邀账号有效期（NULL=永久）''');
CALL guarded_add_column_019('auth_email_registration_invite', 'assigned_role',
  'varchar(50) DEFAULT NULL COMMENT ''预设角色: superuser/staff/tenant/member(NULL=默认member)''');

DROP PROCEDURE IF EXISTS guarded_add_column_019;
```

### 字段说明

| 字段 | 类型 | 默认值 | 说明 |
|------|------|--------|------|
| `invite_reason` | VARCHAR(500) | NULL | 管理员填写的邀请原因，自由文本 |
| `account_expires_at` | DATETIME | NULL | 受邀账号的过期时间，NULL表示永久有效 |
| `assigned_role` | VARCHAR(50) | NULL | 预设角色：`superuser`/`staff`/`tenant`/`member`，NULL表示默认 member |

> **设计决策**：`account_expires_at` 与现有 `expires_at`（邀请链接有效期7天）是独立概念：
> - `expires_at` = 邀请链接有效期（点击注册的窗口期）
> - `account_expires_at` = 注册成功后账号的有效期（NULL=永久）

## 后端变更 (taskAuth Go)

### 1. 更新 `emailInviteRow` 结构体

```go
type emailInviteRow struct {
    // ... 已有字段 ...
    InviteReason     sql.NullString  // 新增
    AccountExpiresAt sql.NullString  // 新增
    AssignedRole     sql.NullString  // 新增
}
```

### 2. 修改 `handleCreateEmailInvitation` — 接受新字段

请求体新增可选字段：
```json
{
  "email": "user@example.com",
  "invite_reason": "新员工入职 - 前端开发",
  "account_expires_at": "2027-08-03T00:00:00Z",
  "assigned_role": "member"
}
```

INSERT 语句更新：
```sql
INSERT INTO auth_email_registration_invite
(id, email, token, inviter_user_id, status, expires_at, created_at,
 email_sent_at, email_send_attempts, delivery_status,
 invite_reason, account_expires_at, assigned_role)
VALUES (?, ?, ?, ?, 'pending', ?, ?, ?, 1, 'pending', ?, ?, ?)
```

### 3. 修改 `handleListEmailInvitations` — 返回新字段 + 邀请人姓名

查询 JOIN `auth_user` 获取邀请人信息：
```sql
SELECT i.id, i.email, i.token, i.inviter_user_id, i.status, i.expires_at,
       COALESCE(i.accepted_user_id, ''),
       COALESCE(i.email_sent_at, ''),
       COALESCE(i.email_send_attempts, 0),
       COALESCE(i.delivery_status, 'pending'),
       COALESCE(i.delivery_error, ''),
       COALESCE(i.invite_reason, ''),
       COALESCE(i.account_expires_at, ''),
       COALESCE(i.assigned_role, ''),
       COALESCE(u.username, '') AS inviter_name
FROM auth_email_registration_invite i
LEFT JOIN auth_user u ON u.id = i.inviter_user_id
ORDER BY i.created_at DESC LIMIT 100
```

响应新增字段：
```json
{
  "invitations": [{
    "id": "...",
    "email": "...",
    "status": "pending",
    "inviterUserId": "...",
    "inviterName": "张三",
    "inviteReason": "新员工入职",
    "assignedRole": "member",
    "accountExpiresAt": "2027-08-03T00:00:00Z",
    "expiresAt": "...",
    "deliveryStatus": "queued",
    ...
  }]
}
```

### 4. 修改 `handleResendEmailInvitation` — 传递新字段到邮件模板

邮件模板中增加邀请原因和账号有效期信息（可选展示）。

### 路由（无变更）

已有路由保持不变：
- `POST /api/system-admin/email-invitations/` — 创建（新增字段）
- `GET /api/system-admin/email-invitations/` — 列表（新增字段+邀请人）
- `POST /api/system-admin/email-invitations/resend/` — 重发
- `DELETE /api/system-admin/email-invitations/{id}/` — 取消
- `POST /api/system-admin/email-invitations/bulk-resend/` — 批量重发

## 前端变更 (taskFE Vue)

### 1. 邀请弹窗增强 (`SystemAdminUsers.vue`)

在现有邮箱邀请模态框中增加：

```
┌──────────────────────────────────────┐
│  发送邮箱注册邀请                    │
│                                      │
│  邮箱地址 *                          │
│  ┌─────────────────────────────────┐ │
│  │ user@example.com                │ │
│  └─────────────────────────────────┘ │
│                                      │
│  用户身份 *                          │
│  ┌─────────────────────────────────┐ │
│  │ 普通用户 ▼                      │ │
│  │  普通用户 (member)              │ │
│  │  员工 (staff)                   │ │
│  │  租户 (tenant)                  │ │
│  │  超级管理员 (superuser)         │ │
│  └─────────────────────────────────┘ │
│                                      │
│  账号有效期                          │
│  ┌─────────────────────────────────┐ │
│  │ 📅 2027-08-03                   │ │  (date picker)
│  │ ☐ 永久有效                      │ │
│  └─────────────────────────────────┘ │
│                                      │
│  邀请原因                            │
│  ┌─────────────────────────────────┐ │
│  │ 新员工入职 - 前端开发           │ │
│  └─────────────────────────────────┘ │
│                                      │
│  邀请链接有效期7天                   │
│                                      │
│  [取消]  [发送邀请]                  │
└──────────────────────────────────────┘
```

### 2. 邀请列表表格增强

现有列：邮箱 | 状态 | 投递状态 | 发送次数 | 最后发送 | 过期时间 | 操作

新增列：
```
邮箱 | 邀请人 | 预设角色 | 账号有效期 | 邀请原因 | 状态 | 投递状态 | 发送次数 | 最后发送 | 过期时间 | 操作
```

变更摘要：
| 新增列 | 数据来源 | 说明 |
|--------|---------|------|
| 邀请人 | `inviterName` (后端 JOIN 返回) | 显示邀请人用户名 |
| 预设角色 | `assignedRole` | `superuser`→超管, `staff`→员工, `tenant`→租户, `member`→普通用户 |
| 账号有效期 | `accountExpiresAt` | 格式化显示日期，NULL显示"永久" |
| 邀请原因 | `inviteReason` | 截断显示（过长用 tooltip） |

### 3. 安全确认（无需变更）

- 所有后端端点已校验 `isSuperAdminUser()` → 非管理员返回 403 `"仅管理员可发送邀请"`
- 前端"发送邮箱邀请"按钮已在 SystemAdmin 页面内（仅超管可访问此路由）

## 业务意图 → 事件对照

| 业务意图 | 事件名（过去式） | 发布点 | 消费者/副作用 | 例外理由 |
|---------|----------------|--------|--------------|---------|
| 管理员创建邮箱邀请 | `EmailInvitationCreated` | `handleCreateEmailInvitation` → Kafka **新增** | `taskEvents` 发邮件 | — |
| 管理员补发邀请 | `EmailInvitationResent` | `handleResendEmailInvitation` → Kafka **新增** | `taskEvents` 发邮件 | — |
| 管理员取消邀请 | `EmailInvitationCancelled` | `handleCancelEmailInvitation` → Kafka **新增** | 审计日志 | — |
| 用户通过邀请注册 | — | **已有** `handleEmailRegister` | 已在现有流程处理 | — |

> ⚠️ **注意**：当前代码在 `handleCreateEmailInvitation` 中直接调用 `publishEmailSent()`（Kafka→SMTP 发送），并未通过独立的领域事件总线。上述事件为本次设计建议新增的结构化事件，便于后续审计和扩展。如果本次迭代范围受限，可暂不实施事件拆分，保持现有 `publishEmailSent` 调用方式。

## Python 服务新增接口评估

**不触发**。本次变更均在 Go 服务 `taskAuth` 中完成，不涉及 Python/Django 新接口。

## 架构变更影响

- **迭代版本**: v58 🎯 target
- **迭代名称**: email-invite-enhancement
- **变更级别**: 组件修改（taskAuth）
- **变更明细**:
  - 🟡 [MODIFIED] taskAuth — 新增 DB 列 + API 字段 + 邀请事件
  - 无新增/删除服务组件
  - 无数据流变更

## Value Stream 影响

| 维度 | 影响 |
|------|------|
| 已有流 | `invitation-email` 步骤（目前引用 Django `auth_invitation`）需更新测试覆盖 Go 邀请字段 |
| 新流 | 无 — 这是对已有流程的增强 |
| 字段影响 | `taskAuth.auth_email_registration_invite.invite_reason/account_expires_at/assigned_role`（新增3字段） |
| 测试影响 | `auth_email_invite_test.go` 需补充新字段测试；前端需添加 Playwright E2E 测试 |

## 文件清单

### 新建文件
| 文件 | 说明 |
|------|------|
| `dataMigrate/taskAuth/019_email_invite_enhancement.sql` | 添加3个新列的幂等迁移 |

### 修改文件
| 文件 | 变更 |
|------|------|
| `taskAuth/src/auth_email_invite.go` | 结构体+创建+列表+邮件模板更新 |
| `taskAuth/src/events.go` | 可选：新增领域事件定义 |
| `taskFE/app/src/views/SystemAdminUsers.vue` | 邀请弹窗+列表表格增强 |

### 测试文件
| 文件 | 说明 |
|------|------|
| `taskAuth/src/auth_email_invite_test.go` | 新字段覆盖 |
| `taskFE/tests/SystemAdminUsers.email-invite-enhancement.playwright.test.js` | E2E 测试 |
