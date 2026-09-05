# 价值流：模拟登录理由与收信箱通知

- **日期**: 2026-08-23

```
管理员打开用户列表
  → 点击「以该用户身份登录」
  → 弹窗填写理由（取消则中止）
  → POST impersonate + Idempotency-Key + reason
  → taskAuth 校验权限/理由并写会话
  → 同事务写收信箱信件
  → 发布 UserImpersonationStarted / UserInboxMessageCreated
  → 网关后续请求带 impersonator 头
  → 访问日志带审计字段
  → 被模拟用户打开账号中心收信箱看到理由
```

测试点：T-reason-modal、T-reason-required、T-inbox-written、T-inbox-idor、T-log-fields、T-header-session。
