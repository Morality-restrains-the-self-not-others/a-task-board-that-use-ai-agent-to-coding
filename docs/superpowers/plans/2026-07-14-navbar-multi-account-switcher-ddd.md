# DDD 领域建模笔记：多账号切换

**日期**: 2026-07-14  
**上下文**: 用户与认证（taskAuth）

## Bounded Context

- **Auth Session Context**（taskAuth + 前端会话层）

## 模型

| 类型 | 名称 | 说明 |
|------|------|------|
| Entity | User | 已有；id string |
| Value Object | AccountSlot | userId, username, avatarUrl, token, addedAt |
| Aggregate (客户端) | SavedAccountList | 槽集合，上限 5，唯一 userId |
| Domain Service | ActivateSession | 校验 token → enrich → 返回 ActiveAccount |
| Repository | TokenStore (auth.db) | 已有 accounts_customtoken |
| Domain Event | （本期无跨服务事件） | 前端刷新即可 |

## 不变量

1. 任一时刻仅一个 ActiveAccount（cookie userId ≡ authToken 所属 user）
2. SavedAccountList.size ≤ 5
3. ActivateSession 不得使无效 token 成为 ActiveAccount

## 落点

- Go: `handleActivateSession` in taskAuth
- JS: `saved_accounts_store.js`, `activate_session_service.js`
