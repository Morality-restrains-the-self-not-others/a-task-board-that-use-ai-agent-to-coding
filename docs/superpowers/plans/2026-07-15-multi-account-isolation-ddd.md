# DDD：多账号切换私有资源隔离

**日期**: 2026-07-15  
**限界上下文**: 用户与认证（taskAuth + Vue 会话 + Django DRF 鉴权）

## 模型要点

| 概念 | 类型 | 说明 |
|------|------|------|
| ActiveAccount | 值对象（客户端） | `userId` + `authToken`；切换后不得残留他人 `sessionid` |
| AccountSlot | 值对象集合 | 本机多账号槽（不变） |
| AuthenticationPrecedence | 策略 | Token > Session（与网关一致） |
| UserSelfResource | 资源边界 | path `user_id` 必须等于认证主体 |

## 领域事件

本期无新业务事件（会话边界修复例外，书面记录于意图文档）。

## 架构变更影响

- Application：Django DEFAULT 与显式 authentication_classes 顺序调整
- 前端：activate / resolveSwitchHref 行为变更
- 产物：`docs/architecture/v28-*`
