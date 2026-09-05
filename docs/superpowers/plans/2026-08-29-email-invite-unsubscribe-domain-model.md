# 邮件邀请退订 — 领域模型

- **日期**: 2026-08-29
- **限界上下文**: Identity / Notifications（taskAuth 持有退订；邀请发送方查询端口）

## 聚合

**EmailUnsubscription**（taskAuth）

- 身份：规范化小写 email（Value Object `InviteEmail`）
- 属性：id (Snowflake)、source、created_at
- 不变量：同一 email 至多一行；source 非空

## 领域服务

- `NormalizeInviteEmail` — trim + lower；须含 `@`
- `SignUnsubscribeToken` / `ParseUnsubscribeToken`
- `RecordUnsubscription` — upsert + 若新插入则领域事件

## 端口

- `UnsubscriptionRepository`：GetByEmail、Upsert
- `UnsubscriptionChecker`（taskTenant / taskEvents）：`IsUnsubscribed(ctx, email) (bool, unsubscribeURL, error)`

## 事件

```
EMAIL_UNSUBSCRIBED {
  email,           // 规范化，作 Kafka key
  source,          // invite_email | list_unsubscribe_post
  unsubscribed_at
}
```

幂等键 = 规范化 email。

## 邀请发送策略

`InvitationDeliveryPolicy.ShouldSendEmail(unsubscribed bool)` → false 时仍创建邀请，delivery=`skipped_unsubscribed`。
