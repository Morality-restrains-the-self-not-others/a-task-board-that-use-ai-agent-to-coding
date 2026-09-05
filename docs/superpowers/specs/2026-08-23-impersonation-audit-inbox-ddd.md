# DDD：模拟登录通知与收信箱

- **日期**: 2026-08-23

## 限界上下文

- **Identity / Auth（taskAuth）**：ImpersonationSession、UserInboxMessage 同属身份上下文。不新建服务。

## 聚合

| 聚合 | 根 | 不变量 |
|------|----|--------|
| ImpersonationSession | id | 独立 token；reason 8～500 字；禁止自模拟/嵌套 |
| UserInboxMessage | id | recipient 不可改；kind=impersonation_notice 时必须有 session_id；同一 session 至多一封 |

## 领域事件

- UserImpersonationStarted（已有，可带 reason_len）
- UserInboxMessageCreated（新）

## 通用语言

- **模拟登录**：管理员以目标用户身份持有独立会话
- **收信箱**：用户只读自己的站内信
- **代登理由**：开始模拟时管理员填写、展示给被模拟用户，不进访问日志全文
