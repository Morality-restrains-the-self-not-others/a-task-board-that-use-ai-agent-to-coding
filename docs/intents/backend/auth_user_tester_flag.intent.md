# 功能意图：测试角色后端标志与网关头

- **日期**: 2026-08-23
- **状态**: 实施中
- **服务**: taskAuth

## 背景与目标

持久化 `is_tester`，在管理 PATCH/创建中可写，经 forward-auth 注入 `X-User-Is-Tester`，成功变更发布事件。

## 范围与边界

- 范围内：DDL、列表过滤、用户 JSON、forward-auth 缓存、事件。
- 范围外：平台 RBAC 角色名 `tester`。

## 验收标准

同 `system_admin_user_tester_role.intent.md` 后端条目。

## 业务意图 → 事件对照

| 业务意图 | 事件名 | MQ | 发布点 | 消费者 |
|----------|--------|----|--------|--------|
| 设置或取消测试角色 | UserTesterFlagChanged | user-tester-flag-changed | patchUserAsAdmin / 创建用户 | 无自动消费者 |

## 变更记录

| 日期 | 变更 |
|------|------|
| 2026-08-23 | 初稿 |
